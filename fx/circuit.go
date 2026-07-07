package fx

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type Circuit struct {
	Routes map[string]*Value
}

func NewCircuit() *Circuit {
	return &Circuit{Routes: make(map[string]*Value)}
}

// Value is one encoded entry, or a bundle of them. A leaf carries Type + Data;
// a bundle carries Values (its children). Name is server-internal — used for
// sort.txt ordering — and is never written to the wire (Encode/Decode), though
// it is preserved when persisted via Save/Load.
type Value struct {
	Name   string
	Type   string
	Data   []byte
	Values []*Value
}

// Encode flattens a Value tree to its leaves in order and writes the wire
// format the client decodes. This is purely the over-the-network format —
// Save/Load use gob instead, so changing this format never requires
// re-saving or re-uploading anything already persisted. Layout:
//
//	[1B typeLen][type] [4B dataLen][data]
//
// There is no leaf count on the wire — the response's length is the
// terminator. Names are not encoded — order is the contract.
func (c *Circuit) Encode(v *Value) []byte {
	var leaves []*Value
	var table []string
	index := make(map[string]byte, 8)
	tableSize, dataSize := 1, 0

	var walk func(*Value)
	walk = func(n *Value) {
		if len(n.Values) > 0 {
			for _, ch := range n.Values {
				walk(ch)
			}
			return
		}
		leaves = append(leaves, n)
		if _, ok := index[n.Type]; !ok {
			index[n.Type] = byte(len(table))
			table = append(table, n.Type)
			tableSize += 1 + len(n.Type)
		}
		dataSize += 1 + 4 + len(n.Data)
	}
	walk(v)

	out := make([]byte, tableSize+dataSize)
	out[0] = byte(len(table))
	pos := 1
	for _, t := range table {
		out[pos] = byte(len(t))
		pos++
		pos += copy(out[pos:], t)
	}
	for _, l := range leaves {
		out[pos] = index[l.Type]
		pos++
		binary.BigEndian.PutUint32(out[pos:], uint32(len(l.Data)))
		pos += 4
		pos += copy(out[pos:], l.Data)
	}
	return out
}

// Decode is the inverse of Encode.
func (c *Circuit) Decode(buf []byte) []*Value {
	tableCount := int(buf[0])
	pos := 1
	table := make([]string, tableCount)
	for i := range table {
		tl := int(buf[pos])
		pos++
		table[i] = string(buf[pos : pos+tl])
		pos += tl
	}
	var leaves []*Value
	for pos < len(buf) {
		idx := buf[pos]
		pos++
		n := int(binary.BigEndian.Uint32(buf[pos : pos+4]))
		pos += 4
		leaves = append(leaves, &Value{Type: table[idx], Data: buf[pos : pos+n]})
		pos += n
	}
	return leaves
}

// ToBytes fetches the content at input, either from an HTTP URL or a local file.
func (c *Circuit) ToBytes(input string) ([]byte, error) {
	if strings.HasPrefix(input, "http://") || strings.HasPrefix(input, "https://") {

		resp, err := http.Get(input)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		return io.ReadAll(resp.Body)
	}
	return os.ReadFile(input)
}

// Load reads a file or directory at path and stores it in Routes keyed by the
// base name of path. A file becomes a leaf Value; a directory becomes a
// bundle. An http(s) path is treated as a gob-encoded Value tree (written by
// Save) and decoded directly — independent of whatever Encode/Decode's wire
// format currently is.
func (c *Circuit) Load(path string) {
	key := filepath.Base(path)
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		if data, err := c.ToBytes(path); err == nil {
			var v Value
			if err := gob.NewDecoder(bytes.NewReader(data)).Decode(&v); err == nil {
				c.Routes[key] = &v
			}
		}
		return
	}
	info, err := os.Stat(path)
	if err != nil {
		return
	}
	if info.IsDir() {
		c.Routes[key] = c.walk(path)
	} else if v := c.read(path); v != nil {
		c.Routes[key] = v
	}
}

// read loads a single file into a leaf Value, inferring MIME type from the
// extension or content detection as a fallback.
func (c *Circuit) read(path string) *Value {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	ct := mime.TypeByExtension(ext)
	if ct == "" {
		ct = http.DetectContentType(raw)
	}
	return &Value{Name: base[:len(base)-len(ext)], Type: ct, Data: raw}
}

// walk reads all files in a directory into a bundle Value. If a sort.txt file
// is present, it defines the order by name; files not listed are appended after.
func (c *Circuit) walk(path string) *Value {
	bundle := &Value{Name: filepath.Base(path)}
	filepath.WalkDir(path, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || filepath.Base(p) == "sort.txt" {
			return err
		}
		if v := c.read(p); v != nil {
			bundle.Values = append(bundle.Values, v)
		}
		return nil
	})
	if raw, err := os.ReadFile(filepath.Join(path, "sort.txt")); err == nil {
		order := strings.Split(strings.TrimSpace(string(raw)), "\n")
		byName := make(map[string]int, len(bundle.Values))
		for i, v := range bundle.Values {
			byName[v.Name] = i
		}
		out := make([]*Value, 0, len(bundle.Values))
		seen := make(map[int]bool)
		for _, name := range order {
			if idx, ok := byName[strings.TrimSpace(name)]; ok && !seen[idx] {
				out = append(out, bundle.Values[idx])
				seen[idx] = true
			}
		}
		for i, v := range bundle.Values {
			if !seen[i] {
				out = append(out, v)
			}
		}
		bundle.Values = out
	}
	return bundle
}

// Save persists the Value tree at key (Name, Type, Data, Values — the full
// structure, not the wire format) to s3/key using gob, so changes to
// Encode/Decode never require re-saving or re-uploading anything already
// in S3. Encode is only ever applied fresh, at serve time, to whatever
// tree Load reconstructs.
func (c *Circuit) Save(key string) error {
	v, ok := c.Routes[key]
	if !ok {
		return fmt.Errorf("save: route %q not found", key)
	}
	if err := os.MkdirAll("s3", 0755); err != nil {
		return err
	}
	f, err := os.Create(filepath.Join("s3", key))
	if err != nil {
		return err
	}
	defer f.Close()
	return gob.NewEncoder(f).Encode(v)
}
