package zero

import (
	"bytes"
	"compress/gzip"
	_ "embed"
	"fmt"
	"html/template"
	"net/http"

	"github.com/timefactoryio/pathless/zero/frames"
	"github.com/timefactoryio/pathless/zero/panels"
)

//go:embed zero/pathless.html
var pathlessHTML string

//go:embed zero/universe.html
var universeHTML []byte

type Zero struct {
	PathlessHTML []byte
	UniverseHTML []byte
	PathlessURL  string
	CircuitURL   string
	Templates    *Templates
	mux          *http.ServeMux
}

func NewZero(args ...string) *Zero {
	z := &Zero{
		UniverseHTML: universeHTML,
		Templates:    NewTemplates(),
		mux:          http.NewServeMux(),
	}
	switch len(args) {
	case 0:
		z.PathlessURL = "*"
		z.CircuitURL = "http://localhost:1001"
	case 2:
		z.PathlessURL = "https://" + args[0]
		z.CircuitURL = "https://" + args[1]
	default:
		panic(fmt.Sprintf(
			"zero.NewZero: expected 0 or 2 arguments, got %d",
			len(args),
		))
	}
	z.pathless()
	z.mux.HandleFunc("/", z.handlePathless)
	return z
}

// Serve starts the pathless shell server (the pre-rendered client page) on :1000.
func (z *Zero) Serve() {
	http.ListenAndServe(":1000", z.mux)
}

func (z *Zero) handlePathless(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" || r.URL.RawQuery != "" {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Encoding", "gzip")
	w.Write(z.PathlessHTML)
}

func (z *Zero) pathless() {
	tmpl := template.Must(template.New("pathless").Parse(pathlessHTML))

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, map[string]string{
		"CIRCUIT": z.CircuitURL,
	}); err != nil {
		panic(err)
	}

	z.PathlessHTML = zip(buf.Bytes())
}

type Templates struct {
	Frames *frames.Frames
	Panels *panels.Panels
}

func NewTemplates() *Templates {
	return &Templates{
		Frames: frames.NewFrames(),
		Panels: panels.NewPanels(),
	}
}

// zip gzip-compresses data once at build time so it can be written as-is on every request.
func zip(data []byte) []byte {
	var gzBuf bytes.Buffer
	w, _ := gzip.NewWriterLevel(&gzBuf, gzip.BestCompression)
	w.Write(data)
	w.Close()
	return gzBuf.Bytes()
}
