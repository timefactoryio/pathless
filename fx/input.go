package fx

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type Input interface {
	Url(string) (*Result, error)
	Path(string) (*Result, error)
}
type Result struct {
	Name    string
	Path    string
	Type    string
	Data    []byte
	Entries []*Result
}

type input struct{}

// baseName drops the extension, except on dot-prefixed bases (.env.local, .gitignore).
func baseName(path string) string {
	base := filepath.Base(path)
	if strings.HasPrefix(base, ".") {
		return base
	}
	return strings.TrimSuffix(base, filepath.Ext(base))
}

func newResult(name, parent string, data []byte) *Result {
	return &Result{Name: name, Path: parent, Type: http.DetectContentType(data), Data: data}
}

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
	return newResult(baseName(resp.Request.URL.Path), "", data), nil
}

func (i *input) Path(path string) (*Result, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("stat %q: %w", path, err)
	}
	if info.IsDir() {
		return readDir(path, "")
	}
	return readFile(path, baseName(path), "")
}

// parent is the containing directory's slash-suffixed name, "" at the walk root.
func readFile(path, name, parent string) (*Result, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file %q: %w", path, err)
	}
	return newResult(name, parent, data), nil
}

func readDir(path, parent string) (*Result, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("read dir %q: %w", path, err)
	}

	// real excludes sequence.txt, so neither pass below needs to special-case it.
	var order []byte
	real := entries[:0:0]
	for _, entry := range entries {
		if entry.Name() == "sequence.txt" {
			if order, err = os.ReadFile(filepath.Join(path, entry.Name())); err != nil {
				return nil, fmt.Errorf("read sequence %q: %w", path, err)
			}
			continue
		}
		real = append(real, entry)
	}

	name := filepath.Base(path) + "/"
	result := &Result{Name: name, Path: parent, Entries: make([]*Result, 0, len(real))}

	names := make(map[string]int, len(real))
	for _, entry := range real {
		if !entry.IsDir() {
			names[baseName(entry.Name())]++
		}
	}

	for _, entry := range real {
		child := filepath.Join(path, entry.Name())
		var nested *Result
		if entry.IsDir() {
			nested, err = readDir(child, name)
		} else {
			fileName := baseName(entry.Name())
			if names[fileName] > 1 {
				fileName = entry.Name()
			}
			nested, err = readFile(child, fileName, name)
		}
		if err != nil {
			return nil, err
		}
		result.Entries = append(result.Entries, nested)
	}

	if order != nil {
		result.Entries = sortBySequence(order, result.Entries)
	}
	return result, nil
}

// sortBySequence reorders entries per the newline-separated names in data;
// named entries come first in listed order, unlisted entries keep their relative order after.
func sortBySequence(data []byte, entries []*Result) []*Result {
	byName := make(map[string][]*Result, len(entries))
	for _, entry := range entries {
		byName[entry.Name] = append(byName[entry.Name], entry)
	}

	ordered := make([]*Result, 0, len(entries))
	used := make(map[*Result]bool, len(entries))
	for name := range strings.SplitSeq(strings.TrimSpace(string(data)), "\n") {
		name = strings.TrimSpace(name)
		matches := byName[name]
		if len(matches) == 0 {
			continue
		}
		next := matches[0]
		byName[name] = matches[1:]
		ordered = append(ordered, next)
		used[next] = true
	}
	for _, entry := range entries {
		if !used[entry] {
			ordered = append(ordered, entry)
		}
	}
	return ordered
}
