package service_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/bcrowe306/linksy/internal/models"
	"github.com/bcrowe306/linksy/internal/service"
	"github.com/bcrowe306/linksy/test/testkit"
)

func TestValidateTargetURL(t *testing.T) {
	valid := []string{
		"https://example.com",
		"http://example.com/some/path?query=1",
		"https://sub.domain.org/app#section",
		"https://8.8.8.8/dns",
	}
	for _, v := range valid {
		if err := service.ValidateTargetURL(v); err != nil {
			t.Errorf("expected %q to be valid, got: %v", v, err)
		}
	}

	invalid := []string{
		"ftp://example.com",
		"javascript:alert(1)",
		"data:text/html,<script>alert(1)</script>",
		"http://localhost:3000",
		"http://LOCALHOST.:3000",
		"http://app.localhost",
		"http://127.0.0.1:8080",
		"http://127.1.2.3",
		"http://[::1]/",
		"http://192.168.1.1/admin",
		"http://10.0.0.5",
		"http://172.16.4.4",
		"http://169.254.169.254/latest/meta-data",
		"http://100.64.0.1",
		"http://2130706433/",
		"http://0x7f000001/",
		"http://user:pass@example.com",
		"http://intranet",
		"not-a-url",
		"",
	}
	for _, v := range invalid {
		if err := service.ValidateTargetURL(v); err == nil {
			t.Errorf("expected %q to be rejected", v)
		}
	}
}

func TestShortLinkLifecycle(t *testing.T) {
	k := testkit.New(t)
	u := k.Login("owner@acme.test")
	a := k.Actor(u, u.TeamID)

	link, err := k.Svc.CreateShortLink(a, service.CreateLinkInput{TargetURL: "https://example.com/original", Title: "Original"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if len(link.Slug) < 7 {
		t.Fatalf("expected generated slug, got %q", link.Slug)
	}

	resolved, err := k.Svc.ResolveShortLink(link.Slug)
	if err != nil || resolved.TargetURL != "https://example.com/original" {
		t.Fatalf("resolve: %v %+v", err, resolved)
	}

	if _, err := k.Svc.UpdateShortLink(a, link.ID, service.UpdateLinkInput{TargetURL: "https://example.com/updated", IsActive: true}); err != nil {
		t.Fatalf("update: %v", err)
	}
	if r, _ := k.Svc.ResolveShortLink(link.Slug); r == nil || r.TargetURL != "https://example.com/updated" {
		t.Fatalf("dynamic destination not applied: %+v", r)
	}

	// Pausing and expiry stop resolution
	if _, err := k.Svc.UpdateShortLink(a, link.ID, service.UpdateLinkInput{IsActive: false}); err != nil {
		t.Fatal(err)
	}
	if _, err := k.Svc.ResolveShortLink(link.Slug); !errors.Is(err, service.ErrLinkNotFound) {
		t.Fatalf("paused link should not resolve, got %v", err)
	}
	past := time.Now().Add(-time.Hour)
	if _, err := k.Svc.UpdateShortLink(a, link.ID, service.UpdateLinkInput{IsActive: true, ExpiresAt: &past}); err != nil {
		t.Fatal(err)
	}
	if _, err := k.Svc.ResolveShortLink(link.Slug); !errors.Is(err, service.ErrLinkNotFound) {
		t.Fatalf("expired link should not resolve, got %v", err)
	}
}

func TestCustomSlugValidationAndCollision(t *testing.T) {
	k := testkit.New(t)
	u := k.Login("slugs@acme.test")
	a := k.Actor(u, u.TeamID)

	if _, err := k.Svc.CreateShortLink(a, service.CreateLinkInput{TargetURL: "https://example.com", CustomSlug: "promo-2026"}); err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := k.Svc.CreateShortLink(a, service.CreateLinkInput{TargetURL: "https://example.com", CustomSlug: "promo-2026"}); !errors.Is(err, service.ErrSlugTaken) {
		t.Fatalf("expected ErrSlugTaken, got %v", err)
	}
	for _, bad := range []string{"a", "has space", "../etc", "api", "emoji🙂", strings.Repeat("x", 65)} {
		if _, err := k.Svc.CreateShortLink(a, service.CreateLinkInput{TargetURL: "https://example.com", CustomSlug: bad}); !errors.Is(err, service.ErrInvalidSlug) {
			t.Errorf("slug %q: expected ErrInvalidSlug, got %v", bad, err)
		}
	}
}

func TestDynamicQRCodeAndExport(t *testing.T) {
	k := testkit.New(t)
	u := k.Login("qr@acme.test")
	a := k.Actor(u, u.TeamID)

	qrCode, err := k.Svc.CreateQRCode(a, service.CreateQRInput{
		Type: models.QRTypeWebsite, Title: "Site", TargetURL: "https://example.com/company", IsDynamic: true,
		Style: models.QRStyle{FgColor: "#0055AA", BgColor: "#FFFFFF"},
	})
	if err != nil {
		t.Fatalf("create QR: %v", err)
	}
	if qrCode.LinkID == nil {
		t.Fatal("expected dynamic QR code to have an associated short link")
	}

	for _, f := range []string{"png", "jpeg", "svg", "pdf"} {
		b, mime, err := k.Svc.ExportQRCode(a, qrCode.ID, f, 256)
		if err != nil || len(b) == 0 || mime == "" {
			t.Fatalf("export %s: %v (len=%d mime=%q)", f, err, len(b), mime)
		}
	}
	if _, _, err := k.Svc.ExportQRCode(a, qrCode.ID, "bmp", 256); err == nil {
		t.Fatal("expected unsupported format error")
	}
}

func TestSignupCreatesIsolatedWorkspaces(t *testing.T) {
	k := testkit.New(t)
	alice := k.Login("alice@acme.test")
	bob := k.Login("bob@other.test")
	if alice.TeamID == "" || bob.TeamID == "" || alice.TeamID == bob.TeamID {
		t.Fatalf("each signup must get its own workspace: %q vs %q", alice.TeamID, bob.TeamID)
	}

	aa := k.Actor(alice, alice.TeamID)
	link, err := k.Svc.CreateShortLink(aa, service.CreateLinkInput{TargetURL: "https://alice.example"})
	if err != nil {
		t.Fatal(err)
	}
	ba := k.Actor(bob, bob.TeamID)
	if _, err := k.Svc.GetLink(ba, link.ID); !service.IsNotFound(err) {
		t.Fatalf("bob must not read alice's link, got %v", err)
	}
	if err := k.Svc.DeleteLink(ba, link.ID); !service.IsNotFound(err) {
		t.Fatalf("bob must not delete alice's link, got %v", err)
	}
	if _, err := k.Svc.UpdateShortLink(ba, link.ID, service.UpdateLinkInput{TargetURL: "https://evil.example", IsActive: true}); !service.IsNotFound(err) {
		t.Fatalf("bob must not update alice's link, got %v", err)
	}
	if links, _ := k.Svc.ListLinks(ba, 50, 0); len(links) != 0 {
		t.Fatalf("bob's workspace must be empty, got %d links", len(links))
	}

	// Bob can't resolve into alice's team even if he asks for it
	if _, _, err := k.Svc.ResolveTeam(bob.ID, alice.TeamID, ""); !errors.Is(err, service.ErrNotMember) {
		t.Fatalf("expected ErrNotMember, got %v", err)
	}

	// Re-login keeps the same workspace rather than creating another
	again := k.Login("alice@acme.test")
	if again.TeamID != alice.TeamID {
		t.Fatalf("re-login switched workspace: %s -> %s", alice.TeamID, again.TeamID)
	}
	teams, _ := k.Svc.ListUserTeams(alice.ID)
	if len(teams) != 1 || teams[0].Role != models.RoleOwner {
		t.Fatalf("expected one owned workspace, got %+v", teams)
	}
}

func TestRoleEnforcement(t *testing.T) {
	k := testkit.New(t)
	owner := k.Login("owner@team.test")
	oa := k.Actor(owner, owner.TeamID)

	viewer := k.Login("viewer@team.test")
	inv, err := k.Svc.CreateInvitation(oa, viewer.Email, models.RoleViewer)
	if err != nil {
		t.Fatal(err)
	}
	token := inv.AcceptURL[strings.LastIndex(inv.AcceptURL, "/")+1:]
	if _, err := k.Svc.AcceptInvitation(token, viewer.ID, ""); err != nil {
		t.Fatalf("accept: %v", err)
	}
	va := k.Actor(viewer, owner.TeamID)
	if va.Role != models.RoleViewer {
		t.Fatalf("expected viewer role, got %s", va.Role)
	}

	if _, err := k.Svc.CreateShortLink(va, service.CreateLinkInput{TargetURL: "https://example.com"}); !errors.Is(err, service.ErrForbidden) {
		t.Fatalf("viewer must not create links, got %v", err)
	}
	if _, err := k.Svc.ListLinks(va, 10, 0); err != nil {
		t.Fatalf("viewer can list links: %v", err)
	}
	if _, err := k.Svc.CreateInvitation(va, "x@y.test", models.RoleMember); !errors.Is(err, service.ErrForbidden) {
		t.Fatalf("viewer must not invite, got %v", err)
	}
	if _, err := k.Svc.ListAudit(va, 10, 0); !errors.Is(err, service.ErrForbidden) {
		t.Fatalf("viewer must not read audit log, got %v", err)
	}
}

func TestMemberHierarchy(t *testing.T) {
	k := testkit.New(t)
	owner := k.Login("o@h.test")
	oa := k.Actor(owner, owner.TeamID)

	join := func(email string, role models.UserRole) *testkit.User {
		u := k.Login(email)
		inv, err := k.Svc.CreateInvitation(oa, email, role)
		if err != nil {
			t.Fatalf("invite %s: %v", email, err)
		}
		if _, err := k.Svc.AcceptInvitation(inv.AcceptURL[strings.LastIndex(inv.AcceptURL, "/")+1:], u.ID, ""); err != nil {
			t.Fatalf("accept %s: %v", email, err)
		}
		return u
	}
	admin := join("a@h.test", models.RoleAdmin)
	admin2 := join("a2@h.test", models.RoleAdmin)
	member := join("m@h.test", models.RoleMember)
	aa := k.Actor(admin, owner.TeamID)

	cases := []struct {
		name string
		err  error
		want error
	}{
		{"admin cannot change owner", k.Svc.UpdateTeamMemberRole(aa, owner.ID, models.RoleMember), service.ErrOwnerProtected},
		{"admin cannot demote another admin", k.Svc.UpdateTeamMemberRole(aa, admin2.ID, models.RoleMember), service.ErrForbidden},
		{"admin cannot grant admin", k.Svc.UpdateTeamMemberRole(aa, member.ID, models.RoleAdmin), service.ErrForbidden},
		{"admin cannot change self", k.Svc.UpdateTeamMemberRole(aa, admin.ID, models.RoleViewer), service.ErrSelfAction},
		{"owner role is not assignable", k.Svc.UpdateTeamMemberRole(oa, member.ID, models.RoleOwner), nil},
		{"admin cannot remove owner", k.Svc.RemoveTeamMember(aa, owner.ID), service.ErrOwnerProtected},
		{"owner cannot leave", k.Svc.LeaveTeam(oa), service.ErrOwnerProtected},
	}
	for _, c := range cases {
		if c.want == nil {
			if c.err == nil {
				t.Errorf("%s: expected an error", c.name)
			}
			continue
		}
		if !errors.Is(c.err, c.want) {
			t.Errorf("%s: expected %v, got %v", c.name, c.want, c.err)
		}
	}

	// Admin may manage members; suspension blocks access; reactivation restores it.
	if err := k.Svc.UpdateTeamMemberRole(aa, member.ID, models.RoleViewer); err != nil {
		t.Fatalf("admin demote member: %v", err)
	}
	if err := k.Svc.SetMemberSuspended(aa, member.ID, true); err != nil {
		t.Fatal(err)
	}
	if _, _, err := k.Svc.ResolveTeam(member.ID, owner.TeamID, ""); !errors.Is(err, service.ErrSuspended) {
		t.Fatalf("suspended member must be blocked, got %v", err)
	}
	if tid, _, err := k.Svc.ResolveTeam(member.ID, "", owner.TeamID); err != nil || tid == owner.TeamID {
		t.Fatalf("suspended member should fall back to their own workspace, got %q %v", tid, err)
	}
	if err := k.Svc.SetMemberSuspended(aa, member.ID, false); err != nil {
		t.Fatal(err)
	}
	if _, _, err := k.Svc.ResolveTeam(member.ID, owner.TeamID, ""); err != nil {
		t.Fatalf("reactivated member should have access: %v", err)
	}

	// Removal deletes only the membership
	if err := k.Svc.RemoveTeamMember(aa, member.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := k.Svc.GetUserProfile(member.ID); err != nil {
		t.Fatalf("removing a member must not delete their account: %v", err)
	}

	// Ownership transfer
	if err := k.Svc.TransferOwnership(oa, admin.ID); err != nil {
		t.Fatalf("transfer: %v", err)
	}
	if _, role, _ := k.Svc.ResolveTeam(admin.ID, owner.TeamID, ""); role != models.RoleOwner {
		t.Fatalf("new owner role = %s", role)
	}
	if _, role, _ := k.Svc.ResolveTeam(owner.ID, owner.TeamID, ""); role != models.RoleAdmin {
		t.Fatalf("previous owner should be admin, got %s", role)
	}

	audit, err := k.Svc.ListAudit(k.Actor(admin, owner.TeamID), 100, 0)
	if err != nil || len(audit) < 5 {
		t.Fatalf("expected audit entries, got %d (%v)", len(audit), err)
	}
}

func TestInvitationRules(t *testing.T) {
	k := testkit.New(t)
	owner := k.Login("own@inv.test")
	oa := k.Actor(owner, owner.TeamID)

	inv, err := k.Svc.CreateInvitation(oa, "Invitee@Inv.Test", models.RoleMember)
	if err != nil {
		t.Fatal(err)
	}
	if inv.Email != "invitee@inv.test" || !strings.Contains(inv.AcceptURL, "/invite/lqi_") {
		t.Fatalf("unexpected invitation: %+v", inv)
	}
	if _, err := k.Svc.CreateInvitation(oa, "invitee@inv.test", models.RoleMember); !errors.Is(err, service.ErrConflict) {
		t.Fatalf("duplicate pending invite should conflict, got %v", err)
	}
	if _, err := k.Svc.CreateInvitation(oa, "not-an-email", models.RoleMember); err == nil {
		t.Fatal("invalid email accepted")
	}
	if _, err := k.Svc.CreateInvitation(oa, "x@inv.test", models.RoleOwner); err == nil {
		t.Fatal("owner role must not be invitable")
	}

	token := inv.AcceptURL[strings.LastIndex(inv.AcceptURL, "/")+1:]
	intruder := k.Login("intruder@evil.test")
	if _, err := k.Svc.AcceptInvitation(token, intruder.ID, ""); !errors.Is(err, service.ErrInvitationMismatch) {
		t.Fatalf("invite must be bound to its email, got %v", err)
	}

	invitee := k.Login("invitee@inv.test")
	if mine, _ := k.Svc.ListMyInvitations(invitee.Email); len(mine) != 1 {
		t.Fatalf("expected 1 pending invite for invitee, got %d", len(mine))
	}
	if _, err := k.Svc.AcceptInvitation(token, invitee.ID, ""); err != nil {
		t.Fatalf("accept: %v", err)
	}
	if _, err := k.Svc.AcceptInvitation(token, invitee.ID, ""); !errors.Is(err, service.ErrInvitationInvalid) {
		t.Fatalf("invitation must be single use, got %v", err)
	}
	if _, err := k.Svc.PreviewInvitation("lqi_bogus"); !errors.Is(err, service.ErrInvitationInvalid) {
		t.Fatalf("bogus token should be invalid, got %v", err)
	}

	// Revoked invites can't be accepted
	inv2, _ := k.Svc.CreateInvitation(oa, "late@inv.test", models.RoleViewer)
	if err := k.Svc.RevokeInvitation(oa, inv2.ID); err != nil {
		t.Fatal(err)
	}
	late := k.Login("late@inv.test")
	if _, err := k.Svc.AcceptInvitation(inv2.AcceptURL[strings.LastIndex(inv2.AcceptURL, "/")+1:], late.ID, ""); !errors.Is(err, service.ErrInvitationInvalid) {
		t.Fatalf("revoked invite accepted: %v", err)
	}
}

func TestSessions(t *testing.T) {
	k := testkit.New(t)
	u := k.Login("sess@acme.test")

	sess, err := k.Svc.ResolveSession(u.Token)
	if err != nil || sess.UserID != u.ID {
		t.Fatalf("resolve: %v", err)
	}
	for _, forged := range []string{"sess_" + u.ID + "_123", u.ID, "lqs_forged", u.Token + "x", ""} {
		if _, err := k.Svc.ResolveSession(forged); err == nil {
			t.Errorf("forged token %q accepted", forged)
		}
	}

	second := k.Login("sess@acme.test")
	list, _ := k.Svc.ListSessions(u.ID, sess.ID)
	if len(list) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(list))
	}
	if err := k.Svc.EndOtherSessions(u.ID, sess.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := k.Svc.ResolveSession(second.Token); err == nil {
		t.Fatal("other session should be revoked")
	}
	if err := k.Svc.EndSession(sess.ID, u.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := k.Svc.ResolveSession(u.Token); err == nil {
		t.Fatal("ended session still valid")
	}
}

func TestAnalyticsAreTeamScoped(t *testing.T) {
	k := testkit.New(t)
	alice := k.Login("an-a@acme.test")
	bob := k.Login("an-b@acme.test")
	aa, ba := k.Actor(alice, alice.TeamID), k.Actor(bob, bob.TeamID)

	link, _ := k.Svc.CreateShortLink(aa, service.CreateLinkInput{TargetURL: "https://example.com"})
	k.Svc.TrackEvent(&models.AnalyticsEvent{TeamID: alice.TeamID, LinkID: &link.ID, EventType: "click", IP: "1.2.3.4",
		UserAgent: "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) Mobile Safari", Country: "US"})

	as, err := k.Svc.GetAnalyticsSummary(aa, service.AnalyticsQuery{})
	if err != nil || as.TotalClicks != 1 || len(as.ByDate) != 1 || as.ByDevice[0].Name != "Mobile" {
		t.Fatalf("alice summary: %v %+v", err, as)
	}
	bs, _ := k.Svc.GetAnalyticsSummary(ba, service.AnalyticsQuery{LinkID: link.ID})
	if bs.TotalClicks != 0 {
		t.Fatalf("bob must not see alice's analytics, got %d clicks", bs.TotalClicks)
	}
	future := time.Now().Add(24 * time.Hour)
	if fs, _ := k.Svc.GetAnalyticsSummary(aa, service.AnalyticsQuery{From: &future}); fs.TotalClicks != 0 {
		t.Fatalf("time range filter not applied: %d", fs.TotalClicks)
	}
}
