package fx

import (
	"fmt"
	"maps"
	"mime"
	"net/http"
	"slices"
	"time"
)

type Fx interface {
	Input(string) (N, error)
	Load(string) (N, error)
	Frame(*One) error
	Panel(*One) error
	Frames(...bool) []byte
}

type fx struct {
	frames N
	panels N
	client *http.Client
}

func NewFx() Fx {
	return &fx{
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (f *fx) Frames(compress ...bool) []byte {
	return slices.Concat(f.frames, f.panels).Encode(compress...)
}

func (f *fx) Frame(one *One) error {
	return f.html(one, "frame", &f.frames)
}

func (f *fx) Panel(one *One) error {
	return f.html(one, "panel", &f.panels)
}

func (f *fx) html(one *One, role string, target *N) error {
	if one == nil {
		return fmt.Errorf("expected an HTML resource, got nil")
	}
	contentType := one.Meta["type"]
	if typ, _, err := mime.ParseMediaType(contentType); err != nil || typ != "text/html" {
		return fmt.Errorf("entry %q: expected text/html, got %q", one.Meta["name"], contentType)
	}
	meta := maps.Clone(one.Meta)
	meta["role"] = role
	*target = append(*target, &One{Meta: meta, Data: one.Data})
	return nil
}
