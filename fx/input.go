package fx

import (
	"fmt"
	"io"
	"io/fs"
	"mime"
	"net/http"
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

func (f *fx) Input(source string) (N, error) {
	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		return f.payload(source)
	}
	info, err := os.Stat(source)
	if err != nil {
		return nil, fmt.Errorf("stat %q: %w", source, err)
	}
	if info.IsDir() {
		return f.dir(source)
	}
	file, err := f.file(source)
	if err != nil {
		return nil, err
	}
	return N{file}, nil
}

func (f *fx) Load(source string) (N, error) {
	payload, err := f.payload(source)
	if err != nil {
		return nil, err
	}
	entries, err := f.Decode(payload[0].Data)
	if err != nil {
		return nil, fmt.Errorf("decode %q: %w", source, err)
	}
	return entries, nil
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

func (f *fx) payload(source string) (N, error) {
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

func (f *fx) dir(path string) (N, error) {
	var groups [][]*One
	directories := make(map[string]int)
	err := fs.WalkDir(os.DirFS(path), ".", func(relative string, entry fs.DirEntry, err error) error {
		fullPath := filepath.Join(path, filepath.FromSlash(relative))
		if err != nil {
			return fmt.Errorf("walk dir %q: %w", fullPath, err)
		}
		if entry.IsDir() {
			name := relative
			if relative == "." {
				name = filepath.Base(path)
			}
			directories[relative] = len(groups)
			groups = append(groups, []*One{{Name: name}})
			return nil
		}
		if relative == "." {
			return fmt.Errorf("read dir %q: not a directory", path)
		}
		file, err := f.file(fullPath)
		if err != nil {
			return err
		}
		file.Name = relative
		index := directories[filepath.ToSlash(filepath.Dir(relative))]
		groups[index] = append(groups[index], file)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return f.normalize(groups), nil
}

func (f *fx) normalize(groups [][]*One) N {
	trim := func(name string) string {
		ext := filepath.Ext(name)
		if ext == filepath.Base(name) {
			return name // Preserve extensionless dotfiles.
		}
		return strings.TrimSuffix(name, ext)
	}

	counts := make(map[string]int)
	for i, group := range groups {
		if i > 0 {
			counts[group[0].Name]++
		}
		for _, file := range group[1:] {
			counts[trim(file.Name)]++
		}
	}

	entries := N{groups[0][0]}
	for _, group := range groups {
		var sequence *One
		byName := make(map[string]*One, len(group)-1)
		for _, file := range group[1:] {
			if name := trim(file.Name); counts[name] == 1 {
				file.Name = name
			}
			name := filepath.Base(file.Name)
			if name == "sequence" {
				sequence = file
			} else {
				byName[name] = file
			}
		}
		if sequence != nil {
			for name := range strings.SplitSeq(string(sequence.Data), "\n") {
				name = strings.TrimSpace(name)
				if file := byName[name]; file != nil {
					entries = append(entries, file)
					delete(byName, name)
				}
			}
		}
		for _, file := range group[1:] {
			if byName[filepath.Base(file.Name)] != nil {
				entries = append(entries, file)
			}
		}
	}
	return entries
}
