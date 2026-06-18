package fx

import (
	_ "embed"

	"github.com/timefactoryio/pathless/zero"
)

type Fx struct {
	*zero.Zero
	Forge
	*Circuit
}

func NewFx(z *zero.Zero) *Fx {
	return &Fx{
		Zero:    z,
		Forge:   NewForge().(*forge),
		Circuit: NewCircuit(),
	}
}
