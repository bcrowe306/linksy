package web

import (
	"os"
	"strings"
	"testing"
)

func TestLoadAndRender(t *testing.T) {
	dir := t.TempDir()
	home := `<html><head><script type="module" crossorigin src="/assets/home-abc.js"></script>
<link rel="stylesheet" crossorigin href="/assets/theme-def.css"><link rel="icon" href="/favicon.svg"></head></html>`
	if err := os.WriteFile(dir+"/home.html", []byte(home), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Guides) < 5 || s.Guides[0].Slug != "getting-started" {
		t.Fatalf("guides not loaded in order: %+v", s.Guides)
	}
	for _, slug := range []string{"terms", "privacy", "acceptable-use"} {
		if s.Legal[slug].Title == "" || s.Legal[slug].Updated == "" {
			t.Fatalf("legal page %s missing front matter", slug)
		}
	}
	var b strings.Builder
	g, _ := s.Guide("getting-started")
	if err := s.Render(&b, "doc", View{Title: g.Title, Path: "/guides/getting-started", Page: g, Guides: s.Guides[1:], JSONLD: `{"@type":"TechArticle"}`}); err != nil {
		t.Fatal(err)
	}
	out := b.String()
	for _, want := range []string{"/assets/home-abc.js", "/assets/theme-def.css", "<kbd>N</kbd>", `href="https://lynqr.app/guides/getting-started"`, "prose-lynqr", "TechArticle", "Start your free trial"} {
		if !strings.Contains(out, want) {
			t.Errorf("rendered guide missing %q", want)
		}
	}
	if strings.Count(out, "<h1") != 1 {
		t.Error("guide should have exactly one h1")
	}
	if strings.Contains(out, `rel="icon" href="/favicon.svg"></head>`) {
		t.Error("only /assets/ tags should be copied from home.html")
	}
}
