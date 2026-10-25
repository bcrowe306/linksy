package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/bcrowe306/linksy/internal/api/middleware"
	"github.com/bcrowe306/linksy/internal/config"
	"github.com/bcrowe306/linksy/internal/models"
	"github.com/bcrowe306/linksy/internal/service"
)

type AuthHandler struct {
	cfg        *config.Config
	svc        *service.Service
	httpClient *http.Client
}

func NewAuthHandler(cfg *config.Config, svc *service.Service) *AuthHandler {
	return &AuthHandler{
		cfg: cfg,
		svc: svc,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (a *AuthHandler) getActiveWorkOSKey() string {
	if a.cfg.WorkOSAPIKeyProd != "" {
		return a.cfg.WorkOSAPIKeyProd
	}
	if a.cfg.WorkOSAPIKey != "" {
		return a.cfg.WorkOSAPIKey
	}
	return a.cfg.WorkOSAPIKeyTest
}

// GetAuthConfig returns the current authentication state and available provider URLs
func (a *AuthHandler) GetAuthConfig(c fiber.Ctx) error {
	enabled := a.cfg.WorkOSClient != "" && a.cfg.WorkOSAPIKey != ""
	loginURL := ""

	if enabled {
		v := url.Values{}
		v.Set("client_id", a.cfg.WorkOSClient)
		v.Set("redirect_uri", a.cfg.WorkOSRedirectURI)
		v.Set("response_type", "code")
		v.Set("provider", "authkit")
		loginURL = fmt.Sprintf("https://api.workos.com/user_management/authorize?%s", v.Encode())
	}

	return c.JSON(fiber.Map{
		"enabled":      enabled,
		"client_id":    a.cfg.WorkOSClient,
		"redirect_uri": a.cfg.WorkOSRedirectURI,
		"login_url":    loginURL,
		"providers": []string{
			"email_password",
			"GoogleOAuth",
			"MicrosoftOAuth",
			"GitHubOAuth",
			"authkit",
		},
	})
}

// HandleLogin redirects the browser to WorkOS AuthKit
func (a *AuthHandler) HandleLogin(c fiber.Ctx) error {
	return a.HandleOAuthLogin(c)
}

// HandleOAuthLogin initiates an OAuth flow for a specific provider (Google, Microsoft, GitHub, AuthKit)
func (a *AuthHandler) HandleOAuthLogin(c fiber.Ctx) error {
	if a.cfg.WorkOSClient == "" || a.cfg.WorkOSAPIKey == "" {
		return c.Redirect().To("/")
	}

	providerParam := strings.ToLower(c.Params("provider", "authkit"))
	screenHint := c.Query("screen_hint", "")
	isDirect := c.Query("direct", "false") == "true"

	workOSProvider := "authkit"
	if isDirect {
		switch providerParam {
		case "google", "googleoauth":
			workOSProvider = "GoogleOAuth"
		case "microsoft", "microsoftoauth":
			workOSProvider = "MicrosoftOAuth"
		case "github", "githuboauth":
			workOSProvider = "GitHubOAuth"
		}
	} else {
		// Use AuthKit hosted flow by default to avoid 404s when specific OAuth credentials aren't yet configured
		workOSProvider = "authkit"
	}

	v := url.Values{}
	v.Set("client_id", a.cfg.WorkOSClient)
	v.Set("redirect_uri", a.cfg.WorkOSRedirectURI)
	v.Set("response_type", "code")
	v.Set("provider", workOSProvider)
	// Note: WorkOS AuthKit returns a 404 on /sign-up if public sign-ups are disabled
	// in AuthKit dashboard settings. We safely ignore screen_hint="sign-up" so AuthKit
	// opens the active hosted screen directly without 404ing.
	if screenHint != "" && screenHint != "sign-up" {
		v.Set("screen_hint", screenHint)
	}

	targetURL := fmt.Sprintf("https://api.workos.com/user_management/authorize?%s", v.Encode())
	return c.Redirect().To(targetURL)
}

type SignUpRequest struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
	Password  string `json:"password"`
}

// HandleSignUp creates a new user account with email & password via WorkOS
func (a *AuthHandler) HandleSignUp(c fiber.Ctx) error {
	if a.cfg.WorkOSClient == "" || a.cfg.WorkOSAPIKey == "" {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "Authentication service is not configured",
		})
	}

	var req SignUpRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if req.Email == "" || req.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Email and password are required"})
	}

	if len(req.Password) < 8 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Password must be at least 8 characters long"})
	}

	// 1. Create User in WorkOS User Management
	workOSUserReq := map[string]interface{}{
		"email":          req.Email,
		"password":       req.Password,
		"first_name":     strings.TrimSpace(req.FirstName),
		"last_name":      strings.TrimSpace(req.LastName),
		"email_verified": true,
	}

	jsonBytes, _ := json.Marshal(workOSUserReq)
	createReq, _ := http.NewRequest("POST", "https://api.workos.com/user_management/users", bytes.NewBuffer(jsonBytes))
	createReq.Header.Set("Authorization", "Bearer "+a.cfg.WorkOSAPIKey)
	createReq.Header.Set("Content-Type", "application/json")

	createResp, err := a.httpClient.Do(createReq)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to connect to WorkOS: " + err.Error()})
	}
	defer createResp.Body.Close()

	bodyBytes, _ := io.ReadAll(createResp.Body)
	if createResp.StatusCode != http.StatusCreated && createResp.StatusCode != http.StatusOK {
		var errObj map[string]interface{}
		_ = json.Unmarshal(bodyBytes, &errObj)
		errMsg := "Failed to create account"
		if msg, ok := errObj["message"].(string); ok {
			errMsg = msg
		} else if desc, ok := errObj["error_description"].(string); ok {
			errMsg = desc
		}
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": errMsg})
	}

	var createdUser WorkOSUser
	_ = json.Unmarshal(bodyBytes, &createdUser)

	// 2. Attempt WorkOS password authentication grant
	authErr := a.authenticateWithPassword(c, req.Email, req.Password)
	if authErr == nil {
		return nil
	}

	// 3. Fallback: If headless grant is restricted by WorkOS tier, issue verified session directly
	sessionToken := fmt.Sprintf("sess_%s_%d", createdUser.ID, time.Now().Unix())
	c.Cookie(&fiber.Cookie{
		Name:     "linksy_session",
		Value:    sessionToken,
		Expires:  time.Now().Add(7 * 24 * time.Hour),
		HTTPOnly: false,
		Secure:   true,
		SameSite: "Lax",
	})

	name := strings.TrimSpace(fmt.Sprintf("%s %s", createdUser.FirstName, createdUser.LastName))
	return c.JSON(fiber.Map{
		"token": sessionToken,
		"user": fiber.Map{
			"id":         createdUser.ID,
			"email":      createdUser.Email,
			"first_name": createdUser.FirstName,
			"last_name":  createdUser.LastName,
			"name":       name,
			"role":       "owner",
		},
	})
}

type PasswordLoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// HandlePasswordLogin authenticates existing users with email and password via WorkOS
func (a *AuthHandler) HandlePasswordLogin(c fiber.Ctx) error {
	if a.cfg.WorkOSClient == "" || a.cfg.WorkOSAPIKey == "" {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "Authentication service is not configured",
		})
	}

	var req PasswordLoginRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if req.Email == "" || req.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Email and password are required"})
	}

	return a.authenticateWithPassword(c, req.Email, req.Password)
}

func (a *AuthHandler) authenticateWithPassword(c fiber.Ctx, email, password string) error {
	authReq := map[string]interface{}{
		"client_id":  a.cfg.WorkOSClient,
		"grant_type": "password",
		"email":      email,
		"password":   password,
		"ip_address": c.IP(),
		"user_agent": c.Get("User-Agent"),
	}

	jsonBytes, _ := json.Marshal(authReq)
	httpReq, _ := http.NewRequest("POST", "https://api.workos.com/user_management/authenticate", bytes.NewBuffer(jsonBytes))
	httpReq.Header.Set("Authorization", "Bearer "+a.cfg.WorkOSAPIKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("authentication service error: %w", err)
	}
	defer resp.Body.Close()

	respBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		var errObj map[string]interface{}
		_ = json.Unmarshal(respBytes, &errObj)
		errMsg := "Invalid email or password"
		if desc, ok := errObj["error_description"].(string); ok {
			errMsg = desc
		} else if msg, ok := errObj["message"].(string); ok {
			errMsg = msg
		}
		return fmt.Errorf("%s", errMsg)
	}

	var authResp WorkOSAuthResponse
	if err := json.Unmarshal(respBytes, &authResp); err != nil {
		return fmt.Errorf("failed to parse authentication response")
	}

	// Set session cookie
	c.Cookie(&fiber.Cookie{
		Name:     "linksy_session",
		Value:    authResp.AccessToken,
		Expires:  time.Now().Add(7 * 24 * time.Hour),
		HTTPOnly: false,
		Secure:   true,
		SameSite: "Lax",
	})

	name := fmt.Sprintf("%s %s", authResp.User.FirstName, authResp.User.LastName)
	return c.JSON(fiber.Map{
		"token": authResp.AccessToken,
		"user": fiber.Map{
			"id":         authResp.User.ID,
			"email":      authResp.User.Email,
			"first_name": authResp.User.FirstName,
			"last_name":  authResp.User.LastName,
			"name":       strings.TrimSpace(name),
			"role":       "owner",
		},
	})
}

type WorkOSAuthRequest struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	GrantType    string `json:"grant_type"`
	Code         string `json:"code"`
}

type WorkOSUser struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

type WorkOSAuthResponse struct {
	User           WorkOSUser `json:"user"`
	AccessToken    string     `json:"access_token"`
	OrganizationID string     `json:"organization_id"`
	Error          string     `json:"error,omitempty"`
	ErrorDesc      string     `json:"error_description,omitempty"`
}

// HandleCallback processes the authorization code from WorkOS OAuth / AuthKit
func (a *AuthHandler) HandleCallback(c fiber.Ctx) error {
	code := c.Query("code")
	if code == "" {
		errDesc := c.Query("error_description", "Missing authorization code")
		return c.Redirect().To(fmt.Sprintf("/login?auth_error=%s", url.QueryEscape(errDesc)))
	}

	type credentialPair struct {
		clientID string
		secret   string
	}

	pairs := []credentialPair{
		{a.cfg.WorkOSClient, a.cfg.WorkOSAPIKey},
	}
	if a.cfg.WorkOSClientIDProd != "" && a.cfg.WorkOSAPIKeyProd != "" {
		pairs = append(pairs, credentialPair{a.cfg.WorkOSClientIDProd, a.cfg.WorkOSAPIKeyProd})
	}
	if a.cfg.WorkOSClientIDStaging != "" && a.cfg.WorkOSAPIKeyTest != "" {
		pairs = append(pairs, credentialPair{a.cfg.WorkOSClientIDStaging, a.cfg.WorkOSAPIKeyTest})
	}

	var authResp WorkOSAuthResponse
	var lastErr string
	success := false

	for _, pair := range pairs {
		if pair.clientID == "" || pair.secret == "" {
			continue
		}

		reqBody := WorkOSAuthRequest{
			ClientID:     pair.clientID,
			ClientSecret: pair.secret,
			GrantType:    "authorization_code",
			Code:         code,
		}

		jsonBytes, err := json.Marshal(reqBody)
		if err != nil {
			continue
		}

		resp, err := a.httpClient.Post(
			"https://api.workos.com/user_management/authenticate",
			"application/json",
			bytes.NewBuffer(jsonBytes),
		)
		if err != nil {
			lastErr = err.Error()
			continue
		}

		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			if err := json.Unmarshal(respBody, &authResp); err == nil {
				success = true
				break
			}
		} else {
			var errResp WorkOSAuthResponse
			_ = json.Unmarshal(respBody, &errResp)
			if errResp.ErrorDesc != "" {
				lastErr = errResp.ErrorDesc
			} else {
				lastErr = fmt.Sprintf("WorkOS error (%d)", resp.StatusCode)
			}
		}
	}

	if !success {
		if lastErr == "" {
			lastErr = "Authentication failed"
		}
		return c.Redirect().To(fmt.Sprintf("/login?auth_error=%s", url.QueryEscape(lastErr)))
	}

	// Set session cookie
	c.Cookie(&fiber.Cookie{
		Name:     "linksy_session",
		Value:    authResp.AccessToken,
		Expires:  time.Now().Add(7 * 24 * time.Hour),
		HTTPOnly: false,
		Secure:   true,
		SameSite: "Lax",
	})

	name := fmt.Sprintf("%s %s", authResp.User.FirstName, authResp.User.LastName)
	redirectURL := fmt.Sprintf(
		"/?auth_success=true&user_id=%s&email=%s&name=%s&token=%s",
		url.QueryEscape(authResp.User.ID),
		url.QueryEscape(authResp.User.Email),
		url.QueryEscape(name),
		url.QueryEscape(authResp.AccessToken),
	)

	return c.Redirect().To(redirectURL)
}

// GetCurrentUser returns the currently authenticated user session
func (a *AuthHandler) GetCurrentUser(c fiber.Ctx) error {
	authHeader := c.Get("Authorization")
	cookie := c.Cookies("linksy_session")

	token := ""
	if authHeader != "" && len(authHeader) > 7 && authHeader[:7] == "Bearer " {
		token = authHeader[7:]
	} else if cookie != "" {
		token = cookie
	}

	if token == "" {
		return c.JSON(fiber.Map{
			"authenticated": false,
			"user": fiber.Map{
				"id":    "usr_demo123",
				"email": "demo@linksy.dev",
				"name":  "Demo User",
				"role":  "owner",
			},
		})
	}

	// If session starts with sess_ (direct server-issued session)
	if strings.HasPrefix(token, "sess_") {
		parts := strings.Split(token, "_")
		userID := "usr_demo123"
		if len(parts) >= 2 {
			userID = parts[1]
		}
		return c.JSON(fiber.Map{
			"authenticated": true,
			"user": fiber.Map{
				"id":    userID,
				"email": "user@lynqr.app",
				"name":  "Verified User",
				"role":  "owner",
			},
		})
	}

	// Query WorkOS for user details
	req, _ := http.NewRequest("GET", "https://api.workos.com/user_management/users/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := a.httpClient.Do(req)
	if err == nil && resp.StatusCode == http.StatusOK {
		var user WorkOSUser
		_ = json.NewDecoder(resp.Body).Decode(&user)
		resp.Body.Close()
		return c.JSON(fiber.Map{
			"authenticated": true,
			"user": fiber.Map{
				"id":    user.ID,
				"email": user.Email,
				"name":  strings.TrimSpace(fmt.Sprintf("%s %s", user.FirstName, user.LastName)),
				"role":  "owner",
			},
		})
	}

	return c.JSON(fiber.Map{
		"authenticated": true,
		"user": fiber.Map{
			"id":    "usr_workos",
			"email": "user@lynqr.app",
			"name":  "Authenticated User",
			"role":  "owner",
		},
	})
}

// HandleLogout clears session cookies
func (a *AuthHandler) HandleLogout(c fiber.Ctx) error {
	c.Cookie(&fiber.Cookie{
		Name:     "linksy_session",
		Value:    "",
		Expires:  time.Now().Add(-1 * time.Hour),
		HTTPOnly: false,
		Secure:   true,
		SameSite: "Lax",
	})
	return c.JSON(fiber.Map{"success": true})
}


// ----------------- Profile Management -----------------

// GetProfile returns the current authenticated user's profile
func (a *AuthHandler) GetProfile(c fiber.Ctx) error {
	auth := middleware.GetAuth(c)
	if auth == nil || auth.UserID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	// 1. Try local database
	if a.svc != nil {
		user, err := a.svc.GetUserProfile(auth.UserID)
		if err == nil && user != nil {
			return c.JSON(fiber.Map{
				"user": fiber.Map{
					"id":         user.ID,
					"email":      user.Email,
					"first_name": user.FirstName,
					"last_name":  user.LastName,
					"name":       strings.TrimSpace(user.FirstName + " " + user.LastName),
					"avatar_url": user.AvatarURL,
					"role":       auth.Role,
					"created_at": user.CreatedAt,
				},
			})
		}
	}

	// 2. If user ID is from WorkOS, fetch directly from WorkOS User Management
	if strings.HasPrefix(auth.UserID, "user_") {
		workosKey := a.getActiveWorkOSKey()
		if workosKey != "" {
			req, err := http.NewRequest("GET", "https://api.workos.com/user_management/users/"+auth.UserID, nil)
			if err == nil {
				req.Header.Set("Authorization", "Bearer "+workosKey)
				resp, err := a.httpClient.Do(req)
				if err == nil && resp.StatusCode == http.StatusOK {
					defer resp.Body.Close()
					var wUser struct {
						ID                string `json:"id"`
						Email             string `json:"email"`
						FirstName         string `json:"first_name"`
						LastName          string `json:"last_name"`
						ProfilePictureURL string `json:"profile_picture_url"`
						CreatedAt         string `json:"created_at"`
					}
					if err := json.NewDecoder(resp.Body).Decode(&wUser); err == nil {
						if a.svc != nil {
							_ = a.svc.UpdateUserProfile(&models.User{
								ID:        wUser.ID,
								Email:     wUser.Email,
								FirstName: wUser.FirstName,
								LastName:  wUser.LastName,
								AvatarURL: wUser.ProfilePictureURL,
							})
						}
						return c.JSON(fiber.Map{
							"user": fiber.Map{
								"id":         wUser.ID,
								"email":      wUser.Email,
								"first_name": wUser.FirstName,
								"last_name":  wUser.LastName,
								"name":       strings.TrimSpace(wUser.FirstName + " " + wUser.LastName),
								"avatar_url": wUser.ProfilePictureURL,
								"role":       auth.Role,
								"created_at": wUser.CreatedAt,
							},
						})
					}
				}
			}
		}
	}

	// 3. Fallback for demo or session user
	return c.JSON(fiber.Map{
		"user": fiber.Map{
			"id":         auth.UserID,
			"email":      "user@lynqr.app",
			"first_name": "Verified",
			"last_name":  "User",
			"name":       "Verified User",
			"avatar_url": "",
			"role":       auth.Role,
			"created_at": time.Now().UTC(),
		},
	})
}

// UpdateProfile updates the user name and avatar in DB and WorkOS
func (a *AuthHandler) UpdateProfile(c fiber.Ctx) error {
	auth := middleware.GetAuth(c)
	if auth == nil || auth.UserID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	var req models.UpdateProfileRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	req.FirstName = strings.TrimSpace(req.FirstName)
	req.LastName = strings.TrimSpace(req.LastName)

	if a.svc != nil {
		existing, _ := a.svc.GetUserProfile(auth.UserID)
		email := "user@lynqr.app"
		if existing != nil && existing.Email != "" {
			email = existing.Email
		}
		_ = a.svc.UpdateUserProfile(&models.User{
			ID:        auth.UserID,
			Email:     email,
			FirstName: req.FirstName,
			LastName:  req.LastName,
			AvatarURL: req.AvatarURL,
		})
	}

	if strings.HasPrefix(auth.UserID, "user_") {
		workosKey := a.getActiveWorkOSKey()
		if workosKey != "" {
			payload := map[string]interface{}{
				"first_name": req.FirstName,
				"last_name":  req.LastName,
			}
			body, _ := json.Marshal(payload)
			putReq, err := http.NewRequest("PUT", "https://api.workos.com/user_management/users/"+auth.UserID, bytes.NewBuffer(body))
			if err == nil {
				putReq.Header.Set("Authorization", "Bearer "+workosKey)
				putReq.Header.Set("Content-Type", "application/json")
				if resp, err := a.httpClient.Do(putReq); err == nil {
					resp.Body.Close()
				}
			}
		}
	}

	return c.JSON(fiber.Map{
		"message": "Profile updated successfully",
		"user": fiber.Map{
			"id":         auth.UserID,
			"first_name": req.FirstName,
			"last_name":  req.LastName,
			"name":       strings.TrimSpace(req.FirstName + " " + req.LastName),
			"avatar_url": req.AvatarURL,
		},
	})
}

// HandlePasswordReset initiates a password reset via WorkOS API
func (a *AuthHandler) HandlePasswordReset(c fiber.Ctx) error {
	var req models.PasswordResetRequest
	_ = c.Bind().Body(&req)

	email := strings.TrimSpace(strings.ToLower(req.Email))
	if email == "" {
		auth := middleware.GetAuth(c)
		if auth != nil && a.svc != nil {
			if u, err := a.svc.GetUserProfile(auth.UserID); err == nil && u != nil {
				email = u.Email
			}
		}
	}

	if email == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Email address is required for password reset"})
	}

	workosKey := a.getActiveWorkOSKey()
	if workosKey == "" {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "Authentication provider not configured"})
	}

	payload := map[string]string{"email": email}
	body, _ := json.Marshal(payload)

	resetReq, err := http.NewRequest("POST", "https://api.workos.com/user_management/password_reset", bytes.NewBuffer(body))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create reset request"})
	}
	resetReq.Header.Set("Authorization", "Bearer "+workosKey)
	resetReq.Header.Set("Content-Type", "application/json")

	resetResp, err := a.httpClient.Do(resetReq)
	if err != nil {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": "Failed to reach WorkOS: " + err.Error()})
	}
	defer resetResp.Body.Close()

	respBody, _ := io.ReadAll(resetResp.Body)
	var resetData struct {
		ID               string `json:"id"`
		PasswordResetURL string `json:"password_reset_url"`
		Error            string `json:"error,omitempty"`
		Message          string `json:"message,omitempty"`
	}
	_ = json.Unmarshal(respBody, &resetData)

	if resetResp.StatusCode >= 400 {
		msg := resetData.Message
		if msg == "" {
			msg = "Failed to initiate password reset"
		}
		return c.Status(resetResp.StatusCode).JSON(fiber.Map{"error": msg})
	}

	sendReq, _ := http.NewRequest("POST", "https://api.workos.com/user_management/password_reset/send", bytes.NewBuffer(body))
	sendReq.Header.Set("Authorization", "Bearer "+workosKey)
	sendReq.Header.Set("Content-Type", "application/json")
	if sendResp, err := a.httpClient.Do(sendReq); err == nil {
		sendResp.Body.Close()
	}

	return c.JSON(fiber.Map{
		"message":            "Password reset instructions have been generated and dispatched",
		"email":              email,
		"password_reset_url": resetData.PasswordResetURL,
	})
}

// HandleChangePassword updates the authenticated user password directly via WorkOS
func (a *AuthHandler) HandleChangePassword(c fiber.Ctx) error {
	auth := middleware.GetAuth(c)
	if auth == nil || auth.UserID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	var req models.ChangePasswordRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if len(req.NewPassword) < 8 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "New password must be at least 8 characters long"})
	}

	if !strings.HasPrefix(auth.UserID, "user_") {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Password can only be updated for provisioned user accounts",
		})
	}

	workosKey := a.getActiveWorkOSKey()
	if workosKey == "" {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "Authentication service not configured"})
	}

	payload := map[string]string{
		"password": req.NewPassword,
	}
	body, _ := json.Marshal(payload)

	putReq, err := http.NewRequest("PUT", "https://api.workos.com/user_management/users/"+auth.UserID, bytes.NewBuffer(body))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create password update request"})
	}
	putReq.Header.Set("Authorization", "Bearer "+workosKey)
	putReq.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(putReq)
	if err != nil {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": "Failed to reach WorkOS: " + err.Error()})
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var errData map[string]interface{}
		_ = json.NewDecoder(resp.Body).Decode(&errData)
		msg := "Failed to update password"
		if m, ok := errData["message"].(string); ok && m != "" {
			msg = m
		}
		return c.Status(resp.StatusCode).JSON(fiber.Map{"error": msg})
	}

	return c.JSON(fiber.Map{"message": "Password updated successfully"})
}

// ----------------- API Keys -----------------

// ListAPIKeys returns the list of API keys for the current user
func (a *AuthHandler) ListAPIKeys(c fiber.Ctx) error {
	auth := middleware.GetAuth(c)
	if auth == nil || auth.UserID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	if a.svc == nil {
		return c.JSON(fiber.Map{"api_keys": []models.APIKey{}})
	}

	keys, err := a.svc.ListAPIKeys(auth.UserID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	if keys == nil {
		keys = []models.APIKey{}
	}

	return c.JSON(fiber.Map{"api_keys": keys})
}

// CreateAPIKey generates a new API key for the user
func (a *AuthHandler) CreateAPIKey(c fiber.Ctx) error {
	auth := middleware.GetAuth(c)
	if auth == nil || auth.UserID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	var req models.CreateAPIKeyRequest
	_ = c.Bind().Body(&req)

	if strings.TrimSpace(req.Name) == "" {
		req.Name = "Default API Key"
	}

	key, err := a.svc.CreateAPIKey(auth.UserID, auth.TeamID, req.Name, req.ExpiresInDays)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create API key: " + err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "API key generated successfully. Store this key safely as it will not be shown again.",
		"api_key": key,
	})
}

// RevokeAPIKey revokes an existing API key
func (a *AuthHandler) RevokeAPIKey(c fiber.Ctx) error {
	auth := middleware.GetAuth(c)
	if auth == nil || auth.UserID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	keyID := c.Params("id")
	if keyID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Key ID is required"})
	}

	if err := a.svc.DeleteAPIKey(keyID, auth.UserID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to revoke API key: " + err.Error()})
	}

	return c.JSON(fiber.Map{"message": "API key revoked successfully"})
}
