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
func (f *Fx) Encode(values ...*Value) []byte {
	size := 0
	for _, v := range values {
		size += 4 + 1 + len(v.Type) + len(v.Data)
	}

	out := make([]byte, size)
	pos := 0
	for _, v := range values {
		n := 1 + len(v.Type) + len(v.Data)
		binary.BigEndian.PutUint32(out[pos:], uint32(n))
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
		values = append(values, &Value{Type: typ, Data: buf[pos : pos+dataLen]})
		pos += dataLen
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
