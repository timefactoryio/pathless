package fx

import (
	"net/http"
	"regexp"
	"strings"

	"github.com/timefactoryio/pathless/zero"
)

type Fx struct {
	*zero.Zero
	Input  Input
	Frames *Output
	Panels *Output
	Routes map[string]*Output
	mux    *http.ServeMux
}

func NewFx(z *zero.Zero) *Fx {
	return &Fx{
		Zero:   z,
		mux:    http.NewServeMux(),
		Input:  NewInput(),
		Frames: &Output{Name: "frames"},
		Panels: &Output{Name: "panels"},
		Routes: make(map[string]*Output),
	}
}

// Frame registers html as a frame, merging any inline style/script blocks first.
func (f *Fx) Frame(html string) {
	f.Frames.One = append(f.Frames.One, f.Build(html))
}

// Panel registers html as a panel, merging any inline style/script blocks first.
func (f *Fx) Panel(html string) {
	f.Panels.One = append(f.Panels.One, f.Build(html))
}

// Build merges any inline style/script blocks in html into a single leaf Output.
func (f *Fx) Build(html string) *Output {
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
	return &Output{Type: "text/html", Zero: []byte(html)}
}

var (
	styleTag  = regexp.MustCompile(`(?s)<style>(.*?)</style>`)
	scriptTag = regexp.MustCompile(`(?s)<script>(.*?)</script>`)
)

// Route registers output to be served at key once Start wires up handlers.
func (f *Fx) Route(key string, output *Output) {
	f.Routes[key] = output
}

// Handle registers path on f's mux as a wire endpoint for output: encoded
// (gzipped) once here, then written from memory on every request.
func (f *Fx) Handle(path string, output *Output) {
	data := output.Encode()
	f.mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Encoding", "gzip")
		w.Write(data)
	})
}

// Start builds the root Output tree (universe.html's shell markup alongside the
// registered Frames and Panels branches) and registers it, and every Route.
func (f *Fx) Start() {
	root := &Output{
		One: []*Output{
			{Name: "universe", Type: "text/html", Zero: f.UniverseHTML},
			f.Frames,
			f.Panels,
		},
	}
	f.Handle("/", root)
	for key, output := range f.Routes {
		f.Handle("/"+key, output)
	}
}

// Serve starts the circuit (wire) server on :1001 behind CORS.
func (f *Fx) Serve() {
	f.Start()
	http.ListenAndServe(":1001", f.cors(f.mux))
}

func (f *Fx) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", f.PathlessURL)
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
