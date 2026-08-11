package fx

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type Input interface {
	Url(string) (*Result, error)
	Path(string) ([]*Result, error)
}
type Result struct {
	Name string
	Type string
	Data []byte
}

type Dir *Result

type input struct{}

func (i *input) Url(source string) (*Result, error) {
	resp, err := http.Get(source)
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
	return &Result{
		Name: baseName(resp.Request.URL.Path),
		Type: http.DetectContentType(data),
		Data: data}, nil
}

func (i *input) Path(path string) ([]*Result, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("stat %q: %w", path, err)
	}
	if info.IsDir() {
		return readDir(path)
	}
	result, err := readFile(path, baseName(path))
	if err != nil {
		return nil, err
	}
	return []*Result{result}, nil
}

// readDir reads path's directory listing, normalizes it into Results, and applies sequencing.
func readDir(path string) ([]*Result, error) {
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

// processFiles reads each non-dir entry into a Result using its full filesystem name (nested
// directories are ignored), then strips extensions from names that don't collide once shortened.
func processFiles(path string, entries []os.DirEntry) ([]*Result, error) {
	results := make([]*Result, 0, len(entries))
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

func readFile(path, name string) (*Result, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file %q: %w", path, err)
	}
	return &Result{Name: name, Type: mime.TypeByExtension(filepath.Ext(path)), Data: data}, nil
}

// stripExtensions shortens each entry's Name to its baseName, except where that would
// collide with another entry's shortened name — those entries keep their full name.
func stripExtensions(entries []*Result) []*Result {
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
func sequence(entries []*Result) []*Result {
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

	byName := make(map[string]*Result, len(rest))
	for _, entry := range rest {
		byName[entry.Name] = entry
	}

	ordered := make([]*Result, 0, len(rest))
	used := make(map[*Result]bool, len(rest))
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
