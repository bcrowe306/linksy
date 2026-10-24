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
	// Colors & Gradients
	FgColor       string `json:"fg_color"`       // Hex, e.g. #000000
	BgColor       string `json:"bg_color"`       // Hex, e.g. #FFFFFF
	IsTransparent bool   `json:"is_transparent"` // Transparent background toggle
	GradientType  string `json:"gradient_type"`  // "none", "linear"
	GradientStart string `json:"gradient_start"` // Hex start color
	GradientEnd   string `json:"gradient_end"`   // Hex end color
	GradientAngle int    `json:"gradient_angle"` // Angle in degrees (0, 45, 90, 135, etc.)

	// Module Shapes & Finder Eyes
	DotStyle      string `json:"dot_style"`       // "square", "dots", "rounded", "classy"
	EyeShape      string `json:"eye_shape"`       // "square", "rounded", "circle"
	EyeColor      string `json:"eye_color"`       // Custom outer finder eye color
	EyeInnerColor string `json:"eye_inner_color"` // Custom inner eyeball color

	// QR Spec Precision
	ErrorCorrection string `json:"error_correction"` // "L", "M", "Q", "H" (Default "H")
	Margin          int    `json:"margin"`           // Quiet zone margin in modules (0-10)

	// Center Logo Customization
	LogoURL     string  `json:"logo_url"`      // URL or storage path
	LogoBase64  string  `json:"logo_base64"`   // Base64 data URI
	LogoSize    float64 `json:"logo_size"`     // 0.15 to 0.28 (fraction of QR dimension)
	LogoBgShape string  `json:"logo_bg_shape"` // "circle", "rounded", "square", "none"
	LogoPadding int     `json:"logo_padding"`  // Padding in px around center logo badge

	// Frame & Border Customization
	FrameStyle   string `json:"frame_style"`    // "none", "banner-bottom", "banner-top", "card", "box"
	BorderWidth  int    `json:"border_width"`   // Border width in px
	BorderColor  string `json:"border_color"`   // Border hex color
	BorderRadius int    `json:"border_radius"`  // Corner rounding in px
	FrameBgColor string `json:"frame_bg_color"` // Frame outer background hex
	LabelText    string `json:"label_text"`     // e.g. "SCAN ME"
	LabelColor   string `json:"label_color"`    // Label font hex color
	LabelSize    int    `json:"label_size"`     // Font size in pt/px
}

type QRCode struct {
	ID        string    `json:"id"`
	TeamID    string    `json:"team_id"`
	UserID    string    `json:"user_id"`
	LinkID    *string   `json:"link_id,omitempty"` // Linked short link for dynamic redirection
	Type      QRType    `json:"type"`
	Title     string    `json:"title"`
	Payload   string    `json:"payload"`  // Encoded destination URL or raw payload
	RawData   string    `json:"raw_data"` // JSON formatted raw type-specific data
	Style     QRStyle   `json:"style"`
	ScanCount int64     `json:"scan_count"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
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
	ID         string    `json:"id"`
	TeamID     string    `json:"team_id"`
	UserID     string    `json:"user_id"`
	Filename   string    `json:"filename"`
	StoredPath string    `json:"stored_path"`
	MimeType   string    `json:"mime_type"`
	FileSize   int64     `json:"file_size"`
	Version    int       `json:"version"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type AnalyticsSummary struct {
	TotalClicks int64        `json:"total_clicks"`
	TotalScans  int64        `json:"total_scans"`
	ByDate      []DateMetric `json:"by_date"`
	ByCountry   []NameCount  `json:"by_country"`
	ByDevice    []NameCount  `json:"by_device"`
	ByBrowser   []NameCount  `json:"by_browser"`
	ByReferrer  []NameCount  `json:"by_referrer"`
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
