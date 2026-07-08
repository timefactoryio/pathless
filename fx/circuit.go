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

// Value is one encoded entry. Name is server-internal — used for sort.txt
// ordering — and is never written to the wire (Encode/Decode), though it is
// preserved when persisted via Save/Load. Length is len(Type)+len(Data),
// set by Encode and populated by Decode.
type Value struct {
	Name   string
	Type   string
	Length int
	Data   []byte
}

// Encode writes the wire format the client decodes: a "railroad track" of
// self-contained entries, one after another — no shared table, no separate
// metadata section. Each entry carries its own type and length, so the
// client hops from one entry straight to the next purely by size. This is
// purely the over-the-network format — Save/Load use gob instead, so
// changing this format never requires re-saving or re-uploading anything
// already persisted. Layout:
//
//	[4B n][1B typeLen][type][data]...  (repeated, one per value)
//	n = 1 + typeLen + len(data)
//
// Names are not encoded — order is the contract.
func (f *Fx) Encode(values []*Value) []byte {
	size := 0
	for _, v := range values {
		v.Length = len(v.Type) + len(v.Data)
		size += 4 + 1 + v.Length
	}

	out := make([]byte, size)
	pos := 0
	for _, v := range values {
		binary.BigEndian.PutUint32(out[pos:], uint32(1+v.Length))
		pos += 4
		out[pos] = byte(len(v.Type))
		pos++
		pos += copy(out[pos:], v.Type)
		pos += copy(out[pos:], v.Data)
	}
	return out
}

// Decode is the inverse of Encode.
func (f *Fx) Decode(buf []byte) []*Value {
	var values []*Value
	pos := 0
	for pos < len(buf) {
		n := int(binary.BigEndian.Uint32(buf[pos:]))
		pos += 4
		tl := int(buf[pos])
		pos++
		typ := string(buf[pos : pos+tl])
		pos += tl
		dataLen := n - 1 - tl
		values = append(values, &Value{Type: typ, Length: n - 1, Data: buf[pos : pos+dataLen]})
		pos += dataLen
	}
	return values
}

// ToBytes fetches the content at input — a local file path or an http(s)
// URL, so custom content can be sourced from S3 exactly like a local file
// — and returns it as a leaf Value. Name is inferred from the base name
// and Type from the extension, falling back to content detection.
func (f *Fx) ToBytes(input string) (*Value, error) {
	var raw []byte
	var err error
	if strings.HasPrefix(input, "http://") || strings.HasPrefix(input, "https://") {
		resp, e := http.Get(input)
		if e != nil {
			return nil, e
		}
		defer resp.Body.Close()
		raw, err = io.ReadAll(resp.Body)
	} else {
		raw, err = os.ReadFile(input)
	}
	if err != nil {
		return nil, err
	}
	base := filepath.Base(input)
	ext := filepath.Ext(base)
	ct := mime.TypeByExtension(ext)
	if ct == "" {
		ct = http.DetectContentType(raw)
	}
	return &Value{Name: base[:len(base)-len(ext)], Type: ct, Data: raw}, nil
}

// Load reads a file or directory at path and stores it in Routes keyed by the
// base name of path. A file becomes a single-entry route; a directory
// becomes a multi-entry route. An http(s) path is treated as a gob-encoded
// []*Value (written by Save) and decoded directly — independent of whatever
// Encode/Decode's wire format currently is.
func (f *Fx) Load(path string) {
	key := filepath.Base(path)
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		if b, err := f.ToBytes(path); err == nil {
			var v []*Value
			if err := gob.NewDecoder(bytes.NewReader(b.Data)).Decode(&v); err == nil {
				f.Routes[key] = v
			}
		}
		return
	}
	info, err := os.Stat(path)
	if err != nil {
		return
	}
	if info.IsDir() {
		f.Routes[key] = f.walk(path)
	} else if v, err := f.ToBytes(path); err == nil {
		f.Routes[key] = []*Value{v}
	}
}

// walk reads all files in a directory into a flat slice of Values. If a
// sort.txt file is present, it defines the order by name; files not listed
// are appended after.
func (f *Fx) walk(path string) []*Value {
	var values []*Value
	filepath.WalkDir(path, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || filepath.Base(p) == "sort.txt" {
			return err
		}
		if v, err := f.ToBytes(p); err == nil {
			values = append(values, v)
		}
		return nil
	})
	if raw, err := os.ReadFile(filepath.Join(path, "sort.txt")); err == nil {
		order := strings.Split(strings.TrimSpace(string(raw)), "\n")
		byName := make(map[string]int, len(values))
		for i, v := range values {
			byName[v.Name] = i
		}
		out := make([]*Value, 0, len(values))
		seen := make(map[int]bool)
		for _, name := range order {
			if idx, ok := byName[strings.TrimSpace(name)]; ok && !seen[idx] {
				out = append(out, values[idx])
				seen[idx] = true
			}
		}
		for i, v := range values {
			if !seen[i] {
				out = append(out, v)
			}
		}
		values = out
	}
	return values
}

// Save persists the []*Value at key (Name, Type, Data — the full structure,
// not the wire format) to s3/key using gob, so changes to Encode/Decode
// never require re-saving or re-uploading anything already in S3. Encode is
// only ever applied fresh, at serve time, to whatever Load reconstructs.
func (f *Fx) Save(key string) error {
	v, ok := f.Routes[key]
	if !ok {
		return fmt.Errorf("save: route %q not found", key)
	}
	if err := os.MkdirAll("s3", 0755); err != nil {
		return err
	}
	data, err := os.Create(filepath.Join("s3", key))
	if err != nil {
		return err
	}
	defer data.Close()
	return gob.NewEncoder(data).Encode(v)
}
