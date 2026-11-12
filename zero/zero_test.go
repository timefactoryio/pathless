package zero

import (
	"bytes"
	"compress/gzip"
	"html/template"
	"io"
	"testing"
)

func TestOrigin(t *testing.T) {
	for _, circuitURL := range []string{
		"https://first.example.test",
		"https://second.example.test/<tag>?a='&b=\"",
	} {
		t.Run(circuitURL, func(t *testing.T) {
			t.Parallel()
			var want bytes.Buffer
			err := template.Must(template.New("").Parse(origin)).Execute(&want, map[string]any{
				"CIRCUIT":  circuitURL,
				"PATHLESS": template.JS(pathlessScript),
				"INPUT":    template.JS(inputScript),
			})
			if err != nil {
				t.Fatal(err)
			}
			shell := NewZero(circuitURL)
			if !bytes.Equal(shell.Origin, want.Bytes()) {
				t.Fatal("cached rendering changed the shell output or escaping")
			}
		})
	}
}

func TestCompression(t *testing.T) {
	const circuitURL = "https://circuit.example.test"
	want := NewZero(circuitURL).Origin
	for _, test := range []struct {
		name       string
		flags      []bool
		compressed bool
	}{
		{"default", nil, false},
		{"false", []bool{false}, false},
		{"true", []bool{true}, true},
		{"first-false", []bool{false, true}, false},
		{"first-true", []bool{true, false}, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := NewZero(circuitURL, test.flags...).Origin
			if test.compressed {
				reader, err := gzip.NewReader(bytes.NewReader(got))
				if err != nil {
					t.Fatal(err)
				}
				got, err = io.ReadAll(reader)
				if err != nil {
					t.Fatal(err)
				}
				if err := reader.Close(); err != nil {
					t.Fatal(err)
				}
			}
			if !bytes.Equal(got, want) {
				t.Fatal("compression changed the rendered shell")
			}
		})
	}
}

func BenchmarkNewZero(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		NewZero("https://circuit.example.test")
	}
}
