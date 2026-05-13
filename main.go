package main

import (
	"github.com/timefactoryio/pathless/one"
)

func main() {
	// o := one.NewOne(os.Getenv("API_URL"))
	o := one.NewOne("")

	o.Home("https://zero.s3.timefactory.io/timefactory.svg", "the perpetual motion machine")
	o.Text("https://raw.githubusercontent.com/timefactoryio/pathless/refs/heads/main/README.md")
	o.Serve()
}
