package service

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/big"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/bcrowe306/linksy/internal/auth"
	"github.com/bcrowe306/linksy/internal/billing"
	"github.com/bcrowe306/linksy/internal/email"
	"github.com/bcrowe306/linksy/internal/models"
	"github.com/bcrowe306/linksy/internal/repository"
)

const (
	// AdminSessionTTL is how long a re-authentication unlocks the admin console.
	AdminSessionTTL   = time.Hour
	adminCodeTTL      = 10 * time.Minute
	adminCodeAttempts = 5
	blocklistTTL      = time.Minute
)

var (
	ErrNotPlatformAdmin   = errors.New("this area is limited to Lynqr staff")
	ErrAdminReauth        = errors.New("confirm it's you to open the admin console")
	ErrAdminCodeInvalid   = errors.New("that code is incorrect or has expired")
	ErrEmailDisabled      = errors.New("email delivery is not configured")
	ErrDomainBlocked      = errors.New("links to this domain aren't allowed on Lynqr")
	ErrAccountSuspended   = errors.New("this account has been suspended. Contact support@lynqr.app if you think this is a mistake")
	ErrWorkspaceSuspended = errors.New("this workspace has been suspended by Lynqr. Contact support@lynqr.app")
	ErrSelfAdminAction    = errors.New("you can't do that to your own account")
)

type adminConfig struct {
	mailer email.Sender
	emails map[string]bool
}

type blocklistCache struct {
	mu      sync.RWMutex
	domains []string
	loaded  time.Time
}

// SetMailer configures outgoing email (admin codes, notifications).
func (s *Service) SetMailer(m email.Sender) { s.admin.mailer = m }

// SetAdminEmails bootstraps platform admins from configuration. Existing
// users are promoted now; others are promoted when they first sign in.
func (s *Service) SetAdminEmails(list []string) {
	s.admin.emails = map[string]bool{}
	var emails []string
	for _, e := range list {
		if e = strings.ToLower(strings.TrimSpace(e)); e != "" {
			s.admin.emails[e] = true
			emails = append(emails, e)
		}
	}
	if n, err := s.repo.GrantAdminsByEmail(emails); err != nil {
		log.Printf("[admin] could not apply ADMIN_EMAILS: %v", err)
	} else if n > 0 {
		log.Printf("[admin] granted platform admin to %d user(s) from ADMIN_EMAILS", n)
	}
}

// onSignIn enforces platform suspension and applies ADMIN_EMAILS.
func (s *Service) onSignIn(u *models.User) error {
	stored, err := s.repo.GetUserByID(u.ID)
	if err != nil {
		return err
	}
	if stored.SuspendedReason != "" {
		return ErrAccountSuspended
	}
	if s.admin.emails[stored.Email] && !stored.IsPlatformAdmin {
		if err := s.repo.SetPlatformAdmin(stored.ID, true); err == nil {
			s.adminAudit(AdminActor{UserID: "system", Email: "ADMIN_EMAILS"}, "admin.grant", "user", stored.ID, stored.Email, map[string]any{"source": "ADMIN_EMAILS"})
		}
	}
	return nil
}

// AdminActor is a platform admin performing an action.
type AdminActor struct {
	UserID string
	Email  string
	IP     string
}

func (s *Service) adminAudit(a AdminActor, action, targetType, targetID, label string, meta map[string]any) {
	metadata := "{}"
	if len(meta) > 0 {
		if b, err := json.Marshal(meta); err == nil {
			metadata = string(b)
		}
	}
	if err := s.repo.RecordPlatformAudit(&models.PlatformAuditEvent{
		ActorID: a.UserID, ActorEmail: a.Email, Action: action, TargetType: targetType, TargetID: targetID,
		TargetLabel: label, Metadata: metadata, IP: a.IP,
	}); err != nil {
		log.Printf("[admin] audit write failed: %v", err)
	}
}

// ----------------- Admin access & re-authentication -----------------

// AdminStatus describes the caller's access to the admin console.
type AdminStatus struct {
	IsAdmin       bool       `json:"is_admin"`
	Elevated      bool       `json:"elevated"`
	ElevatedUntil *time.Time `json:"elevated_until,omitempty"`
	EmailCode     bool       `json:"email_code"`
}

func (s *Service) AdminStatus(sessionID, userID string) (*AdminStatus, error) {
	u, err := s.repo.GetUserByID(userID)
	if err != nil {
		return nil, ErrUnauthorized
	}
	st := &AdminStatus{IsAdmin: u.IsPlatformAdmin, EmailCode: s.admin.mailer != nil}
	if !u.IsPlatformAdmin {
		return st, nil
	}
	until, _ := s.repo.GetSessionAdminUntil(sessionID)
	if until != nil && until.After(time.Now()) {
		st.Elevated, st.ElevatedUntil = true, until
	}
	return st, nil
}

// RequireAdmin returns the admin actor for an elevated session.
func (s *Service) RequireAdmin(sessionID, userID, ip string) (AdminActor, error) {
	u, err := s.repo.GetUserByID(userID)
	if err != nil || !u.IsPlatformAdmin || u.SuspendedReason != "" {
		return AdminActor{}, ErrNotPlatformAdmin
	}
	until, _ := s.repo.GetSessionAdminUntil(sessionID)
	if until == nil || !until.After(time.Now()) {
		return AdminActor{}, ErrAdminReauth
	}
	return AdminActor{UserID: u.ID, Email: u.Email, IP: ip}, nil
}

// ElevateAdmin unlocks the admin console for this session after the caller
// proved their identity (password checked by the identity provider, or an
// emailed code).
func (s *Service) ElevateAdmin(sessionID, userID, ip, method string) (*time.Time, error) {
	u, err := s.repo.GetUserByID(userID)
	if err != nil || !u.IsPlatformAdmin {
		return nil, ErrNotPlatformAdmin
	}
	until := time.Now().UTC().Add(AdminSessionTTL)
	if err := s.repo.SetSessionAdminUntil(sessionID, &until); err != nil {
		return nil, err
	}
	s.repo.DeleteAdminChallenge(sessionID)
	s.adminAudit(AdminActor{UserID: u.ID, Email: u.Email, IP: ip}, "admin.reauth", "session", sessionID, "", map[string]any{"method": method})
	return &until, nil
}

// LeaveAdmin ends the elevated admin session early.
func (s *Service) LeaveAdmin(sessionID string) error {
	return s.repo.SetSessionAdminUntil(sessionID, nil)
}

// SendAdminCode emails a one-time code that elevates this session.
func (s *Service) SendAdminCode(sessionID, userID string) error {
	u, err := s.repo.GetUserByID(userID)
	if err != nil || !u.IsPlatformAdmin {
		return ErrNotPlatformAdmin
	}
	if s.admin.mailer == nil {
		return ErrEmailDisabled
	}
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return err
	}
	code := fmt.Sprintf("%06d", n.Int64())
	if err := s.repo.PutAdminChallenge(sessionID, auth.HashToken(sessionID+":"+code), time.Now().UTC().Add(adminCodeTTL)); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return s.admin.mailer.Send(ctx, email.Message{
		To:      u.Email,
		Subject: "Your Lynqr admin code: " + code,
		Text: "Use this code to open the Lynqr admin console:\n\n    " + code + "\n\n" +
			"It expires in 10 minutes. If you didn't try to sign in to the admin console, someone may have access to your session: sign out of all devices from your profile.",
		HTML: `<div style="font-family:system-ui,sans-serif;max-width:480px;margin:auto;padding:24px;color:#111">` +
			`<h2 style="margin:0 0 12px">Lynqr admin console</h2><p>Use this code to confirm it's you:</p>` +
			`<p style="font-size:32px;font-weight:700;letter-spacing:6px;margin:16px 0">` + code + `</p>` +
			`<p style="color:#555;font-size:14px">It expires in 10 minutes. If you didn't request it, sign out of all devices from your Lynqr profile.</p></div>`,
		Tag: "admin_code",
	})
}

// VerifyAdminCode checks an emailed code and elevates the session.
func (s *Service) VerifyAdminCode(sessionID, userID, code, ip string) (*time.Time, error) {
	hash, attempts, expires, err := s.repo.TakeAdminChallengeAttempt(sessionID)
	if err != nil || attempts > adminCodeAttempts || time.Now().After(expires) ||
		hash != auth.HashToken(sessionID+":"+strings.TrimSpace(code)) {
		if err == nil && attempts >= adminCodeAttempts {
			s.repo.DeleteAdminChallenge(sessionID)
		}
		return nil, ErrAdminCodeInvalid
	}
	return s.ElevateAdmin(sessionID, userID, ip, "email_code")
}

// ----------------- Workspace access (suspension + billing) -----------------

// TeamAccess reports whether a workspace is read-only (lapsed plan or
// suspended) and whether Lynqr suspended it. Cached for 30s.
func (s *Service) TeamAccess(teamID string) (readOnly, suspended bool) {
	if teamID == "" {
		return false, false
	}
	now := time.Now()
	s.lapses.mu.Lock()
	if e, ok := s.lapses.m[teamID]; ok && now.Before(e.until) {
		s.lapses.mu.Unlock()
		return e.lapsed || e.suspended, e.suspended
	}
	s.lapses.mu.Unlock()

	var e lapseEntry
	if t, err := s.repo.GetTeam(teamID); err == nil {
		e.lapsed = s.billing != nil && t.Billing.Lapsed(now)
		e.suspended = t.SuspendedReason != ""
	}
	e.until = now.Add(30 * time.Second)
	s.lapses.mu.Lock()
	if s.lapses.m == nil {
		s.lapses.m = map[string]lapseEntry{}
	}
	s.lapses.m[teamID] = e
	s.lapses.mu.Unlock()
	return e.lapsed || e.suspended, e.suspended
}

// ----------------- Platform metrics -----------------

type PlatformMetrics struct {
	Days       int   `json:"days"`
	Users      int64 `json:"users"`
	Workspaces int64 `json:"workspaces"`
	Links      int64 `json:"links"`
	QRCodes    int64 `json:"qr_codes"`
	// Workspace plan states right now.
	Plans map[string]int `json:"plans"`
	// MRR at list price (discounts aren't subtracted), in cents.
	MRRCents int64 `json:"mrr_cents"`
	// Share of workspaces whose trial has ended that now pay.
	ConversionPct float64 `json:"conversion_pct"`
	TrialsEnded   int     `json:"trials_ended"`
	Series        struct {
		Signups    []models.DayCount `json:"signups"`
		Workspaces []models.DayCount `json:"workspaces"`
		Links      []models.DayCount `json:"links"`
		QRCodes    []models.DayCount `json:"qr_codes"`
		Clicks     []models.DayCount `json:"clicks"`
		Scans      []models.DayCount `json:"scans"`
	} `json:"series"`
}

func (s *Service) PlatformMetrics(days int) (*PlatformMetrics, error) {
	if days < 1 || days > 365 {
		days = 30
	}
	c, err := s.repo.PlatformCounts()
	if err != nil {
		return nil, err
	}
	m := &PlatformMetrics{Days: days, Users: c.Users, Workspaces: c.Teams, Links: c.Links, QRCodes: c.QRCodes, Plans: map[string]int{}}
	teams, err := s.repo.AllTeamBilling()
	if err != nil {
		return nil, err
	}
	now := time.Now()
	paid := 0
	for _, t := range teams {
		state := t.Billing.State(now)
		m.Plans[state]++
		switch state {
		case models.BillingActive, models.BillingPastDue:
			if t.Billing.Interval == "year" {
				m.MRRCents += billing.AnnualCents / 12
			} else {
				m.MRRCents += billing.MonthlyCents
			}
		}
		if t.Billing.TrialEndsAt != nil && t.Billing.TrialEndsAt.Before(now) {
			m.TrialsEnded++
			if state == models.BillingActive || state == models.BillingPastDue || state == models.BillingTrialing {
				paid++
			}
		}
	}
	if m.TrialsEnded > 0 {
		m.ConversionPct = float64(paid) * 100 / float64(m.TrialsEnded)
	}
	since := now.UTC().AddDate(0, 0, -days+1).Truncate(24 * time.Hour)
	series := []struct {
		dst          *[]models.DayCount
		table, where string
	}{
		{&m.Series.Signups, "users", ""}, {&m.Series.Workspaces, "teams", ""},
		{&m.Series.Links, "links", ""}, {&m.Series.QRCodes, "qr_codes", ""},
		{&m.Series.Clicks, "analytics_events", "event_type = 'click'"}, {&m.Series.Scans, "analytics_events", "event_type = 'scan'"},
	}
	for _, x := range series {
		if *x.dst, err = s.repo.DailyCounts(x.table, x.where, since); err != nil {
			return nil, err
		}
	}
	return m, nil
}

// ----------------- Workspaces -----------------

func (s *Service) AdminListTeams(q string, limit, offset int) ([]models.AdminTeam, error) {
	return s.repo.AdminListTeams(q, clampLimit(limit), offset)
}

type AdminTeamDetail struct {
	Team    *models.AdminTeam         `json:"team"`
	Members []models.TeamMemberDetail `json:"members"`
	Audit   []models.AuditEvent       `json:"audit"`
}

func (s *Service) AdminGetTeam(id string) (*AdminTeamDetail, error) {
	t, err := s.repo.AdminGetTeam(id)
	if err != nil {
		return nil, err
	}
	d := &AdminTeamDetail{Team: t}
	if d.Members, err = s.repo.ListTeamMembers(id); err != nil {
		return nil, err
	}
	if d.Audit, err = s.repo.ListAudit(id, 25, 0); err != nil {
		return nil, err
	}
	return d, nil
}

// AdminExtendTrial adds days to a workspace's trial (from today if it already ended).
func (s *Service) AdminExtendTrial(a AdminActor, teamID string, days int) (*models.AdminTeam, error) {
	if days < 1 || days > 365 {
		return nil, errors.New("extend by 1 to 365 days")
	}
	t, err := s.repo.GetTeam(teamID)
	if err != nil {
		return nil, err
	}
	start := time.Now().UTC()
	if t.Billing.TrialEndsAt != nil && t.Billing.TrialEndsAt.After(start) {
		start = *t.Billing.TrialEndsAt
	}
	end := start.AddDate(0, 0, days)
	if err := s.repo.SetTeamTrialEnd(teamID, end); err != nil {
		return nil, err
	}
	s.forgetLapse(teamID)
	s.adminAudit(a, "workspace.trial_extend", "team", teamID, t.Name, map[string]any{"days": days, "before": t.Billing.TrialEndsAt, "after": end})
	return s.repo.AdminGetTeam(teamID)
}

func (s *Service) AdminSetComped(a AdminActor, teamID string, comped bool) (*models.AdminTeam, error) {
	t, err := s.repo.GetTeam(teamID)
	if err != nil {
		return nil, err
	}
	if err := s.repo.SetTeamComped(teamID, comped); err != nil {
		return nil, err
	}
	s.forgetLapse(teamID)
	s.adminAudit(a, "workspace.comp", "team", teamID, t.Name, map[string]any{"before": t.Billing.Comped, "after": comped})
	return s.repo.AdminGetTeam(teamID)
}

// AdminSuspendTeam suspends a workspace (reason required) or lifts it (empty reason).
func (s *Service) AdminSuspendTeam(a AdminActor, teamID, reason string) (*models.AdminTeam, error) {
	t, err := s.repo.GetTeam(teamID)
	if err != nil {
		return nil, err
	}
	reason = truncate(strings.TrimSpace(reason), 500)
	if err := s.repo.SetTeamSuspended(teamID, reason); err != nil {
		return nil, err
	}
	s.forgetLapse(teamID)
	action := "workspace.suspend"
	if reason == "" {
		action = "workspace.unsuspend"
	}
	s.adminAudit(a, action, "team", teamID, t.Name, map[string]any{"reason": reason, "before": t.SuspendedReason})
	return s.repo.AdminGetTeam(teamID)
}

// ----------------- Users -----------------

func (s *Service) AdminListUsers(q string, limit, offset int) ([]models.AdminUser, error) {
	return s.repo.AdminListUsers(q, clampLimit(limit), offset)
}

type AdminUserDetail struct {
	User       *models.AdminUser       `json:"user"`
	Workspaces []models.TeamMembership `json:"workspaces"`
}

func (s *Service) AdminGetUser(id string) (*AdminUserDetail, error) {
	u, err := s.repo.AdminGetUser(id)
	if err != nil {
		return nil, err
	}
	teams, err := s.repo.ListUserTeams(id)
	if err != nil {
		return nil, err
	}
	return &AdminUserDetail{User: u, Workspaces: teams}, nil
}

// AdminSuspendUser blocks sign-in and API keys for a user (empty reason lifts it).
func (s *Service) AdminSuspendUser(a AdminActor, userID, reason string) (*models.AdminUser, error) {
	if userID == a.UserID {
		return nil, ErrSelfAdminAction
	}
	u, err := s.repo.GetUserByID(userID)
	if err != nil {
		return nil, err
	}
	reason = truncate(strings.TrimSpace(reason), 500)
	if err := s.repo.SetUserSuspended(userID, reason); err != nil {
		return nil, err
	}
	action := "user.unsuspend"
	if reason != "" {
		action = "user.suspend"
		_ = s.repo.DeleteUserSessionsExcept(userID, "")
	}
	s.adminAudit(a, action, "user", userID, u.Email, map[string]any{"reason": reason, "before": u.SuspendedReason})
	return s.repo.AdminGetUser(userID)
}

func (s *Service) AdminSetPlatformAdmin(a AdminActor, userID string, admin bool) (*models.AdminUser, error) {
	if userID == a.UserID && !admin {
		return nil, ErrSelfAdminAction
	}
	u, err := s.repo.GetUserByID(userID)
	if err != nil {
		return nil, err
	}
	if err := s.repo.SetPlatformAdmin(userID, admin); err != nil {
		return nil, err
	}
	action := "admin.revoke"
	if admin {
		action = "admin.grant"
	}
	s.adminAudit(a, action, "user", userID, u.Email, map[string]any{"before": u.IsPlatformAdmin, "after": admin})
	return s.repo.AdminGetUser(userID)
}

// ----------------- Checkout & trial codes -----------------

// CodeAdmin is the Stripe surface for managing checkout codes.
type CodeAdmin interface {
	CreateCode(ctx context.Context, n billing.NewCode) (*billing.PromotionCode, error)
	ListCodes(ctx context.Context) ([]billing.PromotionCode, error)
	SetCodeActive(ctx context.Context, id string, active bool) error
}

func (s *Service) codeAdmin() (CodeAdmin, error) {
	if s.billing == nil {
		return nil, ErrBillingDisabled
	}
	ca, ok := s.billing.provider.(CodeAdmin)
	if !ok {
		return nil, ErrBillingDisabled
	}
	return ca, nil
}

type AdminCodes struct {
	BillingEnabled bool                    `json:"billing_enabled"`
	Stripe         []billing.PromotionCode `json:"stripe"`
	Trial          []models.TrialCode      `json:"trial"`
}

func (s *Service) AdminListCodes() (*AdminCodes, error) {
	out := &AdminCodes{Stripe: []billing.PromotionCode{}}
	var err error
	if out.Trial, err = s.repo.ListTrialCodes(); err != nil {
		return nil, err
	}
	if ca, err := s.codeAdmin(); err == nil {
		out.BillingEnabled = true
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if out.Stripe, err = ca.ListCodes(ctx); err != nil {
			return nil, fmt.Errorf("could not load codes from Stripe: %w", err)
		}
	}
	return out, nil
}

var codePattern = regexp.MustCompile(`^[A-Z0-9][A-Z0-9_-]{2,39}$`)

func normalizeCode(code string) (string, error) {
	code = strings.ToUpper(strings.TrimSpace(code))
	if !codePattern.MatchString(code) {
		return "", errors.New("codes are 3–40 letters, numbers, dashes or underscores")
	}
	return code, nil
}

type NewCheckoutCode struct {
	Code             string     `json:"code"`
	PercentOff       float64    `json:"percent_off"`
	AmountOffCents   int64      `json:"amount_off_cents"`
	Duration         string     `json:"duration"`
	DurationInMonths int        `json:"duration_in_months"`
	MaxRedemptions   int64      `json:"max_redemptions"`
	ExpiresAt        *time.Time `json:"expires_at"`
	FirstTimeOnly    bool       `json:"first_time_only"`
}

func (s *Service) AdminCreateCheckoutCode(a AdminActor, in NewCheckoutCode) (*billing.PromotionCode, error) {
	ca, err := s.codeAdmin()
	if err != nil {
		return nil, err
	}
	code, err := normalizeCode(in.Code)
	if err != nil {
		return nil, err
	}
	if _, err := s.repo.GetTrialCode(code); err == nil {
		return nil, fmt.Errorf("%w: a trial code with that name exists", ErrConflict)
	}
	switch {
	case in.PercentOff > 0 && in.AmountOffCents > 0, in.PercentOff <= 0 && in.AmountOffCents <= 0:
		return nil, errors.New("set either a percentage or an amount off")
	case in.PercentOff > 100:
		return nil, errors.New("percent off must be 100 or less")
	case in.AmountOffCents > billing.AnnualCents:
		return nil, errors.New("amount off can't exceed the annual price")
	}
	switch in.Duration {
	case "once", "forever":
	case "repeating":
		if in.DurationInMonths < 1 || in.DurationInMonths > 36 {
			return nil, errors.New("repeating discounts last 1 to 36 months")
		}
	default:
		return nil, errors.New("duration must be once, repeating or forever")
	}
	n := billing.NewCode{Code: code, PercentOff: in.PercentOff, AmountOffCents: in.AmountOffCents, Duration: in.Duration,
		DurationInMonths: in.DurationInMonths, MaxRedemptions: in.MaxRedemptions, FirstTimeOnly: in.FirstTimeOnly, CreatedBy: a.Email}
	if in.ExpiresAt != nil {
		if in.ExpiresAt.Before(time.Now()) {
			return nil, errors.New("the expiry date is in the past")
		}
		n.ExpiresAt = *in.ExpiresAt
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	pc, err := ca.CreateCode(ctx, n)
	if err != nil {
		var se *billing.Error
		if errors.As(err, &se) && se.Status < 500 {
			return nil, errors.New(se.Message)
		}
		return nil, fmt.Errorf("could not create the code in Stripe: %w", err)
	}
	s.adminAudit(a, "code.create", "promotion_code", pc.ID, pc.Code, map[string]any{"discount": pc.Coupon.Describe(),
		"max_redemptions": in.MaxRedemptions, "expires_at": in.ExpiresAt, "first_time_only": in.FirstTimeOnly})
	return pc, nil
}

func (s *Service) AdminSetCheckoutCodeActive(a AdminActor, id, label string, active bool) error {
	ca, err := s.codeAdmin()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := ca.SetCodeActive(ctx, id, active); err != nil {
		return fmt.Errorf("could not update the code in Stripe: %w", err)
	}
	action := "code.deactivate"
	if active {
		action = "code.activate"
	}
	s.adminAudit(a, action, "promotion_code", id, label, nil)
	return nil
}

type NewTrialCode struct {
	Code           string     `json:"code"`
	Days           int        `json:"days"`
	MaxRedemptions int        `json:"max_redemptions"`
	ExpiresAt      *time.Time `json:"expires_at"`
	Note           string     `json:"note"`
}

func (s *Service) AdminCreateTrialCode(a AdminActor, in NewTrialCode) (*models.TrialCode, error) {
	code, err := normalizeCode(in.Code)
	if err != nil {
		return nil, err
	}
	if in.Days < 1 || in.Days > 90 {
		return nil, errors.New("trial codes add 1 to 90 days")
	}
	if in.MaxRedemptions < 0 {
		return nil, errors.New("max redemptions can't be negative")
	}
	if in.ExpiresAt != nil && in.ExpiresAt.Before(time.Now()) {
		return nil, errors.New("the expiry date is in the past")
	}
	c := &models.TrialCode{Code: code, Days: in.Days, MaxRedemptions: in.MaxRedemptions, ExpiresAt: in.ExpiresAt,
		Note: truncate(strings.TrimSpace(in.Note), 200), CreatedBy: a.Email}
	if err := s.repo.CreateTrialCode(c); err != nil {
		if errors.Is(err, repository.ErrConflict) {
			return nil, fmt.Errorf("%w: that code already exists", ErrConflict)
		}
		return nil, err
	}
	s.adminAudit(a, "trial_code.create", "trial_code", code, code, map[string]any{"days": in.Days, "max_redemptions": in.MaxRedemptions})
	return c, nil
}

func (s *Service) AdminSetTrialCodeActive(a AdminActor, code string, active bool) error {
	if err := s.repo.SetTrialCodeActive(code, active); err != nil {
		return err
	}
	action := "trial_code.deactivate"
	if active {
		action = "trial_code.activate"
	}
	s.adminAudit(a, action, "trial_code", strings.ToUpper(code), strings.ToUpper(code), nil)
	return nil
}

// trialCodePreview returns a preview when code is a usable Lynqr trial code.
func (s *Service) trialCodePreview(code string) (*PromoPreview, bool) {
	c, err := s.repo.GetTrialCode(code)
	if err != nil || !c.Usable(time.Now()) {
		return nil, false
	}
	return &PromoPreview{Kind: "trial", Code: c.Code, TrialDays: c.Days,
		Description: fmt.Sprintf("Adds %d days to your free trial", c.Days)}, true
}

// RedeemTrialCode lengthens the workspace's free trial.
func (s *Service) RedeemTrialCode(a Actor, code string) (*BillingSummary, error) {
	if err := a.require(auth.PermManageBilling); err != nil {
		return nil, err
	}
	c, err := s.repo.GetTrialCode(code)
	if err != nil || !c.Usable(time.Now()) {
		return nil, billing.ErrInvalidCode
	}
	t, err := s.repo.GetTeam(a.TeamID)
	if err != nil {
		return nil, err
	}
	switch t.Billing.State(time.Now()) {
	case models.BillingTrial, models.BillingExpired:
	default:
		return nil, errors.New("trial codes only apply to workspaces that haven't subscribed")
	}
	start := time.Now().UTC()
	if t.Billing.TrialEndsAt != nil && t.Billing.TrialEndsAt.After(start) {
		start = *t.Billing.TrialEndsAt
	}
	end := start.AddDate(0, 0, c.Days)
	if err := s.repo.RedeemTrialCode(c.Code, t.ID, end); err != nil {
		if errors.Is(err, repository.ErrConflict) {
			return nil, fmt.Errorf("%w: this workspace already used that code", ErrConflict)
		}
		if errors.Is(err, repository.ErrNotFound) {
			return nil, billing.ErrInvalidCode
		}
		return nil, err
	}
	s.forgetLapse(t.ID)
	s.audit(a, "billing.trial_code", "team", t.ID, map[string]any{"code": c.Code, "days": c.Days})
	return s.Billing(a)
}

// ----------------- Moderation -----------------

func (s *Service) AdminSearchAssets(q string) ([]models.AdminAsset, error) {
	if len(strings.TrimSpace(q)) < 2 {
		return []models.AdminAsset{}, nil
	}
	list, err := s.repo.AdminSearchAssets(q, 100)
	if err != nil {
		return nil, err
	}
	for i := range list {
		switch {
		case list[i].Type == "link":
			list[i].PublicURL = s.appURL + "/s/" + list[i].Label
		case list[i].PublicURL != "":
			list[i].PublicURL = s.ScanURL(list[i].PublicURL)
		}
	}
	return list, nil
}

// AdminModerate disables a link or QR code for abuse (empty reason restores it).
func (s *Service) AdminModerate(a AdminActor, targetType, id, reason string) error {
	reason = truncate(strings.TrimSpace(reason), 300)
	var label, before string
	switch targetType {
	case "link":
		l, err := s.repo.GetLinkByID(id)
		if err != nil {
			return ErrLinkNotFound
		}
		label, before = "/s/"+l.Slug, l.ModerationReason
		err = s.repo.SetLinkModeration(id, reason)
		if err != nil {
			return err
		}
	case "qr":
		q, err := s.repo.GetQRCodeByID(id)
		if err != nil {
			return ErrQRNotFound
		}
		label, before = q.Title, q.ModerationReason
		if err := s.repo.SetQRModeration(id, reason); err != nil {
			return err
		}
	default:
		return errors.New("unknown target type")
	}
	action := targetType + ".disable"
	if reason == "" {
		action = targetType + ".restore"
	}
	s.adminAudit(a, action, targetType, id, label, map[string]any{"reason": reason, "before": before})
	return nil
}

func normalizeDomain(d string) (string, error) {
	d = strings.ToLower(strings.TrimSpace(d))
	if strings.Contains(d, "://") {
		if u, err := url.Parse(d); err == nil {
			d = u.Hostname()
		}
	}
	d = strings.TrimSuffix(strings.TrimPrefix(strings.SplitN(d, "/", 2)[0], "*."), ".")
	if len(d) < 3 || len(d) > 253 || !strings.Contains(d, ".") || strings.ContainsAny(d, " @:?#") {
		return "", errors.New("enter a domain like example.com")
	}
	return d, nil
}

func (s *Service) AdminListBlocked() ([]models.BlockedDomain, error) {
	return s.repo.ListBlockedDomains()
}

func (s *Service) AdminBlockDomain(a AdminActor, domain, reason string) (*models.BlockedDomain, error) {
	d, err := normalizeDomain(domain)
	if err != nil {
		return nil, err
	}
	bd := &models.BlockedDomain{Domain: d, Reason: truncate(strings.TrimSpace(reason), 300), CreatedBy: a.Email}
	if err := s.repo.AddBlockedDomain(bd); err != nil {
		if errors.Is(err, repository.ErrConflict) {
			return nil, fmt.Errorf("%w: that domain is already blocked", ErrConflict)
		}
		return nil, err
	}
	s.invalidateBlocklist()
	s.adminAudit(a, "domain.block", "domain", d, d, map[string]any{"reason": bd.Reason})
	return bd, nil
}

func (s *Service) AdminUnblockDomain(a AdminActor, domain string) error {
	d, err := normalizeDomain(domain)
	if err != nil {
		return err
	}
	if err := s.repo.RemoveBlockedDomain(d); err != nil {
		return err
	}
	s.invalidateBlocklist()
	s.adminAudit(a, "domain.unblock", "domain", d, d, nil)
	return nil
}

func (s *Service) invalidateBlocklist() {
	s.blocklist.mu.Lock()
	s.blocklist.loaded = time.Time{}
	s.blocklist.mu.Unlock()
}

func (s *Service) blockedDomains() []string {
	s.blocklist.mu.RLock()
	if time.Since(s.blocklist.loaded) < blocklistTTL {
		d := s.blocklist.domains
		s.blocklist.mu.RUnlock()
		return d
	}
	s.blocklist.mu.RUnlock()
	list, err := s.repo.ListBlockedDomains()
	if err != nil {
		log.Printf("[moderation] could not load blocklist: %v", err)
		return nil
	}
	domains := make([]string, len(list))
	for i, d := range list {
		domains[i] = d.Domain
	}
	s.blocklist.mu.Lock()
	s.blocklist.domains, s.blocklist.loaded = domains, time.Now()
	s.blocklist.mu.Unlock()
	return domains
}

// URLBlocked reports whether a destination's host is on the domain blocklist
// (the domain itself or any subdomain).
func (s *Service) URLBlocked(raw string) bool {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return false
	}
	host := strings.TrimSuffix(strings.ToLower(u.Hostname()), ".")
	if host == "" {
		return false
	}
	for _, d := range s.blockedDomains() {
		if host == d || strings.HasSuffix(host, "."+d) {
			return true
		}
	}
	return false
}

// checkTarget validates a destination URL and enforces the domain blocklist.
func (s *Service) checkTarget(u string) error {
	if err := ValidateTargetURL(u); err != nil {
		return err
	}
	if s.URLBlocked(u) {
		return ErrDomainBlocked
	}
	return nil
}

// ----------------- Abuse reports -----------------

var reportReasons = map[string]bool{"phishing": true, "malware": true, "spam": true, "scam": true, "illegal": true, "other": true}

type AbuseReportInput struct {
	URL     string `json:"url" form:"url"`
	Reason  string `json:"reason" form:"reason"`
	Details string `json:"details" form:"details"`
	Email   string `json:"email" form:"email"`
}

var ErrReportTarget = errors.New("enter the full Lynqr link or QR code address you want to report, like https://lynqr.app/s/abc123")

// SubmitAbuseReport records a visitor's report about a link or QR code.
func (s *Service) SubmitAbuseReport(in AbuseReportInput, ip string) error {
	if !reportReasons[in.Reason] {
		return errors.New("choose a reason for the report")
	}
	raw := strings.TrimSpace(in.URL)
	if !strings.Contains(raw, "://") {
		raw = "https://" + strings.TrimPrefix(raw, "//")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return ErrReportTarget
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	r := &models.AbuseReport{TargetURL: truncate(strings.TrimSpace(in.URL), 500), Reason: in.Reason,
		Details: truncate(strings.TrimSpace(in.Details), 2000), ReporterEmail: truncate(strings.TrimSpace(in.Email), 254), ReporterIP: s.hashIP(ip)}
	switch {
	case len(parts) == 2 && parts[0] == "s":
		l, err := s.repo.GetLinkBySlug(parts[1])
		if err != nil {
			return ErrReportTarget
		}
		r.TargetType, r.TargetID = "link", l.ID
	case len(parts) == 2 && parts[0] == "q":
		q, err := s.repo.GetQRCodeByCode(parts[1])
		if err != nil {
			return ErrReportTarget
		}
		r.TargetType, r.TargetID = "qr", q.ID
	case len(parts) == 3 && parts[0] == "qr" && parts[1] == "r":
		q, err := s.repo.GetQRCodeByID(parts[2])
		if err != nil {
			return ErrReportTarget
		}
		r.TargetType, r.TargetID = "qr", q.ID
	default:
		return ErrReportTarget
	}
	if s.repo.CountRecentReports(r.ReporterIP, time.Now().UTC().Add(-time.Hour)) >= 5 {
		return errors.New("you've sent several reports recently. Please try again later")
	}
	if err := s.repo.CreateAbuseReport(r); err != nil {
		return err
	}
	s.notifyAbuseReport(r)
	return nil
}

func (s *Service) AdminListReports(status string) ([]models.AbuseReport, error) {
	if status != "" && status != "open" && status != "actioned" && status != "dismissed" {
		status = "open"
	}
	return s.repo.ListAbuseReports(status, 200)
}

// AdminResolveReport dismisses a report or disables its target.
func (s *Service) AdminResolveReport(a AdminActor, id, action string) error {
	r, err := s.repo.GetAbuseReport(id)
	if err != nil {
		return err
	}
	status := "dismissed"
	if action == "disable" {
		status = "actioned"
		if err := s.AdminModerate(a, r.TargetType, r.TargetID, "Reported for "+r.Reason); err != nil {
			return err
		}
	} else if action != "dismiss" {
		return errors.New("action must be disable or dismiss")
	}
	if err := s.repo.ResolveAbuseReport(id, status, a.Email); err != nil {
		return err
	}
	s.adminAudit(a, "report."+status, "report", id, r.TargetURL, map[string]any{"reason": r.Reason})
	return nil
}

// ----------------- Platform audit -----------------

func (s *Service) AdminListAudit(q string, limit, offset int) ([]models.PlatformAuditEvent, error) {
	return s.repo.ListPlatformAudit(q, clampLimit(limit), offset)
}
