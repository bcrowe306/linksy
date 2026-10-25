package service_test

import (
	"testing"
	"time"

	"github.com/bcrowe306/linksy/internal/models"
	"github.com/bcrowe306/linksy/internal/service"
	"github.com/bcrowe306/linksy/test/testkit"
)

const iphone = "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 Mobile/15E148 Safari/604.1"
const chromeMac = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 Chrome/120.0 Safari/537.36"

func TestIngestionEnrichment(t *testing.T) {
	k := testkit.New(t)
	u := k.Login("ingest@acme.test")
	a := k.Actor(u, u.TeamID)
	link, _ := k.Svc.CreateShortLink(a, service.CreateLinkInput{TargetURL: "https://example.com"})

	track := func(ua, ref, country string) {
		k.Svc.TrackEvent(&models.AnalyticsEvent{TeamID: u.TeamID, LinkID: &link.ID, EventType: "click", IP: "203.0.113.9", UserAgent: ua, Referrer: ref, Country: country})
	}
	track(iphone, "https://l.instagram.com/?u=x", "us")
	track(chromeMac, "https://mail.google.com/mail/u/0/", "GB")
	track(chromeMac, "https://www.google.com/search?q=lynqr", "XX")
	track(chromeMac, "", "DE")
	// Dropped: bots and link-preview fetchers
	track("Slackbot-LinkExpanding 1.0 (+https://api.slack.com/robots)", "", "US")
	track("facebookexternalhit/1.1", "", "US")
	track("Googlebot/2.1 (+http://www.google.com/bot.html)", "", "US")
	track("", "", "US")

	now := time.Now()
	r, err := k.Svc.Report(a, service.ReportQuery{From: now.Add(-time.Hour), To: now.Add(time.Hour)})
	if err != nil {
		t.Fatal(err)
	}
	if r.Totals.Clicks != 4 {
		t.Fatalf("bots must be excluded: got %d clicks", r.Totals.Clicks)
	}
	if r.Totals.Visitors != 1 {
		t.Fatalf("one IP should be one visitor, got %d", r.Totals.Visitors)
	}
	got := map[string]int64{}
	for _, c := range r.Channels {
		got[c.Name] = c.Count
	}
	if got["social"] != 1 || got["email"] != 1 || got["search"] != 1 || got["direct"] != 1 {
		t.Fatalf("channel classification wrong: %v", got)
	}
	refs := map[string]bool{}
	for _, x := range r.Referrers {
		refs[x.Name] = true
	}
	if !refs["l.instagram.com"] || !refs["google.com"] || refs["https://www.google.com/search?q=lynqr"] {
		t.Fatalf("referrers should be stored as domains only: %v", r.Referrers)
	}
	countries := map[string]bool{}
	for _, c := range r.Countries {
		countries[c.Name] = true
	}
	if !countries["US"] || !countries["GB"] || countries["XX"] {
		t.Fatalf("country normalization: %v", r.Countries)
	}
	if r.Devices[0].Name != "Desktop" || len(r.TopLinks) != 1 || r.TopLinks[0].Count != 4 {
		t.Fatalf("devices/top links: %v %v", r.Devices, r.TopLinks)
	}
}

func TestReportSeriesTimezoneAndComparison(t *testing.T) {
	k := testkit.New(t)
	u := k.Login("series@acme.test")
	a := k.Actor(u, u.TeamID)
	qrc, _ := k.Svc.CreateQRCode(a, service.CreateQRInput{Type: models.QRTypeText, Title: "T", RawData: "hi"})

	ny, _ := time.LoadLocation("America/New_York")
	day := time.Date(2026, 3, 10, 0, 0, 0, 0, ny) // a Tuesday
	at := func(t time.Time, typ string) {
		k.Svc.TrackEvent(&models.AnalyticsEvent{TeamID: u.TeamID, QRCodeID: &qrc.ID, EventType: typ, IP: t.String(), UserAgent: iphone, CreatedAt: t.UTC()})
	}
	at(day.Add(1*time.Hour), "scan")                  // Tue 01:00 local (05:00/06:00 UTC)
	at(day.Add(23*time.Hour+30*time.Minute), "scan")  // Tue 23:30 local = Wed UTC
	at(day.AddDate(0, 0, 2).Add(9*time.Hour), "scan") // Thu 09:00 local
	at(day.AddDate(0, 0, -5), "scan")                 // previous period

	r, err := k.Svc.Report(a, service.ReportQuery{From: day, To: day.AddDate(0, 0, 7), TZ: ny})
	if err != nil {
		t.Fatal(err)
	}
	if r.Granularity != "day" || len(r.Series) != 7 {
		t.Fatalf("expected 7 daily buckets, got %s/%d", r.Granularity, len(r.Series))
	}
	if r.Series[0].Scans != 2 || r.Series[1].Scans != 0 || r.Series[2].Scans != 1 {
		t.Fatalf("events must land in local days: %+v", r.Series[:3])
	}
	if r.Heatmap[2][23] != 1 || r.Heatmap[2][1] != 1 || r.Heatmap[4][9] != 1 {
		t.Fatalf("heatmap must use local weekday/hour: tue23=%d tue1=%d thu9=%d", r.Heatmap[2][23], r.Heatmap[2][1], r.Heatmap[4][9])
	}
	if r.Totals.Scans != 3 || r.Previous.Scans != 1 {
		t.Fatalf("totals %d / previous %d", r.Totals.Scans, r.Previous.Scans)
	}
	if len(r.TopQRCodes) != 1 || r.TopQRCodes[0].Count != 3 {
		t.Fatalf("top QR codes: %+v", r.TopQRCodes)
	}

	hourly, _ := k.Svc.Report(a, service.ReportQuery{From: day, To: day.Add(24 * time.Hour), TZ: ny})
	if hourly.Granularity != "hour" || len(hourly.Series) != 24 || hourly.Series[1].Scans != 1 || hourly.Series[23].Scans != 1 {
		t.Fatalf("hourly series wrong: %s %d", hourly.Granularity, len(hourly.Series))
	}
	if _, err := k.Svc.Report(a, service.ReportQuery{From: day, To: day}); err == nil {
		t.Fatal("empty range must error")
	}
	if _, err := k.Svc.Report(a, service.ReportQuery{From: day.AddDate(-3, 0, 0), To: day}); err == nil {
		t.Fatal("ranges over two years must error")
	}
}
