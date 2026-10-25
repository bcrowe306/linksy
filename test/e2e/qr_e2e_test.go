package e2e_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/bcrowe306/linksy/internal/models"
	"github.com/bcrowe306/linksy/test/testkit"
)

func createQR(t *testing.T, k *testkit.Kit, token string, body map[string]any) models.QRCode {
	t.Helper()
	r := k.Do("POST", "/api/v1/qr", token, body)
	k.Expect(r, http.StatusCreated, "create QR "+body["type"].(string))
	var q models.QRCode
	_ = r.JSON(&q)
	return q
}

func path(url string) string { return strings.TrimPrefix(url, testkit.AppURL) }

func TestDynamicWebsiteQRCountsScans(t *testing.T) {
	k := testkit.New(t)
	u := k.Login("scan@acme.test")
	q := createQR(t, k, u.Token, map[string]any{"type": "website", "title": "Poster", "target_url": "https://example.com/a", "is_dynamic": true})
	if !q.IsDynamic || q.Code == "" || q.Payload != testkit.AppURL+"/q/"+q.Code || q.ScanURL != q.Payload || q.TargetURL != "https://example.com/a" {
		t.Fatalf("unexpected dynamic QR: %+v", q)
	}

	scan := k.Do("GET", "/q/"+q.Code, "", nil, "User-Agent", "Mozilla/5.0 (iPhone)")
	k.Expect(scan, http.StatusFound, "scan")
	if scan.Header.Get("Location") != "https://example.com/a" {
		t.Fatalf("scan redirect: %s", scan.Header.Get("Location"))
	}

	// Re-point the printed code without changing what it encodes
	upd := k.Do("PUT", "/api/v1/qr/"+q.ID, u.Token, map[string]any{"target_url": "https://example.com/b"})
	k.Expect(upd, http.StatusOK, "update")
	var q2 models.QRCode
	_ = upd.JSON(&q2)
	if q2.Payload != q.Payload {
		t.Fatal("dynamic payload must never change")
	}
	if k.Do("GET", "/q/"+q.Code, "", nil).Header.Get("Location") != "https://example.com/b" {
		t.Fatal("scan did not follow the new destination")
	}

	// Scans count as scans, not link clicks
	var got models.QRCode
	_ = k.Do("GET", "/api/v1/qr/"+q.ID, u.Token, nil).JSON(&got)
	if got.ScanCount != 2 {
		t.Fatalf("expected 2 scans, got %d", got.ScanCount)
	}
	var link models.Link
	_ = k.Do("GET", "/api/v1/links/"+*q.LinkID, u.Token, nil).JSON(&link)
	if link.ClickCount != 0 {
		t.Fatalf("QR scans must not count as link clicks, got %d", link.ClickCount)
	}

	// Pausing the destination shows the branded inactive page
	k.Expect(k.Do("PUT", "/api/v1/links/"+link.ID, u.Token, map[string]any{"target_url": "https://example.com/b", "is_active": false}), http.StatusOK, "pause")
	k.Expect(k.Do("GET", "/q/"+q.Code, "", nil), http.StatusGone, "paused scan")

	k.Expect(k.Do("GET", "/q/nope123", "", nil), http.StatusNotFound, "unknown code")

	// Deleting the QR removes its private destination link
	k.Expect(k.Do("DELETE", "/api/v1/qr/"+q.ID, u.Token, nil), http.StatusNoContent, "delete")
	k.Expect(k.Do("GET", "/api/v1/links/"+link.ID, u.Token, nil), http.StatusNotFound, "link cleaned up")
}

func TestAppStoreQRRoutesByDevice(t *testing.T) {
	k := testkit.New(t)
	u := k.Login("app@acme.test")
	raw, _ := json.Marshal(map[string]string{
		"app_store_url": "https://apps.apple.com/app/id1", "play_store_url": "https://play.google.com/store/apps/details?id=x",
	})
	q := createQR(t, k, u.Token, map[string]any{"type": "app", "title": "App", "raw_data": string(raw)})
	cases := map[string]string{
		"Mozilla/5.0 (iPhone; CPU iPhone OS 17_0)": "https://apps.apple.com/app/id1",
		"Mozilla/5.0 (Linux; Android 14; Pixel 8)": "https://play.google.com/store/apps/details?id=x",
	}
	for ua, want := range cases {
		r := k.Do("GET", "/q/"+q.Code, "", nil, "User-Agent", ua)
		if r.Status != http.StatusFound || r.Header.Get("Location") != want {
			t.Errorf("%s: got %d %s", ua, r.Status, r.Header.Get("Location"))
		}
	}
	desk := k.Do("GET", "/q/"+q.Code, "", nil, "User-Agent", "Mozilla/5.0 (Macintosh)")
	if desk.Status != http.StatusOK || !strings.Contains(string(desk.Body), "Google Play") {
		t.Fatalf("desktop should get a store chooser, got %d", desk.Status)
	}
}

func TestDynamicVCardServesContactFile(t *testing.T) {
	k := testkit.New(t)
	u := k.Login("card@acme.test")
	q := createQR(t, k, u.Token, map[string]any{"type": "vcard", "title": "Card", "is_dynamic": true,
		"raw_data": `{"first_name":"Ada","last_name":"Lovelace","email":"ada@example.com"}`})
	r := k.Do("GET", "/q/"+q.Code, "", nil)
	k.Expect(r, http.StatusOK, "vcard scan")
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "text/vcard") || !strings.Contains(string(r.Body), "FN:Ada Lovelace") {
		t.Fatalf("unexpected vcard response: %s %s", r.Header.Get("Content-Type"), r.Body)
	}
}

func TestDocumentQRSurvivesReplacement(t *testing.T) {
	k := testkit.New(t)
	u := k.Login("menu@acme.test")
	up := upload(k, "/api/v1/documents/upload", "POST", u.Token, "menu.pdf", []byte("%PDF-1.4 monday"))
	var res struct {
		Document models.Document `json:"document"`
		QRCode   models.QRCode   `json:"qr_code"`
	}
	_ = up.JSON(&res)
	if res.QRCode.Code == "" || res.QRCode.Type != models.QRTypeDocument {
		t.Fatalf("upload should create a dynamic document QR: %+v", res.QRCode)
	}

	first := k.Do("GET", "/q/"+res.QRCode.Code, "", nil)
	k.Expect(first, http.StatusFound, "first scan")
	k.Expect(k.Do("GET", path(first.Header.Get("Location")), "", nil), http.StatusOK, "v1 download")

	upload(k, "/api/v1/documents/"+res.Document.ID, "PUT", u.Token, "menu.pdf", []byte("%PDF-1.4 tuesday"))
	second := k.Do("GET", "/q/"+res.QRCode.Code, "", nil)
	if second.Header.Get("Location") == first.Header.Get("Location") {
		t.Fatal("scan should follow the replaced file")
	}
	body := k.Do("GET", path(second.Header.Get("Location")), "", nil)
	if !strings.Contains(string(body.Body), "tuesday") {
		t.Fatalf("expected the new version, got %q", body.Body)
	}

	// Stable document link
	d := k.Do("GET", "/d/"+res.Document.ID, "", nil)
	k.Expect(d, http.StatusFound, "stable document link")
	if d.Header.Get("Location") != second.Header.Get("Location") {
		t.Fatal("/d/ should point at the current version")
	}
}

func TestStaticQRTypes(t *testing.T) {
	k := testkit.New(t)
	u := k.Login("static@acme.test")
	cases := []map[string]any{
		{"type": "wifi", "title": "Wi-Fi", "raw_data": `{"ssid":"Cafe","password":"secret","encryption":"WPA"}`},
		{"type": "text", "title": "Note", "raw_data": "Hello from Lynqr"},
		{"type": "email", "title": "Mail", "raw_data": `{"email":"hi@lynqr.app","subject":"Hi"}`},
		{"type": "sms", "title": "SMS", "raw_data": `{"phone":"+15550100","message":"Yo"}`},
		{"type": "phone", "title": "Call", "raw_data": `{"phone":"+15550100"}`},
		{"type": "vcard", "title": "Card", "raw_data": `{"first_name":"Ada"}`},
		{"type": "website", "title": "Static site", "target_url": "https://example.com"},
	}
	for _, c := range cases {
		q := createQR(t, k, u.Token, c)
		if q.IsDynamic || q.Code != "" {
			t.Errorf("%s should be static: %+v", c["type"], q)
		}
		k.Expect(k.Do("GET", "/api/v1/qr/"+q.ID+"/export?format=svg", u.Token, nil), http.StatusOK, "export "+c["type"].(string))
	}
	bad := k.Do("POST", "/api/v1/qr", u.Token, map[string]any{"type": "wifi", "title": "x", "raw_data": `{"password":"p"}`})
	k.Expect(bad, http.StatusBadRequest, "wifi without ssid")
	k.Expect(k.Do("POST", "/api/v1/qr", u.Token, map[string]any{"type": "website", "title": "x", "target_url": "http://10.0.0.1/"}), http.StatusBadRequest, "private target")
	k.Expect(k.Do("POST", "/api/v1/qr", u.Token, map[string]any{"type": "document", "title": "x"}), http.StatusBadRequest, "document without file")
	k.Expect(k.Do("POST", "/api/v1/qr", u.Token, map[string]any{"type": "website", "target_url": "https://example.com"}), http.StatusBadRequest, "missing title")

	// Static content changes re-encode the payload
	q := createQR(t, k, u.Token, map[string]any{"type": "text", "title": "T", "raw_data": "one"})
	var upd models.QRCode
	_ = k.Do("PUT", "/api/v1/qr/"+q.ID, u.Token, map[string]any{"raw_data": "two"}).JSON(&upd)
	if upd.Payload != "two" {
		t.Fatalf("static payload not updated: %q", upd.Payload)
	}
}

func TestQRPreview(t *testing.T) {
	k := testkit.New(t)
	u := k.Login("preview@acme.test")
	r := k.Do("POST", "/api/v1/qr/preview", u.Token, map[string]any{
		"type": "website", "target_url": "https://example.com", "is_dynamic": true,
		"style": map[string]any{"fg_color": "#4f46e5", "dot_style": "dots", "frame_style": "banner-bottom", "label_text": "SCAN ME"},
	})
	k.Expect(r, http.StatusOK, "preview")
	var p struct {
		SVG     string `json:"svg"`
		Payload string `json:"payload"`
		Dynamic bool   `json:"dynamic"`
		Check   struct {
			Scannable bool `json:"scannable"`
		} `json:"check"`
	}
	_ = r.JSON(&p)
	if !strings.HasPrefix(p.SVG, "<svg") || !strings.Contains(p.SVG, "SCAN ME") || !p.Dynamic || !p.Check.Scannable || !strings.Contains(p.Payload, "/q/") {
		t.Fatalf("unexpected preview: dyn=%v scan=%v payload=%s", p.Dynamic, p.Check.Scannable, p.Payload)
	}
	k.Expect(k.Do("POST", "/api/v1/qr/preview", u.Token, map[string]any{"type": "wifi", "raw_data": `{}`}), http.StatusBadRequest, "invalid preview")
	k.Expect(k.Do("POST", "/api/v1/qr/preview", "", map[string]any{"type": "text", "raw_data": "x"}), http.StatusUnauthorized, "anonymous preview")
}

func TestExportOptions(t *testing.T) {
	k := testkit.New(t)
	u := k.Login("export@acme.test")
	q := createQR(t, k, u.Token, map[string]any{"type": "text", "title": "T", "raw_data": "hello"})
	for _, qs := range []string{"format=png&size=4096", "format=pdf&layout=card", "format=pdf&layout=sheet", "format=jpg&size=256"} {
		r := k.Do("GET", "/api/v1/qr/"+q.ID+"/export?"+qs+"&download=true", u.Token, nil)
		k.Expect(r, http.StatusOK, qs)
		if !strings.Contains(r.Header.Get("Content-Disposition"), "attachment") {
			t.Errorf("%s: missing attachment disposition", qs)
		}
	}
}
