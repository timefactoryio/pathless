package panels

import (
	_ "embed"
)

//go:embed keyboard.html
var keyboardHTML string

type Panels struct {
	Keyboard string
}

func NewPanels() *Panels {
	return &Panels{
		Keyboard: keyboardHTML,
	}
}
