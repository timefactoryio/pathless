package one

import (
	"bytes"
	"mime"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/timefactoryio/pathless/fx"
)

func TestRegistration(t *testing.T) {
	root := t.TempDir()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		typ := mime.TypeByExtension(".html")
		if r.URL.Path == "/invalid" {
			typ = "text/plain"
		}
		w.Header().Set("Content-Type", typ)
		w.Write([]byte("<main>Remote</main>"))
	}))
	defer server.Close()
	for name, data := range map[string]string{
		"first.html":  "<main>First</main>",
		"second.html": "<aside>Second</aside>",
		"invalid.txt": "<main>Not an HTML MIME type</main>",
	} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(data), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	for _, method := range []string{"Frame", "Panel"} {
		t.Run(method, func(t *testing.T) {
			content := NewOne()
			register, target, other := content.Frame, &content.frames, &content.panels
			if method == "Panel" {
				register, target, other = content.Panel, &content.panels, &content.frames
			}
			for _, source := range []string{filepath.Join(root, "second.html"), filepath.Join(root, "first.html"), server.URL + "/remote"} {
				if err := register(source); err != nil {
					t.Fatal(err)
				}
			}
			typ := mime.TypeByExtension(".html")
			want := fx.N{
				{Name: "second.html", Type: typ, Data: []byte("<aside>Second</aside>")},
				{Name: "first.html", Type: typ, Data: []byte("<main>First</main>")},
				{Name: server.URL + "/remote", Type: typ, Data: []byte("<main>Remote</main>")},
			}
			if !reflect.DeepEqual(*target, want) || len(*other) != 0 {
				t.Fatalf("registration routed or ordered entries incorrectly: %#v", *target)
			}
			for _, source := range []string{root, filepath.Join(root, "invalid.txt"), server.URL + "/invalid"} {
				if err := register(source); err == nil {
					t.Fatalf("accepted invalid input %q", source)
				}
				if !reflect.DeepEqual(*target, want) || len(*other) != 0 {
					t.Fatal("failed registration modified collections")
				}
			}
		})
	}
}

func TestFrames(t *testing.T) {
	content := NewOne()
	if got := content.Frames(); !bytes.Equal(got, []byte{0, 0}) {
		t.Fatalf("empty frame/panel sequences = %v", got)
	}
	frame, panel, spare := &fx.One{Name: "frame", Data: []byte{}}, &fx.One{Name: "panel", Data: []byte{}}, &fx.One{Name: "spare"}
	backing := fx.N{frame, spare}
	content.frames, content.panels = backing[:1], fx.N{panel}
	wire := content.Frames()
	want := append(content.Encode(fx.N{frame}), content.Encode(fx.N{panel})...)
	entries, err := content.Decode(wire)
	if err != nil || !bytes.Equal(wire, want) || !reflect.DeepEqual(entries, fx.N{frame, panel}) {
		t.Fatalf("Frames = %v, decoded = %#v, error = %v", wire, entries, err)
	}
	if backing[1] != spare || len(content.frames) != 1 || content.panels[0] != panel {
		t.Fatal("Frames modified its source collections")
	}
}
