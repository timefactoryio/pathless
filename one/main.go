package one

import (
	"bytes"
	"compress/gzip"
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
		[]byte(o.Fx.Input),
		[]byte(o.Fx.Layout),
		[]byte(o.Fx.Keyboard),
	}
	o.Hello = o.Compress(fx.Encode(append(objects, o.Fx.FrameBytes()...)))
}

func (o *One) HandleHello(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Encoding", "gzip")
	w.Write(o.Hello)
}

func (o *One) Compress(data []byte) []byte {
	var buf bytes.Buffer
	w, _ := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	w.Write(data)
	w.Close()
	return buf.Bytes()
}
