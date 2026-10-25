package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/bcrowe306/linksy/internal/auth"
	"github.com/bcrowe306/linksy/internal/billing"
	"github.com/bcrowe306/linksy/internal/models"
)

var (
	ErrPaymentRequired   = errors.New("this workspace's trial has ended. Subscribe to keep creating and editing")
	ErrBillingDisabled   = errors.New("billing is not configured")
	ErrAlreadySubscribed = errors.New("this workspace already has a subscription. Use Manage billing to change it")
)

// writePermissions are blocked while a workspace's subscription has lapsed.
var writePermissions = map[auth.Permission]bool{
	auth.PermEditAssets: true, auth.PermManageMembers: true, auth.PermManageTeam: true,
}

// BillingProvider is the subset of Stripe Lynqr uses (faked in tests).
type BillingProvider interface {
	CreateCustomer(ctx context.Context, email, name, teamID, teamName string) (string, error)
	CreateCheckout(ctx context.Context, p billing.CheckoutParams) (string, error)
	CreatePortal(ctx context.Context, customer, returnURL string) (string, error)
	FindPromotionCode(ctx context.Context, code string) (*billing.PromotionCode, error)
	GetSubscription(ctx context.Context, id string) (*billing.Subscription, error)
}

type billingConfig struct {
	provider      BillingProvider
	prices        billing.Prices
	webhookSecret string
}

// lapseCache remembers which teams have lost access, for the hot redirect path.
type lapseCache struct {
	mu sync.Mutex
	m  map[string]lapseEntry
}

type lapseEntry struct {
	lapsed    bool
	suspended bool
	until     time.Time
}

// EnableBilling turns on subscriptions and enforcement. Without it (local dev,
// tests) every workspace keeps full access.
func (s *Service) EnableBilling(p BillingProvider, prices billing.Prices, webhookSecret string) {
	s.billing = &billingConfig{provider: p, prices: prices, webhookSecret: webhookSecret}
}

func (s *Service) BillingEnabled() bool { return s.billing != nil }

// TeamLapsed reports whether a workspace has lost access: its plan lapsed or
// Lynqr suspended it. Cached for 30s (see TeamAccess).
func (s *Service) TeamLapsed(teamID string) bool {
	readOnly, _ := s.TeamAccess(teamID)
	return readOnly
}

func (s *Service) forgetLapse(teamID string) {
	s.lapses.mu.Lock()
	delete(s.lapses.m, teamID)
	s.lapses.mu.Unlock()
}

// EffectivePermissions removes write permissions from lapsed workspaces.
func EffectivePermissions(role models.UserRole, readOnly bool) []auth.Permission {
	perms := auth.Permissions(role)
	if !readOnly {
		return perms
	}
	out := perms[:0:0]
	for _, p := range perms {
		if !writePermissions[p] {
			out = append(out, p)
		}
	}
	return out
}

// BillingSummary is what the UI shows about a workspace's plan.
type BillingSummary struct {
	Enabled           bool       `json:"enabled"`
	State             string     `json:"state"`
	ReadOnly          bool       `json:"read_only"`
	Suspended         bool       `json:"suspended"`
	TrialEndsAt       *time.Time `json:"trial_ends_at,omitempty"`
	TrialDaysLeft     int        `json:"trial_days_left"`
	Interval          string     `json:"interval,omitempty"`
	CurrentPeriodEnd  *time.Time `json:"current_period_end,omitempty"`
	CancelAtPeriodEnd bool       `json:"cancel_at_period_end"`
	DiscountCode      string     `json:"discount_code,omitempty"`
	DiscountLabel     string     `json:"discount_label,omitempty"`
	HasCustomer       bool       `json:"has_customer"`
	CanManage         bool       `json:"can_manage"`
	MonthlyCents      int        `json:"monthly_cents"`
	AnnualCents       int        `json:"annual_cents"`
}

func (s *Service) Billing(a Actor) (*BillingSummary, error) {
	if a.UserID == "" || a.TeamID == "" {
		return nil, ErrUnauthorized
	}
	t, err := s.repo.GetTeam(a.TeamID)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	b := t.Billing
	sum := &BillingSummary{
		Enabled: s.billing != nil, State: b.State(now), TrialEndsAt: b.TrialEndsAt, Interval: b.Interval,
		CurrentPeriodEnd: b.CurrentPeriodEnd, CancelAtPeriodEnd: b.CancelAtPeriodEnd, DiscountCode: b.DiscountCode,
		DiscountLabel: b.DiscountLabel, HasCustomer: b.StripeCustomerID != "", CanManage: auth.Can(a.Role, auth.PermManageBilling),
		MonthlyCents: billing.MonthlyCents, AnnualCents: billing.AnnualCents,
	}
	sum.Suspended = t.SuspendedReason != ""
	sum.ReadOnly = (s.billing != nil && b.Lapsed(now)) || sum.Suspended
	if b.TrialEndsAt != nil && now.Before(*b.TrialEndsAt) {
		sum.TrialDaysLeft = int(b.TrialEndsAt.Sub(now).Hours()/24) + 1
	}
	return sum, nil
}

// PromoPreview describes a code's effect on a plan before checkout.
type PromoPreview struct {
	// Kind is "discount" (Stripe checkout code) or "trial" (adds trial days).
	Kind          string `json:"kind"`
	TrialDays     int    `json:"trial_days,omitempty"`
	Code          string `json:"code"`
	Description   string `json:"description"`
	Interval      string `json:"interval"`
	OriginalCents int64  `json:"original_cents"`
	DiscountCents int64  `json:"discounted_cents"`
	Free          bool   `json:"free"`
	FirstTimeOnly bool   `json:"first_time_only"`
}

func (s *Service) PreviewPromo(a Actor, code, interval string) (*PromoPreview, error) {
	if err := a.require(auth.PermManageBilling); err != nil {
		return nil, err
	}
	if s.billing == nil {
		return nil, ErrBillingDisabled
	}
	cents := int64(billing.MonthlyCents)
	priceID := s.billing.prices.Monthly
	if interval == "year" {
		cents, priceID = billing.AnnualCents, s.billing.prices.Annual
	} else {
		interval = "month"
	}
	if p, ok := s.trialCodePreview(code); ok {
		return p, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pc, err := s.billing.provider.FindPromotionCode(ctx, code)
	if err != nil {
		if errors.Is(err, billing.ErrInvalidCode) {
			return nil, err
		}
		return nil, fmt.Errorf("could not check that code: %w", err)
	}
	if pc.Coupon.AppliesTo != nil && len(pc.Coupon.AppliesTo.Products) > 0 && priceID != "" {
		// Product restriction is checked by Stripe at checkout; nothing to compute here.
		_ = priceID
	}
	if pc.Restrictions.MinimumAmount > 0 && cents < pc.Restrictions.MinimumAmount {
		return nil, fmt.Errorf("that code needs a minimum purchase of $%.2f; try the annual plan", float64(pc.Restrictions.MinimumAmount)/100)
	}
	disc := pc.Coupon.Discounted(cents)
	return &PromoPreview{Kind: "discount", Code: pc.Code, Description: pc.Coupon.Describe(), Interval: interval, OriginalCents: cents,
		DiscountCents: disc, Free: disc == 0, FirstTimeOnly: pc.Restrictions.FirstTimeTransaction}, nil
}

// StartCheckout returns a Stripe Checkout URL for the workspace.
func (s *Service) StartCheckout(a Actor, interval, code string) (string, error) {
	if err := a.require(auth.PermManageBilling); err != nil {
		return "", err
	}
	if s.billing == nil {
		return "", ErrBillingDisabled
	}
	t, err := s.repo.GetTeam(a.TeamID)
	if err != nil {
		return "", err
	}
	switch t.Billing.State(time.Now()) {
	case models.BillingActive, models.BillingTrialing, models.BillingPastDue:
		return "", ErrAlreadySubscribed
	}
	priceID := s.billing.prices.Monthly
	if interval == "year" {
		priceID = s.billing.prices.Annual
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	var promoID string
	if strings.TrimSpace(code) != "" {
		pc, err := s.billing.provider.FindPromotionCode(ctx, code)
		if err != nil {
			return "", err
		}
		promoID = pc.ID
	}

	customer := t.Billing.StripeCustomerID
	if customer == "" {
		u, err := s.repo.GetUserByID(a.UserID)
		if err != nil {
			return "", err
		}
		name := strings.TrimSpace(u.FirstName + " " + u.LastName)
		if customer, err = s.billing.provider.CreateCustomer(ctx, u.Email, name, t.ID, t.Name); err != nil {
			return "", fmt.Errorf("could not start checkout: %w", err)
		}
		if err := s.repo.SetTeamCustomer(t.ID, customer); err != nil {
			return "", err
		}
	}
	params := billing.CheckoutParams{
		Customer: customer, PriceID: priceID, TeamID: t.ID, PromotionCodeID: promoID,
		SuccessURL: s.appURL + "/settings?tab=billing&checkout=success",
		CancelURL:  s.appURL + "/settings?tab=billing&checkout=canceled",
	}
	if t.Billing.TrialEndsAt != nil {
		params.TrialEnd = *t.Billing.TrialEndsAt // keep the remaining free days
	}
	url, err := s.billing.provider.CreateCheckout(ctx, params)
	if err != nil {
		return "", fmt.Errorf("could not start checkout: %w", err)
	}
	s.audit(a, "billing.checkout_started", "team", t.ID, map[string]any{"interval": interval, "code": strings.ToUpper(code)})
	return url, nil
}

// PortalURL returns a Stripe customer-portal session for managing the subscription.
func (s *Service) PortalURL(a Actor) (string, error) {
	if err := a.require(auth.PermManageBilling); err != nil {
		return "", err
	}
	if s.billing == nil {
		return "", ErrBillingDisabled
	}
	t, err := s.repo.GetTeam(a.TeamID)
	if err != nil {
		return "", err
	}
	if t.Billing.StripeCustomerID == "" {
		return "", errors.New("subscribe first to manage billing")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return s.billing.provider.CreatePortal(ctx, t.Billing.StripeCustomerID, s.appURL+"/settings?tab=billing")
}

// HandleStripeEvent verifies and applies a webhook. Events are processed once.
func (s *Service) HandleStripeEvent(payload []byte, signature string) error {
	if s.billing == nil {
		return ErrBillingDisabled
	}
	ev, err := billing.VerifyWebhook(payload, signature, s.billing.webhookSecret, 5*time.Minute, time.Now())
	if err != nil {
		return err
	}
	var subID, customer string
	switch {
	case strings.HasPrefix(ev.Type, "customer.subscription."):
		var sub billing.Subscription
		if err := json.Unmarshal(ev.Data.Object, &sub); err != nil {
			return err
		}
		subID, customer = sub.ID, sub.Customer
	case ev.Type == "checkout.session.completed":
		var cs struct {
			Subscription string `json:"subscription"`
			Customer     string `json:"customer"`
		}
		if err := json.Unmarshal(ev.Data.Object, &cs); err != nil {
			return err
		}
		subID, customer = cs.Subscription, cs.Customer
	case strings.HasPrefix(ev.Type, "invoice."):
		var inv struct {
			Subscription string `json:"subscription"`
			Customer     string `json:"customer"`
			Parent       struct {
				SubscriptionDetails struct {
					Subscription string `json:"subscription"`
				} `json:"subscription_details"`
			} `json:"parent"`
		}
		if err := json.Unmarshal(ev.Data.Object, &inv); err != nil {
			return err
		}
		subID, customer = inv.Subscription, inv.Customer
		if subID == "" {
			subID = inv.Parent.SubscriptionDetails.Subscription
		}
	default:
		return nil // not relevant to Lynqr
	}
	if subID == "" {
		return nil
	}
	fresh, err := s.repo.RecordStripeEvent(ev.ID, ev.Type)
	if err != nil || !fresh {
		return err
	}
	return s.syncSubscription(subID, customer)
}

// syncSubscription fetches the current subscription from Stripe (never trusting
// event payload ordering) and stores it on the workspace.
func (s *Service) syncSubscription(subID, customer string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	sub, err := s.billing.provider.GetSubscription(ctx, subID)
	if err != nil {
		return err
	}
	var t *models.Team
	if id := sub.Metadata["team_id"]; id != "" {
		t, err = s.repo.GetTeam(id)
	}
	if t == nil {
		if customer == "" {
			customer = sub.Customer
		}
		t, err = s.repo.GetTeamByCustomer(customer)
	}
	if err != nil || t == nil {
		// Checkouts started on another environment (e.g. test.lynqr.app shares
		// the Stripe account) aren't ours: acknowledge so Stripe stops retrying.
		log.Printf("[billing] ignoring subscription %s: no matching workspace here", subID)
		return nil
	}

	before := t.Billing.State(time.Now())
	b := t.Billing
	b.StripeCustomerID = sub.Customer
	b.StripeSubscriptionID = sub.ID
	b.SubscriptionStatus = sub.Status
	b.Interval = sub.Interval()
	if pe := sub.PeriodEnd(); !pe.IsZero() {
		b.CurrentPeriodEnd = &pe
	}
	b.CancelAtPeriodEnd = sub.CancelAtPeriodEnd
	b.DiscountCode, b.DiscountLabel = "", ""
	if len(sub.Discounts) > 0 {
		d := sub.Discounts[0]
		b.DiscountLabel = d.Coupon.Describe()
		if d.PromotionCode != nil {
			b.DiscountCode = d.PromotionCode.Code
		}
	}
	if err := s.repo.UpdateTeamSubscription(t.ID, b); err != nil {
		return err
	}
	s.forgetLapse(t.ID)
	if after := b.State(time.Now()); after != before {
		s.audit(Actor{UserID: "stripe", TeamID: t.ID}, "billing.status_change", "team", t.ID, map[string]any{"from": before, "to": after})
		s.notifyBillingChange(t.ID, before, after)
	}
	return nil
}

// ForgetLapseForTest drops a cached lapse decision (tests change state directly).
func (s *Service) ForgetLapseForTest(teamID string) { s.forgetLapse(teamID) }
