package fx

import (
	"bytes"
	"encoding/gob"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// Output is one processed content entry. A leaf carries Data; a directory
// carries Inputs — its files as nested Values, in sort.txt order. Name is
// server-internal (sort.txt ordering and Save) and is never sent on the
// wire. Turning this tree into the wire format is one's job, not fx's — fx
// only sources and processes the bytes.
type Output struct {
	Name   string
	Type   string
	Data   []byte
	Inputs []*Output
}

// Save gob-encodes v (Name, Type, Data, Inputs — the full tree) for the
// caller to persist wherever it chooses. This is deliberately decoupled from
// the wire format (one.Encode), so wire format changes never invalidate
// anything already saved.
func (v *Output) Save() ([]byte, error) {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// toBytes reads input's raw bytes — a local file or an http(s) URL — so
// custom content can be sourced from S3 exactly like a local file. input
// must name a file, not a directory; use Input when input may be either.
func (f *Fx) toBytes(input string) ([]byte, error) {
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

// Input is the universal means of sourcing data into a *Output — a local
// file, a local directory, or an http(s) URL, so custom content can be
// sourced from S3 exactly like a local file. A directory's files are read
// via walk into Inputs, so the result is always exactly one Output
// regardless of source. Name is inferred from the base name and Type from
// the extension, falling back to content detection.
func (f *Fx) Input(input string) (*Output, error) {
	if !strings.HasPrefix(input, "http://") && !strings.HasPrefix(input, "https://") {
		if info, err := os.Stat(input); err == nil && info.IsDir() {
			return f.walk(input), nil
		}
	}

	raw, err := f.toBytes(input)
	if err != nil {
		return nil, err
	}
	return f.leaf(input, raw), nil
}

// leaf builds a leaf Output from input's base name and raw content: Type from
// the extension, falling back to content detection.
func (f *Fx) leaf(input string, raw []byte) *Output {
	base := filepath.Base(input)
	ext := filepath.Ext(base)
	ct := mime.TypeByExtension(ext)
	if ct == "" {
		ct = http.DetectContentType(raw)
	}
	return &Output{Name: strings.TrimSuffix(base, ext), Type: ct, Data: raw}
}

// walk builds the bundle Output for a directory: its files as nested Values,
// in sort.txt order if present (files not listed are appended after).
func (f *Fx) walk(path string) *Output {
	var values []*Output
	filepath.WalkDir(path, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || filepath.Base(p) == "sort.txt" {
			return err
		}
		if v, err := f.Input(p); err == nil {
			values = append(values, v)
		}
		return nil
	})

	if raw, err := os.ReadFile(filepath.Join(path, "sort.txt")); err == nil {
		byName := make(map[string]*Output, len(values))
		for _, v := range values {
			byName[v.Name] = v
		}
		out := make([]*Output, 0, len(values))
		for name := range strings.SplitSeq(strings.TrimSpace(string(raw)), "\n") {
			if v, ok := byName[strings.TrimSpace(name)]; ok {
				out = append(out, v)
				delete(byName, v.Name)
			}
		}
		for _, v := range values {
			if _, ok := byName[v.Name]; ok {
				out = append(out, v)
			}
		}
		values = out
	}

	return &Output{
		Name:   filepath.Base(path),
		Type:   "application/x-bundle",
		Inputs: values,
	}
}
