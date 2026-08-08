package fx

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/timefactoryio/pathless/zero"
)

type Fx struct {
	*zero.Zero
	Frames []*Output
	Panels []*Output
	Routes map[string][]*Output
}

func NewFx(zero *zero.Zero) *Fx {
	return &Fx{
		Zero:   zero,
		Frames: []*Output{},
		Panels: []*Output{},
		Routes: make(map[string][]*Output),
	}
}

// Root builds the wire blob served at "/" once — universe.html, then the
// frame pool and panel pool each nested as their own manifest — and
// replaces Universe with it. Like Pathless, it's built once and never
// changes, so it's a static blob by the time it's served. Its entries carry
// no Type; root's shape is hardcoded on both sides.
func (f *Fx) Root() []byte {
	f.Universe = f.Marshal(
		&Output{Data: f.Universe},
		f.Marshal(f.Frames...),
		f.Marshal(f.Panels...),
	).Data
	return f.Universe
}

// build consolidates a fragment's <style>/<script> assets into a single
// leaf Output.
func (f *Fx) build(s string) *Output {
	if styles := style.FindAllStringSubmatch(s, -1); len(styles) > 1 {
		var merged strings.Builder
		for _, m := range styles {
			merged.WriteString(m[1])
			merged.WriteByte('\n')
		}
		s = "<style>" + merged.String() + "</style>" + style.ReplaceAllString(s, "")
	}

	if matches := script.FindAllStringSubmatch(s, -1); len(matches) > 0 {
		var merged strings.Builder
		for _, m := range matches {
			if t := strings.TrimSpace(m[1]); !strings.HasPrefix(t, "{") {
				merged.WriteString("{" + m[1] + "}\n")
			} else {
				merged.WriteString(m[1] + "\n")
			}
		}
		s = script.ReplaceAllString(s, "") + "<script>{" + merged.String() + "}</script>"
	}

	return &Output{Data: []byte(s)}
}

// Frame reads a custom .html file at path (local or S3) and registers it
// into the frame pool. Everything a program serves must be available at
// startup, so a failed read is fatal: fix the path.
func (f *Fx) Frame(path string) {
	v, err := f.Input(path)
	if err != nil {
		log.Fatalf("fx: Frame %q: %v", path, err)
	}
	f.Frames = append(f.Frames, f.build(string(v.Data)))
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

// Save gob-encodes a registered route's wire Output for the caller to
// persist wherever it chooses (e.g. syncing to S3 via an external process).
func (f *Fx) Save(key string) ([]byte, error) {
	values, ok := f.Routes[key]
	if !ok {
		return nil, fmt.Errorf("fx: Save %q: route not found", key)
	}
	return f.Marshal(values...).Save()
}

// SaveBinary gob-encodes a registered route's Output and writes it to disk
// under s3/<key>.
func (f *Fx) SaveBinary(key string) error {
	data, err := f.Save(key)
	if err != nil {
		return fmt.Errorf("fx: SaveBinary %q: %w", key, err)
	}
	if err := os.MkdirAll("s3", 0755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join("s3", key), data, 0644)
}

// Panel reads a custom .html file at path (local or S3) and registers it
// into the panel pool. Everything a program serves must be available at
// startup, so a failed read is fatal: fix the path.
func (f *Fx) Panel(path string) {
	v, err := f.Input(path)
	if err != nil {
		log.Fatalf("fx: Panel %q: %v", path, err)
	}
	f.Panels = append(f.Panels, f.build(string(v.Data)))
}

var (
	style  = regexp.MustCompile(`(?s)<style>(.*?)</style>`)
	script = regexp.MustCompile(`(?s)<script>(.*?)</script>`)
)
