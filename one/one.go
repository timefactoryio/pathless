package one

import (
	"net/http"

	"github.com/timefactoryio/pathless/zero"
)

type One struct {
	*zero.Zero
	Pathless *http.ServeMux
	Path     *http.ServeMux
	frames   []byte
}

func NewOne(pathless, path string) *One {
	return &One{
		Zero:     zero.NewZero(pathless, path, true),
		Pathless: http.NewServeMux(),
		Path:     http.NewServeMux(),
		frames:   nil,
	}
}

func (o *One) origin(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" || r.URL.RawQuery != "" {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Encoding", "gzip")
	w.Write(o.Zero.Pathless)
}

func (o *One) circuit(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/octet-stream")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	w.Header().Set("Content-Encoding", "gzip")
	w.Write(o.frames)
}

func (o *One) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", o.Domain)
		w.Header().Set("Access-Control-Allow-Methods", "GET")
		next.ServeHTTP(w, r)
	})
}

func (o *One) Start() {
	o.frames = o.Frames(true)
	o.Pathless.HandleFunc("/", o.origin)
	o.Path.HandleFunc("/", o.circuit)
	go func() {
		if err := http.ListenAndServe(":1000", o.Pathless); err != nil {
			panic(err)
		}
	}()
	if err := http.ListenAndServe(":1001", o.cors(o.Path)); err != nil {
		panic(err)
	}
}
