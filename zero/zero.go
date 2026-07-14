package zero

import (
	_ "embed"
	"html/template"
	"regexp"
	"strings"
)

//go:embed pathless.html
var pathlessHtml string

//go:embed universe.html
var universeHtml string

// Zero compiles the two browser-runtime assets every request is built from:
// Pathless, the HTML shell, and Universe, the client controller payload. The
// circuit URL is baked into the shell at build time (as window.circuit) and
// not retained — nothing reads it after compilation.
type Zero struct {
	Pathless []byte
	Universe []byte
}

// NewZero constructs Zero, compiling the HTML shell (Pathless) against
// circuit and carrying the universe payload untouched.
//
// Pathless is templated with circuit and minified. Universe needs no
// consolidation — it is a single, already-wrapped <script> with no <style>,
// so it is served as-is (one wraps it in a wire Value at serve time).
func NewZero(circuit string) *Zero {
	var b strings.Builder
	tmpl := template.Must(template.New("pathless").Parse(pathlessHtml))
	if err := tmpl.Execute(&b, map[string]string{"CIRCUIT": circuit}); err != nil {
		panic(err)
	}

	return &Zero{
		Pathless: minify(b.String()),
		Universe: []byte(universeHtml),
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
