package fx

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type Circuit struct {
	Routes map[string][]byte
}

func NewCircuit() *Circuit {
	return &Circuit{Routes: make(map[string][]byte)}
}

// Value is the unit of work for encoding. Name and Type populate the manifest;
// Data is the raw blob. Data is excluded from JSON and freed after encoding.
type Value struct {
	Name string `json:"name,omitempty"`
	Type string `json:"type,omitempty"`
	Size uint32 `json:"size"`
	Data []byte `json:"-"`
}

func (c *Circuit) Wire(values []*Value) []byte {
	totalData := 0
	for i, v := range values {
		values[i].Size = uint32(len(v.Data))
		totalData += len(v.Data)
	}
	buf := bytes.NewBuffer(make([]byte, 0, 2+6*len(values)+totalData))
	buf.WriteByte(byte(len(values) >> 8))
	buf.WriteByte(byte(len(values)))
	var hdr [4]byte
	for _, v := range values {
		buf.WriteByte(byte(len(v.Name)))
		buf.WriteString(v.Name)
		buf.WriteByte(byte(len(v.Type)))
		buf.WriteString(v.Type)
		binary.BigEndian.PutUint32(hdr[:], v.Size)
		buf.Write(hdr[:])
		buf.Write(v.Data)
	}
	return buf.Bytes()
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

// Load reads a file or directory at path, encodes and compresses the contents,
// and stores the bundle in Routes keyed by the base name of path.
// Raw data is freed from each Value after the bundle is stored.
func (c *Circuit) Load(path string) {
	key := filepath.Base(path)
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		if data, err := c.ToBytes(path); err == nil {
			c.Routes[key] = c.Compress(data)
		}
		return
	}
	info, err := os.Stat(path)
	if err != nil {
		return
	}
	var values []*Value
	if info.IsDir() {
		values = c.walk(path)
	} else {
		if v := c.read(path); v != nil {
			values = []*Value{v}
		}
	}
	c.Routes[key] = c.Compress(c.Wire(values))
	for _, v := range values {
		v.Data = nil
	}
}

// read loads a single file into a Value, inferring MIME type from extension
// or content detection as a fallback.
func (c *Circuit) read(path string) *Value {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	ct := mime.TypeByExtension(ext)
	if ct == "" {
		ct = http.DetectContentType(data)
	}
	return &Value{Name: base[:len(base)-len(ext)], Size: uint32(len(data)), Type: ct, Data: data}
}

// walk reads all files in a directory into an ordered slice of Values.
// If a sort.json file is present, it defines the blob order by name;
// files not listed in sort.json are appended after the ordered entries.
func (c *Circuit) walk(path string) []*Value {
	var all []*Value
	filepath.WalkDir(path, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || filepath.Base(p) == "sort.json" {
			return err
		}
		if v := c.read(p); v != nil {
			all = append(all, v)
		}
		return nil
	})

	if data, err := os.ReadFile(filepath.Join(path, "sort.json")); err == nil {
		var order []string
		if json.Unmarshal(data, &order) == nil {
			byName := make(map[string]*Value, len(all))
			for _, v := range all {
				byName[v.Name] = v
			}
			out := make([]*Value, 0, len(all))
			for _, name := range order {
				if v, ok := byName[name]; ok {
					out = append(out, v)
					delete(byName, name)
				}
			}
			for _, v := range all {
				if _, ok := byName[v.Name]; ok {
					out = append(out, v)
				}
			}
			return out
		}
	}

	return all
}
