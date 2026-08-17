package pathless

import (
	"github.com/timefactoryio/pathless/fx"
	"github.com/timefactoryio/pathless/zero"
)

type Pathless struct {
	fx.Fx
}

func NewPathless(args ...string) *Pathless {
	z := zero.NewZero(args...)
	return &Pathless{Fx: fx.NewFx(z)}
}
