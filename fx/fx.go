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
	md     *markdown.Markdown
}

func NewFx() *Fx {
	return &Fx{
		frames:  []*template.HTML{},
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

func (f *Fx) Frames(frame ...*template.HTML) [][]byte {
	if len(frame) > 0 && frame[0] != nil {
		f.frames = append(f.frames, frame[0])
	}
	out := make([][]byte, 0, len(f.frames))
	for _, fr := range f.frames {
		if fr != nil {
			out = append(out, []byte(*fr))
		}
	}
	return out
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
