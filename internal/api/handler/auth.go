package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/bcrowe306/linksy/internal/config"
)

type AuthHandler struct {
	cfg        *config.Config
	httpClient *http.Client
}

func NewAuthHandler(cfg *config.Config) *AuthHandler {
	return &AuthHandler{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// GetAuthConfig returns the current authentication state and WorkOS login URL
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
	})
}

// HandleLogin redirects the browser directly to WorkOS AuthKit
func (a *AuthHandler) HandleLogin(c fiber.Ctx) error {
	if a.cfg.WorkOSClient == "" || a.cfg.WorkOSAPIKey == "" {
		return c.Redirect().To("/")
	}

	v := url.Values{}
	v.Set("client_id", a.cfg.WorkOSClient)
	v.Set("redirect_uri", a.cfg.WorkOSRedirectURI)
	v.Set("response_type", "code")
	v.Set("provider", "authkit")
	targetURL := fmt.Sprintf("https://api.workos.com/user_management/authorize?%s", v.Encode())

	return c.Redirect().To(targetURL)
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

// HandleCallback processes the authorization code from WorkOS
func (a *AuthHandler) HandleCallback(c fiber.Ctx) error {
	code := c.Query("code")
	if code == "" {
		errDesc := c.Query("error_description", "Missing authorization code")
		return c.Redirect().To(fmt.Sprintf("/?auth_error=%s", url.QueryEscape(errDesc)))
	}

	reqBody := WorkOSAuthRequest{
		ClientID:     a.cfg.WorkOSClient,
		ClientSecret: a.cfg.WorkOSAPIKey,
		GrantType:    "authorization_code",
		Code:         code,
	}

	jsonBytes, err := json.Marshal(reqBody)
	if err != nil {
		return c.Redirect().To("/?auth_error=internal_error")
	}

	resp, err := a.httpClient.Post(
		"https://api.workos.com/user_management/authenticate",
		"application/json",
		bytes.NewBuffer(jsonBytes),
	)
	if err != nil {
		return c.Redirect().To(fmt.Sprintf("/?auth_error=%s", url.QueryEscape(err.Error())))
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		var errResp WorkOSAuthResponse
		_ = json.Unmarshal(respBody, &errResp)
		msg := errResp.ErrorDesc
		if msg == "" {
			msg = fmt.Sprintf("WorkOS error (%d)", resp.StatusCode)
		}
		return c.Redirect().To(fmt.Sprintf("/?auth_error=%s", url.QueryEscape(msg)))
	}

	var authResp WorkOSAuthResponse
	if err := json.Unmarshal(respBody, &authResp); err != nil {
		return c.Redirect().To("/?auth_error=invalid_response")
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
		// If WorkOS not configured or no token, return demo profile
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

	// Query WorkOS for user details if token is present
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
				"name":  fmt.Sprintf("%s %s", user.FirstName, user.LastName),
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
