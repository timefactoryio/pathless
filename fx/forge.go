package fx

import (
	"fmt"
	"html"
	"html/template"
	"os"
	"regexp"
	"strings"

	"github.com/timefactoryio/markdown"
)

var (
	style  = regexp.MustCompile(`(?s)<style>(.*?)</style>`)
	script = regexp.MustCompile(`(?s)<script>(.*?)</script>`)
)

func NewForge() Forge {
	return &forge{
		frames: []*template.HTML{},
		md:     markdown.New(""),
	}
}

type forge struct {
	frames []*template.HTML
	md     *markdown.Markdown
}

type Forge interface {
	HTML(raw string) *template.HTML
	Build(class string, elements ...*template.HTML)
	Builder(class string, elements ...*template.HTML) *template.HTML
	Frames(frame ...*template.HTML) [][]byte
	Markdown() *markdown.Markdown
	JS(path string) *template.HTML
	CSS(css string) *template.HTML
	Elem(tag, text string, attrs ...Attr) *template.HTML
	Block(tag string, attrs Attr, children ...*template.HTML) *template.HTML
	Void(tag string, attrs Attr) *template.HTML
}

// Attr is a map of HTML attribute key-value pairs passed to any element builder.
type Attr map[string]string

// HTML wraps a raw HTML string as a trusted Frame value without escaping.
// Use only with content you control — caller is responsible for safety.
func (f *forge) HTML(raw string) *template.HTML {
	o := template.HTML(raw)
	return &o
}

func (f *forge) Build(class string, elements ...*template.HTML) {
	f.Frames(f.Builder(class, elements...))
}

func (f *forge) Builder(class string, elements ...*template.HTML) *template.HTML {
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
	result := template.HTML(cleaned)
	return &result
}

func (f *forge) Frames(frame ...*template.HTML) [][]byte {
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
func (f *forge) JS(path string) *template.HTML {
	data, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	o := template.HTML("<script>{" + string(data) + "}</script>")
	return &o
}

// CSS wraps a raw CSS string in a <style> tag without escaping.
func (f *forge) CSS(css string) *template.HTML {
	var b strings.Builder
	b.WriteString(`<style>`)
	b.WriteString(css)
	b.WriteString(`</style>`)
	o := template.HTML(b.String())
	return &o
}

// Elem builds a paired tag with escaped text content: <tag attrs>text</tag>.
// Covers h1-h6, p, span, strong, em, code, button, a, li, td, label, etc.
// Attrs are optional.
func (f *forge) Elem(tag, text string, attrs ...Attr) *template.HTML {
	a := Attr{}
	if len(attrs) > 0 {
		a = attrs[0]
	}
	o := template.HTML(fmt.Sprintf(
		"<%s%s>%s</%s>",
		tag, renderAttrs(a), html.EscapeString(text), tag,
	))
	return &o
}

// Block builds a paired tag with zero or more child Frame nodes: <tag attrs>children</tag>.
// Covers div, section, article, nav, ul, ol, table, tr, video, audio, canvas, etc.
// Pass nil for attrs when no attributes are needed.
func (f *forge) Block(tag string, attrs Attr, children ...*template.HTML) *template.HTML {
	var b strings.Builder
	fmt.Fprintf(&b, "<%s%s>", tag, renderAttrs(attrs))
	for _, child := range children {
		if child != nil {
			b.WriteString(string(*child))
		}
	}
	fmt.Fprintf(&b, "</%s>", tag)
	o := template.HTML(b.String())
	return &o
}

// Void builds a self-closing tag with no children or text: <tag attrs/>.
// Covers img, input, br, hr, meta, link, source, embed, etc.
func (f *forge) Void(tag string, attrs Attr) *template.HTML {
	o := template.HTML(fmt.Sprintf("<%s%s/>", tag, renderAttrs(attrs)))
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
