package testkit

import (
	"context"
	"sync"
	"time"

	"github.com/bcrowe306/linksy/internal/billing"
)

// WebhookSecret signs fake Stripe webhooks in tests.
const WebhookSecret = "whsec_test_secret"

// FakeStripe records calls and serves canned subscriptions and promotion codes.
type FakeStripe struct {
	mu            sync.Mutex
	Checkouts     []billing.CheckoutParams
	Customers     int
	Subscriptions map[string]*billing.Subscription
	Codes         map[string]*billing.PromotionCode
}

func (f *FakeStripe) CreateCustomer(_ context.Context, email, name, teamID, teamName string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Customers++
	return "cus_" + teamID, nil
}

func (f *FakeStripe) CreateCheckout(_ context.Context, p billing.CheckoutParams) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Checkouts = append(f.Checkouts, p)
	return "https://checkout.stripe.com/c/pay/cs_test_" + p.TeamID, nil
}

func (f *FakeStripe) CreatePortal(_ context.Context, customer, returnURL string) (string, error) {
	return "https://billing.stripe.com/p/session/" + customer, nil
}

func (f *FakeStripe) FindPromotionCode(_ context.Context, code string) (*billing.PromotionCode, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if pc, ok := f.Codes[code]; ok {
		return pc, nil
	}
	return nil, billing.ErrInvalidCode
}

func (f *FakeStripe) GetSubscription(_ context.Context, id string) (*billing.Subscription, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if s, ok := f.Subscriptions[id]; ok {
		cp := *s
		return &cp, nil
	}
	return nil, &billing.Error{Status: 404, Message: "No such subscription"}
}

// EnableBilling turns on billing enforcement backed by a fake Stripe.
func (k *Kit) EnableBilling() *FakeStripe {
	f := &FakeStripe{Subscriptions: map[string]*billing.Subscription{}, Codes: map[string]*billing.PromotionCode{}}
	k.Svc.EnableBilling(f, billing.Prices{Monthly: "price_month", Annual: "price_year"}, WebhookSecret)
	return f
}

// ExpireTrial moves a workspace's trial end into the past.
func (k *Kit) ExpireTrial(teamID string) {
	k.T.Helper()
	if err := k.Repo.SetTeamTrialEnd(teamID, time.Now().Add(-time.Hour)); err != nil {
		k.T.Fatal(err)
	}
}
