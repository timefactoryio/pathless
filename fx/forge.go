package fx

import (
	"fmt"
	"html"
	"html/template"
	"regexp"
	"strings"

	"github.com/timefactoryio/markdown"
)

type Frame template.HTML

var (
	style  = regexp.MustCompile(`(?s)<style>(.*?)</style>`)
	script = regexp.MustCompile(`(?s)<script>(.*?)</script>`)
)

func NewForge() Forge {
	return &forge{
		frames: []*Frame{},
		md:     markdown.New(""),
	}
}

type forge struct {
	frames []*Frame
	md     *markdown.Markdown
}

type Forge interface {
	HTML(raw string) *Frame
	Build(class string, elements ...*Frame)
	Builder(class string, elements ...*Frame) *Frame
	Frames(frame ...*Frame) [][]byte
	Markdown() *markdown.Markdown
	JS(js string) *Frame
	CSS(css string) *Frame
	Elem(tag, text string, attrs ...Attr) *Frame
	Block(tag string, attrs Attr, children ...*Frame) *Frame
	Void(tag string, attrs Attr) *Frame
}

// Attr is a map of HTML attribute key-value pairs passed to any element builder.
type Attr map[string]string

// HTML wraps a raw HTML string as a trusted Frame value without escaping.
// Use only with content you control — caller is responsible for safety.
func (f *forge) HTML(raw string) *Frame {
	o := Frame(template.HTML(raw))
	return &o
}

func (f *forge) Build(class string, elements ...*Frame) {
	f.Frames(f.Builder(class, elements...))
}

func (f *forge) Builder(class string, elements ...*Frame) *Frame {
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
	result := Frame(template.HTML(cleaned))
	return &result
}

func (f *forge) Frames(frame ...*Frame) [][]byte {
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

// Markdown returns the configured goldmark instance for rendering markdown to HTML.
func (f *forge) Markdown() *markdown.Markdown {
	return f.md
}

// JS wraps a raw JavaScript string in a <script> tag without escaping.
func (f *forge) JS(js string) *Frame {
	var b strings.Builder
	b.WriteString(`<script>`)
	b.WriteString(js)
	b.WriteString(`</script>`)
	o := Frame(template.HTML(b.String()))
	return &o
}

// CSS wraps a raw CSS string in a <style> tag without escaping.
func (f *forge) CSS(css string) *Frame {
	var b strings.Builder
	b.WriteString(`<style>`)
	b.WriteString(css)
	b.WriteString(`</style>`)
	o := Frame(template.HTML(b.String()))
	return &o
}

// Elem builds a paired tag with escaped text content: <tag attrs>text</tag>.
// Covers h1-h6, p, span, strong, em, code, button, a, li, td, label, etc.
// Attrs are optional.
func (f *forge) Elem(tag, text string, attrs ...Attr) *Frame {
	a := Attr{}
	if len(attrs) > 0 {
		a = attrs[0]
	}
	o := Frame(template.HTML(fmt.Sprintf(
		"<%s%s>%s</%s>",
		tag, renderAttrs(a), html.EscapeString(text), tag,
	)))
	return &o
}

// Block builds a paired tag with zero or more child Frame nodes: <tag attrs>children</tag>.
// Covers div, section, article, nav, ul, ol, table, tr, video, audio, canvas, etc.
// Pass nil for attrs when no attributes are needed.
func (f *forge) Block(tag string, attrs Attr, children ...*Frame) *Frame {
	var b strings.Builder
	fmt.Fprintf(&b, "<%s%s>", tag, renderAttrs(attrs))
	for _, child := range children {
		if child != nil {
			b.WriteString(string(*child))
		}
	}
	fmt.Fprintf(&b, "</%s>", tag)
	o := Frame(template.HTML(b.String()))
	return &o
}

// Void builds a self-closing tag with no children or text: <tag attrs/>.
// Covers img, input, br, hr, meta, link, source, embed, etc.
func (f *forge) Void(tag string, attrs Attr) *Frame {
	o := Frame(template.HTML(fmt.Sprintf("<%s%s/>", tag, renderAttrs(attrs))))
	return &o
}

const (
	openStyle   = "<style>"
	closeStyle  = "</style>"
	openScript  = "<script>"
	closeScript = "</script>"
	openBlock   = "<script>{"
	closeBlock  = "}</script>"
)

func (f *forge) consolidateAssets(s string) string {
	if styles := style.FindAllStringSubmatch(s, -1); len(styles) > 1 {
		var sb strings.Builder
		for _, m := range styles {
			sb.WriteString(m[1])
			sb.WriteByte('\n')
		}
		s = openStyle + sb.String() + closeStyle + style.ReplaceAllString(s, "")
	}

	var scripts strings.Builder
	s = script.ReplaceAllStringFunc(s, func(m string) string {
		content := m[len(openScript) : len(m)-len(closeScript)]
		if t := strings.TrimSpace(content); !strings.HasPrefix(t, "{") {
			scripts.WriteByte('{')
			scripts.WriteString(content)
			scripts.WriteString("}\n")
		} else {
			scripts.WriteString(content)
			scripts.WriteByte('\n')
		}
		return ""
	})
	if scripts.Len() > 0 {
		s += openBlock + scripts.String() + closeBlock
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
