package fx

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// Output is one processed content entry. Name never travels; Type and Data
// both do, encoded by Marshal.
type Output struct {
	Name string
	Type string
	Data []byte
}

// Save gob-encodes v (Name, Type, Data) for the caller to persist wherever
// it chooses. Deliberately decoupled from Marshal, so wire format changes
// never invalidate anything already saved.
func (v *Output) Save() ([]byte, error) {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// contentType resolves a leaf's MIME type from its file name, falling back
// to content sniffing when the extension is unknown or absent.
func contentType(name string, raw []byte) string {
	if ct := mime.TypeByExtension(filepath.Ext(name)); ct != "" {
		return ct
	}
	return http.DetectContentType(raw)
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
// file, a local directory, or an http(s) URL. A directory's files are
// collapsed via walk, so the result is always exactly one Output.
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
	base := filepath.Base(input)
	ext := filepath.Ext(base)
	return &Output{
		Name: strings.TrimSuffix(base, ext),
		Type: contentType(base, raw),
		Data: raw,
	}, nil
}

// list returns path's files as an ordered slice, honoring sort.txt — the
// raw ingredients, before any wire decision is made.
func (f *Fx) list(path string) []*Output {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil
	}

	var values []*Output
	for _, e := range entries {
		if e.Name() == "sort.txt" {
			continue
		}
		p := filepath.Join(path, e.Name())
		if e.IsDir() {
			values = append(values, f.walk(p))
			continue
		}
		if v, err := f.Input(p); err == nil {
			values = append(values, v)
		}
	}

	raw, err := os.ReadFile(filepath.Join(path, "sort.txt"))
	if err != nil {
		return values
	}
	byName := make(map[string]*Output, len(values))
	for _, v := range values {
		byName[v.Name] = v
	}
	ordered := make([]*Output, 0, len(values))
	for name := range strings.SplitSeq(strings.TrimSpace(string(raw)), "\n") {
		if v, ok := byName[strings.TrimSpace(name)]; ok {
			ordered = append(ordered, v)
			delete(byName, v.Name)
		}
	}
	for _, v := range values {
		if _, ok := byName[v.Name]; ok {
			ordered = append(ordered, v)
		}
	}
	return ordered
}

// walk collapses a directory into one Output, honoring Input's contract
// that any input yields exactly one.
func (f *Fx) walk(path string) *Output {
	out := f.Marshal(f.list(path)...)
	out.Name = filepath.Base(path)
	return out
}

// Marshal concatenates values into one Output's Data:
// [4B count][4B length x count][blob x count], each blob [1B typeLen][type][data].
// Type travels because a route's consumer (Logo, Slides) can't infer it from
// bytes alone; entries without one cost a single zero byte.
func (f *Fx) Marshal(values ...*Output) *Output {
	header := 4 + 4*len(values)
	total := header
	for _, v := range values {
		total += 1 + len(v.Type) + len(v.Data)
	}

	data := make([]byte, total)
	binary.BigEndian.PutUint32(data, uint32(len(values)))

	pos, body := 4, header
	for _, v := range values {
		binary.BigEndian.PutUint32(data[pos:], uint32(1+len(v.Type)+len(v.Data)))
		pos += 4
		data[body] = byte(len(v.Type))
		body++
		body += copy(data[body:], v.Type)
		body += copy(data[body:], v.Data)
	}
	return &Output{Data: data}
}
