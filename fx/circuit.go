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

// Value is the unit of work for encoding. Name and Type populate the manifest;
// Data is the raw blob. Data is excluded from JSON and freed after encoding.
type Value struct {
	Name string `json:"name,omitempty"`
	Type string `json:"type,omitempty"`
	Size uint32 `json:"size"`
	Data []byte `json:"-"`
}

// Encode produces a self-describing asset bundle:
// [4B: manifest length][JSON manifest][blob][blob]...[blob]
// Sizes are written into the manifest so the decoder can slice blobs without
// per-blob length prefixes in the binary data.
func Encode(values []*Value) []byte {
	totalData := 0
	for i, v := range values {
		values[i].Size = uint32(len(v.Data))
		totalData += len(v.Data)
	}
	j, _ := json.Marshal(values)
	buf := bytes.NewBuffer(make([]byte, 0, 4+len(j)+totalData))
	var hdr [4]byte
	binary.BigEndian.PutUint32(hdr[:], uint32(len(j)))
	buf.Write(hdr[:])
	buf.Write(j)
	for _, v := range values {
		buf.Write(v.Data)
	}
	return buf.Bytes()
}

// Compress gzip-compresses data at maximum compression.
// Bundles are compressed once at build time and served directly
// with Content-Encoding: gzip.
func Compress(data []byte) []byte {
	var buf bytes.Buffer
	w, _ := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	w.Write(data)
	w.Close()
	return buf.Bytes()
}

// ToBytes fetches the content at input, either from an HTTP URL or a local file.
func (fx *Fx) ToBytes(input string) ([]byte, error) {
	if strings.HasPrefix(input, "http") {
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
func (fx *Fx) Load(path string) {
	info, err := os.Stat(path)
	if err != nil {
		return
	}
	key := filepath.Base(path)
	var values []*Value
	if info.IsDir() {
		values = fx.walk(path)
	} else {
		if v := fx.read(path); v != nil {
			values = []*Value{v}
		}
	}
	fx.Routes[key] = Compress(Encode(values))
	for _, v := range values {
		v.Data = nil
	}
}

// read loads a single file into a Value, inferring MIME type from extension
// or content detection as a fallback.
func (fx *Fx) read(path string) *Value {
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
func (fx *Fx) walk(path string) []*Value {
	var values []*Value
	index := map[string]int{}

	if data, err := os.ReadFile(filepath.Join(path, "sort.json")); err == nil {
		var order []string
		if json.Unmarshal(data, &order) == nil {
			values = make([]*Value, len(order))
			for i, name := range order {
				values[i] = &Value{Name: name}
				index[name] = i
			}
		}
	}

	filepath.WalkDir(path, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || filepath.Base(p) == "sort.json" {
			return err
		}
		v := fx.read(p)
		if v == nil {
			return nil
		}
		if i, ok := index[v.Name]; ok {
			values[i] = v
		} else {
			values = append(values, v)
		}
		return nil
	})
	return values
}
