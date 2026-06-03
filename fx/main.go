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

func NewFx(apiUrl string) *Fx {
	return &Fx{
		Forge:  NewForge().(*forge),
		Zero:   zero.NewZero(apiUrl),
		Routes: make(map[string][]byte),
	}
}
