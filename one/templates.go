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

//go:embed embed/home.html
var homeHtml string

//go:embed embed/slides.html
var slidesHtml string

//go:embed embed/text.html
var textHtml string

//go:embed embed/keyboard.html
var keyboardHtml string

//go:embed embed/app.html
var appHtml string

type One struct {
	*fx.Fx
	SlidesTemplate string
	TextTemplate   string
	HomeTemplate   string
	Keyboard       string
	AppTemplate    string
}

func NewOne(fx *fx.Fx) *One {
	return &One{
		Fx:             fx,
		SlidesTemplate: slidesHtml,
		TextTemplate:   textHtml,
		HomeTemplate:   homeHtml,
		Keyboard:       keyboardHtml,
		AppTemplate:    appHtml,
	}
}

func (o *One) Home(logo, heading string) {
	logoEmbed := o.ToBytes(logo)
	if logoEmbed == nil {
		return
	}

	tmpl := template.Must(template.New("home").Parse(o.HomeTemplate))

	var buf strings.Builder
	if err := tmpl.Execute(&buf, map[string]template.HTML{
		"LOGO":    template.HTML(string(logoEmbed)),
		"HEADING": template.HTML(heading),
	}); err != nil {
		return
	}

	result := fx.One(template.HTML(buf.String()))
	o.Build("", &result)
}

func (o *One) Text(path string) {
	content := o.ToBytes(path)
	if content == nil {
		return
	}

	var buf bytes.Buffer
	if err := (*o.Markdown()).Convert(content, &buf); err != nil {
		return
	}

	html := buf.String()
	html = strings.ReplaceAll(html, "<p><img", "<img")
	html = strings.ReplaceAll(html, "\"></p>", "\">")
	html = strings.ReplaceAll(html, "\" /></p>", "\" />")
	html = strings.ReplaceAll(html, "\"/></p>", "\"/>")

	markdown := fx.One(template.HTML(html))
	textTmpl := fx.One(template.HTML(o.TextTemplate))
	o.Build("text", &markdown, &textTmpl)
}

func (o *One) Slides(dir string) {
	prefix := o.Reader(dir)
	tmpl, err := template.New("slides").Parse(o.SlidesTemplate)
	if err != nil {
		return
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, map[string]string{"PREFIX": prefix}); err != nil {
		return
	}

	result := fx.One(template.HTML(buf.String()))
	o.Build("slides", &result)
}

func (o *One) App(title, url string) {
	tmpl, err := template.New("frame").Parse(o.AppTemplate)
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
	result := fx.One(template.HTML(buf.String()))
	o.Build("app", &result)
}

func (o *One) Logo(path string) template.HTML {
	if strings.ToLower(filepath.Ext(path)) == ".svg" {
		if b := o.ToBytes(path); b != nil {
			return template.HTML(b)
		}
		return ""
	}

	if !strings.HasPrefix(path, "http://") && !strings.HasPrefix(path, "https://") {
		o.Read(path, "")
		name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		path = o.APIURL + "/" + name
	}

	alt := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	return template.HTML(fmt.Sprintf(`<img src="%s" alt="%s">`,
		html.EscapeString(path),
		html.EscapeString(alt),
	))
}
