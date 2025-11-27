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
	title := os.Getenv("TITLE")
	apiURL := os.Getenv("API_URL")

	if title == "" {
		title = "hello_universe"
	}
	if apiURL == "" {
		apiURL = "http://localhost:1001"
	}

	favicon := apiURL + "/img/favicon"
	pathless, err := template.New("pathless").Parse(zero)
	if err != nil {
		panic("template parse error: " + err.Error())
	}

	var b strings.Builder
	if err := pathless.Execute(&b, map[string]string{
		"Title":   title,
		"APIURL":  apiURL,
		"Favicon": favicon,
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
	// Minify CSS in <style> tags
	html = regexp.MustCompile(`<style>([\s\S]*?)</style>`).ReplaceAllStringFunc(html, func(s string) string {
		s = regexp.MustCompile(`/\*[\s\S]*?\*/`).ReplaceAllString(s, "")    // Remove CSS comments
		s = regexp.MustCompile(`\s*([{}:;,])\s*`).ReplaceAllString(s, "$1") // Remove spaces around CSS syntax
		s = regexp.MustCompile(`;\s*}`).ReplaceAllString(s, "}")            // Remove last semicolon before }
		s = regexp.MustCompile(`\s+`).ReplaceAllString(s, " ")              // Collapse whitespace
		return strings.TrimSpace(s)
	})

	// Minify JavaScript in <script> tags - preserve modern syntax
	html = regexp.MustCompile(`<script>([\s\S]*?)</script>`).ReplaceAllStringFunc(html, func(s string) string {
		s = regexp.MustCompile(`//[^\n]*\n`).ReplaceAllString(s, "\n")   // Remove single-line comments
		s = regexp.MustCompile(`/\*[\s\S]*?\*/`).ReplaceAllString(s, "") // Remove multi-line comments
		s = regexp.MustCompile(`\n+`).ReplaceAllString(s, "\n")          // Collapse newlines
		s = regexp.MustCompile(`\t+`).ReplaceAllString(s, "")            // Remove tabs
		// Only collapse multiple spaces, don't remove necessary whitespace
		s = regexp.MustCompile(`  +`).ReplaceAllString(s, " ")
		return strings.TrimSpace(s)
	})
	// Remove whitespace between HTML tags, collapse multiple spaces/newlines to single space, trim leading/trailing whitespace
	html = regexp.MustCompile(`>\s+<`).ReplaceAllString(html, "><")
	html = regexp.MustCompile(`\s+`).ReplaceAllString(html, " ")
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
