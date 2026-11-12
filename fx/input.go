package fx

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
)

type One struct {
	Name string
	Type string
	Data []byte
}

type N []*One

func (f *fx) Input(path string) (N, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file %q: %w", path, err)
	}
	typ := mime.TypeByExtension(filepath.Ext(path))
	if typ == "" {
		typ = http.DetectContentType(data)
	}
	return N{{
		Name: filepath.Base(path),
		Type: typ,
		Data: data,
	}}, nil
}

func (f *fx) file(path string) (*One, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file %q: %w", path, err)
	}
	typ := mime.TypeByExtension(filepath.Ext(path))
	if typ == "" {
		typ = http.DetectContentType(data)
	}
	return &One{
		Name: filepath.Base(path),
		Type: typ,
		Data: data,
	}, nil
}

func (f *fx) InputPayload(source string) (N, error) {
	resp, err := f.client.Get(source)
	if err != nil {
		return nil, fmt.Errorf("get %q: %w", source, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK ||
		resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("get %q: %s", source, resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read %q: %w", source, err)
	}
	typ := resp.Header.Get("Content-Type")
	if _, _, err := mime.ParseMediaType(typ); err != nil {
		typ = mime.TypeByExtension(filepath.Ext(resp.Request.URL.Path))
	}
	if typ == "" {
		typ = http.DetectContentType(data)
	}
	return N{{
		Name: source,
		Type: typ,
		Data: data,
	}}, nil
}

func (f *fx) Load(source string) (N, error) {
	payload, err := f.InputPayload(source)
	if err != nil {
		return nil, err
	}
	entries, err := f.Decode(payload[0].Data)
	if err != nil {
		return nil, fmt.Errorf("decode %q: %w", source, err)
	}
	return entries, nil
}
