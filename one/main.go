package one

import (
	"net/http"

	"github.com/timefactoryio/pathless/fx"
	"github.com/timefactoryio/pathless/zero"
)

type One struct {
	*zero.Zero
	*fx.Fx
	Pathless *http.ServeMux
	Frame    *http.ServeMux
	Hello    []byte
}

func NewOne(apiUrl string) *One {
	o := &One{
		Zero:     zero.NewZero(apiUrl),
		Fx:       fx.NewFx(apiUrl),
		Pathless: http.NewServeMux(),
		Frame:    http.NewServeMux(),
	}
	o.Pathless.HandleFunc("/", o.HandlePathless)
	return o
}

func binaryHandler(data []byte) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Encoding", "gzip")
		w.Write(data)
	}
}

func (o *One) HandlePathless(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" || r.URL.RawQuery != "" {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Encoding", "gzip")
	w.Write(o.Zero.One)
}

func (o *One) BuildHello() {
	var values []*fx.Value
	for _, b := range o.Frames() {
		values = append(values, &fx.Value{Data: b})
	}
	o.Hello = fx.Compress(fx.Encode(values))
}

// func (o *One) BuildHello() {
// 	values := []*fx.Value{
// 		{Data: o.Input},
// 		{Data: o.Layout},
// 		{Data: o.Keyboard},
// 	}
// 	for _, b := range o.Frames() {
// 		values = append(values, &fx.Value{Data: b})
// 	}
// 	o.Hello = fx.Compress(fx.Encode(values))
// }

func (o *One) Serve() {
	o.BuildHello()
	o.Frame.Handle("/", binaryHandler(o.Hello))
	o.Register()
	go http.ListenAndServe(":1001", o.Frame)
	http.ListenAndServe(":1000", o.Pathless)
}

func (o *One) Register() {
	for key, data := range o.Fx.Routes {
		o.Frame.Handle("/"+key, binaryHandler(data))
	}
}
