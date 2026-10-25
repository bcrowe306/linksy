package middleware

import (
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/limiter"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/gofiber/fiber/v3/middleware/recover"

	"github.com/bcrowe306/linksy/internal/models"
)

type AuthContext struct {
	UserID string
	TeamID string
	Role   models.UserRole
	APIKey *models.APIKey
}

type KeyValidator interface {
	ValidateAPIKey(rawKey string) (*models.APIKey, error)
}

func SetupCommon(app *fiber.App) {
	app.Use(recover.New())
	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${status} - ${latency} ${method} ${path}\n",
	}))
	app.Use(cors.New(cors.Config{
		AllowOrigins: []string{"*"},
		AllowHeaders: []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Team-ID"},
		AllowMethods: []string{"GET", "POST", "HEAD", "PUT", "DELETE", "PATCH", "OPTIONS"},
	}))
}

// RedirectRateLimiter applies rate limits on redirect resolution to prevent DDoS
func RedirectRateLimiter() fiber.Handler {
	return limiter.New(limiter.Config{
		Max:               300,
		Expiration:        1 * time.Minute,
		LimiterMiddleware: limiter.SlidingWindow{},
	})
}

// AuthMiddleware handles WorkOS / API token authorization
func AuthMiddleware(validators ...KeyValidator) fiber.Handler {
	var validator KeyValidator
	if len(validators) > 0 {
		validator = validators[0]
	}

	return func(c fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		cookieToken := c.Cookies("linksy_session")
		teamHeader := c.Get("X-Team-ID")

		userID := "usr_demo123"
		teamID := "team_default"
		role := models.RoleOwner
		var apiKey *models.APIKey

		if teamHeader != "" {
			teamID = teamHeader
		}

		token := ""
		if strings.HasPrefix(authHeader, "Bearer ") {
			token = strings.TrimPrefix(authHeader, "Bearer ")
		} else if cookieToken != "" {
			token = cookieToken
		}

		if token != "" {
			// 1. Check if token is a Linksy API Key (lk_live_...)
			if strings.HasPrefix(token, "lk_live_") && validator != nil {
				if key, err := validator.ValidateAPIKey(token); err == nil && key != nil {
					userID = key.UserID
					if key.TeamID != "" {
						teamID = key.TeamID
					}
					apiKey = key
				}
			} else if strings.HasPrefix(token, "user_") {
				userID = token
			} else if strings.HasPrefix(token, "sess_") {
				clean := strings.TrimPrefix(token, "sess_")
				if strings.HasPrefix(clean, "user_") {
					parts := strings.Split(clean, "_")
					if len(parts) >= 2 {
						userID = "user_" + parts[1]
					}
				} else {
					parts := strings.Split(clean, "_")
					userID = parts[0]
				}
			} else if strings.HasPrefix(token, "workos_sess_") {
				userID = strings.TrimPrefix(token, "workos_sess_")
			} else {
				userID = "usr_" + token[:min(len(token), 10)]
			}
		}

		c.Locals("auth", &AuthContext{
			UserID: userID,
			TeamID: teamID,
			Role:   role,
			APIKey: apiKey,
		})

		return c.Next()
	}
}

func GetAuth(c fiber.Ctx) *AuthContext {
	if auth, ok := c.Locals("auth").(*AuthContext); ok {
		return auth
	}
	return &AuthContext{
		UserID: "usr_demo123",
		TeamID: "team_default",
		Role:   models.RoleOwner,
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
