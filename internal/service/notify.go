package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/bcrowe306/linksy/internal/auth"
	"github.com/bcrowe306/linksy/internal/billing"
	"github.com/bcrowe306/linksy/internal/email"
	"github.com/bcrowe306/linksy/internal/models"
)

// Notification preference keys (see models.Preferences).
const (
	prefSecurity = "security_alerts"
	prefTeam     = "team_activity"
	prefWeekly   = "weekly_report"
	prefUsage    = "usage_alerts"
	prefProduct  = "product_updates"
)

var unsubscribable = map[string]string{
	prefTeam: "team activity", prefWeekly: "the weekly report", prefUsage: "trial and billing alerts", prefProduct: "product updates",
}

var ErrBadUnsubscribe = errors.New("this unsubscribe link is invalid")

type mailer struct {
	wg       sync.WaitGroup
	suppress *regexp.Regexp
}

// DefaultEmailSuppression matches the throwaway addresses the automated test
// suites sign up with, so they never bounce and hurt sender reputation.
const DefaultEmailSuppression = `(?i)^(qa_|testdrive_|api_test_|ui_probe_)[^@]*@lynqr\.app$`

// SuppressEmailsMatching skips recipients matching the pattern ("" disables).
func (s *Service) SuppressEmailsMatching(pattern string) error {
	if pattern == "" {
		s.mail.suppress = nil
		return nil
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return err
	}
	s.mail.suppress = re
	return nil
}

// FlushMail waits for queued emails (tests and graceful shutdown).
func (s *Service) FlushMail() { s.mail.wg.Wait() }

// deliver renders and sends an email in the background. Failures are logged,
// never surfaced to the action that triggered them.
func (s *Service) deliver(to, subject, tag string, c email.Content) {
	if s.admin.mailer == nil || to == "" || (s.mail.suppress != nil && s.mail.suppress.MatchString(to)) {
		return
	}
	htmlBody, text := c.Render()
	m := email.Message{To: to, Subject: subject, HTML: htmlBody, Text: text, Tag: tag, Unsubscribe: c.Unsubscribe}
	s.mail.wg.Add(1)
	go func() {
		defer s.mail.wg.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		if err := s.admin.mailer.Send(ctx, m); err != nil {
			log.Printf("[email] %s to %s failed: %v", tag, to, err)
		}
	}()
}

// wants reports whether a user opted in to a kind of email. Security emails are always sent.
func (s *Service) wants(userID, pref string) bool {
	if pref == prefSecurity {
		return true
	}
	p, err := s.GetPreferences(userID)
	if err != nil {
		return false
	}
	switch pref {
	case prefTeam:
		return p.TeamActivity
	case prefWeekly:
		return p.WeeklyReport
	case prefUsage:
		return p.UsageAlerts
	case prefProduct:
		return p.ProductUpdates
	}
	return false
}

func (s *Service) manageURL() string { return s.appURL + "/profile?tab=notifications" }

func (s *Service) unsubscribeSig(userID, pref string) string {
	mac := hmac.New(sha256.New, []byte(s.ipSalt+"|unsubscribe"))
	mac.Write([]byte(userID + "|" + pref))
	return hex.EncodeToString(mac.Sum(nil))[:32]
}

// UnsubscribeURL is a signed one-click link that turns off one kind of email.
func (s *Service) UnsubscribeURL(userID, pref string) string {
	return fmt.Sprintf("%s/email/unsubscribe?u=%s&k=%s&s=%s", s.appURL, userID, pref, s.unsubscribeSig(userID, pref))
}

// Unsubscribe turns off one kind of email for a user from a signed link.
// It returns a human description of what was turned off.
func (s *Service) Unsubscribe(userID, pref, sig string) (string, error) {
	label, ok := unsubscribable[pref]
	if !ok || !hmac.Equal([]byte(sig), []byte(s.unsubscribeSig(userID, pref))) {
		return "", ErrBadUnsubscribe
	}
	p, err := s.GetPreferences(userID)
	if err != nil {
		return "", ErrBadUnsubscribe
	}
	switch pref {
	case prefTeam:
		p.TeamActivity = false
	case prefWeekly:
		p.WeeklyReport = false
	case prefUsage:
		p.UsageAlerts = false
	case prefProduct:
		p.ProductUpdates = false
	}
	if _, err := s.UpdatePreferences(userID, p); err != nil {
		return "", err
	}
	return label, nil
}

// optional builds the footer for emails a user can turn off.
func (s *Service) optional(c email.Content, userID, pref string) email.Content {
	c.ManageURL = s.manageURL()
	c.Unsubscribe = s.UnsubscribeURL(userID, pref)
	return c
}

func displayName(u *models.User) string {
	if n := strings.TrimSpace(u.FirstName + " " + u.LastName); n != "" {
		return n
	}
	return u.Email
}

func firstName(u *models.User) string {
	if u.FirstName != "" {
		return u.FirstName
	}
	return "there"
}

// ----------------- Account & security -----------------

func (s *Service) notifyWelcome(u *models.User) {
	s.deliver(u.Email, "Welcome to Lynqr", "welcome", email.Content{
		Preheader: "Your 7-day free trial has started. Here's how to get going.",
		Heading:   "Welcome to Lynqr, " + firstName(u),
		Paragraphs: []string{
			"Your workspace is ready and your 7-day free trial has started. No card needed.",
			"Create a short link or design a dynamic QR code in QR Studio, and every click and scan will show up on your dashboard as it happens.",
		},
		Button:   &email.Button{Label: "Open your dashboard", URL: s.appURL + "/"},
		Footnote: "Need a hand? Press ? in the app for guides and shortcuts, or reply to support@lynqr.app.",
		Reason:   "You received this because you created a Lynqr account.",
	})
}

// notifyNewSignIn warns about a sign-in from a browser the user hasn't used recently.
func (s *Service) notifyNewSignIn(u *models.User, ip, userAgent string, seen []models.Session) {
	for _, prev := range seen {
		if prev.UserAgent == userAgent {
			return
		}
	}
	_, browser, os := parseUA(userAgent)
	s.deliver(u.Email, "New sign-in to your Lynqr account", "security_signin", email.Content{
		Preheader:  "A new browser signed in to your account.",
		Heading:    "New sign-in to your account",
		Paragraphs: []string{"Your Lynqr account was just used to sign in from a browser we haven't seen recently."},
		Details: [][2]string{
			{"Browser", strings.TrimSpace(browser + " on " + os)},
			{"IP address", ip},
			{"Time", time.Now().UTC().Format("Jan 2, 2006 15:04 UTC")},
		},
		Button:    &email.Button{Label: "Review your sessions", URL: s.appURL + "/profile?tab=security"},
		Footnote:  "If this was you, there's nothing to do. If not, change your password and sign out of other sessions right away.",
		Reason:    "Security alerts are always sent to protect your account.",
		ManageURL: s.manageURL(),
	})
}

// NotifyPasswordChanged confirms a password change to the account owner.
func (s *Service) NotifyPasswordChanged(userID, ip string) {
	u, err := s.repo.GetUserByID(userID)
	if err != nil {
		return
	}
	s.deliver(u.Email, "Your Lynqr password was changed", "security_password", email.Content{
		Heading:    "Your password was changed",
		Paragraphs: []string{"The password for your Lynqr account was just changed, and your other sessions were signed out."},
		Details:    [][2]string{{"IP address", ip}, {"Time", time.Now().UTC().Format("Jan 2, 2006 15:04 UTC")}},
		Footnote:   "If you didn't do this, reset your password from the sign-in page now and contact support@lynqr.app.",
		Button:     &email.Button{Label: "Review your sessions", URL: s.appURL + "/profile?tab=security"},
		Reason:     "Security alerts are always sent to protect your account.",
		ManageURL:  s.manageURL(),
	})
}

// ----------------- Team activity -----------------

func (s *Service) notifyInvitation(inv *models.Invitation, inviterID string) {
	inviter := "A teammate"
	if u, err := s.repo.GetUserByID(inviterID); err == nil {
		inviter = displayName(u)
	}
	s.deliver(inv.Email, fmt.Sprintf("%s invited you to %s on Lynqr", inviter, inv.TeamName), "invitation", email.Content{
		Preheader: fmt.Sprintf("Join %s on Lynqr as %s.", inv.TeamName, inv.Role),
		Heading:   fmt.Sprintf("Join %s on Lynqr", inv.TeamName),
		Paragraphs: []string{
			fmt.Sprintf("%s invited you to the %s workspace as %s %s. Lynqr is where the team manages its short links, QR codes and analytics.", inviter, inv.TeamName, article(string(inv.Role)), inv.Role),
		},
		Button:   &email.Button{Label: "Accept invitation", URL: inv.AcceptURL},
		Footnote: "This invitation expires in 7 days. Sign in or create an account with " + inv.Email + " to accept it.",
		Reason:   fmt.Sprintf("You received this because %s invited %s. If you weren't expecting it, you can ignore this email.", inviter, inv.Email),
	})
}

func article(word string) string {
	if word != "" && strings.ContainsRune("aeiou", rune(word[0])) {
		return "an"
	}
	return "a"
}

func (s *Service) notifyInviteAccepted(inv *models.Invitation, member *models.User) {
	inviter, err := s.repo.GetUserByID(inv.InvitedBy)
	if err != nil || !s.wants(inviter.ID, prefTeam) {
		return
	}
	team, _ := s.repo.GetTeam(inv.TeamID)
	teamName := inv.TeamName
	if team != nil {
		teamName = team.Name
	}
	s.deliver(inviter.Email, fmt.Sprintf("%s joined %s", displayName(member), teamName), "team_joined", s.optional(email.Content{
		Heading:    fmt.Sprintf("%s joined %s", displayName(member), teamName),
		Paragraphs: []string{fmt.Sprintf("%s accepted your invitation and joined as %s %s.", member.Email, article(string(inv.Role)), inv.Role)},
		Button:     &email.Button{Label: "View members", URL: s.appURL + "/settings?tab=members"},
		Reason:     "You received this because you invited them.",
	}, inviter.ID, prefTeam))
}

func (s *Service) notifyRoleChanged(a Actor, userID string, role models.UserRole) {
	u, err := s.repo.GetUserByID(userID)
	if err != nil || !s.wants(userID, prefTeam) {
		return
	}
	team, err := s.repo.GetTeam(a.TeamID)
	if err != nil {
		return
	}
	by := "An admin"
	if actor, err := s.repo.GetUserByID(a.UserID); err == nil {
		by = displayName(actor)
	}
	s.deliver(u.Email, fmt.Sprintf("Your role in %s changed", team.Name), "team_role", s.optional(email.Content{
		Heading:    fmt.Sprintf("You're now %s %s in %s", article(string(role)), role, team.Name),
		Paragraphs: []string{fmt.Sprintf("%s changed your role in the %s workspace to %s.", by, team.Name, role)},
		Button:     &email.Button{Label: "Open Lynqr", URL: s.appURL + "/"},
		Reason:     "You received this because you're a member of " + team.Name + ".",
	}, userID, prefTeam))
}

// ----------------- Billing -----------------

func (s *Service) teamOwner(teamID string) (*models.Team, *models.User, bool) {
	t, err := s.repo.GetTeam(teamID)
	if err != nil {
		return nil, nil, false
	}
	u, err := s.repo.GetUserByID(t.OwnerID)
	if err != nil {
		return nil, nil, false
	}
	return t, u, true
}

// notifyBillingChange emails the owner when the plan state changes meaningfully.
func (s *Service) notifyBillingChange(teamID, before, after string) {
	if before == after {
		return
	}
	t, owner, ok := s.teamOwner(teamID)
	if !ok || !s.wants(owner.ID, prefUsage) {
		return
	}
	billingURL := s.appURL + "/settings?tab=billing"
	var subject string
	var c email.Content
	switch after {
	case models.BillingActive, models.BillingTrialing:
		if before == models.BillingActive || before == models.BillingTrialing {
			return
		}
		subject = "You're subscribed to Lynqr Pro"
		c = email.Content{Heading: "Thanks for subscribing", Paragraphs: []string{
			fmt.Sprintf("%s is now on Lynqr Pro. Your links and QR codes keep working and your whole team can keep creating.", t.Name),
			"Stripe emails your receipts. You can change your plan or payment method any time from Billing.",
		}, Button: &email.Button{Label: "Manage billing", URL: billingURL}}
	case models.BillingPastDue:
		subject = "Action needed: your Lynqr payment failed"
		c = email.Content{Preheader: "Update your payment method to keep your links and QR codes working.", Heading: "Your payment didn't go through",
			Paragraphs: []string{fmt.Sprintf("We couldn't charge the card on file for %s. Stripe will retry, but please update your payment method so your links and QR codes keep working.", t.Name)},
			Button:     &email.Button{Label: "Update payment method", URL: billingURL}}
	case models.BillingCanceled:
		subject = "Your Lynqr subscription has ended"
		c = email.Content{Heading: "Your subscription has ended", Paragraphs: []string{
			fmt.Sprintf("The subscription for %s has ended. The workspace is now read-only and its links and QR codes are paused.", t.Name),
			"Nothing has been deleted. Subscribe again and everything starts working right away.",
		}, Button: &email.Button{Label: "Reactivate Lynqr Pro", URL: billingURL}}
	default:
		return
	}
	c.Reason = "You received this because you own the " + t.Name + " workspace."
	s.deliver(owner.Email, subject, "billing_"+after, s.optional(c, owner.ID, prefUsage))
}

// ----------------- Scheduled emails -----------------

// StartScheduler sends trial reminders and weekly reports in the background.
func (s *Service) StartScheduler(ctx context.Context) {
	go func() {
		tick := time.NewTicker(15 * time.Minute)
		defer tick.Stop()
		for {
			s.RunScheduledEmails(time.Now().UTC())
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
			}
		}
	}()
}

// RunScheduledEmails sends any due trial reminders and weekly reports. Each
// email is recorded so it goes out once, even across restarts.
func (s *Service) RunScheduledEmails(now time.Time) {
	if s.admin.mailer == nil {
		return
	}
	if s.billing != nil {
		s.sendTrialReminders(now)
	}
	if now.Weekday() == time.Monday && now.Hour() >= 13 {
		s.sendWeeklyReports(now)
	}
}

func (s *Service) sendTrialReminders(now time.Time) {
	teams, err := s.repo.AllTeamBilling()
	if err != nil {
		log.Printf("[email] trial reminders: %v", err)
		return
	}
	for _, t := range teams {
		end := t.Billing.TrialEndsAt
		if end == nil {
			continue
		}
		state := t.Billing.State(now)
		left := end.Sub(now)
		var kind string
		switch {
		case state == models.BillingTrial && left <= 24*time.Hour:
			kind = "trial_1d"
		case state == models.BillingTrial && left <= 3*24*time.Hour:
			kind = "trial_3d"
		case state == models.BillingExpired && now.Sub(*end) < 3*24*time.Hour:
			kind = "trial_ended"
		default:
			continue
		}
		// Keyed by the trial end, so an extended trial gets its own reminders.
		ref := t.ID + "|" + end.UTC().Format("2006-01-02")
		owner, err := s.repo.GetUserByID(t.OwnerID)
		if err != nil || !s.wants(owner.ID, prefUsage) {
			continue
		}
		if sent, err := s.repo.MarkEmailSent(kind, ref); err != nil || !sent {
			continue
		}
		s.deliver(owner.Email, trialSubject(kind), kind, s.optional(trialContent(kind, t.Name, s.appURL), owner.ID, prefUsage))
	}
}

func trialSubject(kind string) string {
	switch kind {
	case "trial_1d":
		return "Your Lynqr trial ends tomorrow"
	case "trial_3d":
		return "3 days left in your Lynqr trial"
	}
	return "Your Lynqr trial has ended"
}

func trialContent(kind, team, appURL string) email.Content {
	price := fmt.Sprintf("$%d/month billed annually or $%.2f monthly", billing.AnnualCents/1200, float64(billing.MonthlyCents)/100)
	c := email.Content{Button: &email.Button{Label: "Choose a plan", URL: appURL + "/settings?tab=billing"},
		Reason: "You received this because you own the " + team + " workspace."}
	switch kind {
	case "trial_ended":
		c.Heading = "Your free trial has ended"
		c.Paragraphs = []string{
			fmt.Sprintf("The %s workspace is now read-only and its links and QR codes are paused.", team),
			"Nothing has been deleted. Subscribe to Lynqr Pro (" + price + ") and everything starts working again right away.",
		}
	default:
		when := "in 3 days"
		if kind == "trial_1d" {
			when = "tomorrow"
		}
		c.Heading = "Your free trial ends " + when
		c.Preheader = "Subscribe to keep your links and QR codes working."
		c.Paragraphs = []string{
			fmt.Sprintf("Your Lynqr Pro trial for %s ends %s. After that, the workspace becomes read-only and printed QR codes stop redirecting.", team, when),
			"Subscribe now to keep everything running: " + price + ". Any trial days you have left carry over.",
		}
	}
	return c
}

func (s *Service) sendWeeklyReports(now time.Time) {
	users, err := s.repo.UsersWithPreference(prefWeekly)
	if err != nil {
		log.Printf("[email] weekly reports: %v", err)
		return
	}
	year, week := now.ISOWeek()
	to := now.UTC().Truncate(24 * time.Hour)
	from := to.AddDate(0, 0, -7)
	for _, u := range users {
		teams, err := s.repo.ListUserTeams(u.ID)
		if err != nil {
			continue
		}
		for _, m := range teams {
			if m.Status != models.MemberActive || !auth.Can(m.Role, auth.PermViewAnalytics) {
				continue
			}
			ref := fmt.Sprintf("%s|%s|%d-%02d", u.ID, m.ID, year, week)
			r, err := s.Report(Actor{UserID: u.ID, TeamID: m.ID, Role: m.Role}, ReportQuery{From: from, To: to, TZ: time.UTC})
			if err != nil || (r.Totals.Total == 0 && r.Previous.Total == 0) {
				continue
			}
			if sent, err := s.repo.MarkEmailSent("weekly_report", ref); err != nil || !sent {
				continue
			}
			s.deliver(u.Email, fmt.Sprintf("Your week on Lynqr: %d clicks and scans", r.Totals.Total), "weekly_report",
				s.optional(weeklyContent(r, m.Name, s.appURL), u.ID, prefWeekly))
		}
	}
}

func change(cur, prev int64) string {
	switch {
	case prev == 0 && cur == 0:
		return "no change"
	case prev == 0:
		return "new this week"
	}
	pct := float64(cur-prev) * 100 / float64(prev)
	if pct >= 0 {
		return fmt.Sprintf("up %.0f%%", pct)
	}
	return fmt.Sprintf("down %.0f%%", -pct)
}

func weeklyContent(r *Report, team, appURL string) email.Content {
	details := [][2]string{
		{"Link clicks", fmt.Sprintf("%d (%s)", r.Totals.Clicks, change(r.Totals.Clicks, r.Previous.Clicks))},
		{"QR scans", fmt.Sprintf("%d (%s)", r.Totals.Scans, change(r.Totals.Scans, r.Previous.Scans))},
		{"Unique visitors", fmt.Sprintf("%d", r.Totals.Visitors)},
	}
	if len(r.TopLinks) > 0 {
		details = append(details, [2]string{"Top link", fmt.Sprintf("%s · %d clicks", firstNonEmpty(r.TopLinks[0].Title, r.TopLinks[0].Label), r.TopLinks[0].Count)})
	}
	if len(r.TopQRCodes) > 0 {
		details = append(details, [2]string{"Top QR code", fmt.Sprintf("%s · %d scans", firstNonEmpty(r.TopQRCodes[0].Title, r.TopQRCodes[0].Label), r.TopQRCodes[0].Count)})
	}
	if len(r.Countries) > 0 {
		details = append(details, [2]string{"Top country", r.Countries[0].Name})
	}
	return email.Content{
		Preheader:  fmt.Sprintf("%d clicks and scans in %s last week.", r.Totals.Total, team),
		Heading:    "Your weekly report for " + team,
		Paragraphs: []string{fmt.Sprintf("Here's how %s performed from %s to %s, compared with the week before.", team, r.From.Format("Jan 2"), r.To.AddDate(0, 0, -1).Format("Jan 2"))},
		Details:    details,
		Button:     &email.Button{Label: "Open full analytics", URL: appURL + "/analytics?range=7d"},
		Reason:     "You received this because you turned on the weekly report.",
	}
}

func firstNonEmpty(v ...string) string {
	for _, s := range v {
		if s != "" {
			return s
		}
	}
	return ""
}

// ----------------- Staff -----------------

func (s *Service) notifyAbuseReport(r *models.AbuseReport) {
	admins, err := s.repo.ListPlatformAdmins()
	if err != nil {
		return
	}
	for _, a := range admins {
		s.deliver(a.Email, "New abuse report: "+r.Reason, "abuse_report", email.Content{
			Heading:    "New abuse report",
			Paragraphs: []string{"A visitor reported a Lynqr " + map[string]string{"link": "short link", "qr": "QR code"}[r.TargetType] + "."},
			Details:    [][2]string{{"Reason", r.Reason}, {"Address", r.TargetURL}, {"Details", firstNonEmpty(r.Details, "—")}},
			Button:     &email.Button{Label: "Review in the admin console", URL: s.appURL + "/admin/moderation"},
			Reason:     "You received this because you're Lynqr staff.",
		})
	}
}
