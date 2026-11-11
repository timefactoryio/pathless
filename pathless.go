package pathless

import (
	"fmt"

	"github.com/timefactoryio/pathless/fx"
	"github.com/timefactoryio/pathless/one"
	"github.com/timefactoryio/pathless/zero"
)

type Pathless struct {
	fx.Fx
	origin      []byte
	pathlessURL string
}

func NewPathless(args ...string) *Pathless {
	pathlessURL := "*"
	circuitURL := "http://localhost:1001"
	switch len(args) {
	case 0:
	case 2:
		pathlessURL = "https://" + args[0]
		circuitURL = "https://" + args[1]
	default:
		panic(fmt.Sprintf(
			"pathless.NewPathless: expected 0 or 2 arguments, got %d",
			len(args),
		))
	}
	return &Pathless{
		Fx:          fx.NewFx(),
		origin:      zero.Origin(circuitURL),
		pathlessURL: pathlessURL,
	}
}

func (p *Pathless) Start() {
	one.NewOne(p.origin, p.Circuit(), p.pathlessURL).Start()
}
