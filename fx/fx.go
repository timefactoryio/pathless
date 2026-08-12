package fx

import "github.com/timefactoryio/pathless/zero"

type Fx struct {
	*zero.Zero
	Input  Input
	Frames *Output
	Panels *Output
	Routes map[string]*Output
}

func NewFx(z *zero.Zero) *Fx {
	return &Fx{
		Zero:   z,
		Input:  NewInput(),
		Frames: &Output{Name: "frames"},
		Panels: &Output{Name: "panels"},
		Routes: make(map[string]*Output),
	}
}

// Frame registers html as a frame, merging any inline style/script blocks first.
func (f *Fx) Frame(html string) {
	f.Frames.One = append(f.Frames.One, &Output{Zero: f.Build(html)})
}

// Panel registers html as a panel, merging any inline style/script blocks first.
func (f *Fx) Panel(html string) {
	f.Panels.One = append(f.Panels.One, &Output{Zero: f.Build(html)})
}

// Route registers output to be served at key once One.Serve wires up handlers.
func (f *Fx) Route(key string, output *Output) {
	f.Routes[key] = output
}

// Root builds the single Output tree served for the root route: universe.html's
// shell markup alongside the registered Frames and Panels branches, in that order.
func (f *Fx) Root() *Output {
	return &Output{
		One: []*Output{
			{Name: "universe", Type: "text/html", Zero: f.UniverseHTML},
			f.Frames,
			f.Panels,
		},
	}
}
