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

//go:embed input.html
var inputHtml []byte

//go:embed keyboard.html
var keyboardHtml []byte

// Zero holds the compiled HTML shell and shared assets.
// Origin is the CORS-allowed root domain; Circuit is the API endpoint URL
// baked into the HTML at build time.
type Zero struct {
	One      []byte
	Input    []byte
	Keyboard []byte
	Origin   string
	Circuit  string
}

// NewZero constructs Zero from a single host string.
// See resolve for how origin and circuit are derived.
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
		One:      minify(b.String()),
		Origin:   origin,
		Circuit:  circuit,
		Input:    inputHtml,
		Keyboard: keyboardHtml,
	}
}

// minify: <style> -> <script> -> <html>
func minify(html string) []byte {
	html = regexp.MustCompile(`<style>([\s\S]*?)</style>`).ReplaceAllStringFunc(html, func(s string) string {
		s = regexp.MustCompile(`/\*[\s\S]*?\*/`).ReplaceAllString(s, "")
		s = regexp.MustCompile(`\s*([{}:;,>+~])\s*`).ReplaceAllString(s, "$1")
		s = regexp.MustCompile(`;\s*}`).ReplaceAllString(s, "}")
		s = regexp.MustCompile(`\s+`).ReplaceAllString(s, " ")
		s = strings.ReplaceAll(s, "0px", "0")
		s = strings.ReplaceAll(s, "0em", "0")
		s = strings.ReplaceAll(s, "0%", "0")
		s = strings.ReplaceAll(s, " 0 0 0 0", " 0")
		s = strings.ReplaceAll(s, ":0 0 0 0", ":0")
		s = strings.ReplaceAll(s, "<style> ", "<style>")
		s = strings.ReplaceAll(s, " </style>", "</style>")
		return s
	})
	html = regexp.MustCompile(`<script>([\s\S]*?)</script>`).ReplaceAllStringFunc(html, func(s string) string {
		s = regexp.MustCompile(`//[^\n]*\n`).ReplaceAllString(s, "\n")
		s = regexp.MustCompile(`/\*[\s\S]*?\*/`).ReplaceAllString(s, "")
		s = regexp.MustCompile(`\s*([=+\-*/<>!&|?:,;{}()\[\]])\s*`).ReplaceAllString(s, "$1")
		for _, kw := range []string{"const", "let", "var", "return", "async", "await", "function", "if", "else", "for", "of", "in", "new", "throw", "typeof", "instanceof"} {
			s = strings.ReplaceAll(s, kw+"(", kw+" (")
			s = strings.ReplaceAll(s, kw+"{", kw+" {")
		}
		s = regexp.MustCompile(`\n+`).ReplaceAllString(s, "\n")
		s = regexp.MustCompile(`\t+`).ReplaceAllString(s, "")
		s = regexp.MustCompile(`  +`).ReplaceAllString(s, " ")
		s = regexp.MustCompile(`\n\s*\n`).ReplaceAllString(s, "\n")
		s = regexp.MustCompile(`([;{},])\n`).ReplaceAllString(s, "$1")
		s = regexp.MustCompile(`\n([;})])`).ReplaceAllString(s, "$1")
		s = strings.ReplaceAll(s, "<script>\n", "<script>")
		s = strings.ReplaceAll(s, "\n</script>", "</script>")
		return s
	})
	html = regexp.MustCompile(`<!--[\s\S]*?-->`).ReplaceAllString(html, "")
	html = regexp.MustCompile(`>\s+<`).ReplaceAllString(html, "><")
	html = regexp.MustCompile(`\s+`).ReplaceAllString(html, " ")
	html = strings.ReplaceAll(html, " />", ">")
	html = strings.ReplaceAll(html, "\" >", "\">")
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
