package fx

import (
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
