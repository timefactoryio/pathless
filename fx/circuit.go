package fx

import (
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

type Value struct {
	Name   string
	Type   string
	Size   uint64
	Offset uint64
	Data   []byte
}

func (fx *Fx) ToBytes(input string) []byte {
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

func Encode(objects [][]byte) []byte {
	n := len(objects)
	headerSize := 2 + n*4
	total := headerSize
	for _, o := range objects {
		total += len(o)
	}
	buf := make([]byte, total)
	binary.BigEndian.PutUint16(buf, uint16(n))
	off := uint32(headerSize)
	for i, o := range objects {
		binary.BigEndian.PutUint32(buf[2+i*4:], off)
		off += uint32(len(o))
	}
	pos := headerSize
	for _, o := range objects {
		copy(buf[pos:], o)
		pos += len(o)
	}
	return buf
}

func Bundle(values []*Value) *Value {
	objects := make([][]byte, len(values))
	for i, v := range values {
		objects[i] = v.Data
	}
	data := Encode(objects)
	return &Value{
		Name: "bundle",
		Type: "application/octet-stream",
		Size: uint64(len(data)),
		Data: data,
	}
}

func (fx *Fx) Read(path string) *Value {
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
	return &Value{Name: base[:len(base)-len(ext)], Type: ct, Size: uint64(len(data)), Data: data}
}

func (fx *Fx) Reader(path string) (string, []*Value) {
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
		if v := fx.Read(p); v != nil {
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

// func (fx *Fx) registerRoute(path, contentType string, data []byte) {
// 	fx.Router().HandleFunc("/"+path, func(w http.ResponseWriter, r *http.Request) {
// 		w.Header().Set("Content-Type", contentType)
// 		w.Header().Set("Content-Encoding", "gzip")
// 		w.Write(data)
// 	})
// }
