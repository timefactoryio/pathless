package one

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

//go:embed templates/home.html
var homeHtml string

//go:embed templates/slides.html
var slidesHtml string

//go:embed templates/text.html
var textHtml string

//go:embed templates/keyboard.html
var keyboardHtml string

func (o *One) Home(logo, heading string) {
	tmpl, err := template.New("home").Parse(homeHtml)
	if err != nil {
		return
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, map[string]any{
		"LOGO":    o.Logo(logo),
		"HEADING": heading,
	}); err != nil {
		return
	}
	result := template.HTML(buf.String())
	o.Build(&result)
}

func (o *One) Logo(path string) template.HTML {
	ext := filepath.Ext(path)
	if strings.ToLower(ext) == ".svg" {
		if b, err := o.ToBytes(path); err == nil {
			return template.HTML(b)
		}
	}

	attr, src := "src", path
	if !strings.HasPrefix(path, "http://") && !strings.HasPrefix(path, "https://") {
		o.Load(path)
		attr, src = "data-src", o.Origin+"/"+filepath.Base(path)
	}
	alt := strings.TrimSuffix(filepath.Base(path), ext)
	return template.HTML(fmt.Sprintf(`<img %s="%s" alt="%s">`,
		attr, html.EscapeString(src), html.EscapeString(alt),
	))
}

// Markdown returns the configured goldmark instance for rendering markdown to HTML.
func (o *One) Text(path string) {
	content, err := o.ToBytes(path)
	if err != nil {
		return
	}
	var md bytes.Buffer
	if err := markdown.New("").Convert(content, &md); err != nil {
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
	result := template.HTML(buf.String())
	o.Build(&result)
}

func (o *One) Slides(dir string) {
	o.Load(dir)
	base := filepath.Base(dir)
	tmpl, err := template.New("slides").Parse(slidesHtml)
	if err != nil {
		return
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, map[string]string{"PREFIX": base}); err != nil {
		return
	}
	result := template.HTML(buf.String())
	o.Build(&result)
}

// Keyboard builds the default keyboard panel frame — an example of a
// self-contained panel frame template, same shape as Home/Text/Slides.
// Pass the result to Panels(...) to register it.
func (o *One) Keyboard() *template.HTML {
	result := template.HTML(keyboardHtml)
	return o.BuildPanel(&result)
}
