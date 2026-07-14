package fx

import (
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// Value is one processed content entry. A leaf carries Data; a directory
// carries Children — its files as nested Values, in sort.txt order. Name is
// server-internal (sort.txt ordering and Save) and is never sent on the
// wire. Turning this tree into the wire format is one's job, not fx's — fx
// only sources and processes the bytes.
type Value struct {
	Name     string
	Type     string
	Data     []byte
	Children []*Value
}

// ToValue builds the single *Value for input — a local file, a local
// directory, or an http(s) URL, so custom content can be sourced from S3
// exactly like a local file. A directory's files are read via walk into
// Children, so the result is always exactly one Value regardless of source.
// Name is inferred from the base name and Type from the extension, falling
// back to content detection.
func (f *Fx) ToValue(input string) (*Value, error) {
	remote := strings.HasPrefix(input, "http://") || strings.HasPrefix(input, "https://")

	if !remote {
		if info, err := os.Stat(input); err == nil && info.IsDir() {
			return &Value{
				Name:     filepath.Base(input),
				Type:     "application/octet-stream",
				Children: f.walk(input),
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
