package repository

import (
	"fmt"
	"strings"
	"time"

	"github.com/bcrowe306/linksy/internal/models"
)

// HourCount is the number of events of one type in one UTC hour.
type HourCount struct {
	Hour  time.Time
	Type  string
	Count int64
}

func (r *Repository) hourExpr() string {
	if r.db.DriverName == "postgres" {
		return `to_char(created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24')`
	}
	return `substr(created_at, 1, 13)`
}

// HourlyCounts returns event counts per UTC hour; callers roll them up into
// local days or hours so time zones (including DST) are handled exactly.
func (r *Repository) HourlyCounts(f AnalyticsFilter) ([]HourCount, error) {
	where, args := f.where()
	rows, err := r.query(`SELECT `+r.hourExpr()+` AS h, event_type, COUNT(*) FROM analytics_events WHERE `+where+` GROUP BY 1, 2 ORDER BY 1`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []HourCount{}
	for rows.Next() {
		var h, typ string
		var n int64
		if err := rows.Scan(&h, &typ, &n); err != nil {
			return nil, err
		}
		t, err := time.Parse("2006-01-02T15", strings.Replace(h, " ", "T", 1))
		if err != nil {
			continue
		}
		out = append(out, HourCount{Hour: t, Type: typ, Count: n})
	}
	return out, rows.Err()
}

// UniqueVisitors counts distinct hashed IPs.
func (r *Repository) UniqueVisitors(f AnalyticsFilter) (int64, error) {
	where, args := f.where()
	var n int64
	err := r.queryRow(`SELECT COUNT(DISTINCT ip_hash) FROM analytics_events WHERE `+where+` AND ip_hash <> ''`, args...).Scan(&n)
	return n, err
}

// Breakdown groups events by a column. Allowed columns are fixed, never user input.
func (r *Repository) Breakdown(f AnalyticsFilter, column, fallback string, limit int) ([]models.NameCount, error) {
	switch column {
	case "country", "region", "city", "device", "browser", "os", "referrer", "channel":
	default:
		return nil, fmt.Errorf("invalid breakdown %q", column)
	}
	where, args := f.where()
	return r.nameCounts(column, fallback, where, args, limit)
}

// CityCount is a city with its country for the locations table.
type CityCount struct {
	City    string `json:"city"`
	Region  string `json:"region"`
	Country string `json:"country"`
	Count   int64  `json:"count"`
}

func (r *Repository) TopCities(f AnalyticsFilter, limit int) ([]CityCount, error) {
	where, args := f.where()
	rows, err := r.query(fmt.Sprintf(`SELECT city, region, country, COUNT(*) AS cnt FROM analytics_events
		WHERE %s AND city <> '' GROUP BY city, region, country ORDER BY cnt DESC LIMIT %d`, where, limit), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []CityCount{}
	for rows.Next() {
		var c CityCount
		if err := rows.Scan(&c.City, &c.Region, &c.Country, &c.Count); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// AssetCount is a link or QR code with its event count in a range.
type AssetCount struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Label string `json:"label"` // slug for links, type for QR codes
	Count int64  `json:"count"`
}

// TopAssets ranks links by clicks or QR codes by scans.
func (r *Repository) TopAssets(f AnalyticsFilter, kind string, limit int) ([]AssetCount, error) {
	where, args := f.where()
	var q string
	if kind == "qr" {
		q = fmt.Sprintf(`SELECT q.id, q.title, q.type, COUNT(*) AS cnt FROM analytics_events e JOIN qr_codes q ON q.id = e.qr_code_id
			WHERE %s AND e.event_type = 'scan' GROUP BY q.id, q.title, q.type ORDER BY cnt DESC LIMIT %d`, prefixed(where, "e."), limit)
	} else {
		q = fmt.Sprintf(`SELECT l.id, l.title, l.slug, COUNT(*) AS cnt FROM analytics_events e JOIN links l ON l.id = e.link_id
			WHERE %s AND e.event_type = 'click' GROUP BY l.id, l.title, l.slug ORDER BY cnt DESC LIMIT %d`, prefixed(where, "e."), limit)
	}
	rows, err := r.query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []AssetCount{}
	for rows.Next() {
		var a AssetCount
		if err := rows.Scan(&a.ID, &a.Title, &a.Label, &a.Count); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// prefixed qualifies the filter's bare column names with a table alias.
func prefixed(where, alias string) string {
	for _, col := range []string{"team_id", "link_id", "qr_code_id", "created_at"} {
		where = strings.ReplaceAll(where, col+" ", alias+col+" ")
	}
	return where
}

// ExportRow is one raw event for CSV export.
type ExportRow struct {
	At      time.Time
	Type    string
	Asset   string
	Short   string
	Country string
	Region  string
	City    string
	Device  string
	Browser string
	OS      string
	Channel string
	Referer string
}

// ExportEvents returns raw events (newest first) with asset names.
func (r *Repository) ExportEvents(f AnalyticsFilter, limit int) ([]ExportRow, error) {
	where, args := f.where()
	rows, err := r.query(fmt.Sprintf(`SELECT e.created_at, e.event_type,
		COALESCE(q.title, l.title, ''), COALESCE(l.slug, q.code, ''),
		e.country, e.region, e.city, e.device, e.browser, e.os, e.channel, e.referrer
		FROM analytics_events e
		LEFT JOIN links l ON l.id = e.link_id
		LEFT JOIN qr_codes q ON q.id = e.qr_code_id
		WHERE %s ORDER BY e.created_at DESC LIMIT %d`, prefixed(where, "e."), limit), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ExportRow{}
	for rows.Next() {
		var x ExportRow
		if err := rows.Scan(&x.At, &x.Type, &x.Asset, &x.Short, &x.Country, &x.Region, &x.City, &x.Device, &x.Browser, &x.OS, &x.Channel, &x.Referer); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}
