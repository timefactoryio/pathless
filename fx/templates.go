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
	tmpl := One(template.HTML(fx.HomeTemplate))
	logoDiv := fx.Block("div", Attr{"class": "logo"}, fx.HTML(string(fx.Logo(logo))))
	h1 := fx.Elem("h1", heading)
	kbd := fx.Elem("kbd", "Z")
	button := fx.Block("button", nil, fx.HTML("Press"), kbd)
	fx.Build("home", &tmpl, logoDiv, h1, button)
}

func (fx *Fx) Text(path string) {
	content, err := fx.ToBytes(path)
	if err != nil {
		return
	}

	var buf bytes.Buffer
	if err := fx.Markdown().Convert(content, &buf); err != nil {
		return
	}

	markdown := One(template.HTML(buf.String()))
	textTmpl := One(template.HTML(fx.TextTemplate))
	fx.Build("text", &markdown, &textTmpl)
}

func (fx *Fx) Slides(dir string) {
	fx.Load(dir)
	base := filepath.Base(dir)
	tmpl, err := template.New("slides").Parse(fx.SlidesTemplate)
	if err != nil {
		return
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, map[string]string{
		"PREFIX": base,
		"URL":    fx.APIURL + "/" + base,
	}); err != nil {
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
		if b, err := fx.ToBytes(path); err == nil {
			return template.HTML(b)
		}
	}

	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		alt := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		return template.HTML(fmt.Sprintf(`<img src="%s" alt="%s">`,
			html.EscapeString(path),
			html.EscapeString(alt),
		))
	}

	fx.Load(path)
	base := filepath.Base(path)
	alt := strings.TrimSuffix(base, filepath.Ext(base))
	return template.HTML(fmt.Sprintf(`<img data-src="%s" alt="%s">`,
		html.EscapeString(fx.APIURL+"/"+base),
		html.EscapeString(alt),
	))
}
