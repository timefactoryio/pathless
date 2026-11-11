package fx

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

type Fx interface {
	Input(string) ([]*One, error)
	Frame(string, ...bool) error
	Save(string, ...bool) ([]byte, error)
	Circuit() map[string][]byte
}

type fx struct {
	frames []string
	panels []string
	routes map[string][]*One
	client *http.Client
}

func NewFx() Fx {
	return &fx{
		frames: []string{},
		panels: []string{},
		routes: make(map[string][]*One),
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (f *fx) Frame(source string, panel ...bool) error {
	html := source
	if !strings.HasPrefix(strings.TrimSpace(source), "<") {
		entries, err := f.Input(source)
		if err != nil {
			return err
		}
		if len(entries) != 1 {
			return fmt.Errorf("%q: expected one entry, got %d", source, len(entries))
		}
		html = string(entries[0].Data)
	}

	target := &f.frames
	if len(panel) > 0 && panel[0] {
		target = &f.panels
	}
	*target = append(*target, f.build(html))
	return nil
}

func (f *fx) Save(key string, binary ...bool) ([]byte, error) {
	route, ok := f.routes[key]
	if !ok {
		return nil, fmt.Errorf("route %q not found", key)
	}

	data := encode(route)
	if len(binary) > 0 && binary[0] {
		if err := os.WriteFile(key, data, 0o644); err != nil {
			return nil, fmt.Errorf("save route %q: %w", key, err)
		}
	}
	return data, nil
}

func (f *fx) Circuit() map[string][]byte {
	var root bytes.Buffer
	writer, _ := gzip.NewWriterLevel(&root, gzip.BestCompression)
	if err := json.NewEncoder(writer).Encode([2][]string{f.frames, f.panels}); err != nil {
		panic(err)
	}
	if err := writer.Close(); err != nil {
		panic(err)
	}
	payloads := make(map[string][]byte, len(f.routes)+1)
	payloads[""] = root.Bytes()
	for key, payload := range f.routes {
		payloads[key] = encode(payload)
	}
	return payloads
}

// build merges inline style/script blocks in HTML.
func (f *fx) build(html string) string {
	if styles := styleTag.FindAllStringSubmatch(html, -1); len(styles) > 1 {
		var merged strings.Builder
		for _, match := range styles {
			merged.WriteString(match[1])
			merged.WriteByte('\n')
		}
		html = "<style>" + merged.String() + "</style>" +
			styleTag.ReplaceAllString(html, "")
	}
	if scripts := scriptTag.FindAllStringSubmatch(html, -1); len(scripts) > 0 {
		var merged strings.Builder
		for _, match := range scripts {
			source := match[1]
			if !strings.HasPrefix(strings.TrimSpace(source), "{") {
				source = "{" + source + "}"
			}
			merged.WriteString(source)
			merged.WriteByte('\n')
		}
		html = scriptTag.ReplaceAllString(html, "") +
			"<script>{" + merged.String() + "}</script>"
	}
	return html
}

var (
	styleTag  = regexp.MustCompile(`(?s)<style>(.*?)</style>`)
	scriptTag = regexp.MustCompile(`(?s)<script>(.*?)</script>`)
)
