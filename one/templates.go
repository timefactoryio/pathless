package one

import (
	"bytes"
	"fmt"
	"html"
	"html/template"
	"path/filepath"
	"strings"

	"github.com/timefactoryio/markdown"
)

func (o *One) Home(logo, heading string) {
	tmpl := template.Must(template.New("home").Parse(o.Templates.Frames.Home))
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, map[string]any{
		"LOGO":    o.Logo(logo),
		"HEADING": heading,
	}); err != nil {
		return
	}
	o.Frame(buf.String())
}

func (o *One) Logo(path string) template.HTML {
	remote := strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://")
	ext := filepath.Ext(path)
	alt := strings.TrimSuffix(filepath.Base(path), ext)

	if !remote || strings.ToLower(ext) == ".svg" {
		if v, err := o.Input(path); err == nil {
			if v.Type == "image/svg+xml" {
				return template.HTML(v.Data)
			}
			return template.HTML(fmt.Sprintf(`<img data-src="%s" alt="%s">`,
				html.EscapeString(o.Route(filepath.Base(path), v)), html.EscapeString(alt)))
		}
	}
	return template.HTML(fmt.Sprintf(`<img src="%s" alt="%s">`,
		html.EscapeString(path), html.EscapeString(alt)))
}

func (o *One) Text(path string) {
	v, err := o.Input(path)
	if err != nil {
		return
	}
	var md bytes.Buffer
	if err := markdown.New("").Convert(v.Data, &md); err != nil {
		return
	}
	tmpl := template.Must(template.New("text").Parse(o.Templates.Frames.Text))
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, map[string]any{
		"MARKDOWN": template.HTML(md.String()),
	}); err != nil {
		return
	}
	o.Frame(buf.String())
}

func (o *One) Slides(dir string) {
	base := filepath.Base(dir)
	o.Route(base, o.list(dir)...)
	tmpl := template.Must(template.New("slides").Parse(o.Templates.Frames.Slides))
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, map[string]string{"PREFIX": base}); err != nil {
		return
	}
	o.Frame(buf.String())
}

// Keyboard builds the default keyboard panel frame from Zero's embedded panel HTML.
func (o *One) Keyboard() {
	o.Panel(o.Templates.Panels.Keyboard)
}
