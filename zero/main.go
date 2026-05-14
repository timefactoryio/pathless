package zero

import (
	"bytes"
	"compress/gzip"
	_ "embed"
	"html/template"
	"regexp"
	"strings"
)

//go:embed core/pathless.html
var pathlessHtml string

//go:embed core/input.js
var inputHtml []byte

//go:embed core/layout.js
var layoutHtml []byte

//go:embed core/keyboard.html
var keyboardHtml []byte

type Zero struct {
	One      []byte
	Input    []byte
	Layout   []byte
	Keyboard []byte
	APIURL   string
}

func NewZero(apiURL string) *Zero {
	if apiURL == "" {
		apiURL = "http://localhost:1001"
	}
	z := &Zero{
		APIURL:   apiURL,
		Input:    inputHtml,
		Layout:   layoutHtml,
		Keyboard: keyboardHtml,
	}
	tmpl, err := template.New("pathless").Parse(pathlessHtml)
	if err != nil {
		panic("template parse error: " + err.Error())
	}
	var b strings.Builder
	if err := tmpl.Execute(&b, map[string]string{"APIURL": apiURL}); err != nil {
		panic("template execute error: " + err.Error())
	}
	z.One = z.minify(b.String())
	return z
}

// minify: <style> -> <script> -> <html>
func (z *Zero) minify(html string) []byte {
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
