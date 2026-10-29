package zero

import (
	"bytes"
	"compress/gzip"
	_ "embed"
	"fmt"
	"html/template"
	"net/http"
)

//go:embed zero.html
var page string

//go:embed pathless.js
var pathlessScript string

//go:embed universe.js
var universeScript string

//go:embed input.js
var inputScript string

type Zero struct {
	PathlessURL string
	circuitURL  string
	mux         *http.ServeMux
}

func NewZero(args ...string) *Zero {
	z := &Zero{
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
	tmpl := template.Must(template.New("zero.html").Parse(page))

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "zero.html", map[string]any{
		"CIRCUIT":  z.circuitURL,
		"PATHLESS": template.JS(pathlessScript),
		"UNIVERSE": template.JS(universeScript),
		"INPUT":    template.JS(inputScript),
	}); err != nil {
		panic(err)
	}
	payload := z.zip(buf.Bytes())
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

func (z *Zero) zip(data []byte) []byte {
	var buf bytes.Buffer
	w, _ := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	w.Write(data)
	w.Close()
	return buf.Bytes()
}
