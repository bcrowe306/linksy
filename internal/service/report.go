package service

import (
	"errors"
	"time"

	"github.com/bcrowe306/linksy/internal/auth"
	"github.com/bcrowe306/linksy/internal/models"
	"github.com/bcrowe306/linksy/internal/repository"
)

const maxReportSpan = 2 * 366 * 24 * time.Hour

// ReportQuery selects a time range (in the viewer's time zone) and optional asset.
type ReportQuery struct {
	From, To time.Time
	TZ       *time.Location
	LinkID   string
	QRID     string
}

type Totals struct {
	Clicks   int64 `json:"clicks"`
	Scans    int64 `json:"scans"`
	Total    int64 `json:"total"`
	Visitors int64 `json:"visitors"`
}

type SeriesPoint struct {
	T      time.Time `json:"t"`
	Clicks int64     `json:"clicks"`
	Scans  int64     `json:"scans"`
}

type Report struct {
	From        time.Time               `json:"from"`
	To          time.Time               `json:"to"`
	Timezone    string                  `json:"timezone"`
	Granularity string                  `json:"granularity"` // hour | day | week
	Totals      Totals                  `json:"totals"`
	Previous    Totals                  `json:"previous"`
	Series      []SeriesPoint           `json:"series"`
	Heatmap     [7][24]int64            `json:"heatmap"` // [weekday Sun=0][local hour]
	Countries   []models.NameCount      `json:"countries"`
	Regions     []models.NameCount      `json:"regions"`
	Cities      []repository.CityCount  `json:"cities"`
	Devices     []models.NameCount      `json:"devices"`
	Browsers    []models.NameCount      `json:"browsers"`
	OS          []models.NameCount      `json:"os"`
	Referrers   []models.NameCount      `json:"referrers"`
	Channels    []models.NameCount      `json:"channels"`
	TopLinks    []repository.AssetCount `json:"top_links"`
	TopQRCodes  []repository.AssetCount `json:"top_qr_codes"`
}

func granularityFor(span time.Duration) string {
	switch {
	case span <= 48*time.Hour:
		return "hour"
	case span <= 120*24*time.Hour:
		return "day"
	}
	return "week"
}

// bucketStart floors t (already in the report's zone) to its bucket.
func bucketStart(t time.Time, g string) time.Time {
	y, m, d := t.Date()
	switch g {
	case "hour":
		return time.Date(y, m, d, t.Hour(), 0, 0, 0, t.Location())
	case "week": // weeks start on Monday
		day := time.Date(y, m, d, 0, 0, 0, 0, t.Location())
		offset := (int(day.Weekday()) + 6) % 7
		return day.AddDate(0, 0, -offset)
	}
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}

func nextBucket(t time.Time, g string) time.Time {
	switch g {
	case "hour":
		return t.Add(time.Hour)
	case "week":
		return t.AddDate(0, 0, 7)
	}
	return t.AddDate(0, 0, 1)
}

func (s *Service) totals(f repository.AnalyticsFilter) (Totals, error) {
	var t Totals
	sum, err := s.repo.GetAnalyticsSummary(f)
	if err != nil {
		return t, err
	}
	t.Clicks, t.Scans = sum.TotalClicks, sum.TotalScans
	t.Total = t.Clicks + t.Scans
	t.Visitors, err = s.repo.UniqueVisitors(f)
	return t, err
}

// Report builds the full analytics report for the actor's team.
func (s *Service) Report(a Actor, q ReportQuery) (*Report, error) {
	if err := a.require(auth.PermViewAnalytics); err != nil {
		return nil, err
	}
	if q.TZ == nil {
		q.TZ = time.UTC
	}
	if !q.To.After(q.From) {
		return nil, errors.New("the end of the range must be after the start")
	}
	if q.To.Sub(q.From) > maxReportSpan {
		return nil, errors.New("reports can cover at most two years")
	}
	from, to := q.From.UTC(), q.To.UTC()
	f := repository.AnalyticsFilter{TeamID: a.TeamID, LinkID: q.LinkID, QRID: q.QRID, From: &from, To: &to}

	r := &Report{From: q.From.In(q.TZ), To: q.To.In(q.TZ), Timezone: q.TZ.String(), Granularity: granularityFor(q.To.Sub(q.From))}
	var err error
	if r.Totals, err = s.totals(f); err != nil {
		return nil, err
	}
	span := to.Sub(from)
	pFrom, pTo := from.Add(-span), from
	if r.Previous, err = s.totals(repository.AnalyticsFilter{TeamID: a.TeamID, LinkID: q.LinkID, QRID: q.QRID, From: &pFrom, To: &pTo}); err != nil {
		return nil, err
	}

	// Zero-filled series in local buckets + weekday×hour heatmap
	hours, err := s.repo.HourlyCounts(f)
	if err != nil {
		return nil, err
	}
	index := map[int64]int{}
	for b := bucketStart(r.From, r.Granularity); b.Before(r.To); b = nextBucket(b, r.Granularity) {
		index[b.Unix()] = len(r.Series)
		r.Series = append(r.Series, SeriesPoint{T: b})
	}
	for _, h := range hours {
		local := h.Hour.In(q.TZ)
		if i, ok := index[bucketStart(local, r.Granularity).Unix()]; ok {
			if h.Type == "scan" {
				r.Series[i].Scans += h.Count
			} else {
				r.Series[i].Clicks += h.Count
			}
		}
		r.Heatmap[int(local.Weekday())][local.Hour()] += h.Count
	}

	breakdowns := []struct {
		dest     *[]models.NameCount
		col, def string
		limit    int
	}{
		{&r.Countries, "country", "Unknown", 250},
		{&r.Regions, "region", "Unknown", 20},
		{&r.Devices, "device", "Unknown", 10},
		{&r.Browsers, "browser", "Unknown", 10},
		{&r.OS, "os", "Unknown", 10},
		{&r.Referrers, "referrer", "Direct", 15},
		{&r.Channels, "channel", "Direct", 10},
	}
	for _, b := range breakdowns {
		if *b.dest, err = s.repo.Breakdown(f, b.col, b.def, b.limit); err != nil {
			return nil, err
		}
	}
	// Referring sites lists real domains only; "no referrer" is the Direct channel.
	refs := r.Referrers[:0]
	for _, x := range r.Referrers {
		if x.Name != "Direct" {
			refs = append(refs, x)
		}
	}
	r.Referrers = refs
	if r.Cities, err = s.repo.TopCities(f, 20); err != nil {
		return nil, err
	}
	if r.TopLinks, err = s.repo.TopAssets(f, "link", 10); err != nil {
		return nil, err
	}
	if r.TopQRCodes, err = s.repo.TopAssets(f, "qr", 10); err != nil {
		return nil, err
	}
	return r, nil
}

// ExportEvents returns raw events for CSV export (capped).
func (s *Service) ExportEvents(a Actor, q ReportQuery, limit int) ([]repository.ExportRow, error) {
	if err := a.require(auth.PermExportAssets); err != nil {
		return nil, err
	}
	from, to := q.From.UTC(), q.To.UTC()
	return s.repo.ExportEvents(repository.AnalyticsFilter{TeamID: a.TeamID, LinkID: q.LinkID, QRID: q.QRID, From: &from, To: &to}, limit)
}
