package e2e_test

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
	"testing"

	"github.com/bcrowe306/linksy/test/testkit"
)

// Routes deliberately left out of the public API reference: browser session,
// account and workspace-administration flows that API keys can't call.
var undocumented = []string{
	"/api/v1/auth/", "/api/v1/user/", "/api/v1/api-keys", "/api/v1/teams", "/api/v1/invitations/",
	"/api/v1/team/members/", "/api/v1/team/invitations/", "/api/v1/team/transfer", "/api/v1/team/leave",
	"/api/v1/documents/files/", "/api/v1/documents/bulk-delete", "/api/v1/analytics/summary",
	"/api/v1/analytics/live", "/api/v1/billing", "/api/v1/admin/", "/api/v1/report", "/api/v1/openapi", "/api/docs", "/api/*",
}

// Session-only methods on otherwise documented paths.
var undocumentedOps = map[string]bool{"PUT /api/v1/team": true, "DELETE /api/v1/team": true, "POST /s/:slug": true}

func TestOpenAPISpecCoversPublicRoutes(t *testing.T) {
	k := testkit.New(t)
	r := k.Do("GET", "/api/v1/openapi.json", "", nil)
	k.Expect(r, http.StatusOK, "spec")
	var spec struct {
		OpenAPI string                    `json:"openapi"`
		Paths   map[string]map[string]any `json:"paths"`
	}
	if err := json.Unmarshal(r.Body, &spec); err != nil || spec.OpenAPI != "3.1.0" {
		t.Fatalf("invalid spec: %v", err)
	}
	k.Expect(k.Do("GET", "/api/v1/openapi.yaml", "", nil), http.StatusOK, "yaml")

	param := regexp.MustCompile(`:(\w+)`)
	for _, route := range k.App.GetRoutes(true) {
		p := route.Path
		if route.Method == "HEAD" || !(strings.HasPrefix(p, "/api/v1/") || strings.HasPrefix(p, "/s/") || strings.HasPrefix(p, "/q/")) {
			continue
		}
		skip := undocumentedOps[route.Method+" "+p]
		for _, u := range undocumented {
			skip = skip || strings.HasPrefix(p, u)
		}
		if skip {
			continue
		}
		specPath := param.ReplaceAllString(p, "{$1}")
		ops, ok := spec.Paths[specPath]
		if !ok {
			t.Errorf("route %s %s is not documented in docs/openapi.yaml", route.Method, p)
			continue
		}
		if _, ok := ops[strings.ToLower(route.Method)]; !ok {
			t.Errorf("method %s for %s is not documented", route.Method, specPath)
		}
	}
}
