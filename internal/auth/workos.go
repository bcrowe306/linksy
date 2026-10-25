package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const workosBaseURL = "https://api.workos.com"

// WorkOS is a minimal client for the WorkOS User Management API.
type WorkOS struct {
	APIKey   string
	ClientID string
	BaseURL  string
	HTTP     *http.Client
}

func NewWorkOS(apiKey, clientID string) *WorkOS {
	return &WorkOS{APIKey: apiKey, ClientID: clientID, BaseURL: workosBaseURL, HTTP: &http.Client{Timeout: 10 * time.Second}}
}

func (w *WorkOS) Enabled() bool { return w != nil && w.APIKey != "" && w.ClientID != "" }

type WorkOSUser struct {
	ID                string `json:"id"`
	Email             string `json:"email"`
	FirstName         string `json:"first_name"`
	LastName          string `json:"last_name"`
	EmailVerified     bool   `json:"email_verified"`
	ProfilePictureURL string `json:"profile_picture_url"`
	CreatedAt         string `json:"created_at"`
}

type AuthResult struct {
	User WorkOSUser `json:"user"`
}

// WorkOSError carries the API's machine-readable code (e.g. email_verification_required).
type WorkOSError struct {
	Status                     int    `json:"-"`
	Code                       string `json:"code"`
	Error_                     string `json:"error"`
	Description                string `json:"error_description"`
	Message                    string `json:"message"`
	PendingAuthenticationToken string `json:"pending_authentication_token"`
	Email                      string `json:"email"`
}

func (e *WorkOSError) Error() string {
	for _, s := range []string{e.Message, e.Description, e.Error_, e.Code} {
		if s != "" {
			return s
		}
	}
	return fmt.Sprintf("WorkOS request failed (%d)", e.Status)
}

func (e *WorkOSError) IsCode(code string) bool { return e.Code == code || e.Error_ == code }

var ErrWorkOSDisabled = errors.New("authentication provider is not configured")

func (w *WorkOS) do(ctx context.Context, method, path string, body any, out any) error {
	if !w.Enabled() {
		return ErrWorkOSDisabled
	}
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, w.BaseURL+path, rdr)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+w.APIKey)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := w.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("failed to reach authentication provider: %w", err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode >= 400 {
		we := &WorkOSError{Status: resp.StatusCode}
		_ = json.Unmarshal(data, we)
		return we
	}
	if out != nil && len(data) > 0 {
		return json.Unmarshal(data, out)
	}
	return nil
}

func (w *WorkOS) authenticate(ctx context.Context, params map[string]any) (*AuthResult, error) {
	params["client_id"] = w.ClientID
	params["client_secret"] = w.APIKey
	var res AuthResult
	if err := w.do(ctx, http.MethodPost, "/user_management/authenticate", params, &res); err != nil {
		return nil, err
	}
	if res.User.ID == "" {
		return nil, errors.New("authentication provider returned no user")
	}
	return &res, nil
}

func (w *WorkOS) AuthenticateWithPassword(ctx context.Context, email, password, ip, ua string) (*AuthResult, error) {
	return w.authenticate(ctx, map[string]any{
		"grant_type": "password", "email": email, "password": password, "ip_address": ip, "user_agent": ua,
	})
}

func (w *WorkOS) AuthenticateWithCode(ctx context.Context, code, ip, ua string) (*AuthResult, error) {
	return w.authenticate(ctx, map[string]any{
		"grant_type": "authorization_code", "code": code, "ip_address": ip, "user_agent": ua,
	})
}

func (w *WorkOS) AuthenticateWithEmailVerification(ctx context.Context, pendingToken, code, ip, ua string) (*AuthResult, error) {
	return w.authenticate(ctx, map[string]any{
		"grant_type":                   "urn:workos:oauth:grant-type:email-verification:code",
		"pending_authentication_token": pendingToken, "code": code, "ip_address": ip, "user_agent": ua,
	})
}

// AuthorizationURL builds the hosted AuthKit / OAuth provider redirect.
func (w *WorkOS) AuthorizationURL(provider, redirectURI, state, screenHint string) string {
	v := url.Values{}
	v.Set("client_id", w.ClientID)
	v.Set("redirect_uri", redirectURI)
	v.Set("response_type", "code")
	v.Set("provider", provider)
	v.Set("state", state)
	if screenHint != "" {
		v.Set("screen_hint", screenHint)
	}
	return w.BaseURL + "/user_management/authorize?" + v.Encode()
}

func (w *WorkOS) CreateUser(ctx context.Context, email, password, first, last string) (*WorkOSUser, error) {
	var u WorkOSUser
	err := w.do(ctx, http.MethodPost, "/user_management/users", map[string]any{
		"email": email, "password": password, "first_name": first, "last_name": last,
	}, &u)
	return &u, err
}

func (w *WorkOS) GetUser(ctx context.Context, id string) (*WorkOSUser, error) {
	var u WorkOSUser
	err := w.do(ctx, http.MethodGet, "/user_management/users/"+url.PathEscape(id), nil, &u)
	return &u, err
}

func (w *WorkOS) UpdateUser(ctx context.Context, id string, fields map[string]any) error {
	return w.do(ctx, http.MethodPut, "/user_management/users/"+url.PathEscape(id), fields, nil)
}

// CreatePasswordReset creates a one-time reset token and returns its URL so the
// app can email it. The URL is a credential and must never be returned to API callers.
func (w *WorkOS) CreatePasswordReset(ctx context.Context, email string) (string, error) {
	var res struct {
		PasswordResetURL string `json:"password_reset_url"`
	}
	err := w.do(ctx, http.MethodPost, "/user_management/password_reset", map[string]any{"email": email}, &res)
	return res.PasswordResetURL, err
}

func (w *WorkOS) ConfirmPasswordReset(ctx context.Context, token, newPassword string) (*WorkOSUser, error) {
	var res struct {
		User WorkOSUser `json:"user"`
	}
	err := w.do(ctx, http.MethodPost, "/user_management/password_reset/confirm", map[string]any{
		"token": token, "new_password": newPassword,
	}, &res)
	return &res.User, err
}
