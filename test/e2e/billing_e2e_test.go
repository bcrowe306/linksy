package e2e_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bcrowe306/linksy/internal/billing"
	"github.com/bcrowe306/linksy/test/testkit"
)

func webhook(k *testkit.Kit, id, typ string, object any, secret string) *testkit.Response {
	obj, _ := json.Marshal(object)
	payload := []byte(fmt.Sprintf(`{"id":%q,"type":%q,"data":{"object":%s}}`, id, typ, obj))
	req := httptest.NewRequest("POST", "/api/webhooks/stripe", strings.NewReader(string(payload)))
	req.Header.Set("Stripe-Signature", billing.SignForTest(payload, secret, time.Now()))
	return k.Send(req)
}

func TestTrialLapseAndReadOnly(t *testing.T) {
	k := testkit.New(t)
	k.EnableBilling()
	u := k.Login("trial@acme.test")

	b := k.Do("GET", "/api/v1/billing", u.Token, nil).Map()
	if b["state"] != "trial" || b["trial_days_left"].(float64) != 7 || b["read_only"] != false {
		t.Fatalf("new workspace should be on a 7-day trial: %v", b)
	}
	l := createLink(t, k, u.Token, map[string]any{"target_url": "https://example.com", "custom_slug": "trial-link"})
	q := createQR(t, k, u.Token, map[string]any{"type": "website", "title": "QR", "target_url": "https://example.com", "is_dynamic": true})
	up := upload(k, "/api/v1/documents/upload", "POST", u.Token, "menu.pdf", []byte("%PDF-1.4 menu"))
	var doc struct {
		Document struct{ ID string } `json:"document"`
		URL      string              `json:"download_url"`
	}
	_ = up.JSON(&doc)

	k.ExpireTrial(u.TeamID)
	k.Svc.ForgetLapseForTest(u.TeamID)

	me := k.Do("GET", "/api/v1/auth/me", u.Token, nil).Map()
	perms := fmt.Sprint(me["permissions"])
	if strings.Contains(perms, "assets:edit") || !strings.Contains(perms, "billing:manage") || me["billing"].(map[string]any)["state"] != "expired" {
		t.Fatalf("lapsed workspace should be read-only: %v", me)
	}
	r := k.Do("POST", "/api/v1/links", u.Token, map[string]any{"target_url": "https://example.com"})
	k.Expect(r, http.StatusPaymentRequired, "create while lapsed")
	if r.Map()["code"] != "subscription_required" {
		t.Fatalf("unexpected body: %s", r.Body)
	}
	k.Expect(k.Do("PUT", "/api/v1/links/"+l.ID, u.Token, map[string]any{"target_url": "https://example.org"}), http.StatusPaymentRequired, "edit while lapsed")
	k.Expect(k.Do("GET", "/api/v1/links", u.Token, nil), http.StatusOK, "reads still work")
	k.Expect(k.Do("GET", "/api/v1/qr/"+q.ID+"/export?format=png", u.Token, nil), http.StatusOK, "downloads still work")

	// Codes stop for visitors
	k.Expect(k.Do("GET", "/s/trial-link", "", nil), http.StatusGone, "link stops")
	k.Expect(k.Do("GET", "/q/"+q.Code, "", nil), http.StatusGone, "scan stops")
	k.Expect(k.Do("GET", "/d/"+doc.Document.ID, "", nil), http.StatusGone, "document link stops")
	k.Expect(k.Do("GET", strings.TrimPrefix(doc.URL, testkit.AppURL), "", nil), http.StatusGone, "file stops")
}

func TestCheckoutAndPromoCodes(t *testing.T) {
	k := testkit.New(t)
	f := k.EnableBilling()
	owner := k.Login("buyer@acme.test")
	f.Codes["LAUNCH50"] = &billing.PromotionCode{ID: "promo_1", Code: "LAUNCH50", Active: true,
		Coupon: billing.Coupon{PercentOff: 50, Duration: "repeating", DurationInMonths: 3, Valid: true}}
	f.Codes["FREEPASS"] = &billing.PromotionCode{ID: "promo_2", Code: "FREEPASS", Active: true,
		Coupon: billing.Coupon{PercentOff: 100, Duration: "forever", Valid: true}}

	p := k.Do("POST", "/api/v1/billing/promo", owner.Token, map[string]any{"code": "LAUNCH50", "interval": "month"})
	k.Expect(p, http.StatusOK, "promo preview")
	if m := p.Map(); m["description"] != "50% off for 3 months" || m["discounted_cents"].(float64) != 449 || m["original_cents"].(float64) != 899 {
		t.Fatalf("preview: %v", m)
	}
	free := k.Do("POST", "/api/v1/billing/promo", owner.Token, map[string]any{"code": "FREEPASS", "interval": "year"}).Map()
	if free["free"] != true || free["description"] != "Free forever" {
		t.Fatalf("free code preview: %v", free)
	}
	k.Expect(k.Do("POST", "/api/v1/billing/promo", owner.Token, map[string]any{"code": "NOPE"}), http.StatusBadRequest, "unknown code")

	c := k.Do("POST", "/api/v1/billing/checkout", owner.Token, map[string]any{"interval": "year", "code": "LAUNCH50"})
	k.Expect(c, http.StatusOK, "checkout")
	if !strings.HasPrefix(c.Map()["url"].(string), "https://checkout.stripe.com/") {
		t.Fatalf("checkout url: %s", c.Body)
	}
	got := f.Checkouts[0]
	if got.PriceID != "price_year" || got.PromotionCodeID != "promo_1" || got.TeamID != owner.TeamID || got.TrialEnd.IsZero() ||
		!strings.Contains(got.SuccessURL, "tab=billing") {
		t.Fatalf("checkout params: %+v", got)
	}

	// Only owners manage billing
	admin := k.Login("admin@acme.test")
	inv := k.Do("POST", "/api/v1/team/members/invite", owner.Token, map[string]any{"email": admin.Email, "role": "admin"}).Map()["invitation"].(map[string]any)
	token := inv["accept_url"].(string)[strings.LastIndex(inv["accept_url"].(string), "/")+1:]
	k.Do("POST", "/api/v1/invitations/"+token+"/accept", admin.Token, nil)
	k.Expect(k.Do("POST", "/api/v1/billing/checkout", admin.Token, map[string]any{"interval": "month"}), http.StatusForbidden, "admin checkout")
	k.Expect(k.Do("GET", "/api/v1/billing", admin.Token, nil), http.StatusOK, "admin can read plan")
}

func TestStripeWebhookSync(t *testing.T) {
	k := testkit.New(t)
	f := k.EnableBilling()
	u := k.Login("hook@acme.test")
	k.ExpireTrial(u.TeamID)
	k.Svc.ForgetLapseForTest(u.TeamID)
	k.Expect(k.Do("POST", "/api/v1/links", u.Token, map[string]any{"target_url": "https://example.com"}), http.StatusPaymentRequired, "lapsed")

	sub := &billing.Subscription{ID: "sub_1", Customer: "cus_x", Status: "active", Metadata: map[string]string{"team_id": u.TeamID},
		CurrentPeriodEnd: time.Now().Add(30 * 24 * time.Hour).Unix()}
	sub.Items.Data = append(sub.Items.Data, struct {
		CurrentPeriodEnd int64 `json:"current_period_end"`
		Price            struct {
			ID        string `json:"id"`
			Recurring struct {
				Interval string `json:"interval"`
			} `json:"recurring"`
		} `json:"price"`
	}{})
	sub.Items.Data[0].Price.Recurring.Interval = "year"
	f.Subscriptions["sub_1"] = sub

	k.Expect(webhook(k, "evt_bad", "customer.subscription.updated", map[string]any{"id": "sub_1"}, "whsec_wrong"), http.StatusBadRequest, "bad signature")
	k.Expect(webhook(k, "evt_1", "checkout.session.completed", map[string]any{"subscription": "sub_1", "customer": "cus_x"}, testkit.WebhookSecret), http.StatusOK, "checkout completed")

	b := k.Do("GET", "/api/v1/billing", u.Token, nil).Map()
	if b["state"] != "active" || b["interval"] != "year" || b["read_only"] != false {
		t.Fatalf("subscription not synced: %v", b)
	}
	k.Expect(k.Do("POST", "/api/v1/links", u.Token, map[string]any{"target_url": "https://example.com"}), http.StatusCreated, "writes restored")
	k.Expect(k.Do("POST", "/api/v1/billing/checkout", u.Token, map[string]any{"interval": "month"}), http.StatusConflict, "already subscribed")
	k.Expect(k.Do("POST", "/api/v1/billing/portal", u.Token, nil), http.StatusOK, "portal")

	// Replays are ignored; cancellation lapses the workspace again
	k.Expect(webhook(k, "evt_1", "checkout.session.completed", map[string]any{"subscription": "sub_1"}, testkit.WebhookSecret), http.StatusOK, "replay")
	f.Subscriptions["sub_1"].Status = "canceled"
	k.Expect(webhook(k, "evt_2", "customer.subscription.deleted", map[string]any{"id": "sub_1", "customer": "cus_x"}, testkit.WebhookSecret), http.StatusOK, "canceled")
	if s := k.Do("GET", "/api/v1/billing", u.Token, nil).Map()["state"]; s != "canceled" {
		t.Fatalf("expected canceled, got %v", s)
	}
	k.Expect(k.Do("POST", "/api/v1/links", u.Token, map[string]any{"target_url": "https://example.com"}), http.StatusPaymentRequired, "lapsed again")
}

func TestBillingDisabledNeverLocksUsers(t *testing.T) {
	k := testkit.New(t)
	u := k.Login("free@acme.test")
	k.ExpireTrial(u.TeamID)
	k.Expect(k.Do("POST", "/api/v1/links", u.Token, map[string]any{"target_url": "https://example.com"}), http.StatusCreated, "no enforcement without Stripe")
	k.Expect(k.Do("POST", "/api/v1/billing/checkout", u.Token, map[string]any{"interval": "month"}), http.StatusServiceUnavailable, "checkout unavailable")
}
