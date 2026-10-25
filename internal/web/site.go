// Package web renders Lynqr's public, server-side pages (guides and legal
// documents) with the same styles and scripts as the landing page.
package web

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"

	"github.com/bcrowe306/linksy/docs"
)

//go:embed templates/*.html
var templateFS embed.FS

// Page is a Markdown document with front matter.
type Page struct {
	Slug        string
	Title       string
	Description string
	Updated     string
	Order       int
	HTML        template.HTML
}

// Site holds parsed templates and pre-rendered pages.
type Site struct {
	tmpl   *template.Template
	assets template.HTML
	Guides []Page
	Legal  map[string]Page
}

var assetTag = regexp.MustCompile(`<(?:script|link)[^>]+(?:src|href)="/assets/[^"]+"[^>]*>(?:</script>)?`)

// Load reads the built landing page to reuse its CSS/JS bundle, then renders
// the embedded guides and legal documents.
func Load(distDir string) (*Site, error) {
	home, err := os.ReadFile(distDir + "/home.html")
	if err != nil {
		return nil, fmt.Errorf("public pages need the built landing page: %w", err)
	}
	s := &Site{assets: template.HTML(strings.Join(assetTag.FindAllString(string(home), -1), "\n    ")), Legal: map[string]Page{}}
	s.tmpl, err = template.ParseFS(templateFS, "templates/*.html")
	if err != nil {
		return nil, err
	}
	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithParserOptions(parser.WithAutoHeadingID(), parser.WithAttribute()),
		goldmark.WithRendererOptions(html.WithUnsafe()), // our own trusted docs (e.g. <kbd>)
	)
	if s.Guides, err = loadPages(md, docs.Guides, "guides"); err != nil {
		return nil, err
	}
	legal, err := loadPages(md, docs.Legal, "legal")
	if err != nil {
		return nil, err
	}
	for _, p := range legal {
		s.Legal[p.Slug] = p
	}
	return s, nil
}

func loadPages(md goldmark.Markdown, fsys fs.FS, dir string) ([]Page, error) {
	entries, err := fs.ReadDir(fsys, dir)
	if err != nil {
		return nil, err
	}
	var out []Page
	for _, e := range entries {
		raw, err := fs.ReadFile(fsys, dir+"/"+e.Name())
		if err != nil {
			return nil, err
		}
		p := Page{Slug: strings.TrimSuffix(e.Name(), ".md"), Order: 99}
		body := string(raw)
		if strings.HasPrefix(body, "---\n") {
			if end := strings.Index(body[4:], "\n---\n"); end >= 0 {
				for _, line := range strings.Split(body[4:4+end], "\n") {
					k, v, ok := strings.Cut(line, ":")
					if !ok {
						continue
					}
					switch v = strings.TrimSpace(v); strings.TrimSpace(k) {
					case "title":
						p.Title = v
					case "description":
						p.Description = v
					case "updated":
						p.Updated = v
					case "order":
						p.Order, _ = strconv.Atoi(v)
					}
				}
				body = body[4+end+5:]
			}
		}
		// The page template renders the title, so drop the document's own H1.
		body = strings.TrimLeft(body, "\n")
		if strings.HasPrefix(body, "# ") {
			if nl := strings.Index(body, "\n"); nl > 0 {
				body = body[nl+1:]
			}
		}
		var buf bytes.Buffer
		if err := md.Convert([]byte(body), &buf); err != nil {
			return nil, fmt.Errorf("%s/%s: %w", dir, e.Name(), err)
		}
		p.HTML = template.HTML(buf.String())
		out = append(out, p)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Order < out[j].Order })
	return out, nil
}

// Guide returns a guide by slug.
func (s *Site) Guide(slug string) (Page, bool) {
	for _, g := range s.Guides {
		if g.Slug == slug {
			return g, true
		}
	}
	return Page{}, false
}

// View is the data passed to page templates.
type View struct {
	Title       string
	Description string
	Path        string
	NoIndex     bool
	Page        Page
	Guides      []Page
	JSONLD      template.JS
	Assets      template.HTML
}

// Render executes a page template.
func (s *Site) Render(w io.Writer, name string, v View) error {
	v.Assets = s.assets
	return s.tmpl.ExecuteTemplate(w, name, v)
}
