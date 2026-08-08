package pathless

import (
	"github.com/timefactoryio/pathless/fx"
	"github.com/timefactoryio/pathless/one"
	"github.com/timefactoryio/pathless/zero"
)

type Pathless struct {
	*one.One
}

func NewPathless(args ...string) *Pathless {
	zero := zero.NewZero(args...)
	fx := fx.NewFx(zero)
	one := one.NewOne(fx)
	return &Pathless{
		One: one,
	}
}
