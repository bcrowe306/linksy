package service

import (
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/bcrowe306/linksy/internal/auth"
	"github.com/bcrowe306/linksy/internal/models"
)

// botPattern matches crawlers, link-preview fetchers and uptime monitors so
// they don't inflate click and scan counts. Redirects still work for them.
var botPattern = regexp.MustCompile(`(?i)bot\b|bot/|crawler|spider|slurp|preview|facebookexternalhit|embedly|whatsapp|telegrambot|discordbot|slackbot|skypeuripreview|bitlybot|curl/|wget/|python-requests|go-http-client|httpclient|headlesschrome|lighthouse|pingdom|uptimerobot|statuscake|monitor`)

// IsBot reports whether a user agent belongs to automated traffic.
func IsBot(ua string) bool {
	return strings.TrimSpace(ua) == "" || botPattern.MatchString(ua)
}

var channelDomains = map[string][]string{
	"social": {"facebook.", "fb.com", "instagram.", "t.co", "twitter.", "x.com", "linkedin.", "lnkd.in", "tiktok.", "reddit.", "youtube.", "youtu.be", "pinterest.", "threads.net", "snapchat.", "bsky.app", "mastodon"},
	"email":  {"mail.google.", "outlook.", "mail.yahoo.", "mail.", "proton.me", "fastmail.", "zoho.com/mail"},
	"search": {"google.", "bing.", "duckduckgo.", "yahoo.", "baidu.", "yandex.", "ecosia.", "search.brave.", "startpage."},
}

// referrerDomain reduces a Referer header to its host (no path or query).
func referrerDomain(ref string) string {
	u, err := url.Parse(strings.TrimSpace(ref))
	if err != nil || u.Host == "" {
		return ""
	}
	return strings.TrimPrefix(strings.ToLower(u.Hostname()), "www.")
}

// classifyChannel buckets a visit into direct / social / email / search / web / qr.
func classifyChannel(eventType, domain string) string {
	if eventType == "scan" {
		return "qr"
	}
	if domain == "" {
		return "direct"
	}
	for _, ch := range []string{"email", "social", "search"} { // email before search: mail.google.*
		for _, d := range channelDomains[ch] {
			if strings.HasPrefix(domain, d) || strings.Contains(domain, "."+d) || domain == strings.TrimSuffix(d, ".") {
				return ch
			}
		}
	}
	return "web"
}

func parseUA(ua string) (device, browser, os string) {
	l := strings.ToLower(ua)
	switch {
	case strings.Contains(l, "ipad") || strings.Contains(l, "tablet") || (strings.Contains(l, "android") && !strings.Contains(l, "mobile")):
		device = "Tablet"
	case strings.Contains(l, "mobile") || strings.Contains(l, "iphone") || strings.Contains(l, "android"):
		device = "Mobile"
	default:
		device = "Desktop"
	}
	switch {
	case strings.Contains(l, "edg"):
		browser = "Edge"
	case strings.Contains(l, "opr/") || strings.Contains(l, "opera"):
		browser = "Opera"
	case strings.Contains(l, "samsungbrowser"):
		browser = "Samsung Internet"
	case strings.Contains(l, "firefox") || strings.Contains(l, "fxios"):
		browser = "Firefox"
	case strings.Contains(l, "chrome") || strings.Contains(l, "crios"):
		browser = "Chrome"
	case strings.Contains(l, "safari"):
		browser = "Safari"
	default:
		browser = "Other"
	}
	switch {
	case strings.Contains(l, "windows"):
		os = "Windows"
	case strings.Contains(l, "iphone") || strings.Contains(l, "ipad") || strings.Contains(l, "ipod"):
		os = "iOS"
	case strings.Contains(l, "mac os") || strings.Contains(l, "macintosh"):
		os = "macOS"
	case strings.Contains(l, "android"):
		os = "Android"
	case strings.Contains(l, "cros"):
		os = "ChromeOS"
	case strings.Contains(l, "linux"):
		os = "Linux"
	default:
		os = "Other"
	}
	return
}

// TrackEvent enriches and stores a click or scan, then notifies live viewers.
// Bot traffic is dropped.
func (s *Service) TrackEvent(e *models.AnalyticsEvent) {
	if IsBot(e.UserAgent) || e.TeamID == "" {
		return
	}
	e.Device, e.Browser, e.OS = parseUA(e.UserAgent)
	e.Referrer = referrerDomain(e.Referrer)
	if e.Referrer == strings.TrimPrefix(hostOf(s.appURL), "www.") {
		e.Referrer = "" // our own pages (e.g. the unlock form) aren't a traffic source
	}
	e.Channel = classifyChannel(e.EventType, e.Referrer)
	if c := strings.ToUpper(strings.TrimSpace(e.Country)); len(c) == 2 && c != "XX" && c != "T1" {
		e.Country = c
	} else {
		e.Country = ""
	}
	e.UserAgent = truncate(e.UserAgent, 512)
	e.Region, e.City = truncate(e.Region, 100), truncate(e.City, 100)
	e.IPHash = s.hashIP(e.IP)
	if e.CreatedAt.IsZero() {
		e.CreatedAt = time.Now().UTC()
	}
	if err := s.repo.RecordEvent(e); err != nil {
		return
	}
	s.live.publish(e.TeamID, s.liveEventFor(e))
}

func hostOf(u string) string {
	p, err := url.Parse(u)
	if err != nil {
		return ""
	}
	return strings.ToLower(p.Hostname())
}

// ----------------- Live activity hub -----------------

// LiveEvent is the payload pushed to dashboards over the websocket.
type LiveEvent struct {
	Type    string    `json:"type"` // click | scan
	Asset   string    `json:"asset"`
	AssetID string    `json:"asset_id"`
	Kind    string    `json:"kind"` // link | qr
	Country string    `json:"country"`
	City    string    `json:"city"`
	Device  string    `json:"device"`
	Browser string    `json:"browser"`
	Channel string    `json:"channel"`
	At      time.Time `json:"at"`
}

func (s *Service) liveEventFor(e *models.AnalyticsEvent) LiveEvent {
	ev := LiveEvent{Type: e.EventType, Country: e.Country, City: e.City, Device: e.Device, Browser: e.Browser, Channel: e.Channel, At: e.CreatedAt}
	if e.QRCodeID != nil {
		ev.Kind, ev.AssetID = "qr", *e.QRCodeID
		if q, err := s.repo.GetQRCodeByID(*e.QRCodeID); err == nil {
			ev.Asset = q.Title
		}
	} else if e.LinkID != nil {
		ev.Kind, ev.AssetID = "link", *e.LinkID
		if l, err := s.repo.GetLinkByID(*e.LinkID); err == nil {
			ev.Asset = l.Title
		}
	}
	return ev
}

type liveHub struct {
	mu   sync.RWMutex
	subs map[string]map[chan LiveEvent]struct{} // team → subscribers
}

func newLiveHub() *liveHub { return &liveHub{subs: map[string]map[chan LiveEvent]struct{}{}} }

func (h *liveHub) publish(team string, ev LiveEvent) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for ch := range h.subs[team] {
		select {
		case ch <- ev:
		default: // slow consumer: drop rather than block ingestion
		}
	}
}

// Subscribe registers for a team's live events; call the returned func to stop.
func (s *Service) Subscribe(a Actor) (<-chan LiveEvent, func(), error) {
	if err := a.require(auth.PermViewAnalytics); err != nil {
		return nil, nil, err
	}
	ch := make(chan LiveEvent, 32)
	h := s.live
	h.mu.Lock()
	if h.subs[a.TeamID] == nil {
		h.subs[a.TeamID] = map[chan LiveEvent]struct{}{}
	}
	h.subs[a.TeamID][ch] = struct{}{}
	h.mu.Unlock()
	return ch, func() {
		h.mu.Lock()
		delete(h.subs[a.TeamID], ch)
		if len(h.subs[a.TeamID]) == 0 {
			delete(h.subs, a.TeamID)
		}
		h.mu.Unlock()
	}, nil
}
