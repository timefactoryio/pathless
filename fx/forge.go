package fx

import (
	"fmt"
	"html"
	"html/template"
	"regexp"
	"strings"

	math "github.com/litao91/goldmark-mathjax"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	h "github.com/yuin/goldmark/renderer/html"
)

type One template.HTML

var (
	style  = regexp.MustCompile(`(?s)<style>(.*?)</style>`)
	script = regexp.MustCompile(`(?s)<script>(.*?)</script>`)
)

func NewForge() Forge {
	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM, math.MathJax),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
			parser.WithAttribute(),
			parser.WithBlockParsers(),
			parser.WithInlineParsers(),
		),
		goldmark.WithRendererOptions(
			h.WithHardWraps(),
			h.WithXHTML(),
		),
	)
	return &forge{
		frames: []*One{},
		Md:     &md,
	}
}

type forge struct {
	frames []*One
	Md     *goldmark.Markdown
}

type Forge interface {
	Build(class string, elements ...*One)
	Builder(class string, elements ...*One) *One
	Frames(frame ...*One) [][]byte
	Markdown() *goldmark.Markdown
	HTML(raw string) *One
	JS(js string) One
	CSS(css string) One
	Elem(tag, text string, attrs ...Attr) *One
	Block(tag string, attrs Attr, children ...*One) *One
	Void(tag string, attrs Attr) *One
}

// Attr is a map of HTML attribute key-value pairs passed to any element builder.
type Attr map[string]string

func (f *forge) Build(class string, elements ...*One) {
	f.Frames(f.Builder(class, elements...))
}

func (f *forge) Builder(class string, elements ...*One) *One {
	var b strings.Builder
	if class != "" {
		fmt.Fprintf(&b, `<div class="%s">`, html.EscapeString(class))
	}
	for _, el := range elements {
		b.WriteString(string(*el))
	}
	if class != "" {
		b.WriteString("</div>")
	}
	cleaned := f.consolidateAssets(b.String())
	result := One(template.HTML(cleaned))
	return &result
}

func (f *forge) Frames(frame ...*One) [][]byte {
	if len(frame) > 0 && frame[0] != nil {
		f.frames = append(f.frames, frame[0])
	}
	out := make([][]byte, 0, len(f.frames))
	for _, fr := range f.frames {
		if fr != nil {
			out = append(out, []byte(*fr))
		}
	}
	return out
}

func (f *forge) consolidateAssets(s string) string {
	if styleMatches := style.FindAllStringSubmatch(s, -1); len(styleMatches) > 1 {
		var sb strings.Builder
		for _, m := range styleMatches {
			sb.WriteString(m[1])
			sb.WriteByte('\n')
		}
		s = fmt.Sprintf("<style>%s</style>%s", sb.String(), style.ReplaceAllString(s, ""))
	}
	if scriptMatches := script.FindAllStringSubmatch(s, -1); len(scriptMatches) > 1 {
		var sb strings.Builder
		for _, m := range scriptMatches {
			sb.WriteString(m[1])
			sb.WriteByte('\n')
		}
		s = fmt.Sprintf("%s<script>%s</script>", script.ReplaceAllString(s, ""), sb.String())
	}
	return s
}

func renderAttrs(a Attr) string {
	if len(a) == 0 {
		return ""
	}
	var b strings.Builder
	for k, v := range a {
		fmt.Fprintf(&b, ` %s="%s"`, html.EscapeString(k), html.EscapeString(v))
	}
	return b.String()
}

// Markdown returns the configured goldmark instance for rendering markdown to HTML.
func (f *forge) Markdown() *goldmark.Markdown {
	return f.Md
}

// HTML wraps a raw HTML string as a trusted One value without escaping.
// Use only with content you control — caller is responsible for safety.
func (f *forge) HTML(raw string) *One {
	o := One(template.HTML(raw))
	return &o
}

// JS wraps a raw JavaScript string in a <script> tag without escaping.
func (f *forge) JS(js string) One {
	var b strings.Builder
	b.WriteString(`<script>`)
	b.WriteString(js)
	b.WriteString(`</script>`)
	return One(template.HTML(b.String()))
}

// CSS wraps a raw CSS string in a <style> tag without escaping.
func (f *forge) CSS(css string) One {
	var b strings.Builder
	b.WriteString(`<style>`)
	b.WriteString(css)
	b.WriteString(`</style>`)
	return One(template.HTML(b.String()))
}

// Elem builds a paired tag with escaped text content: <tag attrs>text</tag>.
// Covers h1-h6, p, span, strong, em, code, button, a, li, td, label, etc.
// Attrs are optional.
func (f *forge) Elem(tag, text string, attrs ...Attr) *One {
	a := Attr{}
	if len(attrs) > 0 {
		a = attrs[0]
	}
	o := One(template.HTML(fmt.Sprintf(
		"<%s%s>%s</%s>",
		tag, renderAttrs(a), html.EscapeString(text), tag,
	)))
	return &o
}

// Block builds a paired tag with zero or more child One nodes: <tag attrs>children</tag>.
// Covers div, section, article, nav, ul, ol, table, tr, video, audio, canvas, etc.
// Pass nil for attrs when no attributes are needed.
func (f *forge) Block(tag string, attrs Attr, children ...*One) *One {
	var b strings.Builder
	fmt.Fprintf(&b, "<%s%s>", tag, renderAttrs(attrs))
	for _, child := range children {
		if child != nil {
			b.WriteString(string(*child))
		}
	}
	fmt.Fprintf(&b, "</%s>", tag)
	o := One(template.HTML(b.String()))
	return &o
}

// Void builds a self-closing tag with no children or text: <tag attrs/>.
// Covers img, input, br, hr, meta, link, source, embed, etc.
func (f *forge) Void(tag string, attrs Attr) *One {
	o := One(template.HTML(fmt.Sprintf("<%s%s/>", tag, renderAttrs(attrs))))
	return &o
}
