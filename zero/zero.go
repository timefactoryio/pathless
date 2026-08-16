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

//go:embed pathless.html
var pathlessHTML string

//go:embed universe.html
var universeHTML []byte

type Zero struct {
	UniverseHTML []byte
	PathlessURL  string
	circuitURL   string
	Templates    *Templates
	mux          *http.ServeMux
}

type Templates struct {
	Frames *frames.Frames
	Panels *panels.Panels
}

func NewZero(args ...string) *Zero {
	z := &Zero{
		UniverseHTML: universeHTML,
		Templates: &Templates{
			Frames: frames.NewFrames(),
			Panels: panels.NewPanels(),
		},
		mux: http.NewServeMux(),
	}

	switch len(args) {
	case 0:
		z.PathlessURL = "*"
		z.circuitURL = "http://localhost:1001"
	case 2:
		z.PathlessURL = "https://" + args[0]
		z.circuitURL = "https://" + args[1]
	default:
		panic(fmt.Sprintf(
			"zero.NewZero: expected 0 or 2 arguments, got %d",
			len(args),
		))
	}

	z.pathless()
	go func() {
		if err := http.ListenAndServe(":1000", z.mux); err != nil {
			panic(err)
		}
	}()
	return z
}

func (z *Zero) pathless() {
	tmpl := template.Must(template.New("pathless").Parse(pathlessHTML))

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, map[string]string{
		"CIRCUIT": z.circuitURL,
	}); err != nil {
		panic(err)
	}
	payload := z.Zip(buf.Bytes())
	z.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" || r.URL.RawQuery != "" {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Content-Encoding", "gzip")
		w.Write(payload)
	})
}

func (z *Zero) Zip(data []byte) []byte {
	var buf bytes.Buffer
	w, _ := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	w.Write(data)
	w.Close()
	return buf.Bytes()
}
