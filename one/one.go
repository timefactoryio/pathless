package main

import "github.com/timefactoryio/pathless"

func main() {
	p := pathless.NewPathless()
	// register content here (Home/Text/Slides/Keyboard), then:
	// p.Keyboard()
	p.Home("https://zero.s3.timefactory.io/timefactory.svg", "the point of origin")
	p.Keyboard()
	p.Start()
}
