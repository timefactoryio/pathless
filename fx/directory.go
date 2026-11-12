package fx

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type Manifest [][]*string

func (f *fx) InputDir(path string) (N, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("read dir %q: %w", path, err)
	}

	return f.processFiles(path, entries)
}

func (f *fx) dir(path string) (N, error) {
	manifest := Manifest{}
	directories := make(map[string]int)
	err := fs.WalkDir(os.DirFS(path), ".", func(relative string, entry fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("walk dir %q: %w", filepath.Join(path, filepath.FromSlash(relative)), err)
		}
		if entry.IsDir() {
			record := []*string{nil}
			if relative != "." {
				record[0] = &relative
			}
			directories[relative] = len(manifest)
			manifest = append(manifest, record)
			return nil
		}
		if relative == "." {
			return fmt.Errorf("read dir %q: not a directory", path)
		}
		index := directories[filepath.ToSlash(filepath.Dir(relative))]
		name := entry.Name()
		manifest[index] = append(manifest[index], &name)
		return nil
	})
	if err != nil {
		return nil, err
	}
	data, err := json.Marshal(manifest)
	if err != nil {
		return nil, fmt.Errorf("marshal dir %q: %w", path, err)
	}
	results := N{{
		Name: filepath.Base(path),
		Type: "dir",
		Data: data,
	}}
	for _, record := range manifest {
		var prefix string
		if record[0] != nil {
			prefix = filepath.FromSlash(*record[0])
		}
		for _, name := range record[1:] {
			relative := filepath.Join(prefix, *name)
			file, err := f.file(filepath.Join(path, relative))
			if err != nil {
				return nil, err
			}
			file.Name = filepath.ToSlash(relative)
			results = append(results, file)
		}
	}
	return results, nil
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
		file, err := f.file(filepath.Join(path, name))
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

// stripExtensions removes the final extension except on dot-prefixed names or where that would
// collide with another entry's shortened name — those entries keep their full name.
func stripExtensions(entries N) N {
	counts := make(map[string]int, len(entries))
	candidates := make([]string, len(entries))
	for i, entry := range entries {
		name := entry.Name
		if !strings.HasPrefix(name, ".") {
			name = strings.TrimSuffix(name, filepath.Ext(name))
		}
		candidates[i] = name
		counts[name]++
	}
	for i, entry := range entries {
		if candidate := candidates[i]; counts[candidate] == 1 {
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
