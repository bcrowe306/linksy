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
	QRTypePhone    QRType = "phone"
)

// QRTypes lists every supported QR content type.
var QRTypes = []QRType{QRTypeWebsite, QRTypeDocument, QRTypeVCard, QRTypeWiFi, QRTypeApp, QRTypeText, QRTypeEmail, QRTypeSMS, QRTypePhone}

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
	OwnerID   string    `json:"owner_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Membership status values
const (
	MemberActive    = "active"
	MemberSuspended = "suspended"
)

type TeamMember struct {
	ID        string    `json:"id"`
	TeamID    string    `json:"team_id"`
	UserID    string    `json:"user_id"`
	Role      UserRole  `json:"role"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// TeamMembership is a team as seen by one of its members (used for team switching).
type TeamMembership struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	OwnerID     string    `json:"owner_id"`
	Role        UserRole  `json:"role"`
	Status      string    `json:"status"`
	MemberCount int       `json:"member_count"`
	CreatedAt   time.Time `json:"created_at"`
}

type Session struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	ActiveTeamID string    `json:"active_team_id"`
	IP           string    `json:"ip"`
	UserAgent    string    `json:"user_agent"`
	CreatedAt    time.Time `json:"created_at"`
	LastSeenAt   time.Time `json:"last_seen_at"`
	ExpiresAt    time.Time `json:"expires_at"`
	Current      bool      `json:"current,omitempty"`
}

type AuditEvent struct {
	ID         string    `json:"id"`
	TeamID     string    `json:"team_id"`
	ActorID    string    `json:"actor_id"`
	ActorName  string    `json:"actor_name,omitempty"`
	ActorEmail string    `json:"actor_email,omitempty"`
	Action     string    `json:"action"`
	TargetType string    `json:"target_type"`
	TargetID   string    `json:"target_id"`
	Metadata   string    `json:"metadata"`
	IP         string    `json:"ip"`
	CreatedAt  time.Time `json:"created_at"`
}

type Link struct {
	ID           string     `json:"id"`
	TeamID       string     `json:"team_id"`
	UserID       string     `json:"user_id"`
	Slug         string     `json:"slug"`
	TargetURL    string     `json:"target_url"`
	Title        string     `json:"title"`
	Description  string     `json:"description"`
	IsActive     bool       `json:"is_active"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
	PasswordHash string     `json:"-"`
	HasPassword  bool       `json:"has_password"`
	UTMSource    string     `json:"utm_source"`
	UTMMedium    string     `json:"utm_medium"`
	UTMCampaign  string     `json:"utm_campaign"`
	UTMTerm      string     `json:"utm_term"`
	UTMContent   string     `json:"utm_content"`
	ClickCount   int64      `json:"click_count"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
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
	EyeShape      string `json:"eye_shape"`       // outer finder ring: square, rounded, extra-rounded, circle, leaf
	EyeInnerShape string `json:"eye_inner_shape"` // finder center: square, rounded, circle, leaf
	EyeColor      string `json:"eye_color"`       // Custom outer finder eye color
	EyeInnerColor string `json:"eye_inner_color"` // Custom inner eyeball color

	// QR Spec Precision
	ErrorCorrection string `json:"error_correction"` // "L", "M", "Q", "H" (Default "H")
	Margin          int    `json:"margin"`           // Quiet zone in modules (1-10, default 2)

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
	LabelSize    int    `json:"label_size"`     // Label scale in percent (50–160, default 100)
}

type QRCode struct {
	ID         string    `json:"id"`
	TeamID     string    `json:"team_id"`
	UserID     string    `json:"user_id"`
	LinkID     *string   `json:"link_id,omitempty"`     // Linked short link for dynamic redirection
	DocumentID *string   `json:"document_id,omitempty"` // Hosted document served by this QR
	Code       string    `json:"code,omitempty"`        // Short scan code for dynamic codes (/q/{code})
	IsDynamic  bool      `json:"is_dynamic"`            // Encodes a Lynqr redirect, so content can change after printing
	TargetURL  string    `json:"target_url,omitempty"`  // Current destination (dynamic website codes); computed
	ScanURL    string    `json:"scan_url,omitempty"`    // Public URL encoded in dynamic codes; computed
	Type       QRType    `json:"type"`
	Title      string    `json:"title"`
	Payload    string    `json:"payload"`  // Encoded destination URL or raw payload
	RawData    string    `json:"raw_data"` // JSON formatted raw type-specific data
	Style      QRStyle   `json:"style"`
	ScanCount  int64     `json:"scan_count"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type AnalyticsEvent struct {
	ID        string    `json:"id"`
	TeamID    string    `json:"team_id"`
	LinkID    *string   `json:"link_id,omitempty"`
	QRCodeID  *string   `json:"qr_code_id,omitempty"`
	EventType string    `json:"event_type"` // "click" or "scan"
	IP        string    `json:"-"`          // Raw IP is never stored; only a salted hash
	IPHash    string    `json:"-"`
	UserAgent string    `json:"user_agent"`
	Browser   string    `json:"browser"`
	OS        string    `json:"os"`
	Device    string    `json:"device"`
	Referrer  string    `json:"referrer"`
	Channel   string    `json:"channel"` // direct, social, email, search, web
	Country   string    `json:"country"`
	Region    string    `json:"region"`
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

type APIKey struct {
	ID         string     `json:"id"`
	UserID     string     `json:"user_id"`
	TeamID     string     `json:"team_id,omitempty"`
	Name       string     `json:"name"`
	KeyHash    string     `json:"-"`
	KeyPrefix  string     `json:"key_prefix"`
	Key        string     `json:"key,omitempty"` // Only populated on creation
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

type UpdateProfileRequest struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	AvatarURL string `json:"avatar_url,omitempty"`
}

type PasswordResetRequest struct {
	Email string `json:"email,omitempty"`
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password,omitempty"`
	NewPassword     string `json:"new_password"`
}

type CreateAPIKeyRequest struct {
	Name          string `json:"name"`
	ExpiresInDays int    `json:"expires_in_days,omitempty"`
}

type Invitation struct {
	ID        string    `json:"id"`
	TeamID    string    `json:"team_id"`
	TeamName  string    `json:"team_name,omitempty"`
	Email     string    `json:"email"`
	Role      UserRole  `json:"role"`
	InvitedBy string    `json:"invited_by"`
	State     string    `json:"state"` // pending, accepted, revoked
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
	AcceptURL string    `json:"accept_url,omitempty"` // Only populated on creation
}

type TeamMemberDetail struct {
	ID        string    `json:"id"`
	TeamID    string    `json:"team_id"`
	UserID    string    `json:"user_id"`
	Role      UserRole  `json:"role"`
	Status    string    `json:"status"`
	Email     string    `json:"email"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Name      string    `json:"name"`
	AvatarURL string    `json:"avatar_url"`
	CreatedAt time.Time `json:"created_at"`
}

type BulkDeleteDocumentsRequest struct {
	DocumentIDs []string `json:"document_ids"`
	DeleteQR    bool     `json:"delete_qr"`
}

type InviteMemberRequest struct {
	Email string   `json:"email"`
	Role  UserRole `json:"role"`
}

type UpdateRoleRequest struct {
	Role UserRole `json:"role"`
}
