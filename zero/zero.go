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
var pathlessScript template.JS

//go:embed input.js
var inputScript template.JS

var originTemplate = template.Must(template.New("").Parse(origin))

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
		buf.Grow(len(origin) + len(pathlessScript) + len(inputScript) + len(circuitURL))
	}
	if err := originTemplate.Execute(writer, struct {
		CIRCUIT  string
		PATHLESS template.JS
		INPUT    template.JS
	}{circuitURL, pathlessScript, inputScript}); err != nil {
		panic(err)
	}
	if compressor, ok := writer.(*gzip.Writer); ok {
		if err := compressor.Close(); err != nil {
			panic(err)
		}
	}
	return &Zero{Origin: buf.Bytes()}
}
