package fx

import (
	"fmt"
	"mime"
	"net/http"
	"time"
)

type Fx interface {
	Input(string) (N, error)
	Load(string) (N, error)
	Encode(...N) []byte
	Zip(...N) []byte
	Decode([]byte) (N, error)
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

func (f *fx) Frames(zip ...bool) []byte {
	if len(zip) > 0 && zip[0] {
		return f.Zip(f.frames, f.panels)
	}
	return f.Encode(f.frames, f.panels)
}

func (f *fx) Frame(one *One) error {
	return f.html(one, &f.frames)
}

func (f *fx) Panel(one *One) error {
	return f.html(one, &f.panels)
}

func (f *fx) html(one *One, target *N) error {
	if one == nil {
		return fmt.Errorf("expected an HTML resource, got nil")
	}
	if typ, _, err := mime.ParseMediaType(one.Type); err != nil || typ != "text/html" {
		return fmt.Errorf("entry %q: expected text/html, got %q", one.Name, one.Type)
	}
	*target = append(*target, one)
	return nil
}
