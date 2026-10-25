package e2e_test

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bcrowe306/linksy/internal/models"
	"github.com/bcrowe306/linksy/test/testkit"
)

func TestHealthEndpoint(t *testing.T) {
	k := testkit.New(t)
	r := k.Do("GET", "/health", "", nil)
	k.Expect(r, http.StatusOK, "health")
	for _, h := range []string{"X-Content-Type-Options", "X-Frame-Options", "Content-Security-Policy"} {
		if r.Header.Get(h) == "" {
			t.Errorf("missing security header %s", h)
		}
	}
}

func TestLinksAndRedirectE2E(t *testing.T) {
	k := testkit.New(t)
	u := k.Login("links@acme.test")

	r := k.Do("POST", "/api/v1/links", u.Token, map[string]any{"target_url": "https://google.com", "custom_slug": "my-goog", "title": "Google"})
	k.Expect(r, http.StatusCreated, "create link")
	var link models.Link
	_ = r.JSON(&link)

	k.Expect(k.Do("POST", "/api/v1/links", u.Token, map[string]any{"target_url": "https://google.com", "custom_slug": "my-goog"}), http.StatusConflict, "duplicate slug")
	k.Expect(k.Do("POST", "/api/v1/links", u.Token, map[string]any{"target_url": "http://169.254.169.254/"}), http.StatusBadRequest, "SSRF target")

	red := k.Do("GET", "/s/my-goog", "", nil)
	k.Expect(red, http.StatusFound, "redirect")
	if loc := red.Header.Get("Location"); loc != "https://google.com" {
		t.Errorf("redirect to %q", loc)
	}

	k.Expect(k.Do("PUT", "/api/v1/links/"+link.ID, u.Token, map[string]any{"target_url": "https://duckduckgo.com"}), http.StatusOK, "update")
	if loc := k.Do("GET", "/s/my-goog", "", nil).Header.Get("Location"); loc != "https://duckduckgo.com" {
		t.Errorf("updated redirect to %q", loc)
	}

	k.Expect(k.Do("GET", "/api/v1/links/"+link.ID, u.Token, nil), http.StatusOK, "get")
	k.Expect(k.Do("DELETE", "/api/v1/links/"+link.ID, u.Token, nil), http.StatusNoContent, "delete")
	k.Expect(k.Do("GET", "/s/my-goog", "", nil, "Accept", "application/json"), http.StatusNotFound, "deleted redirect")

	// Links default their title to the destination host/path
	auto := k.Do("POST", "/api/v1/links", u.Token, map[string]any{"target_url": "https://www.example.com/spring/"})
	var al models.Link
	_ = auto.JSON(&al)
	if al.Title != "example.com/spring" {
		t.Errorf("default title = %q", al.Title)
	}
}

func TestBrandedErrorEscapesInput(t *testing.T) {
	k := testkit.New(t)
	r := k.Do("GET", "/s/%3Cscript%3Ealert(1)%3C%2Fscript%3E", "", nil, "Accept", "text/html")
	k.Expect(r, http.StatusNotFound, "missing slug")
	if strings.Contains(string(r.Body), "<script>alert(1)") {
		t.Fatal("slug reflected unescaped into HTML error page")
	}
}

func TestQRCodeAndExportE2E(t *testing.T) {
	k := testkit.New(t)
	u := k.Login("qr@acme.test")

	r := k.Do("POST", "/api/v1/qr", u.Token, map[string]any{"title": "Office", "type": "website", "target_url": "https://example.com", "is_dynamic": true})
	k.Expect(r, http.StatusCreated, "create QR")
	var qr models.QRCode
	_ = r.JSON(&qr)

	for format, mime := range map[string]string{"png": "image/png", "jpeg": "image/jpeg", "svg": "image/svg+xml", "pdf": "application/pdf"} {
		e := k.Do("GET", "/api/v1/qr/"+qr.ID+"/export?format="+format, u.Token, nil)
		k.Expect(e, http.StatusOK, "export "+format)
		if ct := e.Header.Get("Content-Type"); ct != mime {
			t.Errorf("export %s content type %q", format, ct)
		}
	}
	k.Expect(k.Do("GET", "/api/v1/qr/"+qr.ID+"/export?format=png", "", nil), http.StatusUnauthorized, "anonymous export")

	scan := k.Do("GET", "/qr/r/"+qr.ID, "", nil)
	k.Expect(scan, http.StatusFound, "scan redirect")

	k.Expect(k.Do("PUT", "/api/v1/qr/"+qr.ID, u.Token, map[string]any{"title": "Office 2", "target_url": "https://example.org"}), http.StatusOK, "update QR")
	k.Expect(k.Do("DELETE", "/api/v1/qr/"+qr.ID, u.Token, nil), http.StatusNoContent, "delete QR")
}

func upload(k *testkit.Kit, path, method, token, name string, content []byte) *testkit.Response {
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	part, _ := w.CreateFormFile("file", name)
	_, _ = part.Write(content)
	w.Close()
	req := httptest.NewRequest(method, path, &b)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	return k.Send(req)
}

func TestDocumentUploadAndSharingE2E(t *testing.T) {
	k := testkit.New(t)
	u := k.Login("docs@acme.test")

	r := upload(k, "/api/v1/documents/upload", "POST", u.Token, "report.pdf", []byte("%PDF-1.4 sample pdf content"))
	k.Expect(r, http.StatusCreated, "upload")
	var res struct {
		Document    models.Document `json:"document"`
		DownloadURL string          `json:"download_url"`
		QRCode      models.QRCode   `json:"qr_code"`
	}
	_ = r.JSON(&res)
	if res.Document.Filename != "report.pdf" || res.QRCode.DocumentID == nil || *res.QRCode.DocumentID != res.Document.ID {
		t.Fatalf("unexpected upload response: %s", r.Body)
	}

	path := strings.TrimPrefix(res.DownloadURL, testkit.AppURL)
	dl := k.Do("GET", path, "", nil)
	k.Expect(dl, http.StatusOK, "public download")
	if !strings.HasPrefix(string(dl.Body), "%PDF") {
		t.Fatal("download body mismatch")
	}

	// HTML uploads are never served inline
	h := upload(k, "/api/v1/documents/upload", "POST", u.Token, "x.html", []byte("<html><script>alert(1)</script></html>"))
	k.Expect(h, http.StatusCreated, "html upload")
	var hres struct {
		DownloadURL string `json:"download_url"`
	}
	_ = h.JSON(&hres)
	hd := k.Do("GET", strings.TrimPrefix(hres.DownloadURL, testkit.AppURL), "", nil)
	if !strings.HasPrefix(hd.Header.Get("Content-Disposition"), "attachment") || strings.Contains(hd.Header.Get("Content-Type"), "html") {
		t.Fatalf("html served inline: %v", hd.Header)
	}

	// Replace keeps the document ID and bumps the version
	rep := upload(k, "/api/v1/documents/"+res.Document.ID, "PUT", u.Token, "report-v2.pdf", []byte("%PDF-1.4 v2"))
	k.Expect(rep, http.StatusOK, "replace")
	var repRes struct {
		Document models.Document `json:"document"`
	}
	_ = rep.JSON(&repRes)
	if repRes.Document.ID != res.Document.ID || repRes.Document.Version != 2 {
		t.Fatalf("replace changed identity: %+v", repRes.Document)
	}

	k.Expect(k.Do("DELETE", "/api/v1/documents/"+res.Document.ID+"?delete_qr=true", u.Token, nil), http.StatusOK, "delete doc")
	k.Expect(k.Do("GET", "/api/v1/qr/"+res.QRCode.ID, u.Token, nil), http.StatusNotFound, "doc QR deleted")
}

func TestOpenAPISpecAndDocsEndpoint(t *testing.T) {
	k := testkit.New(t)
	k.Expect(k.Do("GET", "/api/v1/openapi.json", "", nil), http.StatusOK, "spec")
	k.Expect(k.Do("GET", "/api/docs", "", nil), http.StatusOK, "docs")
	k.Expect(k.Do("GET", "/api/v1/does-not-exist", "", nil), http.StatusUnauthorized, "unknown api route requires auth")
}

func TestUnauthenticatedAndForgedCredentials(t *testing.T) {
	k := testkit.New(t)
	victim := k.Login("victim@acme.test")
	k.Expect(k.Do("POST", "/api/v1/links", victim.Token, map[string]any{"target_url": "https://victim.example"}), http.StatusCreated, "seed")

	forged := []string{
		"",
		victim.ID,                        // raw user id used to be accepted
		"sess_" + victim.ID + "_1700000", // legacy unsigned session format
		"workos_sess_" + victim.ID,
		"eyJhbGciOiJub25lIn0.eyJzdWIiOiJ1c2VyX3ZpY3RpbV9hY21lX3Rlc3QifQ.", // unsigned JWT
		"lqs_forged",
		"lk_live_forged",
		"randomgarbage",
	}
	for _, tok := range forged {
		r := k.Do("GET", "/api/v1/links", tok, nil)
		if r.Status != http.StatusUnauthorized {
			t.Errorf("token %q: expected 401, got %d", tok, r.Status)
		}
	}
	k.Expect(k.Do("GET", "/api/v1/user/profile", "", nil), http.StatusUnauthorized, "anonymous profile")

	me := k.Do("GET", "/api/v1/auth/me", "", nil).Map()
	if me["authenticated"] != false || me["user"] != nil {
		t.Fatalf("anonymous /me leaked a user: %v", me)
	}
}

func TestTeamHeaderSpoofingIsRejected(t *testing.T) {
	k := testkit.New(t)
	alice := k.Login("alice@acme.test")
	mallory := k.Login("mallory@evil.test")
	k.Expect(k.Do("POST", "/api/v1/links", alice.Token, map[string]any{"target_url": "https://alice.example"}), http.StatusCreated, "seed")

	r := k.Do("GET", "/api/v1/links", mallory.Token, nil, "X-Team-ID", alice.TeamID)
	k.Expect(r, http.StatusForbidden, "spoofed X-Team-ID")

	r = k.Do("GET", "/api/v1/analytics/summary", mallory.Token, nil, "X-Team-ID", "team_default")
	k.Expect(r, http.StatusForbidden, "legacy default team")

	var list struct {
		Data []models.Link `json:"data"`
	}
	_ = k.Do("GET", "/api/v1/links", mallory.Token, nil).JSON(&list)
	if len(list.Data) != 0 {
		t.Fatalf("mallory sees %d foreign links", len(list.Data))
	}
}

func TestCookieAuthRequiresSameOrigin(t *testing.T) {
	k := testkit.New(t)
	u := k.Login("cookie@acme.test")

	send := func(origin string) int {
		req := httptest.NewRequest("POST", "/api/v1/links", strings.NewReader(`{"target_url":"https://example.com"}`))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(&http.Cookie{Name: "lynqr_session", Value: u.Token})
		if origin != "" {
			req.Header.Set("Origin", origin)
		}
		return k.Send(req).Status
	}
	if s := send("https://evil.example"); s != http.StatusForbidden {
		t.Fatalf("cross-origin cookie POST: expected 403, got %d", s)
	}
	if s := send("http://example.com"); s != http.StatusCreated {
		t.Fatalf("same-origin cookie POST: expected 201, got %d", s)
	}
}

func TestAPIKeyLifecycleAndScope(t *testing.T) {
	k := testkit.New(t)
	u := k.Login("dev@acme.test")

	r := k.Do("POST", "/api/v1/api-keys", u.Token, map[string]any{"name": "CI"})
	k.Expect(r, http.StatusCreated, "create key")
	var created struct {
		APIKey models.APIKey `json:"api_key"`
	}
	_ = r.JSON(&created)
	key := created.APIKey.Key
	if !strings.HasPrefix(key, "lk_live_") {
		t.Fatalf("bad key %q", key)
	}

	k.Expect(k.Do("POST", "/api/v1/links", key, map[string]any{"target_url": "https://example.com"}), http.StatusCreated, "key creates link")
	k.Expect(k.Do("POST", "/api/v1/api-keys", key, map[string]any{"name": "nested"}), http.StatusForbidden, "keys cannot mint keys")

	other := k.Login("other@acme.test")
	k.Expect(k.Do("GET", "/api/v1/links", key, nil, "X-Team-ID", other.TeamID), http.StatusOK, "key ignores X-Team-ID")

	k.Expect(k.Do("DELETE", "/api/v1/api-keys/"+created.APIKey.ID, u.Token, nil), http.StatusOK, "revoke")
	k.Expect(k.Do("GET", "/api/v1/links", key, nil), http.StatusUnauthorized, "revoked key")
}

func TestTeamManagementOverHTTP(t *testing.T) {
	k := testkit.New(t)
	owner := k.Login("boss@acme.test")
	friend := k.Login("friend@acme.test")

	r := k.Do("POST", "/api/v1/team/members/invite", owner.Token, map[string]any{"email": "friend@acme.test", "role": "viewer"})
	k.Expect(r, http.StatusCreated, "invite")
	var inv struct {
		Invitation models.Invitation `json:"invitation"`
	}
	_ = r.JSON(&inv)
	token := inv.Invitation.AcceptURL[strings.LastIndex(inv.Invitation.AcceptURL, "/")+1:]

	prev := k.Do("GET", "/api/v1/invitations/"+token, "", nil)
	k.Expect(prev, http.StatusOK, "preview")
	if prev.Map()["email"] != "friend@acme.test" {
		t.Fatalf("preview: %s", prev.Body)
	}

	k.Expect(k.Do("POST", "/api/v1/invitations/"+token+"/accept", friend.Token, nil), http.StatusOK, "accept")

	me := k.Do("GET", "/api/v1/auth/me", friend.Token, nil).Map()
	if me["role"] != "viewer" || len(me["teams"].([]any)) != 2 {
		t.Fatalf("after accept, /me = %v", me)
	}

	// Viewer can read but not write in the shared workspace
	k.Expect(k.Do("GET", "/api/v1/links", friend.Token, nil), http.StatusOK, "viewer list")
	k.Expect(k.Do("POST", "/api/v1/links", friend.Token, map[string]any{"target_url": "https://example.com"}), http.StatusForbidden, "viewer create")
	k.Expect(k.Do("POST", "/api/v1/team/members/invite", friend.Token, map[string]any{"email": "x@y.test"}), http.StatusForbidden, "viewer invite")
	k.Expect(k.Do("GET", "/api/v1/team/audit", friend.Token, nil), http.StatusForbidden, "viewer audit")

	// Switch back to personal workspace where the friend is owner
	k.Expect(k.Do("POST", "/api/v1/teams/switch", friend.Token, map[string]any{"team_id": friend.TeamID}), http.StatusOK, "switch")
	k.Expect(k.Do("POST", "/api/v1/links", friend.Token, map[string]any{"target_url": "https://example.com"}), http.StatusCreated, "owner create")
	k.Expect(k.Do("POST", "/api/v1/teams/switch", friend.Token, map[string]any{"team_id": "team_nope"}), http.StatusForbidden, "switch to foreign team")

	// Owner manages the member
	k.Expect(k.Do("PUT", "/api/v1/team/members/"+friend.ID+"/role", owner.Token, map[string]any{"role": "member"}), http.StatusOK, "promote")
	k.Expect(k.Do("POST", "/api/v1/team/members/"+friend.ID+"/suspend", owner.Token, nil), http.StatusOK, "suspend")
	k.Expect(k.Do("GET", "/api/v1/links", friend.Token, nil, "X-Team-ID", owner.TeamID), http.StatusForbidden, "suspended access")
	k.Expect(k.Do("POST", "/api/v1/team/members/"+friend.ID+"/reactivate", owner.Token, nil), http.StatusOK, "reactivate")
	k.Expect(k.Do("DELETE", "/api/v1/team/members/"+friend.ID, owner.Token, nil), http.StatusOK, "remove")
	k.Expect(k.Do("GET", "/api/v1/links", friend.Token, nil, "X-Team-ID", owner.TeamID), http.StatusForbidden, "removed access")

	k.Expect(k.Do("PUT", "/api/v1/team", owner.Token, map[string]any{"name": "Acme HQ"}), http.StatusOK, "rename")
	audit := k.Do("GET", "/api/v1/team/audit", owner.Token, nil)
	k.Expect(audit, http.StatusOK, "audit")
	if !strings.Contains(string(audit.Body), "member.suspend") {
		t.Fatalf("audit log missing suspend event: %s", audit.Body)
	}

	k.Expect(k.Do("DELETE", "/api/v1/team", owner.Token, map[string]any{"confirm_name": "wrong"}), http.StatusBadRequest, "delete needs confirmation")
	k.Expect(k.Do("DELETE", "/api/v1/team", owner.Token, map[string]any{"confirm_name": "Acme HQ"}), http.StatusOK, "delete team")
	me = k.Do("GET", "/api/v1/auth/me", owner.Token, nil).Map()
	if me["authenticated"] != true || me["team"] == nil {
		t.Fatalf("owner should land in a fresh workspace after deletion: %v", me)
	}
}

func TestLogoutRevokesSession(t *testing.T) {
	k := testkit.New(t)
	u := k.Login("bye@acme.test")
	k.Expect(k.Do("GET", "/api/v1/links", u.Token, nil), http.StatusOK, "before logout")
	k.Expect(k.Do("POST", "/api/v1/auth/logout", u.Token, nil), http.StatusOK, "logout")
	k.Expect(k.Do("GET", "/api/v1/links", u.Token, nil), http.StatusUnauthorized, "after logout")
}

func TestDevLoginDisabledByDefault(t *testing.T) {
	k := testkit.New(t)
	k.Expect(k.Do("POST", "/api/v1/auth/dev-login", "", map[string]any{"email": "a@b.test"}), http.StatusNotFound, "dev login")
	r := k.Do("POST", "/api/v1/auth/password-reset", "", map[string]any{"email": "a@b.test"})
	if strings.Contains(string(r.Body), "password_reset_url") {
		t.Fatal("password reset must never return the reset URL")
	}
}
