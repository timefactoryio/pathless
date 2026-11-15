package fx

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestHTMLRegistrationRejectsInvalidResources(t *testing.T) {
	f := NewFx()
	before := string(f.Frames())
	for _, register := range []func(*One) error{f.Frame, f.Panel} {
		for _, one := range []*One{nil, {Name: "notes.txt", Type: "text/plain", Data: []byte("not HTML")}} {
			if err := register(one); err == nil {
				t.Fatal("expected invalid resource to fail registration")
			}
			if string(f.Frames()) != before {
				t.Fatal("failed registration changed stored entries")
			}
		}
	}
}

func TestHTMLDirectoryAndURLRegistration(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "nested"), 0700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"first.html", "nested/second.html"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("<main>"+name+"</main>"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte("\n<main>Remote</main>\n"))
	}))
	defer server.Close()

	for _, source := range []string{dir, server.URL} {
		t.Run(source, func(t *testing.T) {
			f := NewFx()
			expected, err := f.Input(source)
			if err != nil {
				t.Fatal(err)
			}
			if source == dir {
				expected = expected[1:]
				if len(expected) != 2 {
					t.Fatalf("got %d directory resources, want 2", len(expected))
				}
			}
			for _, register := range []func(*One) error{f.Frame, f.Panel} {
				for _, one := range expected {
					if err := register(one); err != nil {
						t.Fatal(err)
					}
				}
			}
			entries, err := f.Decode(f.Frames())
			if err != nil {
				t.Fatal(err)
			}
			if len(entries) != 2 {
				t.Fatalf("got %d root resources, want 2", len(entries))
			}
			for i, name := range []string{"frames", "panels"} {
				if entries[i].Name != name || entries[i].Type != "application/octet-stream" {
					t.Fatalf("unexpected root resource: %+v", entries[i])
				}
				stored, err := f.Decode(entries[i].Data)
				if err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(stored, expected) {
					t.Fatalf("%s entries differ from Input: got %+v, want %+v", name, stored, expected)
				}
			}
		})
	}
}

func TestFrameFileRegistration(t *testing.T) {
	f := NewFx()
	dir := t.TempDir()
	path := filepath.Join(dir, "home.html")
	if err := os.WriteFile(path, []byte("<main>Home</main>"), 0600); err != nil {
		t.Fatal(err)
	}
	input, err := f.Input(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Frame(input[0]); err != nil {
		t.Fatal(err)
	}
	entries, err := f.Decode(f.Frames())
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 || entries[0].Name != "frames" || entries[1].Name != "panels" {
		t.Fatalf("unexpected root resources: %+v", entries)
	}
	panels, err := f.Decode(entries[1].Data)
	if err != nil {
		t.Fatal(err)
	}
	if len(panels) != 0 {
		t.Fatalf("unexpected panel resources: %+v", panels)
	}
	entries, err = f.Decode(entries[0].Data)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name != "home.html" || string(entries[0].Data) != "<main>Home</main>" {
		t.Fatalf("unexpected stored file: %+v", entries)
	}
}
