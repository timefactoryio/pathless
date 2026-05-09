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

func NewFx(z *zero.Zero) *Fx {
	return &Fx{
		Forge:     NewForge().(*forge),
		Zero:      z,
		Templates: templates.Init(),
		Routes:    make(map[string][]byte),
	}
}
