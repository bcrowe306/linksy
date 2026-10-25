// Package auth holds Lynqr's authorization model: the role/permission matrix
// and the opaque token primitives used for sessions, invitations and API keys.
package auth

import "github.com/bcrowe306/linksy/internal/models"

type Permission string

const (
	PermViewAssets    Permission = "assets:view"    // Read links, QR codes, documents
	PermExportAssets  Permission = "assets:export"  // Download QR exports and reports
	PermViewAnalytics Permission = "analytics:view" // Read analytics
	PermEditAssets    Permission = "assets:edit"    // Create, update, delete links/QR/docs
	PermManageMembers Permission = "members:manage" // Invite, change roles, suspend, remove
	PermManageTeam    Permission = "team:manage"    // Rename workspace
	PermViewAudit     Permission = "audit:view"     // Read the audit log
	PermManageBilling Permission = "billing:manage" // Change plan, payment details
	PermDeleteTeam    Permission = "team:delete"    // Delete the workspace
	PermTransferOwner Permission = "team:transfer"  // Hand ownership to another member
)

var rolePermissions = map[models.UserRole][]Permission{
	models.RoleViewer: {PermViewAssets, PermExportAssets, PermViewAnalytics},
	models.RoleMember: {PermViewAssets, PermExportAssets, PermViewAnalytics, PermEditAssets},
	models.RoleAdmin: {PermViewAssets, PermExportAssets, PermViewAnalytics, PermEditAssets,
		PermManageMembers, PermManageTeam, PermViewAudit},
	models.RoleOwner: {PermViewAssets, PermExportAssets, PermViewAnalytics, PermEditAssets,
		PermManageMembers, PermManageTeam, PermViewAudit, PermManageBilling, PermDeleteTeam, PermTransferOwner},
}

// Can reports whether a role grants a permission. Unknown roles grant nothing.
func Can(role models.UserRole, p Permission) bool {
	for _, granted := range rolePermissions[role] {
		if granted == p {
			return true
		}
	}
	return false
}

// Permissions lists every permission granted to a role (exposed to the UI).
func Permissions(role models.UserRole) []Permission {
	out := make([]Permission, len(rolePermissions[role]))
	copy(out, rolePermissions[role])
	return out
}

// ValidRole reports whether r is a recognised role.
func ValidRole(r models.UserRole) bool {
	_, ok := rolePermissions[r]
	return ok
}

// AssignableRole reports whether r may be granted via invite or role change.
// Ownership only moves through an explicit transfer.
func AssignableRole(r models.UserRole) bool {
	return r == models.RoleAdmin || r == models.RoleMember || r == models.RoleViewer
}
