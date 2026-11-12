package fx

import (
	"bytes"
	"mime"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func writeFiles(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for name, data := range files {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestInput(t *testing.T) {
	root := writeFiles(t, map[string]string{
		"sequence":          "slides\nintro\nslides\nunknown\n",
		"intro.txt":         "intro",
		"pair.md":           "markdown",
		"pair.txt":          "text",
		"slides/sequence":   "second\nfirst\n",
		"slides/first.txt":  "first",
		"slides/second.txt": "second",
	})
	content := NewFx()
	entries, err := content.InputDir(root)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, entry := range entries {
		names = append(names, entry.Name)
	}
	want := []string{"slides/second", "slides/first", "intro", "pair.md", "pair.txt"}
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("names = %v, want %v", names, want)
	}
	single, err := content.Input(filepath.Join(root, "intro.txt"))
	expected := N{{Name: "intro", Type: mime.TypeByExtension(".txt"), Data: []byte("intro")}}
	if err != nil || !reflect.DeepEqual(single, expected) {
		t.Fatalf("file = %#v, error = %v", single, err)
	}
	if string(entries[0].Data) != "second" || entries[0].Type != expected[0].Type {
		t.Fatalf("nested resource = %#v", entries[0])
	}
	if got, err := content.Input(filepath.Join(root, "missing")); err == nil || got != nil {
		t.Fatalf("missing file = %#v, error = %v", got, err)
	}
	if got, err := content.Input(root); err == nil || got != nil {
		t.Fatalf("Input must reject directories: %#v, error = %v", got, err)
	}
	if got, err := content.InputDir(filepath.Join(root, "intro.txt")); err == nil || got != nil {
		t.Fatalf("InputDir must reject files: %#v, error = %v", got, err)
	}
	if got, err := content.InputDir(t.TempDir()); err != nil || len(got) != 0 {
		t.Fatalf("empty directory = %#v, error = %v", got, err)
	}
}

func TestInputAndLoadHTTP(t *testing.T) {
	content := NewFx()
	entries := N{{Name: "one", Type: "text/plain", Data: []byte("first")}, {Name: "two", Data: []byte{0, 255}}}
	payload := content.Encode(entries[:1], N{}, entries[1:])
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/payload.txt":
			w.Write(payload)
		case "/redirect":
			http.Redirect(w, r, "/payload.txt", http.StatusFound)
		case "/raw.txt":
			w.Write([]byte("raw"))
		case "/empty":
			w.Write([]byte{0})
		case "/invalid":
			w.Write(append(bytes.Clone(payload), 0x80))
		case "/short":
			w.Header().Set("Content-Length", "10")
			w.Write([]byte{0})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	rawPayload := N{{Name: "payload", Type: mime.TypeByExtension(".txt"), Data: payload}}
	for _, test := range []struct {
		name    string
		fetch   func(string) (N, error)
		path    string
		want    N
		wantErr bool
	}{
		{"input/raw", content.Input, "/raw.txt", N{{Name: "raw", Type: mime.TypeByExtension(".txt"), Data: []byte("raw")}}, false},
		{"input/encoded", content.Input, "/payload.txt", rawPayload, false},
		{"input/redirect", content.Input, "/redirect", rawPayload, false},
		{"load/encoded", content.Load, "/payload.txt", entries, false},
		{"load/redirect", content.Load, "/redirect", entries, false},
		{"load/empty", content.Load, "/empty", N{}, false},
		{"load/raw", content.Load, "/raw.txt", nil, true},
		{"load/invalid", content.Load, "/invalid", nil, true},
		{"load/status", content.Load, "/missing", nil, true},
		{"load/read", content.Load, "/short", nil, true},
		{"input/status", content.Input, "/missing", nil, true},
		{"input/read", content.Input, "/short", nil, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := test.fetch(server.URL + test.path)
			if (err != nil) != test.wantErr || !reflect.DeepEqual(got, test.want) {
				t.Fatalf("resources = %#v, error = %v; want %#v, error = %v", got, err, test.want, test.wantErr)
			}
		})
	}
}

func TestCodec(t *testing.T) {
	content := NewFx()
	first := N{{Name: "a", Type: "b", Data: []byte("c")}}
	second := N{{Name: "d", Type: "e", Data: []byte{0, 255}}}
	for _, test := range []struct {
		name        string
		collections []N
		wire        []byte
	}{
		{"none", nil, []byte{}},
		{"empty", []N{nil}, []byte{0}},
		{"empty-record", []N{{{Data: []byte{}}}}, []byte{1, 0, 0, 0}},
		{"multiple", []N{first, {}, second}, []byte{1, 1, 'a', 1, 'b', 1, 'c', 0, 1, 1, 'd', 1, 'e', 2, 0, 255}},
	} {
		t.Run(test.name, func(t *testing.T) {
			wire := content.Encode(test.collections...)
			if !bytes.Equal(wire, test.wire) {
				t.Fatalf("wire = %v, want %v", wire, test.wire)
			}
			want := N{}
			for _, entries := range test.collections {
				want = append(want, entries...)
			}
			got, err := content.Decode(wire)
			if err != nil || !reflect.DeepEqual(got, want) {
				t.Fatalf("decoded = %#v, error = %v; want %#v", got, err, want)
			}
		})
	}
}

func TestEncodeAllocation(t *testing.T) {
	content := NewFx()
	entry := &One{
		Name: string(bytes.Repeat([]byte{'n'}, 127)),
		Type: string(bytes.Repeat([]byte{'t'}, 128)),
		Data: bytes.Repeat([]byte{'d'}, 16384),
	}
	entries := make(N, 129)
	for i := range entries {
		entries[i] = entry
	}
	var wire []byte
	allocations := testing.AllocsPerRun(10, func() {
		wire = content.Encode(entries[:1], N{}, entries[1:])
	})
	if allocations != 1 || cap(wire) != len(wire) {
		t.Fatalf("allocations = %g, capacity = %d, length = %d", allocations, cap(wire), len(wire))
	}
	got, err := content.Decode(wire)
	if err != nil || !reflect.DeepEqual(got, entries) {
		t.Fatalf("large records did not round-trip: error = %v", err)
	}
}

func TestDecodeInvalid(t *testing.T) {
	content := NewFx()
	for _, wire := range [][]byte{
		{0x80},
		bytes.Repeat([]byte{0x80}, 11),
		append(bytes.Repeat([]byte{0xff}, 9), 2),
		{1, 0, 0},
		{1, 5, 0, 0},
		{1, 0, 0, 0x80},
		append([]byte{1, 0, 0}, bytes.Repeat([]byte{0x80}, 11)...),
		{1, 0, 0, 0, 0x80},
		{0, 1, 0, 0},
	} {
		if got, err := content.Decode(wire); err == nil || got != nil {
			t.Fatalf("malformed wire %v returned %#v, error = %v", wire, got, err)
		}
	}
}

func TestDecodeOwnership(t *testing.T) {
	content := NewFx()
	wire := content.Encode(N{{Name: "name", Type: "type", Data: []byte("payload")}})
	entries, err := content.Decode(wire)
	if err != nil {
		t.Fatal(err)
	}
	saved := bytes.Clone(wire)
	entries[0].Data[0] = 'P'
	if !bytes.Equal(wire, saved) {
		t.Fatal("decoded data aliases the encoded input")
	}
	clear(wire)
	if got := entries[0]; got.Name != "name" || got.Type != "type" || string(got.Data) != "Payload" {
		t.Fatalf("input mutation changed decoded fields: %#v", got)
	}
}
