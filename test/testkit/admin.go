package testkit

import (
	"context"
	"fmt"
	"net/http"
	"regexp"

	"github.com/bcrowe306/linksy/internal/billing"
	"github.com/bcrowe306/linksy/internal/email"
)

// UseMailbox captures outgoing email in memory.
func (k *Kit) UseMailbox() *email.Memory {
	m := &email.Memory{}
	k.Svc.SetMailer(m)
	return m
}

var sixDigits = regexp.MustCompile(`\b\d{6}\b`)

// AdminLogin signs in a platform admin (via ADMIN_EMAILS) and unlocks the
// admin console with an emailed code.
func (k *Kit) AdminLogin(addr string) *User {
	k.T.Helper()
	mail := k.UseMailbox()
	k.Svc.SetAdminEmails([]string{addr})
	u := k.Login(addr)
	k.Expect(k.Do("POST", "/api/v1/admin/session/code", u.Token, nil), http.StatusOK, "send admin code")
	msg, ok := mail.LastTagged(addr, "admin_code")
	if !ok {
		k.T.Fatal("no admin code email")
	}
	k.Expect(k.Do("POST", "/api/v1/admin/session", u.Token, map[string]any{"code": sixDigits.FindString(msg.Text)}), http.StatusOK, "verify admin code")
	return u
}

// Code-management surface of the fake Stripe.

func (f *FakeStripe) CreateCode(_ context.Context, n billing.NewCode) (*billing.PromotionCode, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.Codes[n.Code]; ok {
		return nil, &billing.Error{Status: 400, Message: "An active promotion code with `code: " + n.Code + "` already exists."}
	}
	pc := &billing.PromotionCode{ID: fmt.Sprintf("promo_%d", len(f.Codes)+1), Code: n.Code, Active: true, MaxRedemptions: n.MaxRedemptions,
		Coupon: billing.Coupon{PercentOff: n.PercentOff, AmountOff: n.AmountOffCents, Duration: n.Duration, DurationInMonths: n.DurationInMonths, Valid: true}}
	f.Codes[n.Code] = pc
	return pc, nil
}

func (f *FakeStripe) ListCodes(context.Context) ([]billing.PromotionCode, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := []billing.PromotionCode{}
	for _, c := range f.Codes {
		out = append(out, *c)
	}
	return out, nil
}

func (f *FakeStripe) SetCodeActive(_ context.Context, id string, active bool) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for code, c := range f.Codes {
		if c.ID == id {
			c.Active = active
			if !active {
				delete(f.Codes, code) // inactive codes can't be looked up
			}
			return nil
		}
	}
	return &billing.Error{Status: 404, Message: "No such promotion code"}
}
