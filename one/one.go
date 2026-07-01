package one

import (
	"net/http"

	"github.com/timefactoryio/pathless/fx"
	"github.com/timefactoryio/pathless/zero"
)

type One struct {
	*zero.Zero
	*fx.Fx
	pathless *http.ServeMux
	circuit  *http.ServeMux
}

func NewOne(z *zero.Zero, f *fx.Fx) *One {
	o := &One{
		Zero:     z,
		Fx:       f,
		pathless: http.NewServeMux(),
		circuit:  http.NewServeMux(),
	}
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

func (o *One) wire(path string, data []byte) {
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
	r := &fx.Response{
		Values: []*fx.Value{
			{Name: "universe"},
			{Name: "coordinates"},
			{Name: "input"},
			{Name: "keyboard"},
		},
		Data: [][]byte{o.Universe, o.Coordinates, o.Input, o.Keyboard},
	}
	for _, b := range o.Frames() {
		r.Values = append(r.Values, &fx.Value{})
		r.Data = append(r.Data, b)
	}
	o.wire("/", o.Fx.Compress(o.Fx.Wire(r)))
	for key, data := range o.Fx.Routes {
		o.wire("/"+key, data)
	}
	go http.ListenAndServe(":1001", o.cors(o.circuit))
	http.ListenAndServe(":1000", o.pathless)
}
