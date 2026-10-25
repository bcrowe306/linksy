package handler

import (
	"encoding/json"
	"fmt"
	"html/template"
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/bcrowe306/linksy/internal/api/middleware"
	"github.com/bcrowe306/linksy/internal/web"
)

// DistDir holds the built frontend (app shell and public landing page).
const DistDir = "./web/dist"

// Home serves the app to signed-in users and the public landing page to everyone else.
func (h *Handler) Home(c fiber.Ctx) error {
	// The same URL serves two different files, so never let a browser
	// revalidate one against the other (both share a build timestamp).
	c.Set("Cache-Control", "no-store")
	c.Set("Vary", "Cookie")
	c.Request().Header.Del("If-Modified-Since")
	c.Request().Header.Del("If-None-Match")
	if token := c.Cookies(middleware.SessionCookie); token != "" {
		if _, err := h.svc.ResolveSession(token); err == nil {
			return c.SendFile(DistDir + "/index.html")
		}
	}
	if !h.indexable(c) {
		c.Set("X-Robots-Tag", "noindex, nofollow")
	}
	return c.SendFile(DistDir + "/home.html")
}

// indexable is true only on the production host; previews and test hosts stay out of search.
func (h *Handler) indexable(c fiber.Ctx) bool {
	host := c.Get("X-Forwarded-Host")
	if host == "" {
		host = c.Host()
	}
	host = strings.ToLower(strings.TrimSpace(strings.Split(host, ",")[0]))
	return host == "lynqr.app" || host == "www.lynqr.app"
}

func (h *Handler) RobotsTxt(c fiber.Ctx) error {
	c.Set("Content-Type", "text/plain; charset=utf-8")
	if !h.indexable(c) {
		return c.SendString("User-agent: *\nDisallow: /\n")
	}
	return c.SendString("User-agent: *\nAllow: /$\nAllow: /api-docs\nAllow: /signup\nAllow: /login\nAllow: /marketing/\n" +
		"Disallow: /api/\nDisallow: /admin\nDisallow: /s/\nDisallow: /q/\nDisallow: /d/\nDisallow: /qr/\nDisallow: /settings\n\n" +
		"Sitemap: https://lynqr.app/sitemap.xml\n")
}

var sitemapPages = []struct{ path, freq, priority string }{
	{"/", "weekly", "1.0"},
	{"/signup", "monthly", "0.8"},
	{"/guides", "weekly", "0.7"},
	{"/terms", "yearly", "0.2"},
	{"/privacy", "yearly", "0.2"},
	{"/acceptable-use", "yearly", "0.2"},
	{"/api-docs", "monthly", "0.5"},
	{"/login", "yearly", "0.3"},
}

func (h *Handler) SitemapXML(c fiber.Ctx) error {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n" + `<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">` + "\n")
	pages := append([]struct{ path, freq, priority string }{}, sitemapPages...)
	for _, slug := range GeneratorSlugs() {
		pages = append(pages, struct{ path, freq, priority string }{generatorPath(slug), "monthly", "0.9"})
	}
	if h.site != nil {
		for _, g := range h.site.Guides {
			pages = append(pages, struct{ path, freq, priority string }{"/guides/" + g.Slug, "monthly", "0.6"})
		}
	}
	for _, p := range pages {
		fmt.Fprintf(&b, "  <url><loc>https://lynqr.app%s</loc><changefreq>%s</changefreq><priority>%s</priority></url>\n", p.path, p.freq, p.priority)
	}
	b.WriteString("</urlset>\n")
	c.Set("Content-Type", "application/xml; charset=utf-8")
	return c.SendString(b.String())
}

// Unsubscribe handles signed one-click unsubscribe links from emails
// (GET shows the result; POST is the RFC 8058 one-click request from mail apps).
func (h *Handler) Unsubscribe(c fiber.Ctx) error {
	label, err := h.svc.Unsubscribe(c.Query("u"), c.Query("k"), c.Query("s"))
	c.Set("X-Robots-Tag", "noindex, nofollow")
	if c.Method() == fiber.MethodPost {
		if err != nil {
			return c.SendStatus(fiber.StatusBadRequest)
		}
		return c.SendStatus(fiber.StatusOK)
	}
	if err != nil {
		return h.renderBrandedError(c, "Link Not Valid", "This unsubscribe link is invalid or has been changed. You can manage email settings from your Lynqr profile.", fiber.StatusBadRequest)
	}
	return h.renderNotice(c, "You're unsubscribed", "You won't get emails about "+label+" any more. You can turn them back on any time in your profile's notification settings.")
}

// SetSite enables the server-rendered public pages (guides and legal documents).
func (h *Handler) SetSite(s *web.Site) { h.site = s }

func (h *Handler) renderPage(c fiber.Ctx, name string, v web.View) error {
	var buf strings.Builder
	v.NoIndex = !h.indexable(c)
	if v.NoIndex {
		c.Set("X-Robots-Tag", "noindex, nofollow")
	}
	if err := h.site.Render(&buf, name, v); err != nil {
		return err
	}
	c.Set("Cache-Control", "public, max-age=300")
	return c.Type("html").SendString(buf.String())
}

func (h *Handler) GuidesIndex(c fiber.Ctx) error {
	return h.renderPage(c, "guides", web.View{Title: "Guides", Path: "/guides",
		Description: "Step-by-step guides for Lynqr short links, dynamic QR codes, analytics and teams.", Guides: h.site.Guides})
}

func (h *Handler) GuidePage(c fiber.Ctx) error {
	g, ok := h.site.Guide(c.Params("slug"))
	if !ok {
		return h.renderBrandedError(c, "Guide Not Found", "That guide doesn't exist. Browse all guides at lynqr.app/guides.", fiber.StatusNotFound)
	}
	others := make([]web.Page, 0, len(h.site.Guides))
	for _, o := range h.site.Guides {
		if o.Slug != g.Slug {
			others = append(others, o)
		}
	}
	ld, _ := json.Marshal(map[string]any{
		"@context": "https://schema.org", "@type": "TechArticle", "headline": g.Title, "description": g.Description,
		"url": "https://lynqr.app/guides/" + g.Slug, "publisher": map[string]any{"@type": "Organization", "name": "Lynqr", "url": "https://lynqr.app/"},
	})
	return h.renderPage(c, "doc", web.View{Title: g.Title, Description: g.Description, Path: "/guides/" + g.Slug, Page: g, Guides: others, JSONLD: template.JS(ld)})
}

// LegalPage serves /terms, /privacy and /acceptable-use.
func (h *Handler) LegalPage(slug string) fiber.Handler {
	return func(c fiber.Ctx) error {
		p, ok := h.site.Legal[slug]
		if !ok {
			return fiber.ErrNotFound
		}
		return h.renderPage(c, "doc", web.View{Title: p.Title, Description: p.Description, Path: "/" + slug, Page: p})
	}
}
