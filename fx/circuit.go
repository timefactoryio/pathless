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
	Routes map[string][]byte
}

func NewCircuit() *Circuit {
	return &Circuit{Routes: make(map[string][]byte)}
}

// Value holds name/type metadata for a single entry in a Response.
// The raw bytes for each Value live in the parallel Response.Data slice
// (same index), so Values can be cheaply built/reordered without copying data.
type Value struct {
	Name string
	Type string
}

// Response pairs a slice of Value metadata with a parallel slice of raw
// byte payloads — Values[i] describes Data[i].
type Response struct {
	Values []*Value
	Data   [][]byte
}

func (c *Circuit) Wire(r *Response) []byte {
	n := len(r.Values)
	totalData := 0
	for _, d := range r.Data {
		totalData += len(d)
	}
	buf := bytes.NewBuffer(make([]byte, 0, 2+6*n+totalData))
	buf.WriteByte(byte(n >> 8))
	buf.WriteByte(byte(n))
	var hdr [4]byte
	for i, v := range r.Values {
		buf.WriteByte(byte(len(v.Name)))
		buf.WriteString(v.Name)
		buf.WriteByte(byte(len(v.Type)))
		buf.WriteString(v.Type)
		data := r.Data[i]
		binary.BigEndian.PutUint32(hdr[:], uint32(len(data)))
		buf.Write(hdr[:])
		buf.Write(data)
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
	var r *Response
	if info.IsDir() {
		r = c.walk(path)
	} else if v, data := c.read(path); v != nil {
		r = &Response{Values: []*Value{v}, Data: [][]byte{data}}
	}
	if r != nil {
		c.Routes[key] = c.Compress(c.Wire(r))
		r.Data = nil
	}
}

// read loads a single file into a Value plus its raw bytes, inferring MIME
// type from extension or content detection as a fallback.
func (c *Circuit) read(path string) (*Value, []byte) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, nil
	}
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	ct := mime.TypeByExtension(ext)
	if ct == "" {
		ct = http.DetectContentType(raw)
	}
	return &Value{Name: base[:len(base)-len(ext)], Type: ct}, raw
}

// walk reads all files in a directory into a Response with parallel
// Values/Data slices. If a sort.txt file is present, it defines the order
// by name; files not listed in sort.txt are appended after the ordered entries.
func (c *Circuit) walk(path string) *Response {
	r := &Response{}
	filepath.WalkDir(path, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || filepath.Base(p) == "sort.txt" {
			return err
		}
		if v, data := c.read(p); v != nil {
			r.Values = append(r.Values, v)
			r.Data = append(r.Data, data)
		}
		return nil
	})
	if raw, err := os.ReadFile(filepath.Join(path, "sort.txt")); err == nil {
		order := strings.Split(strings.TrimSpace(string(raw)), "\n")
		byName := make(map[string]int, len(r.Values))
		for i, v := range r.Values {
			byName[v.Name] = i
		}
		outValues := make([]*Value, 0, len(r.Values))
		outData := make([][]byte, 0, len(r.Data))
		seen := make(map[int]bool)
		for _, name := range order {
			if idx, ok := byName[strings.TrimSpace(name)]; ok && !seen[idx] {
				outValues = append(outValues, r.Values[idx])
				outData = append(outData, r.Data[idx])
				seen[idx] = true
			}
		}
		for i := range r.Values {
			if !seen[i] {
				outValues = append(outValues, r.Values[i])
				outData = append(outData, r.Data[i])
			}
		}
		r.Values, r.Data = outValues, outData
	}
	return r
}

// Save decompresses the bundle stored in Routes at key and writes it to a file named key+".bin".
// used to get raw bytes to be stored in an S3 bucket.
func (c *Circuit) Save(key string) error {
	data, ok := c.Routes[key]
	if !ok {
		return fmt.Errorf("save: route %q not found", key)
	}
	r, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("save: %w", err)
	}
	defer r.Close()
	raw, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("save: %w", err)
	}
	return os.WriteFile(key+".bin", raw, 0644)
}
