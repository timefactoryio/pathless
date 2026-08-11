package zero

import (
	"bytes"
	_ "embed"
	"fmt"
	"html/template"
	"regexp"
	"strings"

	"github.com/timefactoryio/pathless/zero/frames"
	"github.com/timefactoryio/pathless/zero/panels"
)

//go:embed zero/pathless.html
var pathlessHTML string

//go:embed zero/universe.html
var universeHTML []byte

type Zero struct {
	PathlessHTML []byte
	UniverseHTML []byte
	PathlessURL  string
	CircuitURL   string
	Templates    *Templates
}

func NewZero(args ...string) *Zero {
	z := &Zero{
		UniverseHTML: universeHTML,
		Templates:    NewTemplates(),
	}
	switch len(args) {
	case 0:
		z.PathlessURL = "*"
		z.CircuitURL = "http://localhost:1001"
	case 2:
		z.PathlessURL = "https://" + args[0]
		z.CircuitURL = "https://" + args[1]
	default:
		panic(fmt.Sprintf(
			"zero.NewZero: expected 0 or 2 arguments, got %d",
			len(args),
		))
	}
	z.pathless()
	return z
}

func (z *Zero) Build(html string) []byte {
	if styles := styleTag.FindAllStringSubmatch(html, -1); len(styles) > 1 {
		var merged strings.Builder
		for _, match := range styles {
			merged.WriteString(match[1])
			merged.WriteByte('\n')
		}
		html = "<style>" + merged.String() + "</style>" +
			styleTag.ReplaceAllString(html, "")
	}
	if scripts := scriptTag.FindAllStringSubmatch(html, -1); len(scripts) > 0 {
		var merged strings.Builder
		for _, match := range scripts {
			source := match[1]
			if !strings.HasPrefix(strings.TrimSpace(source), "{") {
				source = "{" + source + "}"
			}
			merged.WriteString(source)
			merged.WriteByte('\n')
		}
		html = scriptTag.ReplaceAllString(html, "") +
			"<script>{" + merged.String() + "}</script>"
	}
	return []byte(html)
}

var (
	styleTag  = regexp.MustCompile(`(?s)<style>(.*?)</style>`)
	scriptTag = regexp.MustCompile(`(?s)<script>(.*?)</script>`)
)

func (z *Zero) pathless() {
	tmpl := template.Must(template.New("pathless").Parse(pathlessHTML))

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, map[string]string{
		"CIRCUIT": z.CircuitURL,
	}); err != nil {
		panic(err)
	}

	z.PathlessHTML = buf.Bytes()
}

type Templates struct {
	Frames *frames.Frames
	Panels *panels.Panels
}

func NewTemplates() *Templates {
	return &Templates{
		Frames: frames.NewFrames(),
		Panels: panels.NewPanels(),
	}
}
