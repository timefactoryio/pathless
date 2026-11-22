package zero

import (
	"bytes"
	"compress/gzip"
	_ "embed"
	"html/template"
	"io"

	"github.com/timefactoryio/pathless/fx"
)

//go:embed zero.html
var origin string

//go:embed pathless.js
var pathless template.JS

//go:embed input.js
var input template.JS

var zero = template.Must(template.New("").Parse(origin))

type Zero struct {
	fx.Fx
	Domain   string
	Path     string
	Pathless []byte
}

func NewZero(domain, path string, compressed ...bool) *Zero {
	return &Zero{
		Pathless: execute(path, len(compressed) > 0 && compressed[0]),
		Path:     path,
		Domain:   domain,
		Fx:       fx.NewFx(),
	}
}

func execute(path string, compressed bool) []byte {
	var buf bytes.Buffer
	var writer io.Writer = &buf
	if compressed {
		var err error
		writer, err = gzip.NewWriterLevel(&buf, gzip.BestCompression)
		if err != nil {
			panic(err)
		}
	} else {
		buf.Grow(len(origin) + len(pathless) + len(input) + len(path))
	}
	if err := zero.Execute(writer, struct {
		PATH     string
		PATHLESS template.JS
		INPUT    template.JS
	}{path, pathless, input}); err != nil {
		panic(err)
	}
	if compressor, ok := writer.(*gzip.Writer); ok {
		if err := compressor.Close(); err != nil {
			panic(err)
		}
	}
	return buf.Bytes()
}
