package fx

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/timefactoryio/pathless/zero"
)

type Fx struct {
	z      *zero.Zero
	Input  Input
	frames []*output
	panels []*output
	Routes map[string]Output
	mux    *http.ServeMux
}

func NewFx(z *zero.Zero) *Fx {
	return &Fx{
		z:      z,
		mux:    http.NewServeMux(),
		Input:  NewInput(),
		Routes: make(map[string]Output),
	}
}

func (f *Fx) Frame(path string) error {
	entries, err := f.Input.String(path)
	if err != nil {
		return err
	}
	if len(entries) != 1 {
		return fmt.Errorf("frame %q: expected one entry, got %d", path, len(entries))
	}
	f.frames = append(f.frames, f.build(entries[0]))
	return nil
}

func (f *Fx) Panel(path string) error {
	entries, err := f.Input.String(path)
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
func (f *Fx) build(entry *output) *output {
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

// Route registers a payload to be served at key once Start wires up handlers.
func (f *Fx) Route(key string, payload Output) {
	f.Routes[key] = payload
}

// handle registers path on f's mux as a wire endpoint for output: encoded
// (gzipped) once here, then written from memory on every request.
func (f *Fx) handle(path string, payload Output) {
	data := payload.Encode()
	f.mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Encoding", "gzip")
		w.Write(data)
	})
}

func (f *Fx) Start() {
	f.handle("/", Output{
		{&output{Name: "universe", Type: "text/html", Data: f.z.UniverseHTML}},
		f.frames,
		f.panels,
	})

	for key, payload := range f.Routes {
		f.handle("/"+key, payload)
	}

	if err := http.ListenAndServe(":1001", f.cors(f.mux)); err != nil {
		panic(err)
	}
}

func (f *Fx) cors(next http.Handler) http.Handler {
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
