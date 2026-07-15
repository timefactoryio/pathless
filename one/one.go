package one

import (
	"bytes"
	"compress/gzip"
	"net/http"

	"github.com/timefactoryio/pathless/fx"
)

// One is the HTTP layer. It takes zero's compiled assets and fx's processed
// Values, encodes them into the client wire format, gzips everything once at
// startup, and serves it from memory: the shell on :1000, the wire gateway
// on :1001.
type One struct {
	origin   string
	shell    []byte
	pathless *http.ServeMux
	circuit  *http.ServeMux
}

// NewOne registers the wire endpoints served from :1001. Every route — the
// root and each registered route alike — is exactly one *fx.Output, served by
// serve as Encode(v): the value's type travels in-band in the wire table, so
// the response needs no meaningful Content-Type. The root is just a bundle
// whose children are the universe payload and the frame and panel pools, so
// it needs no special handling. origin is the CORS allow-origin; shell and
// universe are zero's asset bytes; f supplies the frame/panel pools and route
// map.
func NewOne(origin string, shell, universe []byte, f *fx.Fx) *One {
	o := &One{
		origin:   origin,
		shell:    zip(shell),
		pathless: http.NewServeMux(),
		circuit:  http.NewServeMux(),
	}
	o.pathless.HandleFunc("/", o.handlePathless)

	o.serve("/", &fx.Output{Type: "application/x-bundle", Inputs: []*fx.Output{
		{Type: "text/html", Data: universe},
		f.Frames,
		f.Panels,
	}})
	for key, v := range f.Routes {
		o.serve("/"+key, v)
	}
	return o
}

func (o *One) handlePathless(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" || r.URL.RawQuery != "" {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Encoding", "gzip")
	w.Write(o.shell)
}

// serve registers one wire endpoint: Encode(v) — the value and its type,
// self-describing — gzipped once at startup and written from memory. The
// response carries no meaningful Content-Type; the bytes are the whole
// contract, and the client decodes them into a Output.
func (o *One) serve(path string, v *fx.Output) {
	data := zip(Encode(v))
	o.circuit.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Encoding", "gzip")
		w.Write(data)
	})
}

func (o *One) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", o.origin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (o *One) Serve() {
	go http.ListenAndServe(":1001", o.cors(o.circuit))
	http.ListenAndServe(":1000", o.pathless)
}

// zip gzip-compresses data at maximum compression. Bundles are compressed
// once at build time and served directly with Content-Encoding: gzip.
func zip(data []byte) []byte {
	var buf bytes.Buffer
	w, _ := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	w.Write(data)
	w.Close()
	return buf.Bytes()
}
