package core

import (
	_ "embed"
)

//go:embed input.html
var inputHtml string

//go:embed layout.html
var layoutHtml string

//go:embed keyboard.html
var keyboardHtml string

type Core struct {
	Input    string
	Layout   string
	Keyboard string
}

func Init() *Core {
	return &Core{
		Input:    inputHtml,
		Layout:   layoutHtml,
		Keyboard: keyboardHtml,
	}
}
