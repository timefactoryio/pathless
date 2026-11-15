package pathless

import (
	"fmt"

	"github.com/timefactoryio/pathless/one"
)

type Pathless struct {
	*one.One
}

func NewPathless(args ...string) *Pathless {
	pathlessURL := "*"
	pathUrl := "http://localhost:1001"
	switch len(args) {
	case 0:
	case 2:
		pathlessURL = "https://" + args[0]
		pathUrl = "https://" + args[1]
	default:
		panic(fmt.Sprintf(
			"pathless.NewPathless: expected 0 or 2 arguments, got %d",
			len(args),
		))
	}
	return &Pathless{
		One: one.NewOne(pathlessURL, pathUrl),
	}
}
