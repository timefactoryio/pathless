package one

import (
	"fmt"
	"mime"

	"github.com/timefactoryio/pathless/fx"
)

func (o *One) Frame(source string) error {
	return o.html(source, &o.frames)
}

func (o *One) Panel(source string) error {
	return o.html(source, &o.panels)
}

func (o *One) Frames() []byte {
	return o.Encode(o.frames, o.panels)
}

func (o *One) html(source string, target *fx.N) error {
	entries, err := o.Input(source)
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		return fmt.Errorf("%q: no entries", source)
	}
	for _, entry := range entries {
		if typ, _, err := mime.ParseMediaType(entry.Type); err != nil || typ != "text/html" {
			return fmt.Errorf("%q: entry %q: expected text/html, got %q", source, entry.Name, entry.Type)
		}
	}

	*target = append(*target, entries...)
	return nil
}
