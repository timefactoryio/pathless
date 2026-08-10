package fx

import "github.com/timefactoryio/pathless/zero"

type Fx struct {
	*zero.Zero
	Input Input
}

func NewFx(z *zero.Zero) *Fx {
	return &Fx{
		Zero:  z,
		Input: &input{},
	}
}
