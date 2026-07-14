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

// NewOne assembles the "/" payload — [universe, frames bundle, panels
// bundle] — plus one endpoint per route, encoding and gzipping each once.
// origin is the CORS allow-origin; shell and universe are zero's asset bytes;
// f supplies the frame/panel pools and route map.
func NewOne(origin string, shell, universe []byte, f *fx.Fx) *One {
	o := &One{
		origin:   origin,
		shell:    zip(shell),
		pathless: http.NewServeMux(),
		circuit:  http.NewServeMux(),
	}
	o.pathless.HandleFunc("/", o.handlePathless)

	hello := []*fx.Value{
		{Type: "text/html", Data: universe},
		{Type: "application/x-bundle", Children: f.Frames},
		{Type: "application/x-bundle", Children: f.Panels},
	}
	o.wire("/", Encode(hello...))

	for key, v := range f.Routes {
		o.wire("/"+key, Encode(v))
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

func (o *One) wire(path string, data []byte) {
	data = zip(data)
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
