package v2

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/timefactoryio/pathless/zero"
)

type Fx struct {
	*zero.Zero
	Input Input
}

func NewFx(z *zero.Zero) *Fx {
	return &Fx{
		Zero:  z,
		Input: &input{},
	}
}

type Input interface {
	File(string) (*Result, error)
	HTTP(string) (*Result, error)
	Directory(string) (map[string][]*Result, error)
}

type Result struct {
	Name string
	Type string
	Data []byte
}

type input struct{}

func (i *input) File(path string) (*Result, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file %q: %w", path, err)
	}

	return &Result{
		Name: resultName(path),
		Type: mime.TypeByExtension(filepath.Ext(path)),
		Data: data,
	}, nil
}

func (i *input) HTTP(source string) (*Result, error) {
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
		Name: resultName(resp.Request.URL.Path),
		Type: http.DetectContentType(data),
		Data: data,
	}, nil
}

func resultName(path string) string {
	base := filepath.Base(path)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

func (i *input) sequence(path string, results []*Result) ([]*Result, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read sequence %q: %w", path, err)
	}

	byName := make(map[string][]*Result, len(results))
	for _, result := range results {
		byName[result.Name] = append(byName[result.Name], result)
	}

	ordered := make([]*Result, 0, len(results))
	used := make(map[*Result]bool, len(results))
	for name := range strings.SplitSeq(strings.TrimSpace(string(data)), "\n") {
		name = strings.TrimSpace(name)
		matches := byName[name]
		if len(matches) == 0 {
			continue
		}
		result := matches[0]
		byName[name] = matches[1:]
		ordered = append(ordered, result)
		used[result] = true
	}
	for _, result := range results {
		if !used[result] {
			ordered = append(ordered, result)
		}
	}
	return ordered, nil
}

func (i *input) Directory(root string) (map[string][]*Result, error) {
	results := make(map[string][]*Result)
	sequences := make(map[string]string)
	parent := filepath.Dir(filepath.Clean(root))

	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		relative, err := filepath.Rel(parent, path)
		if err != nil {
			return err
		}

		if entry.IsDir() {
			results[filepath.ToSlash(relative)] = []*Result{}
			return nil
		}

		key := filepath.ToSlash(filepath.Dir(relative))
		if entry.Name() == "sequence" {
			sequences[key] = path
			return nil
		}

		result, err := i.File(path)
		if err != nil {
			return err
		}
		results[key] = append(results[key], result)
		return nil
	})
	if err != nil {
		return nil, err
	}

	for key, path := range sequences {
		ordered, err := i.sequence(path, results[key])
		if err != nil {
			return nil, err
		}
		results[key] = ordered
	}
	return results, nil
}
