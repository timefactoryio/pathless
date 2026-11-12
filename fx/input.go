package fx

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type One struct {
	Name string
	Type string
	Data []byte
}

type N []*One

func (f *fx) Input(path string) (N, error) {
	if strings.HasPrefix(path, "http://") ||
		strings.HasPrefix(path, "https://") {
		return f.url(path)
	}

	entry, err := readFile(path, baseName(path))
	if err != nil {
		return nil, err
	}
	return N{entry}, nil
}

// detectType resolves an extension-based MIME type first, falling back to content
// sniffing (which itself defaults to application/octet-stream) when the extension
// is unknown or missing.
func detectType(path string, data []byte) string {
	if typ := mime.TypeByExtension(filepath.Ext(path)); typ != "" {
		return typ
	}
	return http.DetectContentType(data)
}

func (f *fx) url(source string) (N, error) {
	data, path, err := f.fetch(source)
	if err != nil {
		return nil, err
	}
	return N{{
		Name: baseName(path),
		Type: detectType(path, data),
		Data: data,
	}}, nil
}

func (f *fx) Load(source string) (N, error) {
	data, _, err := f.fetch(source)
	if err != nil {
		return nil, err
	}
	entries, err := f.Decode(data)
	if err != nil {
		return nil, fmt.Errorf("decode %q: %w", source, err)
	}
	return entries, nil
}

func (f *fx) fetch(source string) ([]byte, string, error) {
	resp, err := f.client.Get(source)
	if err != nil {
		return nil, "", fmt.Errorf("get %q: %w", source, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK ||
		resp.StatusCode >= http.StatusMultipleChoices {
		return nil, "", fmt.Errorf("get %q: %s", source, resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("read %q: %w", source, err)
	}
	return data, resp.Request.URL.Path, nil
}

func readFile(path, name string) (*One, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file %q: %w", path, err)
	}
	return &One{Name: name, Type: detectType(path, data), Data: data}, nil
}

// baseName drops the extension, except on dot-prefixed bases (.env.local, .gitignore).
func baseName(source string) string {
	if parsed, err := url.Parse(source); err == nil && parsed.Host != "" {
		source = parsed.Path
	}
	base := filepath.Base(source)
	if base == "." || base == "/" {
		return ""
	}
	if strings.HasPrefix(base, ".") {
		return base
	}
	return strings.TrimSuffix(base, filepath.Ext(base))
}
