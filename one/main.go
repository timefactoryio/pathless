package one

import (
	"net/http"

	"github.com/timefactoryio/pathless/fx"
	"github.com/timefactoryio/pathless/zero"
)

type One struct {
	Zero     *zero.Zero
	Fx       *fx.Fx
	Pathless *http.ServeMux
	Frame    *http.ServeMux
	Hello    []byte
}

func NewOne(z *zero.Zero, fx *fx.Fx) *One {
	o := &One{
		Zero:     z,
		Fx:       fx,
		Pathless: http.NewServeMux(),
		Frame:    http.NewServeMux(),
	}
	o.Pathless.HandleFunc("/", o.HandlePathless)
	return o
}

func (o *One) BuildHello() {
	objects := [][]byte{
		[]byte(o.Fx.Input),    // [0]
		[]byte(o.Fx.Layout),   // [1]
		[]byte(o.Fx.Keyboard), // [2]
	}
	for _, f := range o.Fx.Frames() {
		if f != nil {
			objects = append(objects, []byte(*f)) // [3+]
		}
	}
	o.Hello = o.Fx.Compress(fx.Encode(objects))
}

func (o *One) HandleHello(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Encoding", "gzip")
	w.Write(o.Hello)
}
