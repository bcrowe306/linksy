package api

import (
	"log"
	"os"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/static"

	"github.com/bcrowe306/linksy/internal/api/handler"
	"github.com/bcrowe306/linksy/internal/api/middleware"
	"github.com/bcrowe306/linksy/internal/auth"
	"github.com/bcrowe306/linksy/internal/config"
	"github.com/bcrowe306/linksy/internal/service"
	"github.com/bcrowe306/linksy/internal/web"
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
	app.Get("/api/v1/openapi.yaml", h.HandleOpenAPIYAML)

	// Authentication (WorkOS password, email verification, OAuth / AuthKit)
	authLimit := middleware.AuthRateLimiter()
	flowLimit := middleware.FlowRateLimiter()
	app.Get("/api/v1/auth/config", authH.GetAuthConfig)
	app.Get("/api/v1/auth/me", authH.GetCurrentUser)
	app.Post("/api/v1/auth/logout", authH.HandleLogout)
	app.Post("/api/v1/auth/signup", authLimit, authH.HandleSignUp)
	app.Post("/api/v1/auth/login", authLimit, authH.HandlePasswordLogin)
	app.Post("/api/v1/auth/verify-email", authLimit, authH.HandleVerifyEmail)
	app.Post("/api/v1/auth/dev-login", authLimit, authH.HandleDevLogin)
	app.Get("/api/v1/auth/login", flowLimit, authH.HandleLogin)
	app.Get("/api/v1/auth/oauth/:provider", flowLimit, authH.HandleOAuthLogin)
	app.Get("/api/v1/auth/callback", flowLimit, authH.HandleCallback)
	app.Post("/api/v1/auth/password-reset", authLimit, authH.HandlePasswordReset)
	app.Post("/api/v1/auth/password-reset/confirm", authLimit, authH.HandlePasswordResetConfirm)

	// Stripe webhooks (signature-verified)
	app.Post("/api/webhooks/stripe", h.StripeWebhook)

	// Free QR code generator (static codes, nothing stored)
	app.Post("/api/v1/public/qr", middleware.PublicToolRateLimiter(), h.PublicQR)

	// Signed one-click unsubscribe links from emails
	app.Get("/email/unsubscribe", flowLimit, h.Unsubscribe)
	app.Post("/email/unsubscribe", flowLimit, h.Unsubscribe)

	// Public abuse reports (form + JSON)
	app.Get("/report", flowLimit, h.ReportPage)
	app.Post("/report", authLimit, h.SubmitReport)
	app.Post("/api/v1/report", authLimit, h.SubmitReport)

	// Invitation preview (public so the accept page can render before sign-in)
	app.Get("/api/v1/invitations/:token", flowLimit, teamH.PreviewInvitation)

	// High throughput public redirectors
	app.Get("/s/:slug", middleware.RedirectRateLimiter(), h.HandleShortLinkRedirect)
	app.Post("/s/:slug", middleware.AuthRateLimiter(), h.HandleShortLinkUnlock)
	app.Get("/q/:code", middleware.RedirectRateLimiter(), h.HandleQRScan)
	app.Get("/qr/r/:id", middleware.RedirectRateLimiter(), h.HandleLegacyQRScan)
	app.Get("/d/:id", middleware.RedirectRateLimiter(), h.HandleDocumentLink)

	// Hosted document files behind document QR codes (random, unguessable keys)
	app.Get("/api/v1/documents/files/:filename", middleware.RedirectRateLimiter(), h.DownloadDocumentFile)

	// ---- Authenticated API (session cookie / bearer session / lk_live_ API key) ----
	api := app.Group("/api/v1", middleware.APIRateLimiter(), middleware.AuthMiddleware(svc))
	can := middleware.RequirePermission
	sessionOnly := middleware.RequireSession()

	// Profile, password & sessions (interactive sessions only)
	api.Get("/user/profile", authH.GetProfile)
	api.Put("/user/profile", sessionOnly, authH.UpdateProfile)
	api.Post("/user/change-password", sessionOnly, authH.HandleChangePassword)
	api.Get("/user/preferences", sessionOnly, authH.GetPreferences)
	api.Put("/user/preferences", sessionOnly, authH.UpdatePreferences)
	api.Get("/user/sessions", sessionOnly, authH.ListSessions)
	api.Delete("/user/sessions/:id", sessionOnly, authH.RevokeSession)
	api.Post("/user/sessions/revoke-others", sessionOnly, authH.RevokeOtherSessions)
	api.Get("/user/invitations", sessionOnly, teamH.ListMyInvitations)
	api.Post("/user/invitations/:id/accept", sessionOnly, teamH.RespondToInvitation(true))
	api.Post("/user/invitations/:id/decline", sessionOnly, teamH.RespondToInvitation(false))
	api.Post("/invitations/:token/accept", sessionOnly, teamH.AcceptInvitation)

	// Personal API keys (bound to the active workspace)
	api.Get("/api-keys", sessionOnly, authH.ListAPIKeys)
	api.Post("/api-keys", sessionOnly, authH.CreateAPIKey)
	api.Delete("/api-keys/:id", sessionOnly, authH.RevokeAPIKey)

	// Short Links
	api.Post("/links", can(auth.PermEditAssets), h.CreateLink)
	api.Get("/links", can(auth.PermViewAssets), h.ListLinks)
	api.Get("/links/:id", can(auth.PermViewAssets), h.GetLink)
	api.Put("/links/:id", can(auth.PermEditAssets), h.UpdateLink)
	api.Delete("/links/:id", can(auth.PermEditAssets), h.DeleteLink)

	// QR Codes
	api.Post("/qr", can(auth.PermEditAssets), h.CreateQR)
	api.Post("/qr/preview", can(auth.PermViewAssets), h.PreviewQR)
	api.Get("/qr", can(auth.PermViewAssets), h.ListQR)
	api.Get("/qr/:id", can(auth.PermViewAssets), h.GetQR)
	api.Get("/qr/:id/export", can(auth.PermExportAssets), h.ExportQR)
	api.Put("/qr/:id", can(auth.PermEditAssets), h.UpdateQR)
	api.Delete("/qr/:id", can(auth.PermEditAssets), h.DeleteQR)

	// Documents
	api.Post("/documents/upload", can(auth.PermEditAssets), h.UploadDocument)
	api.Put("/documents/:id", can(auth.PermEditAssets), h.UpdateDocumentFile)
	api.Get("/documents", can(auth.PermViewAssets), h.ListDocuments)
	api.Delete("/documents/:id", can(auth.PermEditAssets), h.DeleteDocument)
	api.Post("/documents/bulk-delete", can(auth.PermEditAssets), h.BulkDeleteDocuments)

	// Analytics
	api.Get("/analytics/summary", can(auth.PermViewAnalytics), h.GetAnalyticsSummary)
	api.Get("/analytics/report", can(auth.PermViewAnalytics), h.GetAnalyticsReport)
	api.Get("/analytics/export", can(auth.PermExportAssets), h.ExportAnalytics)
	api.Get("/analytics/live", can(auth.PermViewAnalytics), h.LiveUpgrade, h.LiveFeed())

	// Billing (owners manage; everyone can read the plan state)
	api.Get("/billing", sessionOnly, h.GetBilling)
	api.Post("/billing/promo", sessionOnly, can(auth.PermManageBilling), h.PreviewPromo)
	api.Post("/billing/checkout", sessionOnly, can(auth.PermManageBilling), h.StartCheckout)
	api.Post("/billing/portal", sessionOnly, can(auth.PermManageBilling), h.BillingPortal)
	api.Post("/billing/redeem", sessionOnly, can(auth.PermManageBilling), h.RedeemTrialCode)

	// Platform admin console: Lynqr staff only, after re-authentication
	api.Get("/admin/session", sessionOnly, authH.AdminStatus)
	api.Post("/admin/session", sessionOnly, authLimit, authH.AdminReauth)
	api.Post("/admin/session/code", sessionOnly, authLimit, authH.AdminSendCode)
	api.Delete("/admin/session", sessionOnly, authH.AdminLeave)
	admin := api.Group("/admin", sessionOnly, middleware.RequirePlatformAdmin(svc))
	admin.Get("/metrics", h.AdminMetrics)
	admin.Get("/workspaces", h.AdminListTeams)
	admin.Get("/workspaces/:id", h.AdminGetTeam)
	admin.Post("/workspaces/:id/:action", h.AdminUpdateTeam)
	admin.Get("/users", h.AdminListUsers)
	admin.Get("/users/:id", h.AdminGetUser)
	admin.Post("/users/:id/:action", h.AdminUpdateUser)
	admin.Get("/codes", h.AdminListCodes)
	admin.Post("/codes/checkout", h.AdminCreateCheckoutCode)
	admin.Post("/codes/checkout/:id", h.AdminSetCheckoutCode)
	admin.Post("/codes/trial", h.AdminCreateTrialCode)
	admin.Post("/codes/trial/:code", h.AdminSetTrialCode)
	admin.Get("/moderation/search", h.AdminSearchAssets)
	admin.Post("/moderation/:type/:id/:action", h.AdminModerate)
	admin.Get("/blocklist", h.AdminListBlocked)
	admin.Post("/blocklist", h.AdminBlockDomain)
	admin.Delete("/blocklist/:domain", h.AdminUnblockDomain)
	admin.Get("/reports", h.AdminListReports)
	admin.Post("/reports/:id", h.AdminResolveReport)
	admin.Get("/audit", h.AdminAudit)

	// Workspaces (multi-team membership & switching)
	api.Get("/teams", sessionOnly, teamH.ListTeams)
	api.Post("/teams", sessionOnly, teamH.CreateTeam)
	api.Post("/teams/switch", sessionOnly, teamH.SwitchTeam)

	// Active workspace settings & members
	api.Get("/team", teamH.GetTeam)
	api.Put("/team", sessionOnly, can(auth.PermManageTeam), teamH.UpdateTeam)
	api.Delete("/team", sessionOnly, can(auth.PermDeleteTeam), teamH.DeleteTeam)
	api.Post("/team/transfer", sessionOnly, can(auth.PermTransferOwner), teamH.TransferOwnership)
	api.Post("/team/leave", sessionOnly, teamH.LeaveTeam)
	api.Get("/team/audit", can(auth.PermViewAudit), teamH.ListAudit)
	api.Get("/team/members", can(auth.PermViewAssets), teamH.ListMembers)

	members := api.Group("/team", sessionOnly, can(auth.PermManageMembers))
	members.Post("/members/invite", teamH.InviteMember)
	members.Put("/members/:userId/role", teamH.UpdateMemberRole)
	members.Post("/members/:userId/suspend", teamH.SuspendMember)
	members.Post("/members/:userId/reactivate", teamH.ReactivateMember)
	members.Delete("/members/:userId", teamH.RemoveMember)
	members.Post("/members/:userId/reset-credentials", teamH.ResetMemberCredentials)
	members.Delete("/invitations/:id", teamH.RevokeInvitation)

	// Unknown API routes return JSON 404s rather than the SPA shell
	app.All("/api/*", func(c fiber.Ctx) error {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Not found"})
	})

	// If web/dist directory exists, serve Vue 3 SPA frontend
	app.Get("/robots.txt", h.RobotsTxt)
	app.Get("/sitemap.xml", h.SitemapXML)
	app.Get("/llms.txt", h.LLMsTxt)
	if _, err := os.Stat(handler.DistDir); err == nil {
		if site, err := web.Load(handler.DistDir); err != nil {
			log.Printf("[Lynqr] public pages disabled: %v", err)
		} else {
			h.SetSite(site)
			app.Get("/guides", h.GuidesIndex)
			app.Get("/guides/:slug", h.GuidePage)
			for _, slug := range handler.GeneratorSlugs() {
				if slug == "" {
					app.Get("/qr-code-generator", h.GeneratorPage(slug))
				} else {
					app.Get("/qr-code-generator/"+slug, h.GeneratorPage(slug))
				}
			}
			for _, slug := range []string{"terms", "privacy", "acceptable-use"} {
				app.Get("/"+slug, h.LegalPage(slug))
			}
		}
		app.Get("/", h.Home)
		app.Use("/", static.New(handler.DistDir))
		app.Get("/*", func(c fiber.Ctx) error {
			return c.SendFile("./web/dist/index.html")
		})
	}
}
