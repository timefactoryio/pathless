package main

import "github.com/timefactoryio/pathless"

func main() {
	p := pathless.NewPathless()
	// register content here (Home/Text/Slides/Keyboard), then:
	// p.Home("https://zero.s3.timefactory.io/timefactory.svg", "the point of origin")
	// p.Text("../mechanics.md")
	// p.Slides("../../origin/slides")
	// p.Keyboard()
	p.Start()
	// p.
}
