package zero

import (
	"bytes"
	_ "embed"
	"html/template"
	"log"
	"regexp"
	"strings"
)

//go:embed pathless.html
var pathlessHtml string

//go:embed universe.html
var universeHtml []byte

//go:embed frames/home.html
var homeHtml string

//go:embed frames/slides.html
var slidesHtml string

//go:embed frames/text.html
var textHtml string

//go:embed panels/keyboard.html
var keyboardHtml string

// Zero compiles the two browser-runtime assets every request is built from:
// Pathless, the HTML shell, and Universe, the client controller payload. The
// circuit URL is baked into the shell at build time (as window.circuit) and
// not retained — nothing reads it after compilation.
type Zero struct {
	Pathless    []byte
	Universe    []byte
	PathlessURL string
	CircuitURL  string
	*Templates
}

var (
	styleTag    = regexp.MustCompile(`<style>([\s\S]*?)</style>`)
	scriptTag   = regexp.MustCompile(`<script>([\s\S]*?)</script>`)
	styleSpace  = regexp.MustCompile(`\s*([{}:;,>+~])\s*`)
	scriptSpace = regexp.MustCompile(`\s*([=+\-*/<>!&|?:,;{}()\[\]])\s*`)
	tagGap      = regexp.MustCompile(`>\s+<`)
	whitespace  = regexp.MustCompile(`\s+`)
)

// NewZero constructs Zero, executing and minifying the HTML shell (Pathless)
// with circuit baked in, and carrying the universe payload untouched.
//
// Universe needs no consolidation — it is a single, already-wrapped
// <script> with no <style>, so it is served as-is (one wraps it in a wire
// Value at serve time).
func NewZero(args ...string) *Zero {
	var pathlessUrl, circuitURL string
	switch len(args) {
	case 0:
		pathlessUrl = "*"
		circuitURL = "http://localhost:1001"
	case 2:
		pathlessUrl = "https://" + args[0]
		circuitURL = "https://" + args[1]
	default:
		log.Fatalf("NewPathless: expected 0 or 2 arguments, got %d", len(args))
	}
	return &Zero{
		Pathless:    pathless(circuitURL),
		Universe:    universeHtml,
		Templates:   NewTemplates(),
		PathlessURL: pathlessUrl,
		CircuitURL:  circuitURL,
	}
}

// pathless executes the shell template with circuit baked in, then
// minifies it; hand-tuned for pathless.html's exact content — not a
// general-purpose minifier.
func pathless(circuit string) []byte {
	tmpl := template.Must(template.New("pathless").Parse(pathlessHtml))
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, map[string]string{"CIRCUIT": circuit}); err != nil {
		panic(err)
	}

	html := buf.String()
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

type Templates struct {
	Frames *Frames
	Panels *Panels
}

func NewTemplates() *Templates {
	return &Templates{
		Frames: frames(),
		Panels: panels(),
	}
}

type Frames struct {
	Home   string
	Slides string
	Text   string
}

func frames() *Frames {
	return &Frames{
		Home:   homeHtml,
		Slides: slidesHtml,
		Text:   textHtml,
	}
}

type Panels struct {
	Keyboard string
}

func panels() *Panels {
	return &Panels{
		Keyboard: keyboardHtml,
	}
}
