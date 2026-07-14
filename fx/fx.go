package fx

import (
	"fmt"
	"log"
	"regexp"
	"strings"
)

// Fx sources and processes content into Values: it builds the frame and
// panel pools and the route map. It knows nothing of the wire format, HTTP,
// or where content is served from — one encodes and serves what fx produces,
// and route URLs are resolved client-side against window.circuit.
type Fx struct {
	Frames []*Value
	Panels []*Value
	Routes map[string]*Value
}

func NewFx() *Fx {
	return &Fx{
		Frames: []*Value{},
		Panels: []*Value{},
		Routes: make(map[string]*Value),
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

// Route registers v as a served route under key and returns key, so a frame
// can fetch it client-side via p.source(key). This is the one operation that
// makes content fetchable — ToValue only builds a Value, it never registers.
// A template that must expose companion data while building its frame (as
// Slides and a non-svg Logo do) builds the Value with ToValue, then hands it
// here and bakes the returned key into the frame's markup.
func (f *Fx) Route(key string, v *Value) string {
	f.Routes[key] = v
	return key
}

// Save gob-encodes a registered route's Value for the caller to persist
// wherever it chooses (e.g. syncing to S3 via an external process).
func (f *Fx) Save(key string) ([]byte, error) {
	v, ok := f.Routes[key]
	if !ok {
		return nil, fmt.Errorf("fx: Save %q: route not found", key)
	}
	return v.Save()
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
