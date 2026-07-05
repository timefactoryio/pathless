package fx

import (
	_ "embed"
	"html/template"
	"os"
	"regexp"
	"strings"

	"github.com/timefactoryio/markdown"
)

type Fx struct {
	*Circuit
	frames []*template.HTML
	panels []*template.HTML
	md     *markdown.Markdown
}

func NewFx() *Fx {
	return &Fx{
		frames:  []*template.HTML{},
		panels:  []*template.HTML{},
		md:      markdown.New(""),
		Circuit: NewCircuit(),
	}
}

var (
	style  = regexp.MustCompile(`(?s)<style>(.*?)</style>`)
	script = regexp.MustCompile(`(?s)<script>(.*?)</script>`)
)

// Frame registers a frame from a local file. The file authors its own root
// container div (by convention <div class="<filename stem>">). Nothing is
// wrapped for you — the content is trusted as-is, so only pass local,
// developer-controlled paths.
func (f *Fx) Frame(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	raw := template.HTML(data)
	f.Build(&raw)
}

// Build registers a frame from its elements. The frame's root container is
// authored by the caller (template builder or custom frame file), not added
// here.
func (f *Fx) Build(elements ...*template.HTML) {
	var b strings.Builder
	for _, el := range elements {
		b.WriteString(string(*el))
	}
	cleaned := f.consolidateAssets(b.String())
	result := template.HTML(cleaned)
	f.Frames(&result)
}

func (f *Fx) Frames(frame ...*template.HTML) *Value {
	if len(frame) > 0 && frame[0] != nil {
		f.frames = append(f.frames, frame[0])
	}
	return f.bundle(f.frames)
}

// Markdown returns the configured goldmark instance for rendering markdown to HTML.
func (f *Fx) Markdown() *markdown.Markdown {
	return f.md
}

const (
	openStyle   = "<style>"
	closeStyle  = "</style>"
	openScript  = "<script>"
	closeScript = "</script>"
	openBlock   = "<script>{"
	closeBlock  = "}</script>"
)

func (f *Fx) consolidateAssets(s string) string {
	if styles := style.FindAllStringSubmatch(s, -1); len(styles) > 1 {
		var sb strings.Builder
		for _, m := range styles {
			sb.WriteString(m[1])
			sb.WriteByte('\n')
		}
		s = openStyle + sb.String() + closeStyle + style.ReplaceAllString(s, "")
	}

	var scripts strings.Builder
	s = script.ReplaceAllStringFunc(s, func(m string) string {
		content := m[len(openScript) : len(m)-len(closeScript)]
		if t := strings.TrimSpace(content); !strings.HasPrefix(t, "{") {
			scripts.WriteByte('{')
			scripts.WriteString(content)
			scripts.WriteString("}\n")
		} else {
			scripts.WriteString(content)
			scripts.WriteByte('\n')
		}
		return ""
	})
	if scripts.Len() > 0 {
		s += openBlock + scripts.String() + closeBlock
	}
	return s
}

// Panel builds a panel frame from a local file. It does not register the
// frame — pass the result to Panels(...) to do that.
func (f *Fx) Panel(path string) *template.HTML {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	raw := template.HTML(data)
	return f.BuildPanel(&raw)
}

// BuildPanel consolidates elements into a single panel frame, the same way
// Build consolidates space frames. It does not register the frame — pass
// the result to Panels(...) to do that.
func (f *Fx) BuildPanel(elements ...*template.HTML) *template.HTML {
	var b strings.Builder
	for _, el := range elements {
		if el != nil {
			b.WriteString(string(*el))
		}
	}
	cleaned := f.consolidateAssets(b.String())
	result := template.HTML(cleaned)
	return &result
}

// Panels registers any given panel frames, then returns the wire payload
// for every panel frame registered so far.
func (f *Fx) Panels(panels ...*template.HTML) *Value {
	for _, p := range panels {
		if p != nil {
			f.panels = append(f.panels, p)
		}
	}
	return f.bundle(f.panels)
}

// bundle wraps frames as text/html leaves and pre-encodes them into a single
// application/octet-stream Value for the client to decode as a nested bundle.
func (f *Fx) bundle(frames []*template.HTML) *Value {
	if len(frames) == 0 {
		return nil
	}
	values := make([]*Value, len(frames))
	for i, fr := range frames {
		values[i] = &Value{Type: "text/html", Data: []byte(*fr)}
	}
	return &Value{Type: "application/octet-stream", Data: f.Encode(&Value{Values: values})}
}
