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

func (f *fx) Input(path string, public ...bool) ([]*One, error) {
	var (
		entries []*One
		name    string
	)

	if strings.HasPrefix(path, "http://") ||
		strings.HasPrefix(path, "https://") {
		var err error
		entries, err = f.url(path)
		if err != nil {
			return nil, err
		}
		name = baseName(path)
		if name == "" {
			return nil, fmt.Errorf("URL %q has no route name", path)
		}
	} else {
		info, err := os.Stat(path)
		if err != nil {
			return nil, fmt.Errorf("stat %q: %w", path, err)
		}

		if info.IsDir() {
			entries, err = readDir(path)
			name = filepath.Base(path)
		} else {
			entry, readErr := readFile(path, baseName(path))
			err = readErr
			entries = []*One{entry}
			name = entry.Name
		}
		if err != nil {
			return nil, err
		}
	}

	if len(public) > 0 && public[0] {
		f.routes[name] = entries
	}
	return entries, nil
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

func (f *fx) url(source string) ([]*One, error) {
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

	if entries, err := decode(data); err == nil {
		return entries, nil
	}

	return []*One{{
		Name: baseName(resp.Request.URL.Path),
		Type: detectType(resp.Request.URL.Path, data),
		Data: data,
	}}, nil
}

// readDir reads path's directory listing, normalizes it into Results, and applies sequencing.
func readDir(path string) ([]*One, error) {
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

// processFiles reads files and recursively populates Ones for directories, then
// strips extensions from names that don't collide once shortened.
func processFiles(path string, entries []os.DirEntry) ([]*One, error) {
	results := make([]*One, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() {
			ones, err := readDir(filepath.Join(path, name))
			if err != nil {
				return nil, err
			}
			results = append(results, &One{Name: name, Ones: ones})
			continue
		}
		file, err := readFile(filepath.Join(path, name), name)
		if err != nil {
			return nil, err
		}
		results = append(results, file)
	}
	return stripExtensions(results), nil
}

func readFile(path, name string) (*One, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file %q: %w", path, err)
	}
	return &One{Name: name, Type: detectType(path, data), Data: data}, nil
}

// stripExtensions shortens each entry's Name to its baseName, except where that would
// collide with another entry's shortened name — those entries keep their full name.
func stripExtensions(entries []*One) []*One {
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
func sequence(entries []*One) []*One {
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

	byName := make(map[string]*One, len(rest))
	for _, entry := range rest {
		byName[entry.Name] = entry
	}

	ordered := make([]*One, 0, len(rest))
	used := make(map[*One]bool, len(rest))
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
