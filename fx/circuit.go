package fx

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
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
// sort.txt ordering — and is never written to the wire.
type Value struct {
	Name   string
	Type   string
	Data   []byte
	Values []*Value
}

// Encode flattens a Value tree to its leaves in order and writes the wire
// format the client decodes:
//
//	[2B count]  then per leaf: [1B typeLen][type] [4B dataLen][data]
//
// Names are not encoded — order is the contract.
func (c *Circuit) Encode(v *Value) []byte {
	var leaves []*Value
	var walk func(*Value)
	walk = func(n *Value) {
		if len(n.Values) > 0 {
			for _, ch := range n.Values {
				walk(ch)
			}
			return
		}
		leaves = append(leaves, n)
	}
	walk(v)
	total := 0
	for _, l := range leaves {
		total += len(l.Type) + len(l.Data)
	}
	buf := bytes.NewBuffer(make([]byte, 0, 2+5*len(leaves)+total))
	buf.WriteByte(byte(len(leaves) >> 8))
	buf.WriteByte(byte(len(leaves)))
	var hdr [4]byte
	for _, l := range leaves {
		buf.WriteByte(byte(len(l.Type)))
		buf.WriteString(l.Type)
		binary.BigEndian.PutUint32(hdr[:], uint32(len(l.Data)))
		buf.Write(hdr[:])
		buf.Write(l.Data)
	}
	return buf.Bytes()
}

// Decode is the inverse of Encode: it parses a wire blob back into leaf
// Values (Type/Data only — Name isn't on the wire, so it's not restored).
// This lets the server read back what it once wrote, e.g. content pre-
// encoded via Save and fetched from S3.
func (c *Circuit) Decode(buf []byte) []*Value {
	count := int(buf[0])<<8 | int(buf[1])
	pos := 2
	leaves := make([]*Value, count)
	for i := range leaves {
		tl := int(buf[pos])
		pos++
		typ := string(buf[pos : pos+tl])
		pos += tl
		n := int(binary.BigEndian.Uint32(buf[pos : pos+4]))
		pos += 4
		leaves[i] = &Value{Type: typ, Data: buf[pos : pos+n]}
		pos += n
	}
	return leaves
}

// Compress gzip-compresses data at maximum compression.
// Bundles are compressed once at build time and served directly
// with Content-Encoding: gzip.
func (c *Circuit) Compress(data []byte) []byte {
	var buf bytes.Buffer
	w, _ := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	w.Write(data)
	w.Close()
	return buf.Bytes()
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
// bundle. An http(s) path is treated as a pre-encoded wire blob (from Save)
// and decoded back into a bundle.
func (c *Circuit) Load(path string) {
	key := filepath.Base(path)
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		if data, err := c.ToBytes(path); err == nil {
			c.Routes[key] = &Value{Name: key, Values: c.Decode(data)}
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

// Save writes the wire-encoded bytes of the route at key to s3/key, ready
// to be synced to S3 (e.g. rclone sync s3 remote:bucket) under the same
// object name Load expects when fetching it back.
func (c *Circuit) Save(key string) error {
	v, ok := c.Routes[key]
	if !ok {
		return fmt.Errorf("save: route %q not found", key)
	}
	if err := os.MkdirAll("s3", 0755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join("s3", key), c.Encode(v), 0644)
}
