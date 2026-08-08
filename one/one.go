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
	*fx.Fx
	pathless *http.ServeMux
	circuit  *http.ServeMux
}

// NewOne registers the wire endpoints served from :1001. Root is built once
// in Serve (after all content is registered) and served as a static blob,
// exactly like Pathless; every other registered route is served by serve.
func NewOne(fx *fx.Fx) *One {
	o := &One{
		Fx:       fx,
		pathless: http.NewServeMux(),
		circuit:  http.NewServeMux(),
	}
	o.Pathless = zip(o.Pathless)
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

func (o *One) handleRoot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Encoding", "gzip")
	w.Write(o.Universe)
}

// serve registers one wire endpoint: values marshaled into a single
// manifest, gzipped once at startup and written from memory.
func (o *One) serve(path string, values []*fx.Output) {
	data := zip(o.Marshal(values...).Data)
	o.circuit.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Encoding", "gzip")
		w.Write(data)
	})
}

func (o *One) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", o.PathlessURL)
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (o *One) Serve() {
	o.Root()
	o.Universe = zip(o.Universe)
	o.circuit.HandleFunc("/", o.handleRoot)
	for key, values := range o.Fx.Routes {
		o.serve("/"+key, values)
	}
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
