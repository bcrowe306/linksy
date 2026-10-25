package e2e_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/bcrowe306/linksy/internal/models"
	"github.com/bcrowe306/linksy/test/testkit"
)

func newForm(path string, form url.Values) *http.Request {
	req := httptest.NewRequest("POST", path, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return req
}

func TestAdminAccessRequiresFlagAndReauth(t *testing.T) {
	k := testkit.New(t)
	mail := k.UseMailbox()
	user := k.Login("someone@acme.test")
	r := k.Do("GET", "/api/v1/admin/metrics", user.Token, nil)
	k.Expect(r, http.StatusForbidden, "non-admin")
	if r.Map()["code"] != "not_admin" {
		t.Fatalf("code: %s", r.Body)
	}
	k.Expect(k.Do("POST", "/api/v1/admin/session/code", user.Token, nil), http.StatusForbidden, "non-admin code")

	k.Svc.SetAdminEmails([]string{"Boss@Lynqr.test"})
	boss := k.Login("boss@lynqr.test")
	if me := k.Do("GET", "/api/v1/auth/me", boss.Token, nil).Map(); me["user"].(map[string]any)["is_platform_admin"] != true {
		t.Fatalf("ADMIN_EMAILS not applied: %v", me["user"])
	}
	r = k.Do("GET", "/api/v1/admin/metrics", boss.Token, nil)
	k.Expect(r, http.StatusForbidden, "admin without reauth")
	if r.Map()["code"] != "admin_reauth" {
		t.Fatalf("code: %s", r.Body)
	}
	st := k.Do("GET", "/api/v1/admin/session", boss.Token, nil).Map()
	if st["is_admin"] != true || st["elevated"] != false {
		t.Fatalf("status: %v", st)
	}
	k.Expect(k.Do("POST", "/api/v1/admin/session/code", boss.Token, nil), http.StatusOK, "send code")
	msg, _ := mail.LastTagged("boss@lynqr.test", "admin_code")
	code := strings.Fields(strings.TrimPrefix(msg.Subject, "Your Lynqr admin code: "))[0]
	k.Expect(k.Do("POST", "/api/v1/admin/session", boss.Token, map[string]any{"code": "000000"}), http.StatusBadRequest, "wrong code")
	k.Expect(k.Do("POST", "/api/v1/admin/session", boss.Token, map[string]any{"code": code}), http.StatusOK, "right code")
	k.Expect(k.Do("POST", "/api/v1/admin/session", boss.Token, map[string]any{"code": code}), http.StatusBadRequest, "code is single-use")
	k.Expect(k.Do("GET", "/api/v1/admin/metrics", boss.Token, nil), http.StatusOK, "elevated")

	// API keys never reach the admin console
	key := k.Do("POST", "/api/v1/api-keys", boss.Token, map[string]any{"name": "ci"}).Map()["api_key"].(map[string]any)["key"].(string)
	k.Expect(k.Do("GET", "/api/v1/admin/metrics", key, nil), http.StatusForbidden, "api key")

	k.Expect(k.Do("DELETE", "/api/v1/admin/session", boss.Token, nil), http.StatusOK, "leave")
	k.Expect(k.Do("GET", "/api/v1/admin/metrics", boss.Token, nil), http.StatusForbidden, "after leaving")

	// Every admin step is in the platform audit log
	boss2 := k.AdminLogin("boss@lynqr.test")
	ev := k.Do("GET", "/api/v1/admin/audit", boss2.Token, nil).Map()["events"].([]any)
	actions := ""
	for _, e := range ev {
		actions += e.(map[string]any)["action"].(string) + " "
	}
	if !strings.Contains(actions, "admin.reauth") || !strings.Contains(actions, "admin.grant") {
		t.Fatalf("audit: %s", actions)
	}
}

func TestAdminWorkspacesAndUsers(t *testing.T) {
	k := testkit.New(t)
	k.EnableBilling()
	admin := k.AdminLogin("admin@lynqr.test")
	cust := k.Login("customer@acme.test")
	k.ExpireTrial(cust.TeamID)
	k.Svc.ForgetLapseForTest(cust.TeamID)

	list := k.Do("GET", "/api/v1/admin/workspaces?q=customer", admin.Token, nil).Map()["workspaces"].([]any)
	if len(list) != 1 || list[0].(map[string]any)["state"] != "expired" {
		t.Fatalf("search: %v", list)
	}
	k.Expect(k.Do("POST", "/api/v1/admin/workspaces/"+cust.TeamID+"/trial", admin.Token, map[string]any{"days": 14}), http.StatusOK, "extend")
	if s := k.Do("GET", "/api/v1/billing", cust.Token, nil).Map(); s["state"] != "trial" || s["trial_days_left"].(float64) != 14 {
		t.Fatalf("extended trial: %v", s)
	}
	k.Expect(k.Do("POST", "/api/v1/admin/workspaces/"+cust.TeamID+"/comp", admin.Token, map[string]any{"comped": true}), http.StatusOK, "comp")
	if s := k.Do("GET", "/api/v1/billing", cust.Token, nil).Map(); s["state"] != "comped" {
		t.Fatalf("comped: %v", s)
	}

	// Suspension: read-only for members, codes show the disabled page
	createLink(t, k, cust.Token, map[string]any{"target_url": "https://example.com", "custom_slug": "cust-link"})
	k.Expect(k.Do("POST", "/api/v1/admin/workspaces/"+cust.TeamID+"/suspend", admin.Token, map[string]any{}), http.StatusBadRequest, "reason required")
	k.Expect(k.Do("POST", "/api/v1/admin/workspaces/"+cust.TeamID+"/suspend", admin.Token, map[string]any{"reason": "phishing campaign"}), http.StatusOK, "suspend")
	r := k.Do("POST", "/api/v1/links", cust.Token, map[string]any{"target_url": "https://example.com"})
	k.Expect(r, http.StatusForbidden, "write while suspended")
	if r.Map()["code"] != "workspace_suspended" {
		t.Fatalf("code: %s", r.Body)
	}
	page := k.Do("GET", "/s/cust-link", "", nil, "Accept", "text/html")
	k.Expect(page, http.StatusGone, "suspended link")
	if !strings.Contains(string(page.Body), "Disabled") {
		t.Fatalf("expected disabled page")
	}
	if b := k.Do("GET", "/api/v1/billing", cust.Token, nil).Map(); b["suspended"] != true || b["read_only"] != true {
		t.Fatalf("billing summary: %v", b)
	}
	k.Expect(k.Do("POST", "/api/v1/admin/workspaces/"+cust.TeamID+"/unsuspend", admin.Token, nil), http.StatusOK, "unsuspend")
	k.Expect(k.Do("GET", "/s/cust-link", "", nil), http.StatusFound, "link works again")

	detail := k.Do("GET", "/api/v1/admin/workspaces/"+cust.TeamID, admin.Token, nil).Map()
	if len(detail["members"].([]any)) != 1 {
		t.Fatalf("detail: %v", detail)
	}

	// Users: suspension ends sessions and blocks sign-in and API keys
	key := k.Do("POST", "/api/v1/api-keys", cust.Token, map[string]any{"name": "k"}).Map()["api_key"].(map[string]any)["key"].(string)
	users := k.Do("GET", "/api/v1/admin/users?q=customer@acme", admin.Token, nil).Map()["users"].([]any)
	uid := users[0].(map[string]any)["id"].(string)
	k.Expect(k.Do("POST", "/api/v1/admin/users/"+uid+"/suspend", admin.Token, map[string]any{"reason": "abuse"}), http.StatusOK, "suspend user")
	k.Expect(k.Do("GET", "/api/v1/links", cust.Token, nil), http.StatusUnauthorized, "session ended")
	k.Expect(k.Do("GET", "/api/v1/links", key, nil), http.StatusUnauthorized, "api key blocked")
	if _, _, err := k.Svc.StartSession(&models.User{ID: uid, Email: "customer@acme.test"}, "1.1.1.1", "x"); err == nil {
		t.Fatal("suspended user signed in")
	}
	k.Expect(k.Do("POST", "/api/v1/admin/users/"+uid+"/unsuspend", admin.Token, nil), http.StatusOK, "unsuspend user")
	k.Expect(k.Do("GET", "/api/v1/links", key, nil), http.StatusOK, "api key works again")

	// Admins can't lock themselves out
	adminID := k.Do("GET", "/api/v1/auth/me", admin.Token, nil).Map()["user"].(map[string]any)["id"].(string)
	k.Expect(k.Do("POST", "/api/v1/admin/users/"+adminID+"/suspend", admin.Token, map[string]any{"reason": "x"}), http.StatusForbidden, "self suspend")
	k.Expect(k.Do("POST", "/api/v1/admin/users/"+adminID+"/admin", admin.Token, map[string]any{"admin": false}), http.StatusForbidden, "self revoke")
	k.Expect(k.Do("POST", "/api/v1/admin/users/"+uid+"/admin", admin.Token, map[string]any{"admin": true}), http.StatusOK, "grant")

	m := k.Do("GET", "/api/v1/admin/metrics?days=7", admin.Token, nil).Map()
	if m["workspaces"].(float64) != 2 || m["users"].(float64) != 2 || len(m["series"].(map[string]any)["signups"].([]any)) == 0 {
		t.Fatalf("metrics: %v", m)
	}
}

func TestAdminCodes(t *testing.T) {
	k := testkit.New(t)
	k.EnableBilling()
	admin := k.AdminLogin("admin@lynqr.test")
	owner := k.Login("owner@acme.test")

	k.Expect(k.Do("POST", "/api/v1/admin/codes/checkout", admin.Token, map[string]any{"code": "x"}), http.StatusBadRequest, "bad code")
	k.Expect(k.Do("POST", "/api/v1/admin/codes/checkout", admin.Token, map[string]any{"code": "SPRING25", "percent_off": 25, "duration": "repeating"}), http.StatusBadRequest, "months required")
	c := k.Do("POST", "/api/v1/admin/codes/checkout", admin.Token, map[string]any{"code": "spring25", "percent_off": 25, "duration": "repeating", "duration_in_months": 3})
	k.Expect(c, http.StatusCreated, "create checkout code")
	promoID := c.Map()["id"].(string)
	if p := k.Do("POST", "/api/v1/billing/promo", owner.Token, map[string]any{"code": "SPRING25", "interval": "month"}).Map(); p["kind"] != "discount" || p["description"] != "25% off for 3 months" {
		t.Fatalf("customer preview: %v", p)
	}
	k.Expect(k.Do("POST", "/api/v1/admin/codes/checkout/"+promoID, admin.Token, map[string]any{"active": false, "code": "SPRING25"}), http.StatusOK, "deactivate")
	k.Expect(k.Do("POST", "/api/v1/billing/promo", owner.Token, map[string]any{"code": "SPRING25"}), http.StatusBadRequest, "deactivated code")

	// Trial codes
	k.Expect(k.Do("POST", "/api/v1/admin/codes/trial", admin.Token, map[string]any{"code": "EXTRA", "days": 500}), http.StatusBadRequest, "too many days")
	k.Expect(k.Do("POST", "/api/v1/admin/codes/trial", admin.Token, map[string]any{"code": "extra14", "days": 14, "max_redemptions": 1}), http.StatusCreated, "trial code")
	k.Expect(k.Do("POST", "/api/v1/admin/codes/trial", admin.Token, map[string]any{"code": "EXTRA14", "days": 7}), http.StatusConflict, "duplicate")
	p := k.Do("POST", "/api/v1/billing/promo", owner.Token, map[string]any{"code": "extra14"}).Map()
	if p["kind"] != "trial" || p["trial_days"].(float64) != 14 {
		t.Fatalf("trial preview: %v", p)
	}
	b := k.Do("POST", "/api/v1/billing/redeem", owner.Token, map[string]any{"code": "Extra14"})
	k.Expect(b, http.StatusOK, "redeem")
	if b.Map()["trial_days_left"].(float64) != 21 {
		t.Fatalf("trial not extended: %s", b.Body)
	}
	k.Expect(k.Do("POST", "/api/v1/billing/redeem", owner.Token, map[string]any{"code": "EXTRA14"}), http.StatusBadRequest, "max redemptions reached")
	other := k.Login("other@acme.test")
	k.Expect(k.Do("POST", "/api/v1/billing/redeem", other.Token, map[string]any{"code": "EXTRA14"}), http.StatusBadRequest, "exhausted")

	codes := k.Do("GET", "/api/v1/admin/codes", admin.Token, nil).Map()
	trial := codes["trial"].([]any)[0].(map[string]any)
	if trial["redemptions"].(float64) != 1 || codes["billing_enabled"] != true {
		t.Fatalf("codes: %v", codes)
	}
}

func TestModerationBlocklistAndReports(t *testing.T) {
	k := testkit.New(t)
	admin := k.AdminLogin("mod@lynqr.test")
	u := k.Login("maker@acme.test")
	l := createLink(t, k, u.Token, map[string]any{"target_url": "https://evil-login.example/verify", "custom_slug": "bank-verify"})
	q := createQR(t, k, u.Token, map[string]any{"type": "website", "title": "Flyer", "target_url": "https://shop.example/", "is_dynamic": true})

	// Visitors report it
	k.Expect(k.Do("POST", "/api/v1/report", "", map[string]any{"url": "https://lynqr.app/s/nope", "reason": "phishing"}), http.StatusBadRequest, "unknown link")
	k.Expect(k.Do("POST", "/api/v1/report", "", map[string]any{"url": "https://lynqr.app/s/bank-verify", "reason": "bogus"}), http.StatusBadRequest, "bad reason")
	k.Expect(k.Do("POST", "/api/v1/report", "", map[string]any{"url": "lynqr.app/s/bank-verify", "reason": "phishing", "details": "asks for my bank login"}), http.StatusCreated, "report")
	form := url.Values{"url": {"https://lynqr.app/q/" + q.Code}, "reason": {"spam"}}
	req := newForm("/report", form)
	page := k.Send(req)
	k.Expect(page, http.StatusOK, "form report")
	if !strings.Contains(string(page.Body), "Thanks") {
		t.Fatal("no confirmation")
	}
	k.Expect(k.Do("GET", "/report?u=https://lynqr.app/s/x", "", nil), http.StatusOK, "report page")

	reports := k.Do("GET", "/api/v1/admin/reports", admin.Token, nil).Map()["reports"].([]any)
	if len(reports) != 2 {
		t.Fatalf("reports: %v", reports)
	}
	var linkReport string
	for _, r := range reports {
		if r.(map[string]any)["target_type"] == "link" {
			linkReport = r.(map[string]any)["id"].(string)
		}
	}
	k.Expect(k.Do("POST", "/api/v1/admin/reports/"+linkReport, admin.Token, map[string]any{"action": "disable"}), http.StatusOK, "disable via report")
	k.Expect(k.Do("GET", "/s/bank-verify", "", nil), http.StatusGone, "disabled link")
	if got := k.Do("GET", "/api/v1/links/"+l.ID, u.Token, nil).Map(); got["moderation_reason"] != "Reported for phishing" {
		t.Fatalf("owner should see moderation: %v", got)
	}
	// Owners can't undo moderation by editing
	k.Do("PUT", "/api/v1/links/"+l.ID, u.Token, map[string]any{"title": "renamed"})
	k.Expect(k.Do("GET", "/s/bank-verify", "", nil), http.StatusGone, "still disabled")
	k.Expect(k.Do("POST", "/api/v1/admin/moderation/link/"+l.ID+"/restore", admin.Token, nil), http.StatusOK, "restore")
	k.Expect(k.Do("GET", "/s/bank-verify", "", nil), http.StatusFound, "restored")

	// Search finds the dynamic QR code's short link; the QR is found by its code
	assets := k.Do("GET", "/api/v1/admin/moderation/search?q=shop.example", admin.Token, nil).Map()["assets"].([]any)
	if len(assets) != 1 || assets[0].(map[string]any)["type"] != "link" {
		t.Fatalf("search: %v", assets)
	}
	assets = k.Do("GET", "/api/v1/admin/moderation/search?q="+q.Code, admin.Token, nil).Map()["assets"].([]any)
	if len(assets) != 1 || assets[0].(map[string]any)["type"] != "qr" || !strings.HasSuffix(assets[0].(map[string]any)["public_url"].(string), "/q/"+q.Code) {
		t.Fatalf("search: %v", assets)
	}
	k.Expect(k.Do("POST", "/api/v1/admin/moderation/qr/"+q.ID+"/disable", admin.Token, map[string]any{}), http.StatusBadRequest, "reason required")
	k.Expect(k.Do("POST", "/api/v1/admin/moderation/qr/"+q.ID+"/disable", admin.Token, map[string]any{"reason": "spam"}), http.StatusOK, "disable qr")
	k.Expect(k.Do("GET", "/q/"+q.Code, "", nil), http.StatusGone, "disabled qr")
	k.Expect(k.Do("POST", "/api/v1/admin/moderation/qr/"+q.ID+"/restore", admin.Token, nil), http.StatusOK, "restore qr")

	// Blocklist stops new links and existing redirects to the domain and its subdomains
	k.Expect(k.Do("POST", "/api/v1/admin/blocklist", admin.Token, map[string]any{"domain": "https://Evil-Login.example/path", "reason": "phishing kit"}), http.StatusCreated, "block")
	k.Expect(k.Do("POST", "/api/v1/admin/blocklist", admin.Token, map[string]any{"domain": "evil-login.example"}), http.StatusConflict, "duplicate block")
	r := k.Do("POST", "/api/v1/links", u.Token, map[string]any{"target_url": "https://secure.evil-login.example/"})
	k.Expect(r, http.StatusBadRequest, "blocked create")
	if r.Map()["code"] != "domain_blocked" {
		t.Fatalf("code: %s", r.Body)
	}
	k.Expect(k.Do("POST", "/api/v1/links", u.Token, map[string]any{"target_url": "https://notevil-login.example/"}), http.StatusCreated, "lookalike suffix allowed")
	k.Expect(k.Do("GET", "/s/bank-verify", "", nil), http.StatusGone, "existing link to blocked domain")
	k.Expect(k.Do("DELETE", "/api/v1/admin/blocklist/evil-login.example", admin.Token, nil), http.StatusNoContent, "unblock")
	k.Expect(k.Do("GET", "/s/bank-verify", "", nil), http.StatusFound, "unblocked")

	ev := string(k.Do("GET", "/api/v1/admin/audit?q=domain", admin.Token, nil).Body)
	if !strings.Contains(ev, "domain.block") || !strings.Contains(ev, "domain.unblock") {
		t.Fatalf("audit: %s", ev)
	}
}
