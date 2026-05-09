package main

import (
	"os"

	"github.com/timefactoryio/pathless/fx"
	"github.com/timefactoryio/pathless/one"
	"github.com/timefactoryio/pathless/zero"
)

func main() {
	z := zero.NewZero(os.Getenv("API_URL"))
	f := fx.NewFx(z)
	o := one.NewOne(z, f)

	// o.Home()
	o.Serve()
}
