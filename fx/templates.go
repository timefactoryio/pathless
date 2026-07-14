package fx

import (
	"bytes"

	_ "embed"
	"fmt"
	"html"
	"html/template"
	"path/filepath"
	"strings"

	"github.com/timefactoryio/markdown"
)

//go:embed frames/home.html
var homeHtml string

//go:embed frames/slides.html
var slidesHtml string

//go:embed frames/text.html
var textHtml string

//go:embed panels/keyboard.html
var keyboardHtml string

func (f *Fx) Home(logo, heading string) {
	tmpl, err := template.New("home").Parse(homeHtml)
	if err != nil {
		return
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, map[string]any{
		"LOGO":    f.Logo(logo),
		"HEADING": heading,
	}); err != nil {
		return
	}
	f.Frames = append(f.Frames, f.build(buf.String()))

}

func (f *Fx) Logo(path string) template.HTML {
	ext := filepath.Ext(path)
	if strings.ToLower(ext) == ".svg" {
		if v, err := f.ToValue(path); err == nil {
			return template.HTML(v.Data)
		}
	}

	attr, src := "src", path
	if !strings.HasPrefix(path, "http://") && !strings.HasPrefix(path, "https://") {
		if v, err := f.ToValue(path); err == nil {
			// data-src is the route key; the client's p.source prepends
			// window.circuit when it lazily fetches the image.
			attr, src = "data-src", f.Route(filepath.Base(path), v)
		}
	}
	alt := strings.TrimSuffix(filepath.Base(path), ext)
	return template.HTML(fmt.Sprintf(`<img %s="%s" alt="%s">`,
		attr, html.EscapeString(src), html.EscapeString(alt),
	))
}

// Markdown returns the configured goldmark instance for rendering markdown to HTML.
func (f *Fx) Text(path string) {
	v, err := f.ToValue(path)
	if err != nil {
		return
	}
	var md bytes.Buffer
	if err := markdown.New("").Convert(v.Data, &md); err != nil {
		return
	}
	tmpl, err := template.New("text").Parse(textHtml)
	if err != nil {
		return
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, map[string]any{
		"MARKDOWN": template.HTML(md.String()),
	}); err != nil {
		return
	}
	f.Frames = append(f.Frames, f.build(buf.String()))
}

func (f *Fx) Slides(dir string) {
	base := filepath.Base(dir)
	if v, err := f.ToValue(dir); err == nil {
		f.Route(base, v)
	}
	tmpl, err := template.New("slides").Parse(slidesHtml)
	if err != nil {
		return
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, map[string]string{"PREFIX": base}); err != nil {
		return
	}
	f.Frames = append(f.Frames, f.build(buf.String()))
}

// Keyboard builds the default keyboard panel frame — an example of a
// self-contained panel frame template, same shape as Home/Text/Slides.
// Pass the result to Panels(...) to register it.
func (f *Fx) Keyboard() {
	f.Panels = append(f.Panels, f.build(keyboardHtml))
}
