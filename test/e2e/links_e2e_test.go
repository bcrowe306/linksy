package e2e_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/bcrowe306/linksy/internal/models"
	"github.com/bcrowe306/linksy/test/testkit"
)

func createLink(t *testing.T, k *testkit.Kit, token string, body map[string]any) models.Link {
	t.Helper()
	r := k.Do("POST", "/api/v1/links", token, body)
	k.Expect(r, http.StatusCreated, "create link")
	var l models.Link
	_ = r.JSON(&l)
	return l
}

func TestUTMAutoTagging(t *testing.T) {
	k := testkit.New(t)
	u := k.Login("utm@acme.test")
	l := createLink(t, k, u.Token, map[string]any{
		"target_url": "https://example.com/p?utm_source=keep&x=1", "custom_slug": "utm-test",
		"utm_source": "newsletter", "utm_medium": "email", "utm_campaign": "spring sale",
	})
	loc, _ := url.Parse(k.Do("GET", "/s/utm-test", "", nil).Header.Get("Location"))
	q := loc.Query()
	if q.Get("utm_source") != "keep" || q.Get("utm_medium") != "email" || q.Get("utm_campaign") != "spring sale" || q.Get("x") != "1" {
		t.Fatalf("unexpected tagged destination: %s", loc)
	}

	// Clearing one UTM field leaves the others intact
	k.Expect(k.Do("PUT", "/api/v1/links/"+l.ID, u.Token, map[string]any{"utm_medium": ""}), http.StatusOK, "clear medium")
	loc, _ = url.Parse(k.Do("GET", "/s/utm-test", "", nil).Header.Get("Location"))
	if loc.Query().Get("utm_medium") != "" || loc.Query().Get("utm_campaign") != "spring sale" {
		t.Fatalf("partial UTM update failed: %s", loc)
	}
	k.Expect(k.Do("POST", "/api/v1/links", u.Token, map[string]any{"target_url": "https://example.com", "utm_source": strings.Repeat("x", 101)}), http.StatusBadRequest, "long utm")
}

func TestPasswordProtectedLink(t *testing.T) {
	k := testkit.New(t)
	u := k.Login("pw@acme.test")
	l := createLink(t, k, u.Token, map[string]any{"target_url": "https://example.com/secret", "custom_slug": "vip", "password": "opensesame"})
	if !l.HasPassword {
		t.Fatal("has_password should be true")
	}
	if strings.Contains(string(k.Do("GET", "/api/v1/links/"+l.ID, u.Token, nil).Body), "$2a$") {
		t.Fatal("password hash leaked in API response")
	}

	gate := k.Do("GET", "/s/vip", "", nil)
	k.Expect(gate, http.StatusUnauthorized, "password page")
	if !strings.Contains(string(gate.Body), "This link is protected") {
		t.Fatal("expected password form")
	}
	if strings.Contains(string(gate.Body), "example.com") {
		t.Fatal("password page must not reveal the protected destination")
	}
	// Browsers apply form-action to the post-unlock redirect, which leaves our origin.
	if !strings.Contains(gate.Header.Get("Content-Security-Policy"), "form-action 'self' https:") {
		t.Fatalf("password page CSP blocks the destination redirect: %s", gate.Header.Get("Content-Security-Policy"))
	}

	post := func(pw string, cookies ...*http.Cookie) *testkit.Response {
		req := httptest.NewRequest("POST", "/s/vip", strings.NewReader(url.Values{"password": {pw}}.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		for _, c := range cookies {
			req.AddCookie(c)
		}
		return k.Send(req)
	}
	k.Expect(post("wrong"), http.StatusUnauthorized, "wrong password")
	ok := post("opensesame")
	k.Expect(ok, http.StatusFound, "correct password")
	if ok.Header.Get("Location") != "https://example.com/secret" {
		t.Fatalf("unlock redirect: %s", ok.Header.Get("Location"))
	}

	// The unlock cookie skips the form on later visits
	var unlock *http.Cookie
	for _, c := range (&http.Response{Header: ok.Header}).Cookies() {
		if strings.HasPrefix(c.Name, "lq_unlock_") {
			unlock = c
		}
	}
	if unlock == nil || !unlock.HttpOnly {
		t.Fatal("expected an HttpOnly unlock cookie")
	}
	req := httptest.NewRequest("GET", "/s/vip", nil)
	req.AddCookie(unlock)
	k.Expect(k.Send(req), http.StatusFound, "unlocked visit")

	// Changing the password revokes existing unlocks
	k.Expect(k.Do("PUT", "/api/v1/links/"+l.ID, u.Token, map[string]any{"password": "newpass1"}), http.StatusOK, "change password")
	req = httptest.NewRequest("GET", "/s/vip", nil)
	req.AddCookie(unlock)
	k.Expect(k.Send(req), http.StatusUnauthorized, "stale unlock")

	// Removing protection
	k.Expect(k.Do("PUT", "/api/v1/links/"+l.ID, u.Token, map[string]any{"password": ""}), http.StatusOK, "remove password")
	k.Expect(k.Do("GET", "/s/vip", "", nil), http.StatusFound, "unprotected")

	k.Expect(k.Do("POST", "/api/v1/links", u.Token, map[string]any{"target_url": "https://example.com", "password": "abc"}), http.StatusBadRequest, "short password")
}

func TestPausedExpiredAndFallback(t *testing.T) {
	k := testkit.New(t)
	u := k.Login("fallback@acme.test")
	l := createLink(t, k, u.Token, map[string]any{"target_url": "https://example.com/live", "custom_slug": "promo"})

	k.Expect(k.Do("PUT", "/api/v1/links/"+l.ID, u.Token, map[string]any{"is_active": false}), http.StatusOK, "pause")
	paused := k.Do("GET", "/s/promo", "", nil, "Accept", "text/html")
	k.Expect(paused, http.StatusGone, "paused")
	if !strings.Contains(string(paused.Body), "Link Paused") {
		t.Fatal("paused page missing")
	}

	k.Expect(k.Do("PUT", "/api/v1/links/"+l.ID, u.Token, map[string]any{"is_active": false, "fallback_url": "https://example.com/ended"}), http.StatusOK, "fallback")
	if loc := k.Do("GET", "/s/promo", "", nil).Header.Get("Location"); loc != "https://example.com/ended" {
		t.Fatalf("fallback redirect: %s", loc)
	}
	k.Expect(k.Do("PUT", "/api/v1/links/"+l.ID, u.Token, map[string]any{"fallback_url": "http://127.0.0.1/"}), http.StatusBadRequest, "private fallback")

	k.Expect(k.Do("PUT", "/api/v1/links/"+l.ID, u.Token, map[string]any{"is_active": true, "fallback_url": "", "expires_at": "2020-01-01T00:00:00Z"}), http.StatusOK, "expire")
	k.Expect(k.Do("GET", "/s/promo", "", nil), http.StatusGone, "expired")

	var got models.Link
	_ = k.Do("GET", "/api/v1/links/"+l.ID, u.Token, nil).JSON(&got)
	if got.ClickCount != 0 {
		t.Fatalf("paused/expired visits must not count as clicks, got %d", got.ClickCount)
	}
}

func TestChangeSlug(t *testing.T) {
	k := testkit.New(t)
	u := k.Login("slug@acme.test")
	l := createLink(t, k, u.Token, map[string]any{"target_url": "https://example.com", "custom_slug": "old-name"})
	createLink(t, k, u.Token, map[string]any{"target_url": "https://example.com", "custom_slug": "taken"})
	k.Expect(k.Do("PUT", "/api/v1/links/"+l.ID, u.Token, map[string]any{"slug": "taken"}), http.StatusConflict, "taken slug")
	k.Expect(k.Do("PUT", "/api/v1/links/"+l.ID, u.Token, map[string]any{"slug": "new-name"}), http.StatusOK, "rename")
	k.Expect(k.Do("GET", "/s/new-name", "", nil), http.StatusFound, "new slug")
	k.Expect(k.Do("GET", "/s/old-name", "", nil), http.StatusNotFound, "old slug")
}

func TestQRAttachedToExistingLink(t *testing.T) {
	k := testkit.New(t)
	u := k.Login("attach@acme.test")
	l := createLink(t, k, u.Token, map[string]any{"target_url": "https://example.com/menu", "custom_slug": "menu", "password": "tablefor2", "utm_source": "qr"})
	q := createQR(t, k, u.Token, map[string]any{"type": "website", "title": "Table tent", "link_id": l.ID})
	if q.OwnsLink || q.LinkID == nil || *q.LinkID != l.ID || !q.IsDynamic {
		t.Fatalf("expected an attached, non-owning dynamic QR: %+v", q)
	}

	// Protected destination: the scan is counted, then the unlock page takes over without counting a click
	scan := k.Do("GET", "/q/"+q.Code, "", nil)
	k.Expect(scan, http.StatusFound, "scan")
	if !strings.HasSuffix(scan.Header.Get("Location"), "/s/menu?src=qr") {
		t.Fatalf("protected scan should go to the unlock page: %s", scan.Header.Get("Location"))
	}

	// Deleting the QR keeps the user's link
	k.Expect(k.Do("DELETE", "/api/v1/qr/"+q.ID, u.Token, nil), http.StatusNoContent, "delete qr")
	k.Expect(k.Do("GET", "/api/v1/links/"+l.ID, u.Token, nil), http.StatusOK, "link kept")

	k.Expect(k.Do("POST", "/api/v1/qr", u.Token, map[string]any{"type": "wifi", "title": "x", "link_id": l.ID, "raw_data": `{"ssid":"a"}`}), http.StatusBadRequest, "non-website attach")
	other := k.Login("other@acme.test")
	k.Expect(k.Do("POST", "/api/v1/qr", other.Token, map[string]any{"type": "website", "title": "x", "link_id": l.ID}), http.StatusNotFound, "foreign link")
}
