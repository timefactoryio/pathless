package fx

import (
	"bytes"

	"fmt"
	"html"
	"html/template"
	"path/filepath"
	"strings"
)

func (fx *Fx) Home(logo, heading string) {
	logoEmbed := fx.ToBytes(logo)
	if logoEmbed == nil {
		return
	}

	tmpl := template.Must(template.New("home").Parse(fx.HomeTemplate))

	var buf strings.Builder
	if err := tmpl.Execute(&buf, map[string]template.HTML{
		"LOGO":    template.HTML(string(logoEmbed)),
		"HEADING": template.HTML(heading),
	}); err != nil {
		return
	}

	result := One(template.HTML(buf.String()))
	fx.Build("", &result)
}

func (fx *Fx) Text(path string) {
	content := fx.ToBytes(path)
	if content == nil {
		return
	}

	var buf bytes.Buffer
	if err := (*fx.Markdown()).Convert(content, &buf); err != nil {
		return
	}

	html := buf.String()
	html = strings.ReplaceAll(html, "<p><img", "<img")
	html = strings.ReplaceAll(html, "\"></p>", "\">")
	html = strings.ReplaceAll(html, "\" /></p>", "\" />")
	html = strings.ReplaceAll(html, "\"/></p>", "\"/>")

	markdown := One(template.HTML(html))
	textTmpl := One(template.HTML(fx.TextTemplate))
	fx.Build("text", &markdown, &textTmpl)
}

func (fx *Fx) Slides(dir string) {
	prefix := fx.Reader(dir)
	tmpl, err := template.New("slides").Parse(fx.SlidesTemplate)
	if err != nil {
		return
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, map[string]string{"PREFIX": prefix}); err != nil {
		return
	}

	result := One(template.HTML(buf.String()))
	fx.Build("slides", &result)
}

func (fx *Fx) App(title, url string) {
	tmpl, err := template.New("frame").Parse(fx.AppTemplate)
	if err != nil {
		return
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, map[string]string{
		"TITLE": title,
		"URL":   url,
	}); err != nil {
		return
	}
	result := One(template.HTML(buf.String()))
	fx.Build("app", &result)
}

func (fx *Fx) Logo(path string) template.HTML {
	if strings.ToLower(filepath.Ext(path)) == ".svg" {
		if b := fx.ToBytes(path); b != nil {
			return template.HTML(b)
		}
		return ""
	}

	if !strings.HasPrefix(path, "http://") && !strings.HasPrefix(path, "https://") {
		fx.Read(path, "")
		name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		path = fx.APIURL + "/" + name
	}

	alt := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	return template.HTML(fmt.Sprintf(`<img src="%s" alt="%s">`,
		html.EscapeString(path),
		html.EscapeString(alt),
	))
}
