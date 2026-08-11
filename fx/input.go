package fx

import (
	"encoding/binary"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type Input interface {
	String(string) (*Output, error)
}
type Output struct {
	Name string
	Type string
	Zero []byte
	One  []*Output
}

type input struct{}

func NewInput() Input {
	return &input{}
}

func (i *input) String(path string) (*Output, error) {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return i.url(path)
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("stat %q: %w", path, err)
	}
	if info.IsDir() {
		return i.dir(path)
	}
	return readFile(path, baseName(path))
}

func (i *input) url(source string) (*Output, error) {
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
	return &Output{
		Name: baseName(resp.Request.URL.Path),
		Type: http.DetectContentType(data),
		Zero: data}, nil
}

// dir reads path as a directory directly, without stat-checking it like String does.
func (i *input) dir(path string) (*Output, error) {
	files, err := readDir(path)
	if err != nil {
		return nil, err
	}
	return &Output{Name: baseName(path), Zero: sizes(files), One: files}, nil
}

// sizes encodes each file's byte size as a big-endian uint64, preceded by their combined total.
func sizes(files []*Output) []byte {
	data := make([]byte, 8*(1+len(files)))
	var total uint64
	for i, file := range files {
		size := uint64(len(file.Zero))
		total += size
		binary.BigEndian.PutUint64(data[8*(i+1):], size)
	}
	binary.BigEndian.PutUint64(data[:8], total)
	return data
}

// readDir reads path's directory listing, normalizes it into Results, and applies sequencing.
func readDir(path string) ([]*Output, error) {
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

// processFiles reads each non-dir entry into a Output using its full filesystem name (nested
// directories are ignored), then strips extensions from names that don't collide once shortened.
func processFiles(path string, entries []os.DirEntry) ([]*Output, error) {
	results := make([]*Output, 0, len(entries))
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

func readFile(path, name string) (*Output, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file %q: %w", path, err)
	}
	return &Output{Name: name, Type: mime.TypeByExtension(filepath.Ext(path)), Zero: data}, nil
}

// stripExtensions shortens each entry's Name to its baseName, except where that would
// collide with another entry's shortened name — those entries keep their full name.
func stripExtensions(entries []*Output) []*Output {
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
func sequence(entries []*Output) []*Output {
	var data []byte
	rest := entries[:0:0]
	for _, entry := range entries {
		if entry.Name == "sequence" {
			data = entry.Zero
			continue
		}
		rest = append(rest, entry)
	}
	if data == nil {
		return entries
	}

	byName := make(map[string]*Output, len(rest))
	for _, entry := range rest {
		byName[entry.Name] = entry
	}

	ordered := make([]*Output, 0, len(rest))
	used := make(map[*Output]bool, len(rest))
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
