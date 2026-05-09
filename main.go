package main

import (
	"os"

	"github.com/timefactoryio/pathless/zero"
)

func main() {
	z := zero.NewZero(os.Getenv("API_URL"))
	z.Serve()
}

type Payload struct {
	MetaData []byte
	Data     []byte
	Payload  []*Payload
}
