package api

import (
	"os"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/static"

	"github.com/bcrowe306/linksy/internal/api/handler"
	"github.com/bcrowe306/linksy/internal/api/middleware"
	"github.com/bcrowe306/linksy/internal/config"
	"github.com/bcrowe306/linksy/internal/service"
)

func SetupRouter(app *fiber.App, svc *service.Service, cfg *config.Config) {
	middleware.SetupCommon(app)

	h := handler.New(svc, cfg.AppURL)
	authH := handler.NewAuthHandler(cfg, svc)
	teamH := handler.NewTeamHandler(cfg, svc)

	// Public health check
	app.Get("/health", h.Health)
	app.Get("/api/v1/health", h.Health)

	// API Interactive Documentation & OpenAPI Spec
	app.Get("/api/docs", h.HandleAPIDocs)
	app.Get("/api/v1/openapi.json", h.HandleOpenAPISpec)

	// Authentication Routes (Custom Headless Signup/Login + WorkOS Social OAuth)
	app.Get("/api/v1/auth/config", authH.GetAuthConfig)
	app.Post("/api/v1/auth/signup", authH.HandleSignUp)
	app.Post("/api/v1/auth/login", authH.HandlePasswordLogin)
	app.Get("/api/v1/auth/login", authH.HandleLogin)
	app.Get("/api/v1/auth/oauth/:provider", authH.HandleOAuthLogin)
	app.Get("/api/v1/auth/callback", authH.HandleCallback)
	app.Get("/api/v1/auth/me", authH.GetCurrentUser)
	app.Post("/api/v1/auth/logout", authH.HandleLogout)

	// Password reset (accessible publicly with email or authenticated)
	app.Post("/api/v1/auth/password-reset", authH.HandlePasswordReset)
	app.Post("/api/v1/auth/change-password", authH.HandleChangePassword)

	// High throughput public redirectors
	app.Get("/s/:slug", middleware.RedirectRateLimiter(), h.HandleShortLinkRedirect)
	app.Get("/qr/r/:id", middleware.RedirectRateLimiter(), h.HandleQRScanRedirect)

	// Public QR Code image export / preview (used in <img> tags and public downloads)
	app.Get("/api/v1/qr/:id/export", h.ExportQR)

	// Static Document file downloads
	app.Get("/api/v1/documents/files/:filename", h.DownloadDocumentFile)

	// Protected / Authenticated API v1 routes (authenticates session tokens, cookies, & lk_live_ API keys)
	api := app.Group("/api/v1", middleware.AuthMiddleware(svc))

	// User Profile & Account Management
	api.Get("/user/profile", authH.GetProfile)
	api.Put("/user/profile", authH.UpdateProfile)
	api.Post("/user/change-password", authH.HandleChangePassword)

	// API Keys Management
	api.Get("/api-keys", authH.ListAPIKeys)
	api.Post("/api-keys", authH.CreateAPIKey)
	api.Delete("/api-keys/:id", authH.RevokeAPIKey)

	// Short Links
	api.Post("/links", h.CreateLink)
	api.Get("/links", h.ListLinks)
	api.Get("/links/:id", h.GetLink)
	api.Put("/links/:id", h.UpdateLink)
	api.Delete("/links/:id", h.DeleteLink)

	// QR Codes
	api.Post("/qr", h.CreateQR)
	api.Get("/qr", h.ListQR)
	api.Get("/qr/:id", h.GetQR)
	api.Put("/qr/:id", h.UpdateQR)
	api.Delete("/qr/:id", h.DeleteQR)

	// Documents
	api.Post("/documents/upload", h.UploadDocument)
	api.Put("/documents/:id", h.UpdateDocumentFile)
	api.Get("/documents", h.ListDocuments)
	api.Delete("/documents/:id", h.DeleteDocument)
	api.Post("/documents/bulk-delete", h.BulkDeleteDocuments)

	// Team & User Management (WorkOS Invitations, Roles, & Credential Reset)
	api.Get("/team/members", teamH.ListMembers)

	// Admin-only Team Operations
	teamAdmin := api.Group("/team", middleware.RequireAdmin())
	teamAdmin.Post("/members/invite", teamH.InviteMember)
	teamAdmin.Put("/members/:userId/role", teamH.UpdateMemberRole)
	teamAdmin.Delete("/members/:userId", teamH.RemoveMember)
	teamAdmin.Post("/members/:userId/reset-credentials", teamH.ResetMemberCredentials)
	teamAdmin.Delete("/invitations/:id", teamH.RevokeInvitation)

	// Analytics
	api.Get("/analytics/summary", h.GetAnalyticsSummary)

	// If web/dist directory exists, serve Vue 3 SPA frontend
	if _, err := os.Stat("./web/dist"); err == nil {
		app.Use("/", static.New("./web/dist"))
		app.Get("/*", func(c fiber.Ctx) error {
			return c.SendFile("./web/dist/index.html")
		})
	}
}
