package one

import (
	"bytes"

	_ "embed"
	"fmt"
	"html"
	"html/template"
	"path/filepath"
	"strings"
)

//go:embed templates/home.html
var homeHtml string

//go:embed templates/slides.html
var slidesHtml string

//go:embed templates/text.html
var textHtml string

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

func (o *One) Text(path string) {
	content, err := o.ToBytes(path)
	if err != nil {
		return
	}
	var md bytes.Buffer
	if err := o.Markdown().Convert(content, &md); err != nil {
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
