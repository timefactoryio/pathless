package fx

import "github.com/timefactoryio/pathless/zero"

type Fx struct {
	*zero.Zero
	Input Input
}

func NewFx(z *zero.Zero) *Fx {
	return &Fx{
		Zero:  z,
		Input: NewInput(),
	}
}

// func (f *Fx) Frame(html string) {
// 	if f.Frames == nil {
// 		f.Frames = &Output{}
// 	}
// 	f.Frames.One = append(f.Frames.One, &Output{Zero: f.Build(html)})
// 	f.Frames.Zero = sizes(f.Frames.One)
// }

// func (f *Fx) Panel(html string) {
// 	if f.Panels == nil {
// 		f.Panels = &Output{}
// 	}
// 	f.Panels.One = append(f.Panels.One, &Output{Zero: f.Build(html)})
// 	f.Panels.Zero = sizes(f.Panels.One)
// }
