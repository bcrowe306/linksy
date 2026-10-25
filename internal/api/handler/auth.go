package handler

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/bcrowe306/linksy/internal/api/middleware"
	"github.com/bcrowe306/linksy/internal/auth"
	"github.com/bcrowe306/linksy/internal/config"
	"github.com/bcrowe306/linksy/internal/models"
	"github.com/bcrowe306/linksy/internal/service"
)

const (
	oauthStateCookie    = "lynqr_oauth_state"
	oauthRedirectCookie = "lynqr_oauth_next"
	minPasswordLength   = 8
)

type AuthHandler struct {
	cfg    *config.Config
	svc    *service.Service
	workos *auth.WorkOS
}

func NewAuthHandler(cfg *config.Config, svc *service.Service) *AuthHandler {
	return &AuthHandler{cfg: cfg, svc: svc, workos: auth.NewWorkOS(cfg.WorkOSAPIKey, cfg.WorkOSClient)}
}

func (a *AuthHandler) ctx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 15*time.Second)
}

func (a *AuthHandler) devLoginEnabled() bool {
	return a.cfg.DevLogin && a.cfg.IsLocal()
}

// redirectURI derives the OAuth callback for the host the user is on, so
// lynqr.app and test.lynqr.app each round-trip to themselves.
func (a *AuthHandler) redirectURI(c fiber.Ctx) string {
	host := c.Get("X-Forwarded-Host")
	if host == "" {
		host = c.Host()
	}
	host = strings.TrimSpace(strings.Split(host, ",")[0])
	if host == "" || strings.HasPrefix(host, "localhost") || strings.HasPrefix(host, "127.0.0.1") {
		if a.cfg.WorkOSRedirectURI != "" {
			return a.cfg.WorkOSRedirectURI
		}
	}
	// Anything that isn't loopback is served over TLS (Render/Cloudflare terminate it).
	proto := "https"
	if strings.HasPrefix(host, "localhost") || strings.HasPrefix(host, "127.0.0.1") {
		proto = "http"
	}
	return proto + "://" + host + "/api/v1/auth/callback"
}

// safeNext only allows same-site relative redirect targets.
func safeNext(next string) string {
	if next == "" || !strings.HasPrefix(next, "/") || strings.HasPrefix(next, "//") || strings.Contains(next, "\\") {
		return "/"
	}
	return next
}

func (a *AuthHandler) setCookie(c fiber.Ctx, name, value string, ttl time.Duration) {
	c.Cookie(&fiber.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		Expires:  time.Now().Add(ttl),
		MaxAge:   int(ttl.Seconds()),
		HTTPOnly: true,
		Secure:   a.cfg.SecureCookies(),
		SameSite: fiber.CookieSameSiteLaxMode,
	})
}

func (a *AuthHandler) clearCookie(c fiber.Ctx, name string) {
	c.Cookie(&fiber.Cookie{
		Name: name, Value: "", Path: "/", Expires: time.Unix(0, 0), MaxAge: -1,
		HTTPOnly: true, Secure: a.cfg.SecureCookies(), SameSite: fiber.CookieSameSiteLaxMode,
	})
}

func userFromWorkOS(w auth.WorkOSUser) *models.User {
	return &models.User{ID: w.ID, Email: w.Email, FirstName: w.FirstName, LastName: w.LastName, AvatarURL: w.ProfilePictureURL}
}

// establishSession issues a session cookie for an authenticated identity.
func (a *AuthHandler) establishSession(c fiber.Ctx, u *models.User) error {
	token, _, err := a.svc.StartSession(u, middleware.ClientIP(c), c.Get("User-Agent"))
	if err != nil {
		return err
	}
	a.setCookie(c, middleware.SessionCookie, token, service.SessionTTL)
	return nil
}

func (a *AuthHandler) sessionJSON(c fiber.Ctx, u *models.User) error {
	if err := a.establishSession(c, u); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"authenticated": true, "user": publicUser(u)})
}

func publicUser(u *models.User) fiber.Map {
	return fiber.Map{
		"id":         u.ID,
		"email":      u.Email,
		"first_name": u.FirstName,
		"last_name":  u.LastName,
		"name":       strings.TrimSpace(u.FirstName + " " + u.LastName),
		"avatar_url": u.AvatarURL,
		"created_at": u.CreatedAt,
	}
}

// authFailure converts WorkOS errors into client responses, surfacing the
// email-verification step so the UI can prompt for the emailed code.
func authFailure(c fiber.Ctx, err error, fallbackStatus int) error {
	var we *auth.WorkOSError
	if errors.As(err, &we) {
		if we.IsCode("email_verification_required") {
			return c.Status(fiber.StatusAccepted).JSON(fiber.Map{
				"verification_required": true,
				"pending_token":         we.PendingAuthenticationToken,
				"email":                 we.Email,
				"message":               "We emailed you a verification code. Enter it to finish signing in.",
			})
		}
		msg := we.Error()
		if we.IsCode("invalid_credentials") || we.Status == fiber.StatusUnauthorized {
			msg = "Invalid email or password"
		}
		return c.Status(fallbackStatus).JSON(fiber.Map{"error": msg, "code": we.Code})
	}
	if errors.Is(err, auth.ErrWorkOSDisabled) {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "Authentication service is not configured"})
	}
	return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": err.Error()})
}

// GetAuthConfig returns the available sign-in methods.
func (a *AuthHandler) GetAuthConfig(c fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"enabled":   a.workos.Enabled(),
		"dev_login": a.devLoginEnabled(),
		"providers": []string{"email_password", "google", "microsoft", "github", "authkit"},
	})
}

// HandleLogin redirects the browser to WorkOS AuthKit
func (a *AuthHandler) HandleLogin(c fiber.Ctx) error {
	return a.HandleOAuthLogin(c)
}

// HandleOAuthLogin starts an OAuth / AuthKit flow with a CSRF state cookie.
func (a *AuthHandler) HandleOAuthLogin(c fiber.Ctx) error {
	if !a.workos.Enabled() {
		return c.Redirect().To("/login?auth_error=" + url.QueryEscape("Single sign-on is not configured"))
	}

	provider := "authkit"
	switch strings.ToLower(c.Params("provider", "authkit")) {
	case "google", "googleoauth":
		provider = "GoogleOAuth"
	case "microsoft", "microsoftoauth":
		provider = "MicrosoftOAuth"
	case "github", "githuboauth":
		provider = "GitHubOAuth"
	}

	state, err := auth.NewToken("st_")
	if err != nil {
		return err
	}
	a.setCookie(c, oauthStateCookie, state, 10*time.Minute)
	a.setCookie(c, oauthRedirectCookie, safeNext(c.Query("redirect")), 10*time.Minute)

	hint := c.Query("screen_hint")
	if hint != "sign-in" && hint != "sign-up" {
		hint = ""
	}
	return c.Redirect().To(a.workos.AuthorizationURL(provider, a.redirectURI(c), state, hint))
}

// HandleCallback exchanges the authorization code and starts a session.
func (a *AuthHandler) HandleCallback(c fiber.Ctx) error {
	fail := func(msg string) error {
		return c.Redirect().To("/login?auth_error=" + url.QueryEscape(msg))
	}
	if e := c.Query("error_description", c.Query("error")); e != "" {
		return fail(e)
	}
	code := c.Query("code")
	if code == "" {
		return fail("Missing authorization code")
	}
	expected := c.Cookies(oauthStateCookie)
	if expected == "" || c.Query("state") != expected {
		return fail("Your sign-in link expired. Please try again.")
	}
	next := safeNext(c.Cookies(oauthRedirectCookie))
	a.clearCookie(c, oauthStateCookie)
	a.clearCookie(c, oauthRedirectCookie)

	ctx, cancel := a.ctx()
	defer cancel()
	res, err := a.workos.AuthenticateWithCode(ctx, code, middleware.ClientIP(c), c.Get("User-Agent"))
	if err != nil {
		return fail(err.Error())
	}
	if err := a.establishSession(c, userFromWorkOS(res.User)); err != nil {
		return fail("Could not start your session")
	}
	return c.Redirect().To(next)
}

type SignUpRequest struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
	Password  string `json:"password"`
}

// HandleSignUp creates a WorkOS user. If the environment requires email
// verification, the response asks the client for the emailed code.
func (a *AuthHandler) HandleSignUp(c fiber.Ctx) error {
	var req SignUpRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if req.Email == "" || req.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Email and password are required"})
	}
	if len(req.Password) < minPasswordLength {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Password must be at least 8 characters long"})
	}

	ctx, cancel := a.ctx()
	defer cancel()
	if _, err := a.workos.CreateUser(ctx, req.Email, req.Password, strings.TrimSpace(req.FirstName), strings.TrimSpace(req.LastName)); err != nil {
		return authFailure(c, err, fiber.StatusBadRequest)
	}
	res, err := a.workos.AuthenticateWithPassword(ctx, req.Email, req.Password, middleware.ClientIP(c), c.Get("User-Agent"))
	if err != nil {
		return authFailure(c, err, fiber.StatusUnauthorized)
	}
	return a.sessionJSON(c, userFromWorkOS(res.User))
}

type PasswordLoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (a *AuthHandler) HandlePasswordLogin(c fiber.Ctx) error {
	var req PasswordLoginRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if req.Email == "" || req.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Email and password are required"})
	}

	ctx, cancel := a.ctx()
	defer cancel()
	res, err := a.workos.AuthenticateWithPassword(ctx, req.Email, req.Password, middleware.ClientIP(c), c.Get("User-Agent"))
	if err != nil {
		return authFailure(c, err, fiber.StatusUnauthorized)
	}
	return a.sessionJSON(c, userFromWorkOS(res.User))
}

// HandleVerifyEmail completes sign-in with the code WorkOS emailed the user.
func (a *AuthHandler) HandleVerifyEmail(c fiber.Ctx) error {
	var req struct {
		PendingToken string `json:"pending_token"`
		Code         string `json:"code"`
	}
	if err := c.Bind().Body(&req); err != nil || req.PendingToken == "" || strings.TrimSpace(req.Code) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Verification code is required"})
	}
	ctx, cancel := a.ctx()
	defer cancel()
	res, err := a.workos.AuthenticateWithEmailVerification(ctx, req.PendingToken, strings.TrimSpace(req.Code), middleware.ClientIP(c), c.Get("User-Agent"))
	if err != nil {
		return authFailure(c, err, fiber.StatusBadRequest)
	}
	return a.sessionJSON(c, userFromWorkOS(res.User))
}

// HandleDevLogin signs in without WorkOS for local development only.
func (a *AuthHandler) HandleDevLogin(c fiber.Ctx) error {
	if !a.devLoginEnabled() {
		return c.SendStatus(fiber.StatusNotFound)
	}
	var req struct {
		Email     string `json:"email"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
	}
	if err := c.Bind().Body(&req); err != nil || !strings.Contains(req.Email, "@") {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "A valid email is required"})
	}
	email := strings.ToLower(strings.TrimSpace(req.Email))
	u := &models.User{ID: "user_dev_" + auth.HashToken(email)[:20], Email: email, FirstName: req.FirstName, LastName: req.LastName}
	if existing, err := a.svc.GetUserProfile(u.ID); err == nil {
		u = existing
	}
	return a.sessionJSON(c, u)
}

// GetCurrentUser reports the session's user, active team, role and permissions.
func (a *AuthHandler) GetCurrentUser(c fiber.Ctx) error {
	ac, err := middleware.Authenticate(c, a.svc)
	if err != nil {
		resp := fiber.Map{"authenticated": false}
		if errors.Is(err, service.ErrSuspended) || errors.Is(err, service.ErrNotMember) {
			resp["error"] = err.Error()
		}
		return c.JSON(resp)
	}
	u, err := a.svc.GetUserProfile(ac.UserID)
	if err != nil {
		return c.JSON(fiber.Map{"authenticated": false})
	}
	teams, _ := a.svc.ListUserTeams(ac.UserID)
	invites, _ := a.svc.ListMyInvitations(u.Email)
	var active *models.TeamMembership
	for i := range teams {
		if teams[i].ID == ac.TeamID {
			active = &teams[i]
		}
	}
	return c.JSON(fiber.Map{
		"authenticated":       true,
		"user":                publicUser(u),
		"team":                active,
		"role":                ac.Role,
		"permissions":         auth.Permissions(ac.Role),
		"teams":               teams,
		"pending_invitations": len(invites),
	})
}

// HandleLogout revokes the server-side session and clears the cookie.
func (a *AuthHandler) HandleLogout(c fiber.Ctx) error {
	if ac, err := middleware.Authenticate(c, a.svc); err == nil && ac.Session != nil {
		_ = a.svc.EndSession(ac.Session.ID, ac.UserID)
	}
	a.clearCookie(c, middleware.SessionCookie)
	return c.JSON(fiber.Map{"success": true})
}

// ----------------- Password Management -----------------

// HandlePasswordReset emails a reset link. It always returns the same response
// so it can't be used to discover which emails have accounts.
func (a *AuthHandler) HandlePasswordReset(c fiber.Ctx) error {
	var req models.PasswordResetRequest
	_ = c.Bind().Body(&req)
	email := strings.TrimSpace(strings.ToLower(req.Email))
	if email == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Email address is required for password reset"})
	}
	if !a.workos.Enabled() {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "Authentication provider not configured"})
	}
	ctx, cancel := a.ctx()
	defer cancel()
	_, _ = a.workos.CreatePasswordReset(ctx, email)
	return c.JSON(fiber.Map{"message": "If an account exists for that email, a password reset link is on its way."})
}

// HandlePasswordResetConfirm sets a new password using the emailed reset token.
func (a *AuthHandler) HandlePasswordResetConfirm(c fiber.Ctx) error {
	var req struct {
		Token       string `json:"token"`
		NewPassword string `json:"new_password"`
	}
	if err := c.Bind().Body(&req); err != nil || req.Token == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Reset token is required"})
	}
	if len(req.NewPassword) < minPasswordLength {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Password must be at least 8 characters long"})
	}
	ctx, cancel := a.ctx()
	defer cancel()
	if _, err := a.workos.ConfirmPasswordReset(ctx, req.Token, req.NewPassword); err != nil {
		return authFailure(c, err, fiber.StatusBadRequest)
	}
	return c.JSON(fiber.Map{"message": "Password updated. You can now sign in."})
}

// HandleChangePassword re-verifies the current password before setting a new one.
func (a *AuthHandler) HandleChangePassword(c fiber.Ctx) error {
	ac := middleware.GetAuth(c)
	var req models.ChangePasswordRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	if len(req.NewPassword) < minPasswordLength {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "New password must be at least 8 characters long"})
	}
	if req.CurrentPassword == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Current password is required"})
	}
	u, err := a.svc.GetUserProfile(ac.UserID)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	ctx, cancel := a.ctx()
	defer cancel()
	if _, err := a.workos.AuthenticateWithPassword(ctx, u.Email, req.CurrentPassword, middleware.ClientIP(c), c.Get("User-Agent")); err != nil {
		var we *auth.WorkOSError
		if errors.As(err, &we) && !we.IsCode("email_verification_required") {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Current password is incorrect"})
		}
		if !errors.As(err, &we) {
			return authFailure(c, err, fiber.StatusBadGateway)
		}
	}
	if err := a.workos.UpdateUser(ctx, ac.UserID, map[string]any{"password": req.NewPassword}); err != nil {
		return authFailure(c, err, fiber.StatusBadRequest)
	}
	if ac.Session != nil {
		_ = a.svc.EndOtherSessions(ac.UserID, ac.Session.ID)
	}
	a.svc.Audit(middleware.Actor(c), "user.password_change", "user", ac.UserID, nil)
	return c.JSON(fiber.Map{"message": "Password updated successfully. Other sessions were signed out."})
}

// ----------------- Profile & Sessions -----------------

func (a *AuthHandler) GetProfile(c fiber.Ctx) error {
	ac := middleware.GetAuth(c)
	u, err := a.svc.GetUserProfile(ac.UserID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Profile not found"})
	}
	pu := publicUser(u)
	pu["role"] = ac.Role
	return c.JSON(fiber.Map{"user": pu})
}

func (a *AuthHandler) UpdateProfile(c fiber.Ctx) error {
	ac := middleware.GetAuth(c)
	var req models.UpdateProfileRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	req.FirstName = strings.TrimSpace(req.FirstName)
	req.LastName = strings.TrimSpace(req.LastName)
	if len(req.FirstName) > 80 || len(req.LastName) > 80 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Names must be 80 characters or fewer"})
	}
	if req.AvatarURL != "" && !strings.HasPrefix(req.AvatarURL, "https://") && !strings.HasPrefix(req.AvatarURL, "data:image/") {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Avatar must be an https URL or an uploaded image"})
	}
	if len(req.AvatarURL) > 512*1024 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Avatar image is too large (max 384KB)"})
	}

	u, err := a.svc.GetUserProfile(ac.UserID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Profile not found"})
	}
	u.FirstName, u.LastName, u.AvatarURL = req.FirstName, req.LastName, req.AvatarURL
	if err := a.svc.UpdateUserProfile(u); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	if a.workos.Enabled() && !strings.HasPrefix(u.ID, "user_dev_") {
		ctx, cancel := a.ctx()
		defer cancel()
		_ = a.workos.UpdateUser(ctx, u.ID, map[string]any{"first_name": u.FirstName, "last_name": u.LastName})
	}
	return c.JSON(fiber.Map{"message": "Profile updated successfully", "user": publicUser(u)})
}

func (a *AuthHandler) ListSessions(c fiber.Ctx) error {
	ac := middleware.GetAuth(c)
	sessions, err := a.svc.ListSessions(ac.UserID, ac.Session.ID)
	if err != nil {
		return fail(c, err)
	}
	return c.JSON(fiber.Map{"data": sessions})
}

func (a *AuthHandler) RevokeSession(c fiber.Ctx) error {
	ac := middleware.GetAuth(c)
	if err := a.svc.EndSession(c.Params("id"), ac.UserID); err != nil {
		return fail(c, err)
	}
	return c.JSON(fiber.Map{"message": "Session revoked"})
}

func (a *AuthHandler) RevokeOtherSessions(c fiber.Ctx) error {
	ac := middleware.GetAuth(c)
	if err := a.svc.EndOtherSessions(ac.UserID, ac.Session.ID); err != nil {
		return fail(c, err)
	}
	return c.JSON(fiber.Map{"message": "Signed out of all other sessions"})
}

// ----------------- API Keys -----------------

func (a *AuthHandler) ListAPIKeys(c fiber.Ctx) error {
	keys, err := a.svc.ListAPIKeys(middleware.Actor(c))
	if err != nil {
		return fail(c, err)
	}
	return c.JSON(fiber.Map{"api_keys": keys})
}

func (a *AuthHandler) CreateAPIKey(c fiber.Ctx) error {
	var req models.CreateAPIKeyRequest
	_ = c.Bind().Body(&req)
	if req.ExpiresInDays < 0 || req.ExpiresInDays > 3650 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "expires_in_days must be between 0 and 3650"})
	}
	key, err := a.svc.CreateAPIKey(middleware.Actor(c), req.Name, req.ExpiresInDays)
	if err != nil {
		return fail(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "API key generated successfully. Store this key safely as it will not be shown again.",
		"api_key": key,
	})
}

func (a *AuthHandler) RevokeAPIKey(c fiber.Ctx) error {
	if err := a.svc.DeleteAPIKey(middleware.Actor(c), c.Params("id")); err != nil {
		return fail(c, err)
	}
	return c.JSON(fiber.Map{"message": "API key revoked successfully"})
}
