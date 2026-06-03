package templates

import (
	_ "embed"
)

//go:embed home.html
var homeHtml string

//go:embed slides.html
var slidesHtml string

//go:embed text.html
var textHtml string

//go:embed app.html
var appHtml string

//go:embed keyboard.html
var keyboardHtml []byte

type Templates struct {
	SlidesTemplate   string
	TextTemplate     string
	HomeTemplate     string
	AppTemplate      string
	KeyboardTemplate []byte
}

func Init() *Templates {
	return &Templates{
		SlidesTemplate:   slidesHtml,
		TextTemplate:     textHtml,
		HomeTemplate:     homeHtml,
		AppTemplate:      appHtml,
		KeyboardTemplate: keyboardHtml,
	}
}
