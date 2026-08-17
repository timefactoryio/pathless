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
	tmpl := template.Must(template.New("home").Parse(f.z.Templates.Frames.Home))
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, map[string]any{
		"LOGO":    f.Logo(logo),
		"HEADING": heading,
	}); err != nil {
		return err
	}
	f.frames = append(f.frames, f.build(&output{Data: buf.Bytes()}))
	return nil
}

func (f *Fx) Logo(path string) template.HTML {
	remote := strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://")
	ext := filepath.Ext(path)
	alt := strings.TrimSuffix(filepath.Base(path), ext)

	if !remote || strings.ToLower(ext) == ".svg" {
		if entries, err := f.Input.String(path); err == nil && len(entries) == 1 {
			entry := entries[0]
			if entry.Type == "image/svg+xml" {
				return template.HTML(string(entry.Data))
			}
			name := filepath.Base(path)
			f.Route(name, Output{entries})
			return template.HTML(fmt.Sprintf(`<img data-src="%s" alt="%s">`,
				html.EscapeString(name), html.EscapeString(alt)))
		}
	}
	return template.HTML(fmt.Sprintf(`<img src="%s" alt="%s">`,
		html.EscapeString(path), html.EscapeString(alt)))
}

func (f *Fx) Text(path string) error {
	entries, err := f.Input.String(path)
	if err != nil {
		return err
	}
	if len(entries) != 1 {
		return fmt.Errorf("text %q: expected one entry, got %d", path, len(entries))
	}
	var md bytes.Buffer
	if err := markdown.New("").Convert(entries[0].Data, &md); err != nil {
		return err
	}
	tmpl := template.Must(template.New("text").Parse(f.z.Templates.Frames.Text))
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, map[string]any{
		"MARKDOWN": template.HTML(md.String()),
	}); err != nil {
		return err
	}
	f.frames = append(f.frames, f.build(&output{Data: buf.Bytes()}))
	return nil
}

func (f *Fx) Slides(dir string) error {
	entries, err := f.Input.String(dir)
	if err != nil {
		return err
	}
	base := filepath.Base(dir)
	f.Route(base, Output{entries})
	tmpl := template.Must(template.New("slides").Parse(f.z.Templates.Frames.Slides))
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, map[string]string{"PREFIX": base}); err != nil {
		return err
	}
	f.frames = append(f.frames, f.build(&output{Data: buf.Bytes()}))
	return nil
}

// Keyboard builds the default keyboard panel frame from Zero's embedded panel HTML.
func (f *Fx) Keyboard() {
	f.panels = append(f.panels, f.build(&output{
		Data: []byte(f.z.Templates.Panels.Keyboard),
	}))
}
