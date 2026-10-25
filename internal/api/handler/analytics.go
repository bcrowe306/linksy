package handler

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/gofiber/contrib/v3/websocket"
	"github.com/gofiber/fiber/v3"

	"github.com/bcrowe306/linksy/internal/api/middleware"
	"github.com/bcrowe306/linksy/internal/reportpdf"
	"github.com/bcrowe306/linksy/internal/service"
)

const maxExportRows = 100_000

// parseReportQuery reads from/to (RFC3339 or YYYY-MM-DD in tz), tz (IANA) and
// asset filters. Defaults to the last 7 days.
func parseReportQuery(c fiber.Ctx) (service.ReportQuery, error) {
	q := service.ReportQuery{TZ: time.UTC, LinkID: c.Query("link_id"), QRID: c.Query("qr_id")}
	if tz := c.Query("tz"); tz != "" {
		loc, err := time.LoadLocation(tz)
		if err != nil {
			return q, fmt.Errorf("unknown time zone %q", tz)
		}
		q.TZ = loc
	}
	parse := func(v string, endOfDay bool) (time.Time, error) {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			return t, nil
		}
		d, err := time.ParseInLocation("2006-01-02", v, q.TZ)
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid date %q: use RFC3339 or YYYY-MM-DD", v)
		}
		if endOfDay {
			d = d.AddDate(0, 0, 1) // inclusive end date
		}
		return d, nil
	}
	now := time.Now().In(q.TZ)
	q.To = now
	q.From = now.AddDate(0, 0, -7)
	var err error
	if v := c.Query("to"); v != "" {
		if q.To, err = parse(v, true); err != nil {
			return q, err
		}
	}
	if v := c.Query("from"); v != "" {
		if q.From, err = parse(v, false); err != nil {
			return q, err
		}
	}
	return q, nil
}

func (h *Handler) GetAnalyticsReport(c fiber.Ctx) error {
	q, err := parseReportQuery(c)
	if err != nil {
		return fail(c, err)
	}
	r, err := h.svc.Report(middleware.Actor(c), q)
	if err != nil {
		return fail(c, err)
	}
	return c.JSON(r)
}

// ExportAnalytics downloads raw events (csv), the report (json) or a printable report (pdf).
func (h *Handler) ExportAnalytics(c fiber.Ctx) error {
	a := middleware.Actor(c)
	q, err := parseReportQuery(c)
	if err != nil {
		return fail(c, err)
	}
	stamp := q.From.Format("2006-01-02") + "_" + q.To.Format("2006-01-02")
	format := strings.ToLower(c.Query("format", "csv"))

	switch format {
	case "csv":
		rows, err := h.svc.ExportEvents(a, q, maxExportRows)
		if err != nil {
			return fail(c, err)
		}
		c.Set("Content-Type", "text/csv; charset=utf-8")
		c.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="lynqr-events-%s.csv"`, stamp))
		w := csv.NewWriter(c.Response().BodyWriter())
		_ = w.Write([]string{"timestamp_utc", "timestamp_local", "type", "asset", "short_code", "country", "region", "city", "device", "browser", "os", "channel", "referrer"})
		for _, r := range rows {
			_ = w.Write([]string{r.At.UTC().Format(time.RFC3339), r.At.In(q.TZ).Format("2006-01-02 15:04:05"), r.Type, csvSafe(r.Asset), r.Short,
				r.Country, csvSafe(r.Region), csvSafe(r.City), r.Device, r.Browser, r.OS, r.Channel, csvSafe(r.Referer)})
		}
		w.Flush()
		return w.Error()

	case "json", "pdf":
		r, err := h.svc.Report(a, q)
		if err != nil {
			return fail(c, err)
		}
		if format == "json" {
			c.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="lynqr-report-%s.json"`, stamp))
			b, _ := json.MarshalIndent(r, "", "  ")
			c.Set("Content-Type", "application/json")
			return c.Send(b)
		}
		team, _ := h.svc.GetTeam(a)
		workspace := ""
		if team != nil {
			workspace = team.Name
		}
		pdf, err := reportpdf.Render(r, workspace, h.reportSubject(a, q))
		if err != nil {
			return fail(c, err)
		}
		c.Set("Content-Type", "application/pdf")
		c.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="lynqr-report-%s.pdf"`, stamp))
		return c.Send(pdf)
	}
	return fail(c, errors.New("format must be csv, json or pdf"))
}

func (h *Handler) reportSubject(a service.Actor, q service.ReportQuery) string {
	if q.LinkID != "" {
		if l, err := h.svc.GetLink(a, q.LinkID); err == nil {
			return "Link: " + l.Title
		}
	}
	if q.QRID != "" {
		if r, err := h.svc.GetQRCode(a, q.QRID); err == nil {
			return "QR code: " + r.Title
		}
	}
	return "All links and QR codes"
}

// csvSafe neutralizes spreadsheet formula injection in user-controlled cells.
func csvSafe(s string) string {
	if s != "" && strings.ContainsRune("=+-@\t\r", rune(s[0])) {
		return "'" + s
	}
	return s
}

// LiveUpgrade gates the analytics websocket: it must be an upgrade request
// from our own origin (browsers send cookies on cross-site websocket requests).
func (h *Handler) LiveUpgrade(c fiber.Ctx) error {
	if !websocket.IsWebSocketUpgrade(c) {
		return fiber.ErrUpgradeRequired
	}
	origin := c.Get("Origin")
	if u, err := url.Parse(origin); origin == "" || err != nil || !strings.EqualFold(u.Host, c.Host()) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Cross-origin websocket rejected"})
	}
	c.Locals("actor", middleware.Actor(c))
	return c.Next()
}

// LiveFeed streams the team's clicks and scans as JSON messages.
func (h *Handler) LiveFeed() fiber.Handler {
	return websocket.New(func(conn *websocket.Conn) {
		actor, _ := conn.Locals("actor").(service.Actor)
		events, stop, err := h.svc.Subscribe(actor)
		if err != nil {
			_ = conn.WriteJSON(fiber.Map{"error": err.Error()})
			return
		}
		defer stop()

		done := make(chan struct{})
		go func() { // reader: detects client disconnects
			defer close(done)
			for {
				if _, _, err := conn.ReadMessage(); err != nil {
					return
				}
			}
		}()
		ping := time.NewTicker(25 * time.Second)
		defer ping.Stop()
		_ = conn.WriteJSON(fiber.Map{"type": "ready"})
		for {
			select {
			case <-done:
				return
			case ev := <-events:
				if conn.WriteJSON(ev) != nil {
					return
				}
			case <-ping.C:
				if conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(5*time.Second)) != nil {
					return
				}
			}
		}
	})
}
