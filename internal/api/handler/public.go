package handler

import (
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/bcrowe306/linksy/internal/api/middleware"
)

// DistDir holds the built frontend (app shell and public landing page).
const DistDir = "./web/dist"

// Home serves the app to signed-in users and the public landing page to everyone else.
func (h *Handler) Home(c fiber.Ctx) error {
	c.Set("Cache-Control", "no-cache")
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
	{"/api-docs", "monthly", "0.5"},
	{"/login", "yearly", "0.3"},
}

func (h *Handler) SitemapXML(c fiber.Ctx) error {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n" + `<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">` + "\n")
	for _, p := range sitemapPages {
		fmt.Fprintf(&b, "  <url><loc>https://lynqr.app%s</loc><changefreq>%s</changefreq><priority>%s</priority></url>\n", p.path, p.freq, p.priority)
	}
	b.WriteString("</urlset>\n")
	c.Set("Content-Type", "application/xml; charset=utf-8")
	return c.SendString(b.String())
}
