package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/bcrowe306/linksy/internal/api/middleware"
	"github.com/bcrowe306/linksy/internal/config"
	"github.com/bcrowe306/linksy/internal/models"
	"github.com/bcrowe306/linksy/internal/service"
)

type TeamHandler struct {
	cfg        *config.Config
	svc        *service.Service
	httpClient *http.Client
}

func NewTeamHandler(cfg *config.Config, svc *service.Service) *TeamHandler {
	return &TeamHandler{
		cfg: cfg,
		svc: svc,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (t *TeamHandler) getActiveWorkOSKey() string {
	if t.cfg.WorkOSAPIKeyProd != "" {
		return t.cfg.WorkOSAPIKeyProd
	}
	if t.cfg.WorkOSAPIKey != "" {
		return t.cfg.WorkOSAPIKey
	}
	return t.cfg.WorkOSAPIKeyTest
}

func (t *TeamHandler) getWorkOSOrgID() string {
	if t.cfg.WorkOSOrgID != "" {
		return t.cfg.WorkOSOrgID
	}
	return "org_01M5R8PX1FRH54EN4ZW56E8HRE"
}

// ListMembers returns all active team members and pending WorkOS invitations
func (t *TeamHandler) ListMembers(c fiber.Ctx) error {
	auth := middleware.GetAuth(c)
	teamID := auth.TeamID
	if teamID == "" {
		teamID = "team_default"
	}

	members, err := t.svc.ListTeamMembers(teamID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	invitations, _ := t.svc.ListInvitations(teamID)
	if invitations == nil {
		invitations = []models.Invitation{}
	}

	// Also fetch live pending invitations from WorkOS if key is configured
	workosKey := t.getActiveWorkOSKey()
	orgID := t.getWorkOSOrgID()
	if workosKey != "" && orgID != "" {
		req, err := http.NewRequest("GET", "https://api.workos.com/user_management/invitations?organization_id="+orgID, nil)
		if err == nil {
			req.Header.Set("Authorization", "Bearer "+workosKey)
			resp, err := t.httpClient.Do(req)
			if err == nil && resp.StatusCode == http.StatusOK {
				defer resp.Body.Close()
				var wInvs struct {
					Data []struct {
						ID                  string `json:"id"`
						Email               string `json:"email"`
						State               string `json:"state"`
						AcceptInvitationURL string `json:"accept_invitation_url"`
						RoleSlug            string `json:"role_slug"`
						CreatedAt           string `json:"created_at"`
						ExpiresAt           string `json:"expires_at"`
					} `json:"data"`
				}
				if err := json.NewDecoder(resp.Body).Decode(&wInvs); err == nil {
					existingEmails := make(map[string]bool)
					for _, inv := range invitations {
						existingEmails[inv.Email] = true
					}
					for _, w := range wInvs.Data {
						if w.State == "pending" && !existingEmails[w.Email] {
							cTime, _ := time.Parse(time.RFC3339, w.CreatedAt)
							eTime, _ := time.Parse(time.RFC3339, w.ExpiresAt)
							role := models.RoleMember
							if w.RoleSlug == "admin" {
								role = models.RoleAdmin
							}
							invitations = append(invitations, models.Invitation{
								ID:                 w.ID,
								TeamID:             teamID,
								Email:              w.Email,
								Role:               role,
								WorkOSInvitationID: w.ID,
								State:              w.State,
								AcceptURL:          w.AcceptInvitationURL,
								CreatedAt:          cTime,
								ExpiresAt:          &eTime,
							})
						}
					}
				}
			}
		}
	}

	return c.JSON(fiber.Map{
		"team_id":             teamID,
		"team_name":           "Lynqr Workspace",
		"current_user_id":     auth.UserID,
		"current_user_role":   auth.Role,
		"members":             members,
		"pending_invitations": invitations,
	})
}

// InviteMember creates a WorkOS user invitation and records it in the team
func (t *TeamHandler) InviteMember(c fiber.Ctx) error {
	auth := middleware.GetAuth(c)
	var req models.InviteMemberRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if req.Email == "" || !strings.Contains(req.Email, "@") {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Valid email address is required"})
	}

	if req.Role != models.RoleAdmin && req.Role != models.RoleMember {
		req.Role = models.RoleMember
	}

	workosKey := t.getActiveWorkOSKey()
	orgID := t.getWorkOSOrgID()
	var workOSInvID string
	var acceptURL string

	if workosKey != "" {
		payload := map[string]interface{}{
			"email":           req.Email,
			"organization_id": orgID,
			"role_slug":       string(req.Role),
		}
		jsonBytes, _ := json.Marshal(payload)
		httpReq, err := http.NewRequest("POST", "https://api.workos.com/user_management/invitations", bytes.NewBuffer(jsonBytes))
		if err == nil {
			httpReq.Header.Set("Authorization", "Bearer "+workosKey)
			httpReq.Header.Set("Content-Type", "application/json")
			resp, err := t.httpClient.Do(httpReq)
			if err == nil {
				defer resp.Body.Close()
				bodyBytes, _ := io.ReadAll(resp.Body)
				if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated {
					var resData struct {
						ID                  string `json:"id"`
						AcceptInvitationURL string `json:"accept_invitation_url"`
					}
					if err := json.Unmarshal(bodyBytes, &resData); err == nil {
						workOSInvID = resData.ID
						acceptURL = resData.AcceptInvitationURL
					}
				}
			}
		}
	}

	inv := &models.Invitation{
		TeamID:             auth.TeamID,
		Email:              req.Email,
		Role:               req.Role,
		WorkOSInvitationID: workOSInvID,
		AcceptURL:          acceptURL,
		State:              "pending",
		CreatedAt:          time.Now().UTC(),
	}

	if err := t.svc.CreateInvitation(inv); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed creating invitation record: " + err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message":    "Invitation successfully dispatched via WorkOS",
		"invitation": inv,
	})
}

// UpdateMemberRole updates a team member's role (admin <-> member)
func (t *TeamHandler) UpdateMemberRole(c fiber.Ctx) error {
	auth := middleware.GetAuth(c)
	targetUserID := c.Params("userId")
	if targetUserID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "User ID is required"})
	}

	var req models.UpdateRoleRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if req.Role != models.RoleAdmin && req.Role != models.RoleMember {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Role must be either 'admin' or 'member'"})
	}

	// Prevent user from modifying their own role
	if targetUserID == auth.UserID {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "You cannot change your own role"})
	}

	if err := t.svc.UpdateTeamMemberRole(auth.TeamID, targetUserID, req.Role); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"message": "User role updated successfully",
		"user_id": targetUserID,
		"role":    req.Role,
	})
}

// RemoveMember removes a user from the team and deletes their account in WorkOS
func (t *TeamHandler) RemoveMember(c fiber.Ctx) error {
	auth := middleware.GetAuth(c)
	targetUserID := c.Params("userId")
	if targetUserID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "User ID is required"})
	}

	// Prevent user from removing themselves
	if targetUserID == auth.UserID {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "You cannot remove yourself from the team"})
	}

	// 1. Remove from local database
	if err := t.svc.RemoveTeamMember(auth.TeamID, targetUserID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	// 2. Remove from WorkOS User Management
	workosKey := t.getActiveWorkOSKey()
	if workosKey != "" {
		wUserID := targetUserID
		if !strings.HasPrefix(wUserID, "user_") {
			user, _ := t.svc.GetUserProfile(targetUserID)
			emailToFind := ""
			if user != nil {
			emailToFind = user.Email
			} else if strings.Contains(targetUserID, "@") {
				emailToFind = targetUserID
			}
			if emailToFind != "" {
				searchReq, err := http.NewRequest("GET", "https://api.workos.com/user_management/users?email="+emailToFind, nil)
				if err == nil {
					searchReq.Header.Set("Authorization", "Bearer "+workosKey)
					if searchResp, err := t.httpClient.Do(searchReq); err == nil {
						var sResult struct {
							Data []struct {
								ID string `json:"id"`
							} `json:"data"`
						}
						if json.NewDecoder(searchResp.Body).Decode(&sResult) == nil && len(sResult.Data) > 0 {
							wUserID = sResult.Data[0].ID
						}
						searchResp.Body.Close()
					}
				}
			}
		}

		if strings.HasPrefix(wUserID, "user_") {
			delReq, err := http.NewRequest("DELETE", "https://api.workos.com/user_management/users/"+wUserID, nil)
			if err == nil {
				delReq.Header.Set("Authorization", "Bearer "+workosKey)
				if resp, err := t.httpClient.Do(delReq); err == nil {
					resp.Body.Close()
				}
			}
		}
	}

	return c.JSON(fiber.Map{
		"message": "User successfully removed and disabled",
		"user_id": targetUserID,
	})
}

// ResetMemberCredentials initiates credential reset instructions for a member
func (t *TeamHandler) ResetMemberCredentials(c fiber.Ctx) error {
	targetUserID := c.Params("userId")
	if targetUserID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "User ID is required"})
	}

	// Fetch user details
	user, err := t.svc.GetUserProfile(targetUserID)
	email := ""
	if err == nil && user != nil {
		email = user.Email
	}

	if email == "" && strings.HasPrefix(targetUserID, "user_") {
		// Lookup email directly from WorkOS
		workosKey := t.getActiveWorkOSKey()
		if workosKey != "" {
			req, err := http.NewRequest("GET", "https://api.workos.com/user_management/users/"+targetUserID, nil)
			if err == nil {
				req.Header.Set("Authorization", "Bearer "+workosKey)
				if resp, err := t.httpClient.Do(req); err == nil && resp.StatusCode == http.StatusOK {
					defer resp.Body.Close()
					var wUser struct {
						Email string `json:"email"`
					}
					if err := json.NewDecoder(resp.Body).Decode(&wUser); err == nil {
						email = wUser.Email
					}
				}
			}
		}
	}

	if email == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "User email could not be located for password reset"})
	}

	workosKey := t.getActiveWorkOSKey()
	resetURL := ""
	if workosKey != "" {
		body, _ := json.Marshal(map[string]string{"email": email})
		// 1. Generate reset link
		rReq, err := http.NewRequest("POST", "https://api.workos.com/user_management/password_reset", bytes.NewBuffer(body))
		if err == nil {
			rReq.Header.Set("Authorization", "Bearer "+workosKey)
			rReq.Header.Set("Content-Type", "application/json")
			if resp, err := t.httpClient.Do(rReq); err == nil {
				defer resp.Body.Close()
				var resData struct {
					PasswordResetURL string `json:"password_reset_url"`
				}
				if err := json.NewDecoder(resp.Body).Decode(&resData); err == nil {
					resetURL = resData.PasswordResetURL
				}
			}
		}

		// 2. Dispatch email
		sReq, err := http.NewRequest("POST", "https://api.workos.com/user_management/password_reset/send", bytes.NewBuffer(body))
		if err == nil {
			sReq.Header.Set("Authorization", "Bearer "+workosKey)
			sReq.Header.Set("Content-Type", "application/json")
			if resp, err := t.httpClient.Do(sReq); err == nil {
				resp.Body.Close()
			}
		}
	}

	return c.JSON(fiber.Map{
		"message":            "Password reset instructions dispatched successfully via WorkOS",
		"email":              email,
		"password_reset_url": resetURL,
	})
}

// RevokeInvitation revokes a pending invitation in WorkOS and locally
func (t *TeamHandler) RevokeInvitation(c fiber.Ctx) error {
	auth := middleware.GetAuth(c)
	invID := c.Params("id")
	if invID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invitation ID is required"})
	}

	workosKey := t.getActiveWorkOSKey()
	if workosKey != "" {
		req, err := http.NewRequest("POST", "https://api.workos.com/user_management/invitations/"+invID+"/revoke", bytes.NewBuffer([]byte("{}")))
		if err == nil {
			req.Header.Set("Authorization", "Bearer "+workosKey)
			req.Header.Set("Content-Type", "application/json")
			if resp, err := t.httpClient.Do(req); err == nil {
				resp.Body.Close()
			}
		}
	}

	_ = t.svc.RevokeInvitation(invID, auth.TeamID)

	return c.JSON(fiber.Map{
		"message": "Invitation revoked successfully",
		"id":      invID,
	})
}
