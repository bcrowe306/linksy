package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/bcrowe306/linksy/internal/api"
	"github.com/bcrowe306/linksy/internal/billing"
	"github.com/bcrowe306/linksy/internal/config"
	"github.com/bcrowe306/linksy/internal/db"
	"github.com/bcrowe306/linksy/internal/repository"
	"github.com/bcrowe306/linksy/internal/service"
	"github.com/bcrowe306/linksy/internal/storage"
)

func main() {
	cfg := config.LoadConfig()
	log.Printf("[Lynqr] Starting server in %s mode (DB: %s, Storage: %s)...", cfg.LogLevel, cfg.DBType, cfg.StorageType)

	// Initialize Database
	database, err := db.Connect(cfg.DBType, cfg.DBDSN)
	if err != nil {
		log.Fatalf("[Lynqr] Database connection failed: %v", err)
	}
	defer database.Close()
	log.Printf("[Lynqr] Database connected successfully via %s", database.DriverName)

	repo := repository.New(database)

	// Initialize Document Storage (Cloudflare R2 or Local)
	var store storage.StorageProvider
	if cfg.StorageType == "r2" {
		log.Printf("[Lynqr] Initializing Cloudflare R2 storage (Bucket: %s, Endpoint: %s)...", cfg.R2Bucket, cfg.R2Endpoint)
		r2Store, err := storage.NewR2Storage(cfg.R2Endpoint, cfg.R2AccessKeyID, cfg.R2SecretAccessKey, cfg.R2Bucket, cfg.AppURL)
		if err != nil {
			log.Fatalf("[Lynqr] Failed to initialize R2 storage: %v", err)
		}
		store = r2Store
	} else {
		localStore, err := storage.NewLocalStorage(cfg.StoragePath, cfg.AppURL)
		if err != nil {
			log.Fatalf("[Lynqr] Failed to initialize storage: %v", err)
		}
		store = localStore
	}

	// Initialize Services
	svc := service.New(repo, store, cfg.AppURL)
	if cfg.AppSecret == "" {
		if !cfg.IsLocal() {
			log.Fatalf("[Lynqr] APP_SECRET must be set outside local development")
		}
		cfg.AppSecret = "lynqr-local-dev-secret"
	}
	svc.SetIPSalt(cfg.AppSecret)

	// Billing: provision the Stripe catalog, then enable enforcement. Until that
	// succeeds every workspace keeps full access, so a Stripe outage never locks users out.
	if cfg.StripeKey != "" {
		go func() {
			client := billing.New(cfg.StripeKey)
			for attempt := 1; ; attempt++ {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				prices, err := client.EnsureCatalog(ctx)
				if err == nil {
					if product, perr := client.ProductOf(ctx, prices.Monthly); perr == nil {
						if err := client.EnsurePortal(ctx, prices, product, cfg.AppURL+"/settings?tab=billing"); err != nil {
							log.Printf("[Lynqr] Stripe portal configuration: %v", err)
						}
					}
					cancel()
					svc.EnableBilling(client, prices, cfg.StripeWebhookSecret)
					log.Printf("[Lynqr] Billing enabled (monthly %s, annual %s)", prices.Monthly, prices.Annual)
					return
				}
				cancel()
				log.Printf("[Lynqr] Stripe catalog setup failed (attempt %d): %v", attempt, err)
				time.Sleep(time.Duration(min(attempt, 10)) * 30 * time.Second)
			}
		}()
	} else {
		log.Printf("[Lynqr] STRIPE_SECRET_KEY not set: billing disabled, all workspaces have full access")
	}

	// Periodically purge expired sessions
	go func() {
		for range time.Tick(6 * time.Hour) {
			_ = repo.DeleteExpiredSessions()
		}
	}()

	// Initialize Fiber v3 App
	app := fiber.New(fiber.Config{
		AppName: "Lynqr",
		// Request values outlive handlers (async analytics), so they must not alias fasthttp's reused buffers.
		Immutable: true,
		BodyLimit: 16 << 20, // documents are capped at 15MB in the upload handler
	})

	api.SetupRouter(app, svc, cfg)

	// Graceful shutdown handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("[Lynqr] Shutting down gracefully...")
		_ = app.Shutdown()
	}()

	addr := fmt.Sprintf(":%s", cfg.Port)
	log.Printf("[Lynqr] Server listening on %s (App URL: %s)", addr, cfg.AppURL)
	if err := app.Listen(addr); err != nil {
		log.Printf("[Lynqr] Server stopped: %v", err)
	}
}
