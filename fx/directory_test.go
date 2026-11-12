package fx

import (
	"mime"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestDir(t *testing.T) {
	root := t.TempDir()
	content := &fx{}
	empty := N{{Name: filepath.Base(root), Type: "dir", Data: []byte("[[null]]")}}
	if got, err := content.dir(root); err != nil || !reflect.DeepEqual(got, empty) {
		t.Fatalf("empty directory = %#v, error = %v", got, err)
	}

	want := N{
		{Name: "first.txt", Type: mime.TypeByExtension(".txt"), Data: []byte("first")},
		{Name: "sequence", Type: "text/plain; charset=utf-8", Data: []byte("first\n")},
	}
	for _, entry := range want {
		if err := os.WriteFile(filepath.Join(root, entry.Name), entry.Data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	want = append(N{{
		Name: filepath.Base(root),
		Type: "dir",
		Data: []byte(`[[null,"first.txt","sequence"]]`),
	}}, want...)
	if got, err := content.dir(root); err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("directory = %#v, error = %v; want %#v", got, err, want)
	}
	if got, err := content.dir(filepath.Join(root, "first.txt")); err == nil || got != nil {
		t.Fatalf("file as directory = %#v, error = %v", got, err)
	}
	link := filepath.Join(t.TempDir(), "linked")
	if err := os.Symlink(root, link); err != nil {
		t.Fatal(err)
	}
	linkedWant := append(N{{Name: "linked", Type: "dir", Data: want[0].Data}}, want[1:]...)
	if got, err := content.dir(link); err != nil || !reflect.DeepEqual(got, linkedWant) {
		t.Fatalf("linked directory = %#v, error = %v; want %#v", got, err, linkedWant)
	}
	if got, err := content.dir(filepath.Join(root, "missing")); err == nil || got != nil {
		t.Fatalf("missing directory = %#v, error = %v", got, err)
	}
	if err := os.Symlink(filepath.Join(root, "missing"), filepath.Join(root, "broken")); err != nil {
		t.Fatal(err)
	}
	if got, err := content.dir(root); err == nil || got != nil {
		t.Fatalf("unreadable entry = %#v, error = %v", got, err)
	}
}

func TestDirNested(t *testing.T) {
	root := t.TempDir()
	want := N{
		{
			Name: filepath.Base(root),
			Type: "dir",
			Data: []byte(`[[null,"index.html"],["assets","logo.svg"],["assets/icons","close.svg"],["empty"]]`),
		},
		{Name: "index.html", Type: mime.TypeByExtension(".html"), Data: []byte("<html></html>")},
		{Name: "assets/logo.svg", Type: mime.TypeByExtension(".svg"), Data: []byte("<svg>logo</svg>")},
		{Name: "assets/icons/close.svg", Type: mime.TypeByExtension(".svg"), Data: []byte("<svg>close</svg>")},
	}
	for _, file := range want[1:] {
		path := filepath.Join(root, filepath.FromSlash(file.Name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, file.Data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Mkdir(filepath.Join(root, "empty"), 0o755); err != nil {
		t.Fatal(err)
	}
	content := &fx{}
	if got, err := content.dir(root); err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("nested directory = %#v, error = %v; want %#v", got, err, want)
	}
	if err := os.Symlink(filepath.Join(root, "missing"), filepath.Join(root, "assets", "broken")); err != nil {
		t.Fatal(err)
	}
	if got, err := content.dir(root); err == nil || got != nil {
		t.Fatalf("unreadable nested entry = %#v, error = %v", got, err)
	}
}
