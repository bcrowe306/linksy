package e2e_test

import (
	"encoding/csv"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	fws "github.com/fasthttp/websocket"

	"github.com/bcrowe306/linksy/internal/models"
	"github.com/bcrowe306/linksy/test/testkit"
)

const ua = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) Chrome/120.0 Safari/537.36"

func TestAnalyticsReportAndExports(t *testing.T) {
	k := testkit.New(t)
	u := k.Login("report@acme.test")
	l := createLink(t, k, u.Token, map[string]any{"target_url": "https://example.com", "custom_slug": "report-me", "title": "=HYPERLINK(\"evil\")"})
	for i := 0; i < 3; i++ {
		k.Do("GET", "/s/report-me", "", nil, "User-Agent", ua, "CF-IPCountry", "CA", "CF-IPCity", "Toronto", "Referer", "https://www.linkedin.com/feed/")
	}
	k.Do("GET", "/s/report-me", "", nil, "User-Agent", "Twitterbot/1.0")
	time.Sleep(200 * time.Millisecond) // events are recorded asynchronously

	r := k.Do("GET", "/api/v1/analytics/report?tz=America/Toronto&link_id="+l.ID, u.Token, nil)
	k.Expect(r, http.StatusOK, "report")
	var rep struct {
		Totals struct{ Clicks int64 }   `json:"totals"`
		Series []struct{ Clicks int64 } `json:"series"`
		Cities []struct {
			City  string
			Count int64
		} `json:"cities"`
		Channels []models.NameCount `json:"channels"`
	}
	_ = r.JSON(&rep)
	if rep.Totals.Clicks != 3 || len(rep.Series) == 0 || len(rep.Cities) != 1 || rep.Cities[0].City != "Toronto" || rep.Channels[0].Name != "social" {
		t.Fatalf("unexpected report: %s", r.Body)
	}

	k.Expect(k.Do("GET", "/api/v1/analytics/report?tz=Mars/Base", u.Token, nil), http.StatusBadRequest, "bad tz")
	k.Expect(k.Do("GET", "/api/v1/analytics/report?from=2020-01-01&to=2026-01-01", u.Token, nil), http.StatusBadRequest, "too long")

	c := k.Do("GET", "/api/v1/analytics/export?format=csv&link_id="+l.ID, u.Token, nil)
	k.Expect(c, http.StatusOK, "csv")
	recs, err := csv.NewReader(strings.NewReader(string(c.Body))).ReadAll()
	if err != nil || len(recs) != 4 || recs[0][0] != "timestamp_utc" {
		t.Fatalf("csv rows=%d err=%v", len(recs), err)
	}
	if !strings.HasPrefix(recs[1][3], "'=") {
		t.Fatalf("formula injection not neutralized: %q", recs[1][3])
	}
	j := k.Do("GET", "/api/v1/analytics/export?format=json", u.Token, nil)
	k.Expect(j, http.StatusOK, "json")
	p := k.Do("GET", "/api/v1/analytics/export?format=pdf&from=2026-01-01&to=2026-12-31", u.Token, nil)
	k.Expect(p, http.StatusOK, "pdf")
	if !strings.HasPrefix(string(p.Body), "%PDF") {
		t.Fatal("pdf export is not a PDF")
	}
	k.Expect(k.Do("GET", "/api/v1/analytics/export?format=xml", u.Token, nil), http.StatusBadRequest, "bad format")

	other := k.Login("nosy@acme.test")
	var o struct {
		Totals struct{ Clicks int64 } `json:"totals"`
	}
	_ = k.Do("GET", "/api/v1/analytics/report?link_id="+l.ID, other.Token, nil).JSON(&o)
	if o.Totals.Clicks != 0 {
		t.Fatal("another workspace can read this link's analytics")
	}
}

func TestLiveAnalyticsWebsocket(t *testing.T) {
	k := testkit.New(t)
	u := k.Login("live@acme.test")
	l := createLink(t, k, u.Token, map[string]any{"target_url": "https://example.com", "custom_slug": "live-one", "title": "Live link"})

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() { _ = k.App.Listener(ln) }()
	defer k.App.Shutdown()
	base := ln.Addr().String()

	dial := func(origin string) (*fws.Conn, *http.Response, error) {
		h := http.Header{}
		h.Set("Origin", origin)
		h.Set("Cookie", "lynqr_session="+u.Token)
		return fws.DefaultDialer.Dial("ws://"+base+"/api/v1/analytics/live", h)
	}
	if _, resp, err := dial("https://evil.example"); err == nil || resp == nil || resp.StatusCode != http.StatusForbidden {
		t.Fatalf("cross-origin websocket should be rejected, got %v %v", err, resp)
	}
	conn, _, err := dial("http://" + base)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()
	var ready map[string]any
	_ = conn.ReadJSON(&ready)

	req, _ := http.NewRequest("GET", "http://"+base+"/s/live-one", nil)
	req.Header.Set("User-Agent", ua)
	req.Header.Set("CF-IPCountry", "FR")
	cl := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	if _, err := cl.Do(req); err != nil {
		t.Fatal(err)
	}
	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	var ev struct {
		Type    string `json:"type"`
		Asset   string `json:"asset"`
		AssetID string `json:"asset_id"`
		Country string `json:"country"`
	}
	if err := conn.ReadJSON(&ev); err != nil {
		t.Fatalf("no live event: %v", err)
	}
	if ev.Type != "click" || ev.Asset != "Live link" || ev.AssetID != l.ID || ev.Country != "FR" {
		t.Fatalf("unexpected live event: %+v", ev)
	}
}
