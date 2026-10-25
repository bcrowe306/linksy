package auth

import (
	"strings"
	"testing"

	"github.com/bcrowe306/linksy/internal/models"
)

func TestRBACMatrix(t *testing.T) {
	cases := []struct {
		role models.UserRole
		perm Permission
		want bool
	}{
		{models.RoleViewer, PermViewAssets, true},
		{models.RoleViewer, PermExportAssets, true},
		{models.RoleViewer, PermEditAssets, false},
		{models.RoleViewer, PermManageMembers, false},
		{models.RoleMember, PermEditAssets, true},
		{models.RoleMember, PermManageMembers, false},
		{models.RoleMember, PermViewAudit, false},
		{models.RoleAdmin, PermManageMembers, true},
		{models.RoleAdmin, PermViewAudit, true},
		{models.RoleAdmin, PermManageBilling, false},
		{models.RoleAdmin, PermDeleteTeam, false},
		{models.RoleOwner, PermManageBilling, true},
		{models.RoleOwner, PermDeleteTeam, true},
		{models.RoleOwner, PermTransferOwner, true},
		{models.UserRole("superuser"), PermViewAssets, false},
		{models.UserRole(""), PermViewAssets, false},
	}
	for _, c := range cases {
		if got := Can(c.role, c.perm); got != c.want {
			t.Errorf("Can(%q, %q) = %v, want %v", c.role, c.perm, got, c.want)
		}
	}
}

func TestAssignableRole(t *testing.T) {
	if AssignableRole(models.RoleOwner) {
		t.Error("owner must not be directly assignable")
	}
	for _, r := range []models.UserRole{models.RoleAdmin, models.RoleMember, models.RoleViewer} {
		if !AssignableRole(r) {
			t.Errorf("%s should be assignable", r)
		}
	}
}

func TestTokens(t *testing.T) {
	a, err := NewToken(SessionPrefix)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := NewToken(SessionPrefix)
	if a == b {
		t.Fatal("tokens must be unique")
	}
	if !strings.HasPrefix(a, SessionPrefix) || len(a) < 40 {
		t.Fatalf("unexpected token shape: %s", a)
	}
	if HashToken(a) == a || HashToken(a) != HashToken(a) || len(HashToken(a)) != 64 {
		t.Fatal("hash must be deterministic 64-char hex and differ from the token")
	}
}
