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

func (f *Fx) Home(logo, heading string) {
	tmpl := template.Must(template.New("home").Parse(f.Zero.Frames.Home))
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
	remote := strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://")
	ext := filepath.Ext(path)
	alt := strings.TrimSuffix(filepath.Base(path), ext)

	if !remote || strings.ToLower(ext) == ".svg" {
		if v, err := f.Input(path); err == nil {
			if v.Type == "image/svg+xml" {
				return template.HTML(v.Data)
			}
			return template.HTML(fmt.Sprintf(`<img data-src="%s" alt="%s">`,
				html.EscapeString(f.Route(filepath.Base(path), v)), html.EscapeString(alt)))
		}
	}
	return template.HTML(fmt.Sprintf(`<img src="%s" alt="%s">`,
		html.EscapeString(path), html.EscapeString(alt)))
}

func (f *Fx) Text(path string) {
	v, err := f.Input(path)
	if err != nil {
		return
	}
	var md bytes.Buffer
	if err := markdown.New("").Convert(v.Data, &md); err != nil {
		return
	}
	tmpl := template.Must(template.New("text").Parse(f.Zero.Frames.Text))
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
	f.Route(base, f.list(dir)...)
	tmpl := template.Must(template.New("slides").Parse(f.Zero.Frames.Slides))
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, map[string]string{"PREFIX": base}); err != nil {
		return
	}
	f.Frames = append(f.Frames, f.build(buf.String()))
}

// Keyboard builds the default keyboard panel frame from Zero's embedded panel HTML.
func (f *Fx) Keyboard() {
	f.Panels = append(f.Panels, f.build(f.Zero.Panels.Keyboard))
}
