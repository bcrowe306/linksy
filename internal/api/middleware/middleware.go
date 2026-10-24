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
func AuthMiddleware() fiber.Handler {
	return func(c fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		teamHeader := c.Get("X-Team-ID")

		userID := "usr_demo123"
		teamID := "team_default"
		role := models.RoleOwner

		if teamHeader != "" {
			teamID = teamHeader
		}

		if strings.HasPrefix(authHeader, "Bearer ") {
			token := strings.TrimPrefix(authHeader, "Bearer ")
			// If real WorkOS token or custom API key, populate here
			if token != "" {
				userID = "usr_" + token[:min(len(token), 10)]
			}
		}

		c.Locals("auth", &AuthContext{
			UserID: userID,
			TeamID: teamID,
			Role:   role,
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
