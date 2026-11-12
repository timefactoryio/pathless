package fx

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func (f *fx) InputDir(path string) (N, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("read dir %q: %w", path, err)
	}

	return f.processFiles(path, entries)
}

// processFiles normalizes and orders siblings before flattening their contents.
func (f *fx) processFiles(path string, entries []os.DirEntry) (N, error) {
	results := make(N, 0, len(entries))
	directories := make(map[*One]N)
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() {
			ones, err := f.InputDir(filepath.Join(path, name))
			if err != nil {
				return nil, err
			}
			directory := &One{Name: name}
			directories[directory] = ones
			results = append(results, directory)
			continue
		}
		file, err := readFile(filepath.Join(path, name), name)
		if err != nil {
			return nil, err
		}
		results = append(results, file)
	}
	flat := make(N, 0, len(results))
	for _, entry := range sequence(stripExtensions(results)) {
		if children, ok := directories[entry]; ok {
			for _, child := range children {
				child.Name = entry.Name + "/" + child.Name
			}
			flat = append(flat, children...)
		} else {
			flat = append(flat, entry)
		}
	}
	return flat, nil
}

// stripExtensions shortens each entry's Name to its baseName, except where that would
// collide with another entry's shortened name — those entries keep their full name.
func stripExtensions(entries N) N {
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
func sequence(entries N) N {
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

	ordered := make(N, 0, len(rest))
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
