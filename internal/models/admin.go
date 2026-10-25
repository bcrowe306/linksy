package models

import "time"

// PlatformAuditEvent records an action taken by a Lynqr platform admin.
type PlatformAuditEvent struct {
	ID          string    `json:"id"`
	ActorID     string    `json:"actor_id"`
	ActorEmail  string    `json:"actor_email"`
	Action      string    `json:"action"`
	TargetType  string    `json:"target_type"`
	TargetID    string    `json:"target_id"`
	TargetLabel string    `json:"target_label"`
	Metadata    string    `json:"metadata"`
	IP          string    `json:"ip"`
	CreatedAt   time.Time `json:"created_at"`
}

type BlockedDomain struct {
	Domain    string    `json:"domain"`
	Reason    string    `json:"reason"`
	CreatedBy string    `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
}

type AbuseReport struct {
	ID            string     `json:"id"`
	TargetType    string     `json:"target_type"` // link | qr
	TargetID      string     `json:"target_id"`
	TargetURL     string     `json:"target_url"`
	Reason        string     `json:"reason"`
	Details       string     `json:"details"`
	ReporterEmail string     `json:"reporter_email"`
	ReporterIP    string     `json:"-"`
	Status        string     `json:"status"` // open | actioned | dismissed
	ResolvedBy    string     `json:"resolved_by,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	ResolvedAt    *time.Time `json:"resolved_at,omitempty"`
}

// TrialCode lengthens a workspace's free trial when redeemed.
type TrialCode struct {
	Code           string     `json:"code"`
	Days           int        `json:"days"`
	MaxRedemptions int        `json:"max_redemptions"`
	Redemptions    int        `json:"redemptions"`
	ExpiresAt      *time.Time `json:"expires_at,omitempty"`
	Active         bool       `json:"active"`
	Note           string     `json:"note"`
	CreatedBy      string     `json:"created_by"`
	CreatedAt      time.Time  `json:"created_at"`
}

// Usable reports whether the code can still be redeemed.
func (c *TrialCode) Usable(now time.Time) bool {
	return c.Active && (c.ExpiresAt == nil || now.Before(*c.ExpiresAt)) && (c.MaxRedemptions == 0 || c.Redemptions < c.MaxRedemptions)
}

// AdminTeam is a workspace row in the admin console.
type AdminTeam struct {
	ID              string     `json:"id"`
	Name            string     `json:"name"`
	OwnerID         string     `json:"owner_id"`
	OwnerEmail      string     `json:"owner_email"`
	CreatedAt       time.Time  `json:"created_at"`
	State           string     `json:"state"`
	Interval        string     `json:"interval,omitempty"`
	TrialEndsAt     *time.Time `json:"trial_ends_at,omitempty"`
	PeriodEnd       *time.Time `json:"current_period_end,omitempty"`
	Comped          bool       `json:"comped"`
	DiscountLabel   string     `json:"discount_label,omitempty"`
	CustomerID      string     `json:"stripe_customer_id,omitempty"`
	SuspendedReason string     `json:"suspended_reason,omitempty"`
	Members         int        `json:"members"`
	Links           int        `json:"links"`
	QRCodes         int        `json:"qr_codes"`
}

// AdminUser is a user row in the admin console.
type AdminUser struct {
	ID              string     `json:"id"`
	Email           string     `json:"email"`
	Name            string     `json:"name"`
	CreatedAt       time.Time  `json:"created_at"`
	LastSeenAt      *time.Time `json:"last_seen_at,omitempty"`
	IsPlatformAdmin bool       `json:"is_platform_admin"`
	SuspendedReason string     `json:"suspended_reason,omitempty"`
	Workspaces      int        `json:"workspaces"`
}

// AdminAsset is a link or QR code found by a moderation search.
type AdminAsset struct {
	Type             string    `json:"type"` // link | qr
	ID               string    `json:"id"`
	TeamID           string    `json:"team_id"`
	TeamName         string    `json:"team_name"`
	Label            string    `json:"label"` // slug or title
	PublicURL        string    `json:"public_url"`
	Destination      string    `json:"destination"`
	ModerationReason string    `json:"moderation_reason,omitempty"`
	Count            int64     `json:"count"` // clicks or scans
	CreatedAt        time.Time `json:"created_at"`
}

// DayCount is one bucket of a daily series.
type DayCount struct {
	Day   string `json:"day"` // YYYY-MM-DD (UTC)
	Count int64  `json:"count"`
}
