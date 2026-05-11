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

type Value struct {
	Name   string `json:"name,omitempty"`
	Type   string `json:"type,omitempty"`
	Size   uint32 `json:"size"`
	Offset uint32 `json:"offset"`
	Data   []byte `json:"-"`
}

func Encode(values []*Value) []byte {
	for i, v := range values {
		values[i].Size = uint32(len(v.Data))
		values[i].Offset = 0
	}
	j, _ := json.Marshal(values)
	for {
		off := uint32(4 + len(j))
		for i, v := range values {
			values[i].Offset = off
			off += v.Size
		}
		j2, _ := json.Marshal(values)
		if len(j2) == len(j) {
			j = j2
			break
		}
		j = j2
	}
	var buf bytes.Buffer
	binary.Write(&buf, binary.BigEndian, uint32(len(j)))
	buf.Write(j)
	for _, v := range values {
		buf.Write(v.Data)
	}
	return buf.Bytes()
}

func Compress(data []byte) []byte {
	var buf bytes.Buffer
	w, _ := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	w.Write(data)
	w.Close()
	return buf.Bytes()
}

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
	for _, v := range values {
		v.Data = nil
	}
	fx.Routes[key] = Compress(Encode(values))
}

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
	return &Value{Name: base[:len(base)-len(ext)], Type: ct, Size: uint32(len(data)), Data: data}
}

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
