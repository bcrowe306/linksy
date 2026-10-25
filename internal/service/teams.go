package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/bcrowe306/linksy/internal/auth"
	"github.com/bcrowe306/linksy/internal/models"
	"github.com/bcrowe306/linksy/internal/repository"
)

const (
	SessionTTL      = 30 * 24 * time.Hour
	sessionTouchGap = 10 * time.Minute
	InvitationTTL   = 7 * 24 * time.Hour
)

var (
	ErrSuspended          = errors.New("your access to this workspace has been suspended")
	ErrNotMember          = errors.New("you are not a member of this workspace")
	ErrInvitationInvalid  = errors.New("this invitation is invalid, expired, or has already been used")
	ErrInvitationMismatch = errors.New("this invitation was sent to a different email address")
	ErrAlreadyMember      = errors.New("that user is already a member of this workspace")
	ErrOwnerProtected     = errors.New("the workspace owner cannot be changed this way; transfer ownership first")
	ErrSelfAction         = errors.New("you cannot perform this action on your own membership")
)

// ----------------- Sessions -----------------

// StartSession records the identity-provider user locally, guarantees they own
// at least one workspace, and issues an opaque session token.
func (s *Service) StartSession(u *models.User, ip, userAgent string) (string, *models.Session, error) {
	if u == nil || u.ID == "" || u.Email == "" {
		return "", nil, errors.New("identity provider returned an incomplete user")
	}
	u.Email = strings.ToLower(strings.TrimSpace(u.Email))
	if err := s.repo.UpsertUser(u); err != nil {
		return "", nil, fmt.Errorf("failed to record user: %w", err)
	}

	teamID, err := s.defaultTeam(u)
	if err != nil {
		return "", nil, err
	}

	token, err := auth.NewToken(auth.SessionPrefix)
	if err != nil {
		return "", nil, err
	}
	t := time.Now().UTC()
	sess := &models.Session{
		UserID:       u.ID,
		ActiveTeamID: teamID,
		IP:           ip,
		UserAgent:    truncate(userAgent, 512),
		CreatedAt:    t,
		LastSeenAt:   t,
		ExpiresAt:    t.Add(SessionTTL),
	}
	if err := s.repo.CreateSession(sess, auth.HashToken(token)); err != nil {
		return "", nil, fmt.Errorf("failed to create session: %w", err)
	}
	return token, sess, nil
}

// ResolveSession validates a raw session token and slides its expiry.
func (s *Service) ResolveSession(token string) (*models.Session, error) {
	if !strings.HasPrefix(token, auth.SessionPrefix) {
		return nil, ErrUnauthorized
	}
	sess, err := s.repo.GetSessionByTokenHash(auth.HashToken(token))
	if err != nil {
		return nil, ErrUnauthorized
	}
	t := time.Now().UTC()
	if sess.ExpiresAt.Before(t) {
		_ = s.repo.DeleteSession(sess.ID, sess.UserID)
		return nil, ErrUnauthorized
	}
	if t.Sub(sess.LastSeenAt) > sessionTouchGap {
		sess.LastSeenAt, sess.ExpiresAt = t, t.Add(SessionTTL)
		_ = s.repo.TouchSession(sess.ID, sess.LastSeenAt, sess.ExpiresAt)
	}
	return sess, nil
}

func (s *Service) EndSession(sessionID, userID string) error {
	return s.repo.DeleteSession(sessionID, userID)
}

func (s *Service) ListSessions(userID, currentID string) ([]models.Session, error) {
	list, err := s.repo.ListUserSessions(userID)
	for i := range list {
		list[i].Current = list[i].ID == currentID
	}
	return list, err
}

func (s *Service) EndOtherSessions(userID, keepID string) error {
	return s.repo.DeleteUserSessionsExcept(userID, keepID)
}

// ResolveTeam determines the team and role for a request. requested (from the
// X-Team-ID header) and preferred (the session's active team) are honoured only
// when the user holds an active membership; otherwise the first active team wins.
func (s *Service) ResolveTeam(userID, requested, preferred string) (string, models.UserRole, error) {
	if requested != "" {
		m, err := s.repo.GetMembership(requested, userID)
		if err != nil {
			return "", "", ErrNotMember
		}
		if m.Status != models.MemberActive {
			return "", "", ErrSuspended
		}
		return m.TeamID, m.Role, nil
	}
	if preferred != "" {
		if m, err := s.repo.GetMembership(preferred, userID); err == nil && m.Status == models.MemberActive {
			return m.TeamID, m.Role, nil
		}
	}
	teams, err := s.repo.ListUserTeams(userID)
	if err != nil {
		return "", "", err
	}
	for _, t := range teams {
		if t.Status == models.MemberActive {
			return t.ID, t.Role, nil
		}
	}
	// Every active member was suspended or removed everywhere: give them a fresh personal workspace.
	u, err := s.repo.GetUserByID(userID)
	if err != nil {
		return "", "", ErrUnauthorized
	}
	team, err := s.createPersonalTeam(u)
	if err != nil {
		return "", "", err
	}
	return team.ID, models.RoleOwner, nil
}

func (s *Service) defaultTeam(u *models.User) (string, error) {
	teams, err := s.repo.ListUserTeams(u.ID)
	if err != nil {
		return "", err
	}
	for _, t := range teams {
		if t.Status == models.MemberActive {
			return t.ID, nil
		}
	}
	team, err := s.createPersonalTeam(u)
	if err != nil {
		return "", err
	}
	return team.ID, nil
}

func (s *Service) createPersonalTeam(u *models.User) (*models.Team, error) {
	name := strings.TrimSpace(u.FirstName)
	if name == "" {
		name = strings.Split(u.Email, "@")[0]
	}
	return s.newTeam(u.ID, name+"'s Workspace")
}

var nonSlug = regexp.MustCompile(`[^a-z0-9]+`)

func (s *Service) newTeam(ownerID, name string) (*models.Team, error) {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 80 {
		return nil, errors.New("workspace name must be between 1 and 80 characters")
	}
	base := strings.Trim(nonSlug.ReplaceAllString(strings.ToLower(name), "-"), "-")
	if base == "" {
		base = "workspace"
	}
	if len(base) > 40 {
		base = base[:40]
	}
	slug := base
	for i := 0; s.repo.TeamSlugExists(slug); i++ {
		suffix, _ := GenerateRandomSlug(5)
		slug = base + "-" + strings.ToLower(suffix)
		if i > 5 {
			return nil, errors.New("could not allocate a workspace slug")
		}
	}
	t := &models.Team{Name: name, Slug: slug, OwnerID: ownerID}
	if err := s.repo.CreateTeam(t); err != nil {
		return nil, err
	}
	return t, nil
}

// ----------------- Teams -----------------

func (s *Service) ListUserTeams(userID string) ([]models.TeamMembership, error) {
	return s.repo.ListUserTeams(userID)
}

func (s *Service) GetTeam(a Actor) (*models.Team, error) {
	if err := a.require(auth.PermViewAssets); err != nil {
		return nil, err
	}
	return s.repo.GetTeam(a.TeamID)
}

func (s *Service) CreateTeam(userID, name, ip string) (*models.Team, error) {
	t, err := s.newTeam(userID, name)
	if err != nil {
		return nil, err
	}
	s.audit(Actor{UserID: userID, TeamID: t.ID, Role: models.RoleOwner, IP: ip}, "team.create", "team", t.ID, map[string]any{"name": t.Name})
	return t, nil
}

// SwitchTeam pins a session to a team the user actively belongs to.
func (s *Service) SwitchTeam(sess *models.Session, teamID string) (models.UserRole, error) {
	m, err := s.repo.GetMembership(teamID, sess.UserID)
	if err != nil {
		return "", ErrNotMember
	}
	if m.Status != models.MemberActive {
		return "", ErrSuspended
	}
	if err := s.repo.SetSessionTeam(sess.ID, teamID); err != nil {
		return "", err
	}
	sess.ActiveTeamID = teamID
	return m.Role, nil
}

func (s *Service) RenameTeam(a Actor, name string) (*models.Team, error) {
	if err := a.require(auth.PermManageTeam); err != nil {
		return nil, err
	}
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 80 {
		return nil, errors.New("workspace name must be between 1 and 80 characters")
	}
	if err := s.repo.RenameTeam(a.TeamID, name); err != nil {
		return nil, err
	}
	s.audit(a, "team.rename", "team", a.TeamID, map[string]any{"name": name})
	return s.repo.GetTeam(a.TeamID)
}

func (s *Service) DeleteTeam(a Actor) error {
	if err := a.require(auth.PermDeleteTeam); err != nil {
		return err
	}
	docs, err := s.repo.ListDocuments(a.TeamID)
	if err != nil {
		return err
	}
	if err := s.repo.DeleteTeam(a.TeamID); err != nil {
		return err
	}
	for _, d := range docs {
		_ = s.storage.Delete(d.StoredPath)
	}
	return nil
}

func (s *Service) TransferOwnership(a Actor, newOwnerID string) error {
	if err := a.require(auth.PermTransferOwner); err != nil {
		return err
	}
	if newOwnerID == a.UserID {
		return ErrSelfAction
	}
	m, err := s.repo.GetMembership(a.TeamID, newOwnerID)
	if err != nil {
		return ErrNotMember
	}
	if m.Status != models.MemberActive {
		return ErrSuspended
	}
	if err := s.repo.TransferOwnership(a.TeamID, a.UserID, newOwnerID); err != nil {
		return err
	}
	s.audit(a, "team.transfer_ownership", "user", newOwnerID, nil)
	return nil
}

// LeaveTeam removes the actor's own membership. Owners must transfer first.
func (s *Service) LeaveTeam(a Actor) error {
	if a.Role == models.RoleOwner {
		return ErrOwnerProtected
	}
	if err := s.repo.RemoveTeamMember(a.TeamID, a.UserID); err != nil {
		return err
	}
	s.audit(a, "member.leave", "user", a.UserID, nil)
	return nil
}

// ----------------- Members -----------------

func (s *Service) ListTeamMembers(a Actor) ([]models.TeamMemberDetail, error) {
	if err := a.require(auth.PermViewAssets); err != nil {
		return nil, err
	}
	return s.repo.ListTeamMembers(a.TeamID)
}

// canManage enforces the member hierarchy: nobody manages the owner or
// themselves, and only the owner manages admins or grants admin.
func (s *Service) canManage(a Actor, targetUserID string, newRole models.UserRole) (*models.TeamMember, error) {
	if err := a.require(auth.PermManageMembers); err != nil {
		return nil, err
	}
	if targetUserID == a.UserID {
		return nil, ErrSelfAction
	}
	target, err := s.repo.GetMembership(a.TeamID, targetUserID)
	if err != nil {
		return nil, ErrNotFound
	}
	if target.Role == models.RoleOwner {
		return nil, ErrOwnerProtected
	}
	if a.Role != models.RoleOwner && (target.Role == models.RoleAdmin || newRole == models.RoleAdmin) {
		return nil, ErrForbidden
	}
	return target, nil
}

func (s *Service) UpdateTeamMemberRole(a Actor, userID string, role models.UserRole) error {
	if !auth.AssignableRole(role) {
		return errors.New("role must be one of admin, member, or viewer")
	}
	target, err := s.canManage(a, userID, role)
	if err != nil {
		return err
	}
	if err := s.repo.UpdateMemberRole(a.TeamID, userID, role); err != nil {
		return err
	}
	s.audit(a, "member.role_change", "user", userID, map[string]any{"from": target.Role, "to": role})
	return nil
}

func (s *Service) SetMemberSuspended(a Actor, userID string, suspended bool) error {
	if _, err := s.canManage(a, userID, ""); err != nil {
		return err
	}
	status, action := models.MemberActive, "member.reactivate"
	if suspended {
		status, action = models.MemberSuspended, "member.suspend"
	}
	if err := s.repo.UpdateMemberStatus(a.TeamID, userID, status); err != nil {
		return err
	}
	s.audit(a, action, "user", userID, nil)
	return nil
}

func (s *Service) RemoveTeamMember(a Actor, userID string) error {
	if _, err := s.canManage(a, userID, ""); err != nil {
		return err
	}
	if err := s.repo.RemoveTeamMember(a.TeamID, userID); err != nil {
		return err
	}
	s.audit(a, "member.remove", "user", userID, nil)
	return nil
}

// ----------------- Invitations -----------------

var emailPattern = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

// CreateInvitation issues a single-use invitation and returns it with its accept URL.
func (s *Service) CreateInvitation(a Actor, email string, role models.UserRole) (*models.Invitation, error) {
	if err := a.require(auth.PermManageMembers); err != nil {
		return nil, err
	}
	email = strings.ToLower(strings.TrimSpace(email))
	if !emailPattern.MatchString(email) {
		return nil, errors.New("a valid email address is required")
	}
	if role == "" {
		role = models.RoleMember
	}
	if !auth.AssignableRole(role) {
		return nil, errors.New("role must be one of admin, member, or viewer")
	}
	if role == models.RoleAdmin && a.Role != models.RoleOwner {
		return nil, ErrForbidden
	}
	if u, err := s.repo.GetUserByEmail(email); err == nil {
		if _, err := s.repo.GetMembership(a.TeamID, u.ID); err == nil {
			return nil, ErrAlreadyMember
		}
	}
	if s.repo.HasPendingInvitation(a.TeamID, email) {
		return nil, fmt.Errorf("%w: an invitation is already pending for %s", ErrConflict, email)
	}

	token, err := auth.NewToken(auth.InvitationPrefix)
	if err != nil {
		return nil, err
	}
	t := time.Now().UTC()
	inv := &models.Invitation{
		TeamID:    a.TeamID,
		Email:     email,
		Role:      role,
		InvitedBy: a.UserID,
		CreatedAt: t,
		ExpiresAt: t.Add(InvitationTTL),
	}
	if err := s.repo.CreateInvitation(inv, auth.HashToken(token)); err != nil {
		return nil, err
	}
	if team, err := s.repo.GetTeam(a.TeamID); err == nil {
		inv.TeamName = team.Name
	}
	inv.AcceptURL = s.appURL + "/invite/" + token
	s.audit(a, "member.invite", "invitation", inv.ID, map[string]any{"email": email, "role": role})
	return inv, nil
}

func (s *Service) ListInvitations(a Actor) ([]models.Invitation, error) {
	if err := a.require(auth.PermManageMembers); err != nil {
		return nil, err
	}
	return s.repo.ListPendingInvitations(a.TeamID)
}

func (s *Service) ListMyInvitations(email string) ([]models.Invitation, error) {
	return s.repo.ListPendingInvitationsForEmail(email)
}

func (s *Service) RevokeInvitation(a Actor, id string) error {
	if err := a.require(auth.PermManageMembers); err != nil {
		return err
	}
	if err := s.repo.SetInvitationState(id, a.TeamID, "revoked"); err != nil {
		return err
	}
	s.audit(a, "member.invite_revoke", "invitation", id, nil)
	return nil
}

// PreviewInvitation returns a still-valid invitation for display on the accept page.
func (s *Service) PreviewInvitation(token string) (*models.Invitation, error) {
	if !strings.HasPrefix(token, auth.InvitationPrefix) {
		return nil, ErrInvitationInvalid
	}
	inv, err := s.repo.GetInvitationByTokenHash(auth.HashToken(token))
	if err != nil || inv.State != "pending" || inv.ExpiresAt.Before(time.Now().UTC()) {
		return nil, ErrInvitationInvalid
	}
	return inv, nil
}

// AcceptInvitation joins the user to the inviting team. The signed-in user's
// email must match the invited address.
func (s *Service) AcceptInvitation(token, userID, ip string) (*models.Invitation, error) {
	inv, err := s.PreviewInvitation(token)
	if err != nil {
		return nil, err
	}
	return inv, s.acceptInvitation(inv, userID, ip)
}

// RespondToInvitation accepts or declines a pending invitation addressed to the user.
func (s *Service) RespondToInvitation(invID, userID string, accept bool, ip string) error {
	u, err := s.repo.GetUserByID(userID)
	if err != nil {
		return ErrUnauthorized
	}
	invs, err := s.repo.ListPendingInvitationsForEmail(u.Email)
	if err != nil {
		return err
	}
	for i := range invs {
		if invs[i].ID != invID {
			continue
		}
		if accept {
			return s.acceptInvitation(&invs[i], userID, ip)
		}
		if err := s.repo.SetInvitationState(invID, invs[i].TeamID, "declined"); err != nil {
			return err
		}
		s.audit(Actor{UserID: userID, TeamID: invs[i].TeamID, IP: ip}, "member.invite_decline", "invitation", invID, nil)
		return nil
	}
	return ErrInvitationInvalid
}

func (s *Service) acceptInvitation(inv *models.Invitation, userID, ip string) error {
	u, err := s.repo.GetUserByID(userID)
	if err != nil {
		return ErrUnauthorized
	}
	if !strings.EqualFold(u.Email, inv.Email) {
		return ErrInvitationMismatch
	}
	if _, err := s.repo.GetMembership(inv.TeamID, userID); err == nil {
		_ = s.repo.SetInvitationState(inv.ID, inv.TeamID, "accepted")
		return ErrAlreadyMember
	}
	if err := s.repo.AddTeamMember(inv.TeamID, userID, inv.Role); err != nil {
		return err
	}
	if err := s.repo.SetInvitationState(inv.ID, inv.TeamID, "accepted"); err != nil {
		return err
	}
	s.audit(Actor{UserID: userID, TeamID: inv.TeamID, Role: inv.Role, IP: ip}, "member.join", "user", userID, map[string]any{"role": inv.Role})
	return nil
}

// ----------------- Audit Log -----------------

func (s *Service) audit(a Actor, action, targetType, targetID string, meta map[string]any) {
	if a.TeamID == "" {
		return
	}
	metadata := "{}"
	if len(meta) > 0 {
		if b, err := json.Marshal(meta); err == nil {
			metadata = string(b)
		}
	}
	_ = s.repo.RecordAudit(&models.AuditEvent{
		TeamID: a.TeamID, ActorID: a.UserID, Action: action,
		TargetType: targetType, TargetID: targetID, Metadata: metadata, IP: a.IP,
	})
}

// Audit records an action performed outside the service layer (e.g. profile changes).
func (s *Service) Audit(a Actor, action, targetType, targetID string, meta map[string]any) {
	s.audit(a, action, targetType, targetID, meta)
}

func (s *Service) ListAudit(a Actor, limit, offset int) ([]models.AuditEvent, error) {
	if err := a.require(auth.PermViewAudit); err != nil {
		return nil, err
	}
	return s.repo.ListAudit(a.TeamID, clampLimit(limit), max(offset, 0))
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}

// IsNotFound reports whether err represents a missing or out-of-team record.
func IsNotFound(err error) bool {
	return errors.Is(err, repository.ErrNotFound) || errors.Is(err, ErrLinkNotFound) ||
		errors.Is(err, ErrQRNotFound) || errors.Is(err, ErrDocNotFound)
}
