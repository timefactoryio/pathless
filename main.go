package main

import (
	"github.com/timefactoryio/pathless/one"
)

func main() {
	// o := one.NewOne(os.Getenv("API_URL"))
	o := one.NewOne("http://10.0.0.100:1001")

	o.Home("https://zero.s3.timefactory.io/timefactory.svg", "the perpetual motion machine")
	o.Text("https://raw.githubusercontent.com/timefactoryio/pathless/refs/heads/main/mechanics.md")
	o.Text("https://raw.githubusercontent.com/timefactoryio/pathless/refs/heads/main/README.md")
	o.Slides("../demo/slides")
	o.Serve()
}
