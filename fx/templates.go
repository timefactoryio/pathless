package fx

import (
	"bytes"
	"embed"

	_ "embed"
	"fmt"
	"html"
	"html/template"
	"path/filepath"
	"strings"

	"github.com/timefactoryio/markdown"
)

//go:embed frames panels
var templatesFS embed.FS

var templates = template.Must(template.ParseFS(templatesFS, "frames/*.html", "panels/*.html"))

func (f *Fx) Home(logo, heading string) {
	var buf bytes.Buffer
	if err := templates.ExecuteTemplate(&buf, "home.html", map[string]any{
		"LOGO":    f.Logo(logo),
		"HEADING": heading,
	}); err != nil {
		return
	}
	f.Frames.Inputs = append(f.Frames.Inputs, f.build(buf.String()))
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

// Markdown returns the configured goldmark instance for rendering markdown to HTML.
func (f *Fx) Text(path string) {
	v, err := f.Input(path)
	if err != nil {
		return
	}
	var md bytes.Buffer
	if err := markdown.New("").Convert(v.Data, &md); err != nil {
		return
	}
	var buf bytes.Buffer
	if err := templates.ExecuteTemplate(&buf, "text.html", map[string]any{
		"MARKDOWN": template.HTML(md.String()),
	}); err != nil {
		return
	}
	f.Frames.Inputs = append(f.Frames.Inputs, f.build(buf.String()))
}

func (f *Fx) Slides(dir string) {
	base := filepath.Base(dir)
	if v, err := f.Input(dir); err == nil {
		f.Route(base, v)
	}
	var buf bytes.Buffer
	if err := templates.ExecuteTemplate(&buf, "slides.html", map[string]string{"PREFIX": base}); err != nil {
		return
	}
	f.Frames.Inputs = append(f.Frames.Inputs, f.build(buf.String()))
}

// Keyboard builds the default keyboard panel frame — an example of a
// self-contained panel frame template, same shape as Home/Text/Slides.
// Pass the result to Panels(...) to register it.
func (f *Fx) Keyboard() {
	kb, err := templatesFS.ReadFile("panels/keyboard.html")
	if err != nil {
		return
	}
	f.Panels.Inputs = append(f.Panels.Inputs, f.build(string(kb)))
}
