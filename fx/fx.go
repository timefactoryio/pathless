package fx

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/timefactoryio/pathless/zero"
)

type Fx interface {
	Input(string, ...bool) ([]*output, error)
	Frame(string) error
	Panel(string) error
	Start()
}

type fx struct {
	z      *zero.Zero
	frames []*output
	panels []*output
	routes map[string]Output
	mux    *http.ServeMux
	client *http.Client
}

func NewFx(z *zero.Zero) Fx {
	return &fx{
		z:      z,
		frames: []*output{},
		panels: []*output{},
		routes: make(map[string]Output),
		mux:    http.NewServeMux(),
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (f *fx) Frame(path string) error {
	entries, err := f.Input(path)
	if err != nil {
		return err
	}
	if len(entries) != 1 {
		return fmt.Errorf("frame %q: expected one entry, got %d", path, len(entries))
	}
	f.frames = append(f.frames, f.build(entries[0]))
	return nil
}

func (f *fx) Panel(path string) error {
	entries, err := f.Input(path)
	if err != nil {
		return err
	}
	if len(entries) != 1 {
		return fmt.Errorf("panel %q: expected one entry, got %d", path, len(entries))
	}
	f.panels = append(f.panels, f.build(entries[0]))
	return nil
}

// build merges any inline style/script blocks in html into a single entry.
func (f *fx) build(entry *output) *output {
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

// handle registers path on f's mux as a wire endpoint for output: encoded
// (gzipped) once here, then written from memory on every request.
func (f *fx) handle(path string, payload Output) {
	data := payload.Encode()
	f.mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Encoding", "gzip")
		w.Write(data)
	})
}

func (f *fx) Start() {
	f.handle("/", Output{
		{&output{Name: "universe", Type: "text/html", Data: f.z.UniverseHTML}},
		f.frames,
		f.panels,
	})

	for key, payload := range f.routes {
		f.handle("/"+key, payload)
	}

	if err := http.ListenAndServe(":1001", f.cors(f.mux)); err != nil {
		panic(err)
	}
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
