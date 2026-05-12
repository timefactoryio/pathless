package fx

import (
	_ "embed"

	"github.com/timefactoryio/pathless/fx/templates"
	"github.com/timefactoryio/pathless/zero"
)

type Fx struct {
	*zero.Zero
	*templates.Templates
	Forge
	Routes map[string][]byte
}

func NewFx(apiUrl string) *Fx {
	return &Fx{
		Forge:     NewForge().(*forge),
		Zero:      zero.NewZero(apiUrl),
		Templates: templates.Init(),
		Routes:    make(map[string][]byte),
	}
}
