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
