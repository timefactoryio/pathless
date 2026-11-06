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

	"github.com/timefactoryio/pathless/zero"
)

type Fx interface {
	Input(string, ...bool) ([]*One, error)
	Frame(string, ...bool) error
	Save(string, ...bool) ([]byte, error)
	Start()
}

type fx struct {
	z      *zero.Zero
	frames []string
	panels []string
	routes map[string][]*One
	mux    *http.ServeMux
	client *http.Client
}

func NewFx(z *zero.Zero) Fx {
	return &fx{
		z:      z,
		frames: []string{},
		panels: []string{},
		routes: make(map[string][]*One),
		mux:    http.NewServeMux(),
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

func (f *fx) Start() {
	var root bytes.Buffer
	writer, _ := gzip.NewWriterLevel(&root, gzip.BestCompression)
	if err := json.NewEncoder(writer).Encode([2][]string{f.frames, f.panels}); err != nil {
		panic(err)
	}
	if err := writer.Close(); err != nil {
		panic(err)
	}
	f.handle("/", "application/json", root.Bytes())
	for key, payload := range f.routes {
		// log.Printf("route: %s", key)
		f.handle("/"+key, "application/octet-stream", encode(payload))
	}

	if err := http.ListenAndServe(":1001", f.cors(f.mux)); err != nil {
		panic(err)
	}
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

// handle registers an encoded wire payload and writes it from memory on every request.
func (f *fx) handle(path, contentType string, data []byte) {
	f.mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Content-Encoding", "gzip")
		w.Write(data)
	})
}

func (f *fx) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", f.z.PathlessURL)
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
