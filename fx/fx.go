package fx

import (
	"fmt"
	"os"

	"github.com/timefactoryio/pathless/zero"
)

type Fx struct {
	*zero.Zero
	Routes map[string][]*Output
}

func NewFx(z *zero.Zero) *Fx {
	return &Fx{
		Zero:   z,
		Routes: make(map[string][]*Output),
	}
}

// Route registers values as a served route under key and returns key, so a
// frame can fetch it client-side via p.source(key). This is the one
// operation that makes content fetchable — Input only builds a Output, it
// never registers. A template that must expose companion data while
// building its frame (as Slides and a non-svg Logo do) builds the Output(s)
// with Input, then hands them here and bakes the returned key into the
// frame's markup.
func (f *Fx) Route(key string, values ...*Output) string {
	f.Routes[key] = values
	return key
}

// Save gob-encodes a registered route. When disk is true, it also writes
// the encoded data to s3/<key>.
func (f *Fx) Save(key string, disk ...bool) ([]byte, error) {
	values, ok := f.Routes[key]
	if !ok {
		return nil, fmt.Errorf("fx: Save %q: route not found", key)
	}

	data, err := f.Marshal(values...).Save()
	if err != nil {
		return nil, fmt.Errorf("fx: Save %q: %w", key, err)
	}

	if len(disk) > 0 && disk[0] {
		if err := os.WriteFile(key, data, 0o644); err != nil {
			return nil, fmt.Errorf("fx: Save %q: %w", key, err)
		}
	}
	return data, nil
}
