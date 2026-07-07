package one

import (
	"bytes"
	"compress/gzip"
	"net/http"

	"github.com/timefactoryio/pathless/fx"
	"github.com/timefactoryio/pathless/zero"
)

type One struct {
	*zero.Zero
	*fx.Fx
	pathless *http.ServeMux
	circuit  *http.ServeMux
	Pathless []byte
	Universe []byte
}

func NewOne(z *zero.Zero, f *fx.Fx) *One {
	pathless, universe := z.Compile()
	o := &One{
		Zero:     z,
		Fx:       f,
		pathless: http.NewServeMux(),
		circuit:  http.NewServeMux(),
		Pathless: zip(pathless),
		Universe: universe,
	}
	o.Panels(o.Keyboard())
	o.pathless.HandleFunc("/", o.handlePathless)
	return o
}

func (o *One) handlePathless(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" || r.URL.RawQuery != "" {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Encoding", "gzip")
	w.Write(o.Pathless)
}

func (o *One) wire(path string, v *fx.Value) {
	data := zip(o.Encode(v))
	o.circuit.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Encoding", "gzip")
		w.Write(data)
	})
}

func (o *One) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", o.Origin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (o *One) Serve() {
	o.wire("/", &fx.Value{Values: []*fx.Value{
		{Type: "text/html", Data: o.Universe},
		o.Frames(),
		o.Panels(),
	}})

	for key, v := range o.Fx.Routes {
		o.wire("/"+key, v)
	}

	go http.ListenAndServe(":1001", o.cors(o.circuit))
	http.ListenAndServe(":1000", o.pathless)
}

// zip gzip-zipes data at maximum zipion.
// Bundles are ziped once at build time and served directly
// with Content-Encoding: gzip.
func zip(data []byte) []byte {
	var buf bytes.Buffer
	w, _ := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	w.Write(data)
	w.Close()
	return buf.Bytes()
}
