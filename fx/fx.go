package fx

import (
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
	frames []*One
	panels []*One
	routes map[string][]*One
	mux    *http.ServeMux
	client *http.Client
}

func NewFx(z *zero.Zero) Fx {
	return &fx{
		z:      z,
		routes: make(map[string][]*One),
		mux:    http.NewServeMux(),
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (f *fx) Frame(path string, panel ...bool) error {
	entries, err := f.Input(path)
	if err != nil {
		return err
	}
	if len(entries) != 1 {
		return fmt.Errorf("%q: expected one entry, got %d", path, len(entries))
	}

	target := &f.frames
	if len(panel) > 0 && panel[0] {
		target = &f.panels
	}
	*target = append(*target, f.build(entries[0]))
	return nil
}

func (f *fx) Save(key string, binary ...bool) ([]byte, error) {
	route, ok := f.routes[key]
	if !ok {
		return nil, fmt.Errorf("route %q not found", key)
	}

	data := encode(route)
	if len(binary) > 0 && binary[0] {
		if err := os.WriteFile(key+".bin", data, 0o644); err != nil {
			return nil, fmt.Errorf("save route %q: %w", key, err)
		}
	}
	return data, nil
}

func (f *fx) Start() {
	f.handle("/", encode(f.universe()))

	for key, payload := range f.routes {
		f.handle("/"+key, encode(payload))
	}

	if err := http.ListenAndServe(":1001", f.cors(f.mux)); err != nil {
		panic(err)
	}
}

func (f *fx) universe() []*One {
	return []*One{
		{
			Name: "universe",
			Type: "text/html",
			Data: f.z.UniverseHTML,
		},
		{Ones: f.frames},
		{Ones: f.panels},
	}
}

// build merges any inline style/script blocks in html into a single entry.
func (f *fx) build(entry *One) *One {
	html := string(entry.Data)
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
	result := *entry
	result.Type = "text/html"
	result.Data = []byte(html)
	return &result
}

var (
	styleTag  = regexp.MustCompile(`(?s)<style>(.*?)</style>`)
	scriptTag = regexp.MustCompile(`(?s)<script>(.*?)</script>`)
)

// handle registers an encoded wire payload and writes it from memory on every request.
func (f *fx) handle(path string, data []byte) {
	f.mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
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
