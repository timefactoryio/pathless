package zero

import (
	_ "embed"
	"html/template"
	"regexp"
	"strings"
)

//go:embed pathless.html
var pathlessHtml string

// Zero holds the circuit host a shell is compiled against. Circuit is the
// API endpoint URL baked into the shell at build time.
type Zero struct {
	Circuit  string
	Pathless []byte
}

// NewZero constructs Zero, compiling the HTML shell (Pathless) against
// circuit.
//
// Pathless is templated with circuit and minified.
func NewZero(circuit string) *Zero {
	var b strings.Builder
	tmpl := template.Must(template.New("pathless").Parse(pathlessHtml))
	if err := tmpl.Execute(&b, map[string]string{"CIRCUIT": circuit}); err != nil {
		panic(err)
	}

	return &Zero{
		Circuit:  circuit,
		Pathless: minify(b.String()),
	}
}

var (
	styleTag    = regexp.MustCompile(`<style>([\s\S]*?)</style>`)
	scriptTag   = regexp.MustCompile(`<script>([\s\S]*?)</script>`)
	styleSpace  = regexp.MustCompile(`\s*([{}:;,>+~])\s*`)
	scriptSpace = regexp.MustCompile(`\s*([=+\-*/<>!&|?:,;{}()\[\]])\s*`)
	tagGap      = regexp.MustCompile(`>\s+<`)
	whitespace  = regexp.MustCompile(`\s+`)
)

// minify: <style> -> <script> -> <html>
// minify aggressively strips pathless.html's <style>/<script> markup down to
// its minimum byte size. This is hand-tuned for pathless.html's exact
// content — not a general-purpose minifier.
func minify(html string) []byte {
	html = styleTag.ReplaceAllStringFunc(html, func(s string) string {
		s = styleSpace.ReplaceAllString(s, "$1")
		return strings.ReplaceAll(s, ";}", "}")
	})
	html = scriptTag.ReplaceAllStringFunc(html, func(s string) string {
		s = scriptSpace.ReplaceAllString(s, "$1")
		return strings.ReplaceAll(s, ";}", "}")
	})
	html = tagGap.ReplaceAllString(html, "><")
	html = whitespace.ReplaceAllString(html, " ")
	html = strings.ReplaceAll(html, " />", ">")
	html = strings.TrimSpace(html)

	return []byte(html)
}
