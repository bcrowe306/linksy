package middleware

import (
	"errors"
	"net/url"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/limiter"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/gofiber/fiber/v3/middleware/recover"

	"github.com/bcrowe306/linksy/internal/auth"
	"github.com/bcrowe306/linksy/internal/models"
	"github.com/bcrowe306/linksy/internal/service"
)

// SessionCookie is the HttpOnly cookie carrying the opaque session token.
const SessionCookie = "lynqr_session"

type AuthContext struct {
	UserID  string
	TeamID  string
	Role    models.UserRole
	APIKey  *models.APIKey
	Session *models.Session
	// ViaCookie is true when the credential came from the browser cookie
	// (and therefore needs CSRF protection on unsafe methods).
	ViaCookie bool
}

// DefaultCSP is applied to every response; the API docs page relaxes it.
const DefaultCSP = "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; " +
	"font-src 'self' https://fonts.gstatic.com data:; img-src 'self' data: blob: https:; connect-src 'self' wss: ws:; " +
	"frame-ancestors 'none'; base-uri 'self'; form-action 'self' https://api.workos.com; object-src 'none'"

func SetupCommon(app *fiber.App) {
	app.Use(recover.New())
	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${status} - ${latency} ${method} ${path} client=${reqHeader:CF-Connecting-IP}\n",
	}))
	app.Use(SecurityHeaders())
	// API keys are bearer credentials, so any origin may call the API. Credentials
	// (cookies) are never allowed cross-origin.
	app.Use(cors.New(cors.Config{
		AllowOrigins: []string{"*"},
		AllowHeaders: []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Team-ID"},
		AllowMethods: []string{"GET", "POST", "HEAD", "PUT", "DELETE", "PATCH", "OPTIONS"},
	}))
}

func SecurityHeaders() fiber.Handler {
	return func(c fiber.Ctx) error {
		c.Set("X-Content-Type-Options", "nosniff")
		c.Set("X-Frame-Options", "DENY")
		c.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		c.Set("Content-Security-Policy", DefaultCSP)
		if c.Protocol() == "https" || c.Get("X-Forwarded-Proto") == "https" {
			c.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		return c.Next()
	}
}

// ClientIP returns the visitor's address. Production traffic arrives through
// Cloudflare (CF-Connecting-IP) and Render's proxy (X-Forwarded-For); the
// result feeds rate limiting and hashed analytics, never authorization.
func ClientIP(c fiber.Ctx) string {
	if ip := strings.TrimSpace(c.Get("CF-Connecting-IP")); ip != "" {
		return ip
	}
	if xff := c.Get("X-Forwarded-For"); xff != "" {
		if ip := strings.TrimSpace(strings.Split(xff, ",")[0]); ip != "" {
			return ip
		}
	}
	return c.IP()
}

// RedirectRateLimiter applies rate limits on redirect resolution to prevent abuse
func RedirectRateLimiter() fiber.Handler {
	return limiter.New(limiter.Config{
		Max:               300,
		Expiration:        1 * time.Minute,
		KeyGenerator:      ClientIP,
		LimiterMiddleware: limiter.SlidingWindow{},
	})
}

// AuthRateLimiter throttles credential endpoints (login, signup, password reset)
// to slow password guessing. Each call creates an independent counter.
func AuthRateLimiter() fiber.Handler {
	return limiter.New(limiter.Config{
		Max:               30,
		Expiration:        1 * time.Minute,
		KeyGenerator:      ClientIP,
		LimiterMiddleware: limiter.SlidingWindow{},
	})
}

// FlowRateLimiter covers OAuth redirects, callbacks and invitation previews,
// which are cheap and may be hit by many users behind one office NAT.
func FlowRateLimiter() fiber.Handler {
	return limiter.New(limiter.Config{
		Max:               120,
		Expiration:        1 * time.Minute,
		KeyGenerator:      ClientIP,
		LimiterMiddleware: limiter.SlidingWindow{},
	})
}

// APIRateLimiter applies a per-credential ceiling on the authenticated API.
func APIRateLimiter() fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        600,
		Expiration: 1 * time.Minute,
		KeyGenerator: func(c fiber.Ctx) string {
			if t := bearerOrCookie(c); t != "" {
				return auth.HashToken(t)
			}
			return ClientIP(c)
		},
		LimiterMiddleware: limiter.SlidingWindow{},
	})
}

func bearerOrCookie(c fiber.Ctx) string {
	if h := c.Get("Authorization"); strings.HasPrefix(h, "Bearer ") {
		return strings.TrimSpace(strings.TrimPrefix(h, "Bearer "))
	}
	return c.Cookies(SessionCookie)
}

func unauthorized(c fiber.Ctx, msg string) error {
	return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": msg})
}

// AuthMiddleware authenticates opaque session tokens (cookie or bearer) and
// lk_live_ API keys, then resolves the caller's team and live role. The
// X-Team-ID header is honoured only for teams the user actively belongs to.
func AuthMiddleware(svc *service.Service) fiber.Handler {
	return func(c fiber.Ctx) error {
		ctx, err := Authenticate(c, svc)
		if err != nil {
			switch {
			case errors.Is(err, service.ErrSuspended):
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": err.Error(), "code": "suspended"})
			case errors.Is(err, service.ErrNotMember):
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": err.Error(), "code": "not_member"})
			}
			return unauthorized(c, "Authentication required. Please sign in or provide a valid API key.")
		}
		if ctx.ViaCookie && !SameOrigin(c) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Cross-origin request rejected"})
		}
		c.Locals("auth", ctx)
		return c.Next()
	}
}

// Authenticate resolves the request credential without writing a response.
func Authenticate(c fiber.Ctx, svc *service.Service) (*AuthContext, error) {
	header := c.Get("Authorization")
	token, viaCookie := "", false
	if strings.HasPrefix(header, "Bearer ") {
		token = strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
	} else if ck := c.Cookies(SessionCookie); ck != "" {
		token, viaCookie = ck, true
	}
	if token == "" {
		return nil, service.ErrUnauthorized
	}

	if strings.HasPrefix(token, auth.APIKeyPrefix) {
		key, err := svc.ValidateAPIKey(token)
		if err != nil {
			return nil, service.ErrUnauthorized
		}
		// Keys are bound to one team; the role is the owner's live role there.
		teamID, role, err := svc.ResolveTeam(key.UserID, key.TeamID, "")
		if err != nil {
			return nil, err
		}
		return &AuthContext{UserID: key.UserID, TeamID: teamID, Role: role, APIKey: key}, nil
	}

	sess, err := svc.ResolveSession(token)
	if err != nil {
		return nil, service.ErrUnauthorized
	}
	teamID, role, err := svc.ResolveTeam(sess.UserID, c.Get("X-Team-ID"), sess.ActiveTeamID)
	if err != nil {
		return nil, err
	}
	return &AuthContext{UserID: sess.UserID, TeamID: teamID, Role: role, Session: sess, ViaCookie: viaCookie}, nil
}

// SameOrigin enforces that unsafe cookie-authenticated requests originate
// from this application (CSRF defence in depth on top of SameSite=Lax).
func SameOrigin(c fiber.Ctx) bool {
	switch c.Method() {
	case fiber.MethodGet, fiber.MethodHead, fiber.MethodOptions:
		return true
	}
	origin := c.Get("Origin")
	if origin == "" {
		ref := c.Get("Referer")
		if ref == "" {
			// Non-browser clients don't send Origin; browsers always do on unsafe requests.
			return true
		}
		origin = ref
	}
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	host := c.Get("X-Forwarded-Host")
	if host == "" {
		host = c.Host()
	}
	return strings.EqualFold(u.Host, strings.TrimSpace(strings.Split(host, ",")[0]))
}

// GetAuth returns the request's auth context, or an empty context (which every
// permission check rejects) when the route is unauthenticated.
func GetAuth(c fiber.Ctx) *AuthContext {
	if a, ok := c.Locals("auth").(*AuthContext); ok && a != nil {
		return a
	}
	return &AuthContext{}
}

// Actor converts the request's auth context into a service Actor.
func Actor(c fiber.Ctx) service.Actor {
	a := GetAuth(c)
	return service.Actor{UserID: a.UserID, TeamID: a.TeamID, Role: a.Role, IP: ClientIP(c)}
}

// RequirePermission rejects requests whose role lacks the permission.
func RequirePermission(p auth.Permission) fiber.Handler {
	return func(c fiber.Ctx) error {
		a := GetAuth(c)
		if a.UserID == "" {
			return unauthorized(c, "Authentication required")
		}
		if !auth.Can(a.Role, p) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Your role does not allow this action",
				"code":  "forbidden",
			})
		}
		return c.Next()
	}
}

// RequireSession rejects API-key callers on routes meant only for interactive sessions.
func RequireSession() fiber.Handler {
	return func(c fiber.Ctx) error {
		if GetAuth(c).Session == nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "This endpoint requires a signed-in session"})
		}
		return c.Next()
	}
}
