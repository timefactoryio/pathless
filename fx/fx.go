package fx

import (
	_ "embed"

	"github.com/timefactoryio/pathless/zero"
)

type Fx struct {
	*zero.Zero
	Forge
	Routes map[string][]byte
}

func NewFx(z *zero.Zero) *Fx {
	return &Fx{
		Zero:   z,
		Forge:  NewForge().(*forge),
		Routes: make(map[string][]byte),
	}
}
