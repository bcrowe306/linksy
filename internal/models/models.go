package models

import (
	"time"
)

type UserRole string

const (
	RoleOwner  UserRole = "owner"
	RoleAdmin  UserRole = "admin"
	RoleMember UserRole = "member"
	RoleViewer UserRole = "viewer"
)

type QRType string

const (
	QRTypeWebsite  QRType = "website"
	QRTypeDocument QRType = "document"
	QRTypeVCard    QRType = "vcard"
	QRTypeWiFi     QRType = "wifi"
	QRTypeApp      QRType = "app"
	QRTypeText     QRType = "text"
	QRTypeEmail    QRType = "email"
	QRTypeSMS      QRType = "sms"
)

type User struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	AvatarURL string    `json:"avatar_url"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Team struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type TeamMember struct {
	ID        string    `json:"id"`
	TeamID    string    `json:"team_id"`
	UserID    string    `json:"user_id"`
	Role      UserRole  `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

type Link struct {
	ID          string     `json:"id"`
	TeamID      string     `json:"team_id"`
	UserID      string     `json:"user_id"`
	Slug        string     `json:"slug"`
	TargetURL   string     `json:"target_url"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	IsActive    bool       `json:"is_active"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	Password    string     `json:"-"` // Hashed or empty
	ClickCount  int64      `json:"click_count"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type QRStyle struct {
	FgColor     string `json:"fg_color"`     // Hex, e.g. #000000
	BgColor     string `json:"bg_color"`     // Hex, e.g. #FFFFFF
	Margin      int    `json:"margin"`       // Quiet zone in pixels/blocks
	LogoURL     string `json:"logo_url"`     // URL or base64 or stored logo path
	LogoBase64  string `json:"logo_base64"`  // Optional inline base64 logo for preview/export
	BorderWidth int    `json:"border_width"` // Border frame width
	BorderColor string `json:"border_color"` // Frame border color
	LabelText   string `json:"label_text"`   // e.g. "SCAN ME"
	LabelColor  string `json:"label_color"`
}

type QRCode struct {
	ID          string    `json:"id"`
	TeamID      string    `json:"team_id"`
	UserID      string    `json:"user_id"`
	LinkID      *string   `json:"link_id,omitempty"` // Linked short link for dynamic redirection
	Type        QRType    `json:"type"`
	Title       string    `json:"title"`
	Payload     string    `json:"payload"`      // Encoded destination URL or raw payload
	RawData     string    `json:"raw_data"`     // JSON formatted raw type-specific data
	Style       QRStyle   `json:"style"`
	ScanCount   int64     `json:"scan_count"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type AnalyticsEvent struct {
	ID        string    `json:"id"`
	LinkID    *string   `json:"link_id,omitempty"`
	QRCodeID  *string   `json:"qr_code_id,omitempty"`
	EventType string    `json:"event_type"` // "click" or "scan"
	IP        string    `json:"ip"`
	UserAgent string    `json:"user_agent"`
	Browser   string    `json:"browser"`
	OS        string    `json:"os"`
	Device    string    `json:"device"`
	Referrer  string    `json:"referrer"`
	Country   string    `json:"country"`
	City      string    `json:"city"`
	CreatedAt time.Time `json:"created_at"`
}

type Document struct {
	ID          string    `json:"id"`
	TeamID      string    `json:"team_id"`
	UserID      string    `json:"user_id"`
	Filename    string    `json:"filename"`
	StoredPath  string    `json:"stored_path"`
	MimeType    string    `json:"mime_type"`
	FileSize    int64     `json:"file_size"`
	Version     int       `json:"version"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type AnalyticsSummary struct {
	TotalClicks     int64            `json:"total_clicks"`
	TotalScans      int64            `json:"total_scans"`
	ByDate          []DateMetric     `json:"by_date"`
	ByCountry       []NameCount      `json:"by_country"`
	ByDevice        []NameCount      `json:"by_device"`
	ByBrowser       []NameCount      `json:"by_browser"`
	ByReferrer      []NameCount      `json:"by_referrer"`
}

type DateMetric struct {
	Date   string `json:"date"`
	Clicks int64  `json:"clicks"`
	Scans  int64  `json:"scans"`
}

type NameCount struct {
	Name  string `json:"name"`
	Count int64  `json:"count"`
}
