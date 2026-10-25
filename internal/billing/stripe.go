// Package billing talks to Stripe: catalog setup, Checkout, the customer
// portal, promotion codes and webhook verification. It uses Stripe's REST API
// directly (form-encoded requests, JSON responses).
package billing

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	apiBase        = "https://api.stripe.com/v1"
	productName    = "Lynqr Pro"
	LookupMonthly  = "lynqr_pro_monthly"
	LookupAnnual   = "lynqr_pro_annual"
	MonthlyCents   = 899  // $8.99 billed monthly
	AnnualCents    = 6000 // $60 billed yearly ($5/month)
	Currency       = "usd"
	TrialDays      = 7
	minTrialRemain = 49 * time.Hour // Stripe requires trial_end ≥ 48h ahead
)

// Prices are the two plan prices.
type Prices struct{ Monthly, Annual string }

// Client is a minimal Stripe API client.
type Client struct {
	Key  string
	HTTP *http.Client
	Base string
}

func New(key string) *Client {
	return &Client{Key: key, HTTP: &http.Client{Timeout: 15 * time.Second}, Base: apiBase}
}

// Error is a Stripe API error.
type Error struct {
	Status  int
	Type    string `json:"type"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *Error) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return fmt.Sprintf("stripe error %d", e.Status)
}

func (c *Client) do(ctx context.Context, method, path string, form url.Values, out any) error {
	var body io.Reader
	u := c.Base + path
	if method == http.MethodGet && form != nil {
		u += "?" + form.Encode()
	} else if form != nil {
		body = strings.NewReader(form.Encode())
	}
	req, err := http.NewRequestWithContext(ctx, method, u, body)
	if err != nil {
		return err
	}
	req.SetBasicAuth(c.Key, "")
	if body != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("could not reach Stripe: %w", err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if resp.StatusCode >= 400 {
		var w struct {
			Error Error `json:"error"`
		}
		_ = json.Unmarshal(data, &w)
		w.Error.Status = resp.StatusCode
		return &w.Error
	}
	if out != nil {
		return json.Unmarshal(data, out)
	}
	return nil
}

type listResp[T any] struct {
	Data []T `json:"data"`
}

type price struct {
	ID        string `json:"id"`
	LookupKey string `json:"lookup_key"`
	Active    bool   `json:"active"`
}

// EnsureCatalog finds (or creates) the Lynqr Pro product and its two prices,
// keyed by lookup_key so it is safe to run on every boot.
func (c *Client) EnsureCatalog(ctx context.Context) (Prices, error) {
	var p Prices
	q := url.Values{"active": {"true"}}
	q.Add("lookup_keys[]", LookupMonthly)
	q.Add("lookup_keys[]", LookupAnnual)
	var existing listResp[price]
	if err := c.do(ctx, http.MethodGet, "/prices", q, &existing); err != nil {
		return p, err
	}
	for _, pr := range existing.Data {
		switch pr.LookupKey {
		case LookupMonthly:
			p.Monthly = pr.ID
		case LookupAnnual:
			p.Annual = pr.ID
		}
	}
	if p.Monthly != "" && p.Annual != "" {
		return p, nil
	}

	var prod struct {
		ID string `json:"id"`
	}
	var products struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := c.do(ctx, http.MethodGet, "/products/search", url.Values{"query": {`metadata['lynqr_plan']:'pro'`}}, &products); err == nil && len(products.Data) > 0 {
		prod.ID = products.Data[0].ID
	} else {
		if err := c.do(ctx, http.MethodPost, "/products", url.Values{
			"name":                 {productName},
			"description":          {"Unlimited dynamic QR codes, short links, documents, analytics and team members."},
			"metadata[lynqr_plan]": {"pro"},
			"tax_code":             {"txcd_10103001"}, // SaaS: business use
		}, &prod); err != nil {
			return p, fmt.Errorf("create product: %w", err)
		}
	}
	create := func(lookup string, cents int, interval, nickname string) (string, error) {
		var pr price
		err := c.do(ctx, http.MethodPost, "/prices", url.Values{
			"product":             {prod.ID},
			"currency":            {Currency},
			"unit_amount":         {strconv.Itoa(cents)},
			"recurring[interval]": {interval},
			"lookup_key":          {lookup},
			"nickname":            {nickname},
			"tax_behavior":        {"exclusive"},
		}, &pr)
		return pr.ID, err
	}
	var err error
	if p.Monthly == "" {
		if p.Monthly, err = create(LookupMonthly, MonthlyCents, "month", "Pro monthly"); err != nil {
			return p, fmt.Errorf("create monthly price: %w", err)
		}
	}
	if p.Annual == "" {
		if p.Annual, err = create(LookupAnnual, AnnualCents, "year", "Pro annual"); err != nil {
			return p, fmt.Errorf("create annual price: %w", err)
		}
	}
	return p, nil
}

// EnsurePortal makes sure a customer-portal configuration exists that lets
// customers update payment details, switch plans, view invoices and cancel.
func (c *Client) EnsurePortal(ctx context.Context, prices Prices, productID string, returnURL string) error {
	var cfgs listResp[struct {
		ID        string `json:"id"`
		IsDefault bool   `json:"is_default"`
		Active    bool   `json:"active"`
	}]
	if err := c.do(ctx, http.MethodGet, "/billing_portal/configurations", url.Values{"active": {"true"}}, &cfgs); err != nil {
		return err
	}
	for _, cfg := range cfgs.Data {
		if cfg.IsDefault {
			return nil
		}
	}
	form := url.Values{
		"business_profile[headline]":                               {"Manage your Lynqr subscription"},
		"default_return_url":                                       {returnURL},
		"features[payment_method_update][enabled]":                 {"true"},
		"features[invoice_history][enabled]":                       {"true"},
		"features[customer_update][enabled]":                       {"true"},
		"features[customer_update][allowed_updates][]":             {"email", "address", "tax_id", "name"},
		"features[subscription_cancel][enabled]":                   {"true"},
		"features[subscription_cancel][mode]":                      {"at_period_end"},
		"features[subscription_update][enabled]":                   {"true"},
		"features[subscription_update][default_allowed_updates][]": {"price", "promotion_code"},
		"features[subscription_update][proration_behavior]":        {"create_prorations"},
		"features[subscription_update][products][0][product]":      {productID},
		"features[subscription_update][products][0][prices][]":     {prices.Monthly, prices.Annual},
	}
	return c.do(ctx, http.MethodPost, "/billing_portal/configurations", form, nil)
}

// ProductOf returns the product a price belongs to.
func (c *Client) ProductOf(ctx context.Context, priceID string) (string, error) {
	var pr struct {
		Product string `json:"product"`
	}
	err := c.do(ctx, http.MethodGet, "/prices/"+url.PathEscape(priceID), nil, &pr)
	return pr.Product, err
}

func (c *Client) CreateCustomer(ctx context.Context, email, name, teamID, teamName string) (string, error) {
	var cu struct {
		ID string `json:"id"`
	}
	err := c.do(ctx, http.MethodPost, "/customers", url.Values{
		"email": {email}, "name": {name},
		"description":         {teamName},
		"metadata[team_id]":   {teamID},
		"metadata[workspace]": {teamName},
	}, &cu)
	return cu.ID, err
}

// CheckoutParams describes a subscription checkout for one workspace.
type CheckoutParams struct {
	Customer        string
	PriceID         string
	TeamID          string
	PromotionCodeID string    // optional, pre-applied
	TrialEnd        time.Time // optional: keep remaining Lynqr trial
	SuccessURL      string
	CancelURL       string
}

func (c *Client) CreateCheckout(ctx context.Context, p CheckoutParams) (string, error) {
	form := url.Values{
		"mode":                                 {"subscription"},
		"customer":                             {p.Customer},
		"client_reference_id":                  {p.TeamID},
		"line_items[0][price]":                 {p.PriceID},
		"line_items[0][quantity]":              {"1"},
		"success_url":                          {p.SuccessURL},
		"cancel_url":                           {p.CancelURL},
		"subscription_data[metadata][team_id]": {p.TeamID},
		"payment_method_collection":            {"if_required"}, // 100%-off codes need no card
		"billing_address_collection":           {"auto"},
		"customer_update[address]":             {"auto"},
		"customer_update[name]":                {"auto"},
		"tax_id_collection[enabled]":           {"true"},
	}
	if p.PromotionCodeID != "" {
		form.Set("discounts[0][promotion_code]", p.PromotionCodeID)
	} else {
		form.Set("allow_promotion_codes", "true")
	}
	if !p.TrialEnd.IsZero() && time.Until(p.TrialEnd) > minTrialRemain {
		form.Set("subscription_data[trial_end]", strconv.FormatInt(p.TrialEnd.Unix(), 10))
	}
	var s struct {
		URL string `json:"url"`
	}
	err := c.do(ctx, http.MethodPost, "/checkout/sessions", form, &s)
	return s.URL, err
}

func (c *Client) CreatePortal(ctx context.Context, customer, returnURL string) (string, error) {
	var s struct {
		URL string `json:"url"`
	}
	err := c.do(ctx, http.MethodPost, "/billing_portal/sessions", url.Values{"customer": {customer}, "return_url": {returnURL}}, &s)
	return s.URL, err
}

// Coupon is the discount behind a promotion code.
type Coupon struct {
	ID               string  `json:"id"`
	PercentOff       float64 `json:"percent_off"`
	AmountOff        int64   `json:"amount_off"`
	Currency         string  `json:"currency"`
	Duration         string  `json:"duration"` // once | repeating | forever
	DurationInMonths int     `json:"duration_in_months"`
	Name             string  `json:"name"`
	Valid            bool    `json:"valid"`
	AppliesTo        *struct {
		Products []string `json:"products"`
	} `json:"applies_to"`
}

// PromotionCode is a customer-facing code.
type PromotionCode struct {
	ID             string `json:"id"`
	Code           string `json:"code"`
	Active         bool   `json:"active"`
	ExpiresAt      int64  `json:"expires_at"`
	MaxRedemptions int64  `json:"max_redemptions"`
	TimesRedeemed  int64  `json:"times_redeemed"`
	Coupon         Coupon `json:"coupon"`
	Restrictions   struct {
		FirstTimeTransaction  bool   `json:"first_time_transaction"`
		MinimumAmount         int64  `json:"minimum_amount"`
		MinimumAmountCurrency string `json:"minimum_amount_currency"`
	} `json:"restrictions"`
}

// ErrInvalidCode is returned for unknown, inactive or exhausted codes.
var ErrInvalidCode = errors.New("that code isn't valid or has expired")

// FindPromotionCode looks up an active promotion code (case-insensitive).
func (c *Client) FindPromotionCode(ctx context.Context, code string) (*PromotionCode, error) {
	code = strings.TrimSpace(code)
	if code == "" || len(code) > 100 {
		return nil, ErrInvalidCode
	}
	q := url.Values{"code": {code}, "active": {"true"}, "limit": {"1"}}
	q.Add("expand[]", "data.coupon.applies_to")
	var res listResp[PromotionCode]
	if err := c.do(ctx, http.MethodGet, "/promotion_codes", q, &res); err != nil {
		return nil, err
	}
	if len(res.Data) == 0 {
		return nil, ErrInvalidCode
	}
	pc := res.Data[0]
	if !pc.Active || !pc.Coupon.Valid || (pc.ExpiresAt > 0 && time.Unix(pc.ExpiresAt, 0).Before(time.Now())) ||
		(pc.MaxRedemptions > 0 && pc.TimesRedeemed >= pc.MaxRedemptions) {
		return nil, ErrInvalidCode
	}
	return &pc, nil
}

// Describe renders a coupon for people: "50% off for 3 months", "Free forever".
func (cp Coupon) Describe() string {
	var amount string
	switch {
	case cp.PercentOff >= 100:
		amount = "Free"
	case cp.PercentOff > 0:
		amount = strconv.FormatFloat(cp.PercentOff, 'f', -1, 64) + "% off"
	case cp.AmountOff > 0:
		amount = fmt.Sprintf("$%.2f off", float64(cp.AmountOff)/100)
	default:
		return cp.Name
	}
	switch cp.Duration {
	case "forever":
		return amount + " forever"
	case "repeating":
		if cp.DurationInMonths == 1 {
			return amount + " for 1 month"
		}
		return fmt.Sprintf("%s for %d months", amount, cp.DurationInMonths)
	}
	return amount + " on the first payment"
}

// Discounted returns the price in cents after the coupon.
func (cp Coupon) Discounted(cents int64) int64 {
	if cp.PercentOff > 0 {
		return int64(float64(cents) * (100 - cp.PercentOff) / 100)
	}
	if cp.AmountOff > 0 {
		return max(0, cents-cp.AmountOff)
	}
	return cents
}

// Subscription is the subset of a Stripe subscription Lynqr stores.
type Subscription struct {
	ID                string            `json:"id"`
	Customer          string            `json:"customer"`
	Status            string            `json:"status"`
	CancelAtPeriodEnd bool              `json:"cancel_at_period_end"`
	CurrentPeriodEnd  int64             `json:"current_period_end"`
	TrialEnd          int64             `json:"trial_end"`
	Metadata          map[string]string `json:"metadata"`
	Items             struct {
		Data []struct {
			CurrentPeriodEnd int64 `json:"current_period_end"`
			Price            struct {
				ID        string `json:"id"`
				Recurring struct {
					Interval string `json:"interval"`
				} `json:"recurring"`
			} `json:"price"`
		} `json:"data"`
	} `json:"items"`
	Discounts []struct {
		Coupon        Coupon `json:"coupon"`
		PromotionCode *struct {
			Code string `json:"code"`
		} `json:"promotion_code"`
	} `json:"discounts"`
}

// Interval returns "month" or "year".
func (s *Subscription) Interval() string {
	if len(s.Items.Data) > 0 {
		return s.Items.Data[0].Price.Recurring.Interval
	}
	return ""
}

// PeriodEnd handles both API shapes (period end on the subscription or its items).
func (s *Subscription) PeriodEnd() time.Time {
	ts := s.CurrentPeriodEnd
	if ts == 0 && len(s.Items.Data) > 0 {
		ts = s.Items.Data[0].CurrentPeriodEnd
	}
	if ts == 0 {
		return time.Time{}
	}
	return time.Unix(ts, 0).UTC()
}

func (c *Client) GetSubscription(ctx context.Context, id string) (*Subscription, error) {
	q := url.Values{}
	q.Add("expand[]", "discounts")
	q.Add("expand[]", "discounts.promotion_code")
	var s Subscription
	err := c.do(ctx, http.MethodGet, "/subscriptions/"+url.PathEscape(id), q, &s)
	return &s, err
}

// ----------------- Webhooks -----------------

// Event is a Stripe event envelope.
type Event struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	Data struct {
		Object json.RawMessage `json:"object"`
	} `json:"data"`
}

var ErrBadSignature = errors.New("invalid Stripe signature")

// VerifyWebhook checks the Stripe-Signature header (HMAC-SHA256 over
// "timestamp.payload") and rejects events older than the tolerance.
func VerifyWebhook(payload []byte, header, secret string, tolerance time.Duration, now time.Time) (*Event, error) {
	if secret == "" {
		return nil, errors.New("webhook secret not configured")
	}
	var ts int64
	var sigs []string
	for _, part := range strings.Split(header, ",") {
		k, v, ok := strings.Cut(strings.TrimSpace(part), "=")
		if !ok {
			continue
		}
		switch k {
		case "t":
			ts, _ = strconv.ParseInt(v, 10, 64)
		case "v1":
			sigs = append(sigs, v)
		}
	}
	if ts == 0 || len(sigs) == 0 {
		return nil, ErrBadSignature
	}
	if d := now.Sub(time.Unix(ts, 0)); d > tolerance || d < -tolerance {
		return nil, ErrBadSignature
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(strconv.FormatInt(ts, 10)))
	mac.Write([]byte("."))
	mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))
	ok := false
	for _, s := range sigs {
		ok = ok || hmac.Equal([]byte(s), []byte(expected))
	}
	if !ok {
		return nil, ErrBadSignature
	}
	var ev Event
	if err := json.Unmarshal(payload, &ev); err != nil {
		return nil, err
	}
	return &ev, nil
}

// SignForTest produces a valid signature header (used by tests).
func SignForTest(payload []byte, secret string, ts time.Time) string {
	mac := hmac.New(sha256.New, []byte(secret))
	fmt.Fprintf(mac, "%d.", ts.Unix())
	mac.Write(payload)
	return fmt.Sprintf("t=%d,v1=%s", ts.Unix(), hex.EncodeToString(mac.Sum(nil)))
}
