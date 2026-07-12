package fx

import (
	_ "embed"
	"log"
	"regexp"
	"strings"
)

//go:embed universe.html
var universeHtml string

type Fx struct {
	Frames []*Value
	Panels []*Value
	Routes map[string]*Value
	Origin string
	Hello  []*Value
}

func NewFx(origin string) *Fx {
	fx := &Fx{
		Frames: []*Value{},
		Panels: []*Value{},
		Routes: make(map[string]*Value),
		Origin: origin,
		Hello:  []*Value{},
	}
	fx.Hello = append(fx.Hello, fx.build(universeHtml))
	return fx
}

// Build encodes the universe shell, frames, and panels each into their own
// self-contained blob, wrapped as the three entries of Hello — this is
// what the circuit server serves for "/". Since each blob is already a
// valid Encode() output, the client decodes Hello once to get the three
// blobs, then decodes each one again the same way. Order is the contract:
// universe, frames, panels — the client doesn't look these up by name.
func (f *Fx) Build() {
	f.Hello = []*Value{
		{Type: "text/html", Data: f.Encode(f.Frames...)},
		{Type: "text/html", Data: f.Encode(f.Panels...)},
	}
}

// build consolidates a fragment's <style>/<script> assets into a single
// text/html leaf Value.
func (f *Fx) build(s string) *Value {
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

	return &Value{Type: "text/html", Data: []byte(s)}
}

// Frame reads a custom .html file at path (local or S3) and registers it
// into the frame pool. Everything a program serves must be available at
// startup, so a failed read is fatal: fix the path.
func (f *Fx) Frame(path string) {
	v, err := f.ToValue(path)
	if err != nil {
		log.Fatalf("fx: Frame %q: %v", path, err)
	}
	f.Frames = append(f.Frames, f.build(string(v.Data)))
}

// Panel reads a custom .html file at path (local or S3) and registers it
// into the panel pool. Everything a program serves must be available at
// startup, so a failed read is fatal: fix the path.
func (f *Fx) Panel(path string) {
	v, err := f.ToValue(path)
	if err != nil {
		log.Fatalf("fx: Panel %q: %v", path, err)
	}
	f.Panels = append(f.Panels, f.build(string(v.Data)))
}

var (
	style  = regexp.MustCompile(`(?s)<style>(.*?)</style>`)
	script = regexp.MustCompile(`(?s)<script>(.*?)</script>`)
)
