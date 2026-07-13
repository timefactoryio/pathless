package fx

import (
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
// preserved when persisted via Save/Load.
type Value struct {
	Name string
	Type string
	Data []byte
}

// Encode writes the wire format the client decodes: distinct types are
// written once, up front, as a small dictionary; each entry then costs
// just a 1-byte type id and a 4-byte length ahead of its data — no shared
// table beyond that, no repeated type strings. This is purely the
// over-the-network format — Save/Load use gob instead, so changing this
// format never requires re-saving or re-uploading anything already
// persisted. Layout:
//
//	[1B typeCount][typeCount x ([1B typeLen][type])]
//	[1B typeID][4B len(data)][data]...  (repeated, one per value)
//
// Names are not encoded — order is the contract.
func (f *Fx) Encode(values ...*Value) []byte {
	typeID := make(map[string]byte, 4)
	var types []string
	for _, v := range values {
		if _, ok := typeID[v.Type]; !ok {
			typeID[v.Type] = byte(len(types))
			types = append(types, v.Type)
		}
	}

	size := 1
	for _, t := range types {
		size += 1 + len(t)
	}
	for _, v := range values {
		size += 1 + 4 + len(v.Data)
	}

	out := make([]byte, size)
	pos := 0
	out[pos] = byte(len(types))
	pos++
	for _, t := range types {
		out[pos] = byte(len(t))
		pos++
		pos += copy(out[pos:], t)
	}
	for _, v := range values {
		out[pos] = typeID[v.Type]
		pos++
		binary.BigEndian.PutUint32(out[pos:], uint32(len(v.Data)))
		pos += 4
		pos += copy(out[pos:], v.Data)
	}
	return out
}

// Decode is the inverse of Encode.
func (f *Fx) Decode(buf []byte) []*Value {
	if len(buf) == 0 {
		return nil
	}
	pos := 0
	types := make([]string, buf[pos])
	pos++
	for i := range types {
		tl := int(buf[pos])
		pos++
		types[i] = string(buf[pos : pos+tl])
		pos += tl
	}

	var values []*Value
	for pos < len(buf) {
		typ := types[buf[pos]]
		pos++
		n := int(binary.BigEndian.Uint32(buf[pos:]))
		pos += 4
		values = append(values, &Value{Type: typ, Data: buf[pos : pos+n]})
		pos += n
	}
	return values
}

// ToValue builds the single *Value for input — a local file, a local
// directory, or an http(s) URL, so custom content can be sourced from S3
// exactly like a local file. A directory's files are read via walk and
// pre-encoded into one blob, so the result is always exactly one Value
// regardless of source. Name is inferred from the base name and Type from
// the extension, falling back to content detection.
func (f *Fx) ToValue(input string) (*Value, error) {
	remote := strings.HasPrefix(input, "http://") || strings.HasPrefix(input, "https://")

	if !remote {
		if info, err := os.Stat(input); err == nil && info.IsDir() {
			return &Value{
				Name: filepath.Base(input),
				Type: "application/octet-stream",
				Data: f.Encode(f.walk(input)...),
			}, nil
		}
	}

	var raw []byte
	var err error
	if remote {
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
	return &Value{Name: strings.TrimSuffix(base, ext), Type: ct, Data: raw}, nil
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
		if v, err := f.ToValue(p); err == nil {
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

// Save persists the *Value at key (Name, Type, Data — the full structure,
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
