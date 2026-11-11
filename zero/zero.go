package zero

import (
	"bytes"
	_ "embed"
	"html/template"
)

//go:embed zero.html
var origin string

//go:embed pathless.js
var pathlessScript string

//go:embed input.js
var inputScript string

func Origin(circuitURL string) []byte {
	var buf bytes.Buffer
	if err := template.Must(template.New("").Parse(origin)).Execute(&buf, map[string]any{
		"CIRCUIT":  circuitURL,
		"PATHLESS": template.JS(pathlessScript),
		"INPUT":    template.JS(inputScript),
	}); err != nil {
		panic(err)
	}
	return buf.Bytes()
}
