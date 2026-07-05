package zero

import (
	"bytes"
	"compress/gzip"
	_ "embed"
	"html/template"
	"regexp"
	"strings"
)

//go:embed pathless.html
var pathlessHtml string

//go:embed universe.html
var universeHtml []byte

// Zero holds the compiled HTML shell (Pathless) and the universe payload.
// Origin is the CORS-allowed root domain; Circuit is the API endpoint URL
// baked into the shell at build time. Universe is the single item-0 blob:
// the plane DOM plus its folded input/keyboard modules.
type Zero struct {
	Pathless []byte
	Universe []byte
	Origin   string
	Circuit  string
}

// NewZero constructs Zero from an origin and circuit host.
func NewZero(origin, circuit string) *Zero {
	if origin == "" {
		origin = "*"
	}
	if circuit == "" {
		circuit = "http://localhost:1001"
	}
	tmpl := template.Must(template.New("pathless").Parse(pathlessHtml))
	var b strings.Builder
	if err := tmpl.Execute(&b, map[string]string{"CIRCUIT": circuit}); err != nil {
		panic(err)
	}
	return &Zero{
		Pathless: minify(b.String()),
		Universe: universeHtml,
		Origin:   origin,
		Circuit:  circuit,
	}
}

// minify: <style> -> <script> -> <html>
// minify aggressively strips pathless.html's <style>/<script> markup down to
// its minimum byte size, then gzip-compresses the result. This is hand-tuned
// for pathless.html's exact content — not a general-purpose minifier.
func minify(html string) []byte {
	html = regexp.MustCompile(`<style>([\s\S]*?)</style>`).ReplaceAllStringFunc(html, func(s string) string {
		s = regexp.MustCompile(`\s*([{}:;,>+~])\s*`).ReplaceAllString(s, "$1")
		return strings.ReplaceAll(s, ";}", "}")
	})
	html = regexp.MustCompile(`<script>([\s\S]*?)</script>`).ReplaceAllStringFunc(html, func(s string) string {
		s = regexp.MustCompile(`\s*([=+\-*/<>!&|?:,;{}()\[\]])\s*`).ReplaceAllString(s, "$1")
		return strings.ReplaceAll(s, ";}", "}")
	})
	html = regexp.MustCompile(`>\s+<`).ReplaceAllString(html, "><")
	html = regexp.MustCompile(`\s+`).ReplaceAllString(html, " ")
	html = strings.ReplaceAll(html, " />", ">")
	html = strings.TrimSpace(html)

	var buf bytes.Buffer
	gz, _ := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	if _, err := gz.Write([]byte(html)); err != nil {
		panic("gzip write error: " + err.Error())
	}
	if err := gz.Close(); err != nil {
		panic("gzip close error: " + err.Error())
	}
	return buf.Bytes()
}
