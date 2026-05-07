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
	"sort"
	"strings"
)

type Circuit interface {
	Router() *http.ServeMux
	Read(path, prefix string)
	Reader(path string) string
	ToBytes(input string) []byte
	Compress(data []byte) []byte
}

type circuit struct {
	router *http.ServeMux
	value  map[string][]*Value
}

type Value struct {
	Name string `json:"name"`
	Type string `json:"type"`
	Size int    `json:"size"`
	Data []byte `json:"-"`
}

func NewCircuit() Circuit {
	return &circuit{
		router: http.NewServeMux(),
		value:  make(map[string][]*Value),
	}
}

func (c *circuit) Router() *http.ServeMux {
	return c.router
}

func (c *circuit) ToBytes(input string) []byte {
	if strings.HasPrefix(input, "http://") || strings.HasPrefix(input, "https://") {
		resp, err := http.Get(input)
		if err != nil {
			return nil
		}
		defer resp.Body.Close()
		b, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil
		}
		return b
	}
	b, err := os.ReadFile(input)
	if err != nil {
		return nil
	}
	return b
}

func (c *circuit) Compress(data []byte) []byte {
	var buf bytes.Buffer
	w, _ := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	w.Write(data)
	w.Close()
	return buf.Bytes()
}

func (c *circuit) Read(path, prefix string) {
	v := c.newValue(path)
	if v == nil {
		return
	}
	route := v.Name
	if prefix != "" {
		route = prefix + "/" + v.Name
	}
	c.value[prefix] = []*Value{v}
	c.registerRoute(route, v.Type, c.Compress(v.Data))
	v.Data = nil
}

func (c *circuit) Reader(path string) string {
	dirName, values := c.loadFiles(path)
	c.value[dirName] = values

	manifestJSON, _ := json.Marshal(values)

	var buf bytes.Buffer
	binary.Write(&buf, binary.BigEndian, uint32(len(manifestJSON)))
	buf.Write(manifestJSON)
	for _, v := range values {
		buf.Write(v.Data)
	}

	c.registerRoute(dirName, "application/octet-stream", c.Compress(buf.Bytes()))
	for _, v := range values {
		v.Data = nil
	}
	return dirName
}

func (c *circuit) newValue(path string) *Value {
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
	return &Value{Name: base[:len(base)-len(ext)], Type: ct, Size: len(data), Data: data}
}

func (c *circuit) loadFiles(path string) (string, []*Value) {
	dirName := filepath.Base(path)
	var values []*Value
	var orderMap map[string]int

	if data, err := os.ReadFile(filepath.Join(path, "sort.json")); err == nil {
		var order []string
		if json.Unmarshal(data, &order) == nil {
			orderMap = make(map[string]int, len(order))
			for i, name := range order {
				orderMap[name] = i
			}
		}
	}

	filepath.WalkDir(path, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || filepath.Base(p) == "sort.json" {
			return err
		}
		if v := c.newValue(p); v != nil {
			values = append(values, v)
		}
		return nil
	})

	if len(orderMap) > 0 {
		sort.Slice(values, func(i, j int) bool {
			posI, foundI := orderMap[values[i].Name]
			posJ, foundJ := orderMap[values[j].Name]
			if foundI && foundJ {
				return posI < posJ
			}
			if foundI {
				return true
			}
			if foundJ {
				return false
			}
			return values[i].Name < values[j].Name
		})
	}

	return dirName, values
}

func (c *circuit) registerRoute(path, contentType string, data []byte) {
	c.Router().HandleFunc("/"+path, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Content-Encoding", "gzip")
		w.Write(data)
	})
}
