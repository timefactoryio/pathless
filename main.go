package main

import (
	"os"

	"github.com/timefactoryio/pathless/zero"
)

func main() {
	z := zero.NewZero(os.Getenv("API_URL"))
	z.Serve()
}
