package frames

import (
	_ "embed"
)

//go:embed home.html
var homeHTML string

//go:embed slides.html
var slidesHTML string

//go:embed text.html
var textHTML string

type Frames struct {
	Home   string
	Slides string
	Text   string
}

func NewFrames() *Frames {
	return &Frames{
		Home:   homeHTML,
		Slides: slidesHTML,
		Text:   textHTML,
	}
}
