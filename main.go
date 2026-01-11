package main

import (
	"bytes"
	"compress/gzip"
	_ "embed"
	"html/template"
	"net/http"
	"os"
	"regexp"
	"strings"
)

//go:embed pathless.html
var zero string
var one []byte

func init() {
	apiURL := os.Getenv("API_URL")
	if apiURL == "" {
		apiURL = "http://localhost:1001"
	}

	pathless, err := template.New("pathless").Parse(zero)
	if err != nil {
		panic("template parse error: " + err.Error())
	}

	var b strings.Builder
	if err := pathless.Execute(&b, map[string]string{
		"APIURL": apiURL,
	}); err != nil {
		panic("template execute error: " + err.Error())
	}
	one = minify(b.String())
}

func Pathless(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" || r.URL.RawQuery != "" {
		http.Redirect(w, r, "/", http.StatusMovedPermanently)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Encoding", "gzip")
	w.Write(one)
}

func main() {
	http.HandleFunc("/", Pathless)
	http.ListenAndServe(":1000", nil)
}

func minify(html string) []byte {
	// <style>
	html = regexp.MustCompile(`<style>([\s\S]*?)</style>`).ReplaceAllStringFunc(html, func(s string) string {
		s = regexp.MustCompile(`/\*[\s\S]*?\*/`).ReplaceAllString(s, "")
		s = regexp.MustCompile(`\s*([{}:;,>+~])\s*`).ReplaceAllString(s, "$1")
		s = regexp.MustCompile(`;\s*}`).ReplaceAllString(s, "}")
		s = regexp.MustCompile(`\s+`).ReplaceAllString(s, " ")
		s = strings.ReplaceAll(s, "0px", "0")
		s = strings.ReplaceAll(s, "0em", "0")
		s = strings.ReplaceAll(s, "0%", "0")
		s = strings.ReplaceAll(s, " 0 0 0 0", " 0")
		s = strings.ReplaceAll(s, ":0 0 0 0", ":0")
		s = strings.ReplaceAll(s, "<style> ", "<style>")
		s = strings.ReplaceAll(s, " </style>", "</style>")
		return s
	})

	// <script>
	html = regexp.MustCompile(`<script>([\s\S]*?)</script>`).ReplaceAllStringFunc(html, func(s string) string {
		s = regexp.MustCompile(`//[^\n]*\n`).ReplaceAllString(s, "\n")
		s = regexp.MustCompile(`/\*[\s\S]*?\*/`).ReplaceAllString(s, "")
		// Remove whitespace around operators and punctuation
		s = regexp.MustCompile(`\s*([=+\-*/<>!&|?:,;{}()\[\]])\s*`).ReplaceAllString(s, "$1")
		// Restore necessary spaces after keywords
		for _, kw := range []string{"const", "let", "var", "return", "async", "await", "function", "if", "else", "for", "of", "in", "new", "throw", "typeof", "instanceof"} {
			s = strings.ReplaceAll(s, kw+"(", kw+" (")
			s = strings.ReplaceAll(s, kw+"{", kw+" {")
		}
		// Collapse whitespace
		s = regexp.MustCompile(`\n+`).ReplaceAllString(s, "\n")
		s = regexp.MustCompile(`\t+`).ReplaceAllString(s, "")
		s = regexp.MustCompile(`  +`).ReplaceAllString(s, " ")
		s = regexp.MustCompile(`\n\s*\n`).ReplaceAllString(s, "\n")
		// Remove newlines where safe (after ; { } ,)
		s = regexp.MustCompile(`([;{},])\n`).ReplaceAllString(s, "$1")
		s = regexp.MustCompile(`\n([;})])`).ReplaceAllString(s, "$1")
		s = strings.ReplaceAll(s, "<script>\n", "<script>")
		s = strings.ReplaceAll(s, "\n</script>", "</script>")
		return s
	})

	// <html>
	html = regexp.MustCompile(`<!--[\s\S]*?-->`).ReplaceAllString(html, "")
	html = regexp.MustCompile(`>\s+<`).ReplaceAllString(html, "><")
	html = regexp.MustCompile(`\s+`).ReplaceAllString(html, " ")
	html = strings.ReplaceAll(html, " />", ">")
	html = strings.ReplaceAll(html, "\" >", "\">")
	html = strings.TrimSpace(html)

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write([]byte(html)); err != nil {
		panic("gzip write error: " + err.Error())
	}
	if err := gz.Close(); err != nil {
		panic("gzip close error: " + err.Error())
	}
	return buf.Bytes()
}
