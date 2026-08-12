// templates.go, moved into package fx
package fx

import (
	"bytes"
	"fmt"
	"html"
	"html/template"
	"path/filepath"
	"strings"

	"github.com/timefactoryio/markdown"
)

func (f *Fx) Home(logo, heading string) error {
	tmpl := template.Must(template.New("home").Parse(f.Templates.Frames.Home))
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, map[string]any{
		"LOGO":    f.Logo(logo),
		"HEADING": heading,
	}); err != nil {
		return err
	}
	f.Frame(buf.String())
	return nil
}

func (f *Fx) Logo(path string) template.HTML {
	remote := strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://")
	ext := filepath.Ext(path)
	alt := strings.TrimSuffix(filepath.Base(path), ext)

	if !remote || strings.ToLower(ext) == ".svg" {
		if v, err := f.Input.String(path); err == nil {
			if v.Type == "image/svg+xml" {
				return template.HTML(string(v.Zero))
			}
			name := filepath.Base(path)
			f.Route(name, v)
			return template.HTML(fmt.Sprintf(`<img data-src="%s" alt="%s">`,
				html.EscapeString(name), html.EscapeString(alt)))
		}
	}
	return template.HTML(fmt.Sprintf(`<img src="%s" alt="%s">`,
		html.EscapeString(path), html.EscapeString(alt)))
}

func (f *Fx) Text(path string) error {
	v, err := f.Input.String(path)
	if err != nil {
		return err
	}
	var md bytes.Buffer
	if err := markdown.New("").Convert(v.Zero, &md); err != nil {
		return err
	}
	tmpl := template.Must(template.New("text").Parse(f.Templates.Frames.Text))
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, map[string]any{
		"MARKDOWN": template.HTML(md.String()),
	}); err != nil {
		return err
	}
	f.Frame(buf.String())
	return nil
}

func (f *Fx) Slides(dir string) error {
	output, err := f.Input.String(dir)
	if err != nil {
		return err
	}
	base := filepath.Base(dir)
	f.Route(base, output)
	tmpl := template.Must(template.New("slides").Parse(f.Templates.Frames.Slides))
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, map[string]string{"PREFIX": base}); err != nil {
		return err
	}
	f.Frame(buf.String())
	return nil
}

// Keyboard builds the default keyboard panel frame from Zero's embedded panel HTML.
func (f *Fx) Keyboard() {
	f.Panel(f.Templates.Panels.Keyboard)
}
