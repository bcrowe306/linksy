package billing

import (
	"testing"
	"time"
)

func TestVerifyWebhook(t *testing.T) {
	payload := []byte(`{"id":"evt_1","type":"invoice.paid","data":{"object":{}}}`)
	now := time.Now()
	ev, err := VerifyWebhook(payload, SignForTest(payload, "whsec_x", now), "whsec_x", 5*time.Minute, now)
	if err != nil || ev.ID != "evt_1" {
		t.Fatalf("valid signature rejected: %v", err)
	}
	cases := map[string]string{
		"wrong secret": SignForTest(payload, "whsec_y", now),
		"stale":        SignForTest(payload, "whsec_x", now.Add(-10*time.Minute)),
		"garbage":      "t=abc,v1=zzz",
		"empty":        "",
	}
	for name, h := range cases {
		if _, err := VerifyWebhook(payload, h, "whsec_x", 5*time.Minute, now); err == nil {
			t.Errorf("%s: expected rejection", name)
		}
	}
	tampered := append([]byte{}, payload...)
	tampered[10] = 'X'
	if _, err := VerifyWebhook(tampered, SignForTest(payload, "whsec_x", now), "whsec_x", 5*time.Minute, now); err == nil {
		t.Error("tampered payload accepted")
	}
}

func TestCouponDescribe(t *testing.T) {
	cases := map[string]Coupon{
		"50% off for 3 months":           {PercentOff: 50, Duration: "repeating", DurationInMonths: 3},
		"Free forever":                   {PercentOff: 100, Duration: "forever"},
		"$2.00 off on the first payment": {AmountOff: 200, Duration: "once"},
		"20% off for 1 month":            {PercentOff: 20, Duration: "repeating", DurationInMonths: 1},
	}
	for want, c := range cases {
		if got := c.Describe(); got != want {
			t.Errorf("got %q want %q", got, want)
		}
	}
	if (Coupon{AmountOff: 1000}).Discounted(899) != 0 || (Coupon{PercentOff: 25}).Discounted(6000) != 4500 {
		t.Error("discount math wrong")
	}
}
