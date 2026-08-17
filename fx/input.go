package fx

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Input interface {
	String(string) ([]*output, error)
}

type fx struct {
	client *http.Client
}

func NewInput() Input {
	return &fx{
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (i *fx) String(path string) ([]*output, error) {
	if strings.HasPrefix(path, "http://") ||
		strings.HasPrefix(path, "https://") {
		entry, err := i.url(path)
		if err != nil {
			return nil, err
		}
		return []*output{entry}, nil
	}

	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("stat %q: %w", path, err)
	}
	if info.IsDir() {
		return readDir(path)
	}

	entry, err := readFile(path, baseName(path))
	if err != nil {
		return nil, err
	}
	return []*output{entry}, nil
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

func (i *fx) url(source string) (*output, error) {
	resp, err := i.client.Get(source)
	if err != nil {
		return nil, fmt.Errorf("get %q: %w", source, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("get %q: %s", source, resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read %q: %w", source, err)
	}
	return &output{
		Name: baseName(resp.Request.URL.Path),
		Type: detectType(resp.Request.URL.Path, data),
		Data: data}, nil
}

// readDir reads path's directory listing, normalizes it into Results, and applies sequencing.
func readDir(path string) ([]*output, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("read dir %q: %w", path, err)
	}

	results, err := processFiles(path, entries)
	if err != nil {
		return nil, err
	}
	return sequence(results), nil
}

// processFiles reads each non-dir entry using its full filesystem name (nested
// directories are ignored), then strips extensions from names that don't collide once shortened.
func processFiles(path string, entries []os.DirEntry) ([]*output, error) {
	results := make([]*output, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		file, err := readFile(filepath.Join(path, name), name)
		if err != nil {
			return nil, err
		}
		results = append(results, file)
	}
	return stripExtensions(results), nil
}

func readFile(path, name string) (*output, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file %q: %w", path, err)
	}
	return &output{Name: name, Type: detectType(path, data), Data: data}, nil
}

// stripExtensions shortens each entry's Name to its baseName, except where that would
// collide with another entry's shortened name — those entries keep their full name.
func stripExtensions(entries []*output) []*output {
	counts := make(map[string]int, len(entries))
	for _, entry := range entries {
		counts[baseName(entry.Name)]++
	}
	for _, entry := range entries {
		if candidate := baseName(entry.Name); counts[candidate] == 1 {
			entry.Name = candidate
		}
	}
	return entries
}

// sequence orders entries by the newline-separated names in a "sequence" entry's Data,
// dropping that entry from the result; unlisted entries keep their relative order after.
// If no "sequence" entry exists, entries is returned unchanged.
func sequence(entries []*output) []*output {
	var data []byte
	rest := entries[:0:0]
	for _, entry := range entries {
		if entry.Name == "sequence" {
			data = entry.Data
			continue
		}
		rest = append(rest, entry)
	}
	if data == nil {
		return entries
	}

	byName := make(map[string]*output, len(rest))
	for _, entry := range rest {
		byName[entry.Name] = entry
	}

	ordered := make([]*output, 0, len(rest))
	used := make(map[*output]bool, len(rest))
	for name := range strings.SplitSeq(strings.TrimSpace(string(data)), "\n") {
		name = strings.TrimSpace(name)
		entry, ok := byName[name]
		if !ok || used[entry] {
			continue
		}
		ordered = append(ordered, entry)
		used[entry] = true
	}
	for _, entry := range rest {
		if !used[entry] {
			ordered = append(ordered, entry)
		}
	}
	return ordered
}

// baseName drops the extension, except on dot-prefixed bases (.env.local, .gitignore).
func baseName(path string) string {
	base := filepath.Base(path)
	if strings.HasPrefix(base, ".") {
		return base
	}
	return strings.TrimSuffix(base, filepath.Ext(base))
}
