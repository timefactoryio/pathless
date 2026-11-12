package zero

import (
	"bytes"
	"compress/gzip"
	_ "embed"
	"html/template"
	"io"
)

//go:embed zero.html
var origin string

//go:embed pathless.js
var pathless template.JS

//go:embed input.js
var input template.JS

var zero = template.Must(template.New("").Parse(origin))

type Zero struct {
	Origin []byte
}

func NewZero(circuitURL string, compressed ...bool) *Zero {
	var buf bytes.Buffer
	var writer io.Writer = &buf
	if len(compressed) > 0 && compressed[0] {
		var err error
		writer, err = gzip.NewWriterLevel(&buf, gzip.BestCompression)
		if err != nil {
			panic(err)
		}
	} else {
		buf.Grow(len(origin) + len(pathless) + len(input) + len(circuitURL))
	}
	if err := zero.Execute(writer, struct {
		CIRCUIT  string
		PATHLESS template.JS
		INPUT    template.JS
	}{circuitURL, pathless, input}); err != nil {
		panic(err)
	}
	if compressor, ok := writer.(*gzip.Writer); ok {
		if err := compressor.Close(); err != nil {
			panic(err)
		}
	}
	return &Zero{Origin: buf.Bytes()}
}
