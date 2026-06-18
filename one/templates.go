package one

import (
	"bytes"

	_ "embed"
	"fmt"
	"html"
	"html/template"
	"path/filepath"
	"strings"

	"github.com/timefactoryio/pathless/fx"
)

//go:embed templates/home.html
var homeHtml string

//go:embed templates/slides.html
var slidesHtml string

//go:embed templates/text.html
var textHtml string

func (o *One) Home(logo, heading string) {
	tmpl := template.HTML(homeHtml)
	logoDiv := o.Block("div", fx.Attr{"class": "logo"}, o.HTML(string(o.Logo(logo))))
	h1 := o.Elem("h1", heading)
	kbd := o.Elem("kbd", "Z")
	button := o.Block("button", nil, o.HTML("Press"), kbd)
	o.Build("home", &tmpl, logoDiv, h1, button)
}

func (o *One) Text(path string) {
	content, err := o.ToBytes(path)
	if err != nil {
		return
	}
	var buf bytes.Buffer
	if err := o.Markdown().Convert(content, &buf); err != nil {
		return
	}
	markdown := template.HTML(buf.String())
	textTmpl := template.HTML(textHtml)
	o.Build("text", &markdown, &textTmpl)
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
	o.Build("slides", &result)
}

func (o *One) Logo(path string) template.HTML {
	if strings.ToLower(filepath.Ext(path)) == ".svg" {
		if b, err := o.ToBytes(path); err == nil {
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
	o.Load(path)
	base := filepath.Base(path)
	alt := strings.TrimSuffix(base, filepath.Ext(base))
	return template.HTML(fmt.Sprintf(`<img data-src="%s" alt="%s">`,
		html.EscapeString(o.Origin+"/"+base),
		html.EscapeString(alt),
	))
}
