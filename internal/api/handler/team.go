package handler

import (
	"context"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/bcrowe306/linksy/internal/api/middleware"
	"github.com/bcrowe306/linksy/internal/auth"
	"github.com/bcrowe306/linksy/internal/config"
	"github.com/bcrowe306/linksy/internal/models"
	"github.com/bcrowe306/linksy/internal/service"
)

type TeamHandler struct {
	cfg    *config.Config
	svc    *service.Service
	workos *auth.WorkOS
}

func NewTeamHandler(cfg *config.Config, svc *service.Service) *TeamHandler {
	return &TeamHandler{cfg: cfg, svc: svc, workos: auth.NewWorkOS(cfg.WorkOSAPIKey, cfg.WorkOSClient)}
}

// ----------------- Workspaces -----------------

// ListTeams returns every workspace the caller belongs to.
func (t *TeamHandler) ListTeams(c fiber.Ctx) error {
	ac := middleware.GetAuth(c)
	teams, err := t.svc.ListUserTeams(ac.UserID)
	if err != nil {
		return fail(c, err)
	}
	return c.JSON(fiber.Map{"data": teams, "active_team_id": ac.TeamID})
}

func (t *TeamHandler) CreateTeam(c fiber.Ctx) error {
	ac := middleware.GetAuth(c)
	var req struct {
		Name string `json:"name"`
	}
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	team, err := t.svc.CreateTeam(ac.UserID, req.Name, middleware.ClientIP(c))
	if err != nil {
		return fail(c, err)
	}
	if ac.Session != nil {
		_, _ = t.svc.SwitchTeam(ac.Session, team.ID)
	}
	return c.Status(fiber.StatusCreated).JSON(team)
}

// SwitchTeam pins the current session to another workspace.
func (t *TeamHandler) SwitchTeam(c fiber.Ctx) error {
	ac := middleware.GetAuth(c)
	var req struct {
		TeamID string `json:"team_id"`
	}
	if err := c.Bind().Body(&req); err != nil || req.TeamID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "team_id is required"})
	}
	role, err := t.svc.SwitchTeam(ac.Session, req.TeamID)
	if err != nil {
		return fail(c, err)
	}
	return c.JSON(fiber.Map{"team_id": req.TeamID, "role": role, "permissions": auth.Permissions(role)})
}

func (t *TeamHandler) GetTeam(c fiber.Ctx) error {
	ac := middleware.GetAuth(c)
	team, err := t.svc.GetTeam(middleware.Actor(c))
	if err != nil {
		return fail(c, err)
	}
	return c.JSON(fiber.Map{"team": team, "role": ac.Role, "permissions": auth.Permissions(ac.Role)})
}

func (t *TeamHandler) UpdateTeam(c fiber.Ctx) error {
	var req struct {
		Name string `json:"name"`
	}
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	team, err := t.svc.RenameTeam(middleware.Actor(c), req.Name)
	if err != nil {
		return fail(c, err)
	}
	return c.JSON(team)
}

// DeleteTeam permanently deletes the workspace. The caller must echo the name.
func (t *TeamHandler) DeleteTeam(c fiber.Ctx) error {
	a := middleware.Actor(c)
	var req struct {
		ConfirmName string `json:"confirm_name"`
	}
	_ = c.Bind().Body(&req)
	team, err := t.svc.GetTeam(a)
	if err != nil {
		return fail(c, err)
	}
	if strings.TrimSpace(req.ConfirmName) != team.Name {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Type the workspace name exactly to confirm deletion"})
	}
	if err := t.svc.DeleteTeam(a); err != nil {
		return fail(c, err)
	}
	return c.JSON(fiber.Map{"message": "Workspace deleted"})
}

func (t *TeamHandler) TransferOwnership(c fiber.Ctx) error {
	var req struct {
		UserID string `json:"user_id"`
	}
	if err := c.Bind().Body(&req); err != nil || req.UserID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "user_id is required"})
	}
	if err := t.svc.TransferOwnership(middleware.Actor(c), req.UserID); err != nil {
		return fail(c, err)
	}
	return c.JSON(fiber.Map{"message": "Ownership transferred"})
}

func (t *TeamHandler) LeaveTeam(c fiber.Ctx) error {
	if err := t.svc.LeaveTeam(middleware.Actor(c)); err != nil {
		return fail(c, err)
	}
	return c.JSON(fiber.Map{"message": "You left the workspace"})
}

func (t *TeamHandler) ListAudit(c fiber.Ctx) error {
	events, err := t.svc.ListAudit(middleware.Actor(c), queryInt(c, "limit", 50), queryInt(c, "offset", 0))
	if err != nil {
		return fail(c, err)
	}
	return c.JSON(fiber.Map{"data": events})
}

// ----------------- Members -----------------

// ListMembers returns members, plus pending invitations for those who manage members.
func (t *TeamHandler) ListMembers(c fiber.Ctx) error {
	ac := middleware.GetAuth(c)
	a := middleware.Actor(c)
	members, err := t.svc.ListTeamMembers(a)
	if err != nil {
		return fail(c, err)
	}
	invitations := []models.Invitation{}
	if auth.Can(ac.Role, auth.PermManageMembers) {
		if invs, err := t.svc.ListInvitations(a); err == nil {
			invitations = invs
		}
	}
	team, _ := t.svc.GetTeam(a)
	name := ""
	if team != nil {
		name = team.Name
	}
	return c.JSON(fiber.Map{
		"team_id":             ac.TeamID,
		"team_name":           name,
		"current_user_id":     ac.UserID,
		"current_user_role":   ac.Role,
		"members":             members,
		"pending_invitations": invitations,
	})
}

func (t *TeamHandler) InviteMember(c fiber.Ctx) error {
	var req models.InviteMemberRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	inv, err := t.svc.CreateInvitation(middleware.Actor(c), req.Email, req.Role)
	if err != nil {
		return fail(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message":    "Invitation created. Share the link with " + inv.Email + ".",
		"invitation": inv,
	})
}

func (t *TeamHandler) UpdateMemberRole(c fiber.Ctx) error {
	var req models.UpdateRoleRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	userID := c.Params("userId")
	if err := t.svc.UpdateTeamMemberRole(middleware.Actor(c), userID, req.Role); err != nil {
		return fail(c, err)
	}
	return c.JSON(fiber.Map{"message": "Role updated", "user_id": userID, "role": req.Role})
}

func (t *TeamHandler) SuspendMember(c fiber.Ctx) error {
	if err := t.svc.SetMemberSuspended(middleware.Actor(c), c.Params("userId"), true); err != nil {
		return fail(c, err)
	}
	return c.JSON(fiber.Map{"message": "Member suspended"})
}

func (t *TeamHandler) ReactivateMember(c fiber.Ctx) error {
	if err := t.svc.SetMemberSuspended(middleware.Actor(c), c.Params("userId"), false); err != nil {
		return fail(c, err)
	}
	return c.JSON(fiber.Map{"message": "Member reactivated"})
}

// RemoveMember removes the membership only; the user's account is untouched.
func (t *TeamHandler) RemoveMember(c fiber.Ctx) error {
	userID := c.Params("userId")
	if err := t.svc.RemoveTeamMember(middleware.Actor(c), userID); err != nil {
		return fail(c, err)
	}
	return c.JSON(fiber.Map{"message": "Member removed from workspace", "user_id": userID})
}

// ResetMemberCredentials emails a password reset link to the member. The link
// is sent only to the member's inbox and is never returned to the admin.
func (t *TeamHandler) ResetMemberCredentials(c fiber.Ctx) error {
	a := middleware.Actor(c)
	userID := c.Params("userId")
	members, err := t.svc.ListTeamMembers(a)
	if err != nil {
		return fail(c, err)
	}
	email := ""
	for _, m := range members {
		if m.UserID == userID {
			email = m.Email
		}
	}
	if email == "" {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Member not found"})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if _, err := t.workos.CreatePasswordReset(ctx, email); err != nil {
		return authFailure(c, err, fiber.StatusBadGateway)
	}
	t.svc.Audit(a, "member.password_reset", "user", userID, nil)
	return c.JSON(fiber.Map{"message": "Password reset email sent to " + email, "email": email})
}

func (t *TeamHandler) RevokeInvitation(c fiber.Ctx) error {
	id := c.Params("id")
	if err := t.svc.RevokeInvitation(middleware.Actor(c), id); err != nil {
		return fail(c, err)
	}
	return c.JSON(fiber.Map{"message": "Invitation revoked", "id": id})
}

// ----------------- Invitation acceptance -----------------

// PreviewInvitation is public so the accept page can show who invited you.
func (t *TeamHandler) PreviewInvitation(c fiber.Ctx) error {
	inv, err := t.svc.PreviewInvitation(c.Params("token"))
	if err != nil {
		return fail(c, err)
	}
	return c.JSON(fiber.Map{"team_name": inv.TeamName, "email": inv.Email, "role": inv.Role, "expires_at": inv.ExpiresAt})
}

func (t *TeamHandler) AcceptInvitation(c fiber.Ctx) error {
	ac := middleware.GetAuth(c)
	inv, err := t.svc.AcceptInvitation(c.Params("token"), ac.UserID, middleware.ClientIP(c))
	if err != nil {
		return fail(c, err)
	}
	if ac.Session != nil {
		_, _ = t.svc.SwitchTeam(ac.Session, inv.TeamID)
	}
	return c.JSON(fiber.Map{"message": "Welcome to " + inv.TeamName, "team_id": inv.TeamID, "role": inv.Role})
}

func (t *TeamHandler) ListMyInvitations(c fiber.Ctx) error {
	u, err := t.svc.GetUserProfile(middleware.GetAuth(c).UserID)
	if err != nil {
		return fail(c, err)
	}
	invs, err := t.svc.ListMyInvitations(u.Email)
	if err != nil {
		return fail(c, err)
	}
	return c.JSON(fiber.Map{"data": invs})
}

func (t *TeamHandler) RespondToInvitation(accept bool) fiber.Handler {
	return func(c fiber.Ctx) error {
		ac := middleware.GetAuth(c)
		id := c.Params("id")
		if err := t.svc.RespondToInvitation(id, ac.UserID, accept, middleware.ClientIP(c)); err != nil {
			return fail(c, err)
		}
		return c.JSON(fiber.Map{"message": map[bool]string{true: "Invitation accepted", false: "Invitation declined"}[accept]})
	}
}
