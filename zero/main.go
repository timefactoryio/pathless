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
var universeHtml []byte

// Zero holds the origin and circuit host a shell is compiled against.
// Origin is the CORS-allowed root domain; Circuit is the API endpoint URL
// baked into the shell at build time.
type Zero struct {
	Origin  string
	Circuit string
}

// NewZero constructs Zero from an origin and circuit host.
func NewZero(origin, circuit string) *Zero {
	if origin == "" {
		origin = "*"
	}
	if circuit == "" {
		circuit = "http://localhost:1001"
	}
	return &Zero{
		Origin:  origin,
		Circuit: circuit,
	}
}

// Compile builds the HTML shell (pathless) and returns it alongside the
// universe payload (universe). pathless is templated with z.Circuit,
// minified, and gzip-compressed. universe is the single item-0 blob — the
// plane DOM plus its folded input/keyboard modules — returned raw: fx
// performs its own consolidation pass over it before anything is
// wire-encoded.
func (z *Zero) Compile() (pathless []byte, universe []byte) {
	tmpl := template.Must(template.New("pathless").Parse(pathlessHtml))
	var b strings.Builder
	if err := tmpl.Execute(&b, map[string]string{"CIRCUIT": z.Circuit}); err != nil {
		panic(err)
	}
	return minify(b.String()), universeHtml
}

// minify: <style> -> <script> -> <html>
// minify aggressively strips pathless.html's <style>/<script> markup down to
// its minimum byte size. This is hand-tuned for pathless.html's exact
// content — not a general-purpose minifier.
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

	return []byte(html)
}
