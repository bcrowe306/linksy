package e2e_test

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/bcrowe306/linksy/internal/models"
	"github.com/bcrowe306/linksy/internal/service"
	"github.com/bcrowe306/linksy/test/testkit"
)

func TestAccountAndTeamEmails(t *testing.T) {
	k := testkit.New(t)
	mail := k.UseMailbox()
	owner := k.Login("owner@acme.test")
	k.Svc.FlushMail()
	if _, ok := mail.LastTagged("owner@acme.test", "welcome"); !ok {
		t.Fatal("no welcome email")
	}

	// Same browser again: no alert. New browser: alert.
	k.Login("owner@acme.test")
	if _, _, err := k.Svc.StartSession(&models.User{ID: owner.ID, Email: owner.Email}, "203.0.113.9", "Mozilla/5.0 (Macintosh; Intel Mac OS X 14_0) Chrome/126"); err != nil {
		t.Fatal(err)
	}
	k.Svc.FlushMail()
	if n := mail.Count("owner@acme.test", "security_signin"); n != 1 {
		t.Fatalf("expected 1 new sign-in alert, got %d", n)
	}
	m, _ := mail.LastTagged("owner@acme.test", "security_signin")
	if !strings.Contains(m.Text, "203.0.113.9") || !strings.Contains(m.Text, "Chrome") || m.HTML == "" {
		t.Fatalf("alert body: %s", m.Text)
	}

	// Invitation email carries the accept link; accepting notifies the inviter
	inv := k.Do("POST", "/api/v1/team/members/invite", owner.Token, map[string]any{"email": "new@acme.test", "role": "member"}).Map()["invitation"].(map[string]any)
	k.Svc.FlushMail()
	invMail, ok := mail.LastTagged("new@acme.test", "invitation")
	if !ok || !strings.Contains(invMail.Text, inv["accept_url"].(string)) || !strings.Contains(invMail.Subject, "invited you") {
		t.Fatalf("invitation email: %+v", invMail)
	}
	member := k.Login("new@acme.test")
	token := inv["accept_url"].(string)[strings.LastIndex(inv["accept_url"].(string), "/")+1:]
	k.Expect(k.Do("POST", "/api/v1/invitations/"+token+"/accept", member.Token, nil), http.StatusOK, "accept")
	k.Svc.FlushMail()
	if _, ok := mail.LastTagged("owner@acme.test", "team_joined"); !ok {
		t.Fatal("inviter not told about the new member")
	}

	// Role change email, and its signed unsubscribe link turns team activity off
	k.Expect(k.Do("PUT", "/api/v1/team/members/"+member.ID+"/role", owner.Token, map[string]any{"role": "viewer"}), http.StatusOK, "role")
	k.Svc.FlushMail()
	role, ok := mail.LastTagged("new@acme.test", "team_role")
	if !ok || role.Unsubscribe == "" {
		t.Fatalf("role email: %+v", role)
	}
	u, _ := url.Parse(role.Unsubscribe)
	bad := strings.Replace(u.RequestURI(), "&s=", "&s=00", 1)
	k.Expect(k.Do("GET", bad, "", nil), http.StatusBadRequest, "tampered unsubscribe")
	k.Expect(k.Do("POST", u.RequestURI(), "", nil), http.StatusOK, "one-click unsubscribe")
	prefs := k.Do("GET", "/api/v1/user/preferences", member.Token, nil).Map()
	if prefs["team_activity"] != false {
		t.Fatalf("preference not updated: %v", prefs)
	}
	k.Expect(k.Do("PUT", "/api/v1/team/members/"+member.ID+"/role", owner.Token, map[string]any{"role": "member"}), http.StatusOK, "role again")
	k.Svc.FlushMail()
	if n := mail.Count("new@acme.test", "team_role"); n != 1 {
		t.Fatalf("unsubscribed user still emailed: %d", n)
	}
}

func TestTrialReminderAndBillingEmails(t *testing.T) {
	k := testkit.New(t)
	f := k.EnableBilling()
	mail := k.UseMailbox()
	u := k.Login("trialer@acme.test")
	now := time.Now().UTC()

	// 2 days left → the 3-day reminder, once
	if err := k.Repo.SetTeamTrialEnd(u.TeamID, now.Add(48*time.Hour)); err != nil {
		t.Fatal(err)
	}
	k.Svc.RunScheduledEmails(now)
	k.Svc.RunScheduledEmails(now.Add(time.Minute))
	k.Svc.FlushMail()
	if n := mail.Count(u.Email, "trial_3d"); n != 1 {
		t.Fatalf("expected one 3-day reminder, got %d", n)
	}
	k.Svc.RunScheduledEmails(now.Add(30 * time.Hour)) // < 24h left
	k.Svc.RunScheduledEmails(now.Add(50 * time.Hour)) // ended
	k.Svc.FlushMail()
	if mail.Count(u.Email, "trial_1d") != 1 || mail.Count(u.Email, "trial_ended") != 1 {
		t.Fatalf("1-day/ended reminders: %d %d", mail.Count(u.Email, "trial_1d"), mail.Count(u.Email, "trial_ended"))
	}
	ended, _ := mail.LastTagged(u.Email, "trial_ended")
	if !strings.Contains(ended.Text, "/settings?tab=billing") {
		t.Fatalf("ended email: %s", ended.Text)
	}

	// Payment failure then cancellation, via Stripe webhooks
	f.Subscriptions["sub_9"] = testkit.Sub{ID: "sub_9", Customer: "cus_9", Status: "past_due", TeamID: u.TeamID}.Build()
	k.Expect(webhook(k, "evt_9", "invoice.payment_failed", map[string]any{"subscription": "sub_9", "customer": "cus_9"}, testkit.WebhookSecret), http.StatusOK, "past due")
	k.Svc.FlushMail()
	if _, ok := mail.LastTagged(u.Email, "billing_past_due"); !ok {
		t.Fatal("no payment-failed email")
	}
	k.ExpireTrial(u.TeamID) // otherwise a canceled subscription falls back to the remaining trial
	f.Subscriptions["sub_9"].Status = "canceled"
	k.Expect(webhook(k, "evt_10", "customer.subscription.deleted", map[string]any{"id": "sub_9", "customer": "cus_9"}, testkit.WebhookSecret), http.StatusOK, "canceled")
	k.Svc.FlushMail()
	if _, ok := mail.LastTagged(u.Email, "billing_canceled"); !ok {
		t.Fatal("no subscription-ended email")
	}
}

func TestWeeklyReport(t *testing.T) {
	k := testkit.New(t)
	mail := k.UseMailbox()
	u := k.Login("weekly@acme.test")
	k.Expect(k.Do("PUT", "/api/v1/user/preferences", u.Token, map[string]any{"weekly_report": true, "security_alerts": true}), http.StatusOK, "opt in")
	createLink(t, k, u.Token, map[string]any{"target_url": "https://example.com", "custom_slug": "weekly-link"})
	monday := time.Now().UTC()
	for monday.Weekday() != time.Monday {
		monday = monday.AddDate(0, 0, 1)
	}
	monday = time.Date(monday.Year(), monday.Month(), monday.Day(), 14, 0, 0, 0, time.UTC)

	k.Svc.RunScheduledEmails(monday)
	k.Svc.FlushMail()
	if mail.Count(u.Email, "weekly_report") != 0 {
		t.Fatal("empty week should not send a report")
	}
	// A click during the reported week
	k.Svc.TrackEvent(&models.AnalyticsEvent{TeamID: u.TeamID, EventType: "click", UserAgent: "Mozilla/5.0 (Windows NT 10.0) Chrome/126", CreatedAt: monday.Add(-48 * time.Hour), Country: "US"})
	k.Svc.RunScheduledEmails(monday)
	k.Svc.RunScheduledEmails(monday.Add(time.Hour))
	k.Svc.FlushMail()
	if n := mail.Count(u.Email, "weekly_report"); n != 1 {
		t.Fatalf("expected one weekly report, got %d", n)
	}
	r, _ := mail.LastTagged(u.Email, "weekly_report")
	if !strings.Contains(r.Text, "Link clicks: 1") || r.Unsubscribe == "" {
		t.Fatalf("report: %s", r.Text)
	}
}

func TestTestSuiteAddressesAreNeverEmailed(t *testing.T) {
	k := testkit.New(t)
	mail := k.UseMailbox()
	if err := k.Svc.SuppressEmailsMatching(service.DefaultEmailSuppression); err != nil {
		t.Fatal(err)
	}
	k.Login("testdrive_123_abcd@lynqr.app")
	k.Login("real.person@lynqr.app")
	k.Svc.FlushMail()
	if mail.Count("testdrive_123_abcd@lynqr.app", "welcome") != 0 || mail.Count("real.person@lynqr.app", "welcome") != 1 {
		t.Fatalf("suppression wrong: %+v", mail.Sent)
	}
}
