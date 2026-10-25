package e2e_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/bcrowe306/linksy/test/testkit"
)

func TestRobotsAndSitemap(t *testing.T) {
	k := testkit.New(t)
	r := k.Do("GET", "/robots.txt", "", nil, "Host", "test.lynqr.app")
	k.Expect(r, http.StatusOK, "robots on test host")
	if !strings.Contains(string(r.Body), "Disallow: /\n") {
		t.Fatalf("test host must not be indexed: %s", r.Body)
	}
	r = k.Do("GET", "/robots.txt", "", nil, "X-Forwarded-Host", "lynqr.app")
	if body := string(r.Body); !strings.Contains(body, "Sitemap: https://lynqr.app/sitemap.xml") || !strings.Contains(body, "Disallow: /admin") {
		t.Fatalf("production robots: %s", body)
	}
	r = k.Do("GET", "/sitemap.xml", "", nil)
	k.Expect(r, http.StatusOK, "sitemap")
	if !strings.Contains(string(r.Body), "<loc>https://lynqr.app/</loc>") {
		t.Fatalf("sitemap: %s", r.Body)
	}
}

func TestPublicQRGenerator(t *testing.T) {
	k := testkit.New(t)
	r := k.Do("POST", "/api/v1/public/qr", "", map[string]any{"type": "website", "target_url": "example.com/menu", "dot_style": "rounded"})
	k.Expect(r, http.StatusOK, "preview")
	m := r.Map()
	if !strings.HasPrefix(m["svg"].(string), "<svg") || m["check"].(map[string]any)["scannable"] != true {
		t.Fatalf("preview: %v", m["check"])
	}
	png := k.Do("POST", "/api/v1/public/qr?format=png", "", map[string]any{"type": "wifi", "raw_data": `{"ssid":"Cafe","password":"pw12345678"}`, "size": 512})
	k.Expect(png, http.StatusOK, "png")
	if png.Header.Get("Content-Type") != "image/png" || !strings.HasPrefix(string(png.Body), "\x89PNG") {
		t.Fatalf("not a PNG: %s", png.Header.Get("Content-Type"))
	}
	svg := k.Do("POST", "/api/v1/public/qr?format=svg", "", map[string]any{"type": "vcard", "raw_data": `{"first_name":"Alex"}`})
	k.Expect(svg, http.StatusOK, "svg")
	if !strings.Contains(svg.Header.Get("Content-Disposition"), "qr-code.svg") {
		t.Fatal("svg should download")
	}
	k.Expect(k.Do("POST", "/api/v1/public/qr", "", map[string]any{"type": "document"}), http.StatusBadRequest, "dynamic types are Pro-only")
	k.Expect(k.Do("POST", "/api/v1/public/qr", "", map[string]any{"type": "website", "target_url": "http://127.0.0.1/admin"}), http.StatusBadRequest, "private targets")
	k.Expect(k.Do("POST", "/api/v1/public/qr", "", map[string]any{"type": "email", "raw_data": `{"email":"nope"}`}), http.StatusBadRequest, "invalid email")

	// Blocked domains can't be turned into QR codes either
	admin := k.AdminLogin("mod@lynqr.test")
	k.Expect(k.Do("POST", "/api/v1/admin/blocklist", admin.Token, map[string]any{"domain": "evil.example"}), http.StatusCreated, "block")
	r = k.Do("POST", "/api/v1/public/qr", "", map[string]any{"type": "website", "target_url": "https://login.evil.example"})
	k.Expect(r, http.StatusBadRequest, "blocked domain")
}

func TestWWWRedirectsToApex(t *testing.T) {
	k := testkit.New(t)
	r := k.Do("GET", "/s/spring-menu?utm_source=qr", "", nil, "X-Forwarded-Host", "www.lynqr.app")
	k.Expect(r, http.StatusMovedPermanently, "www redirect")
	if loc := r.Header.Get("Location"); loc != "https://lynqr.app/s/spring-menu?utm_source=qr" {
		t.Fatalf("location: %s", loc)
	}
	k.Expect(k.Do("GET", "/health", "", nil, "X-Forwarded-Host", "lynqr.app"), http.StatusOK, "apex untouched")
}
