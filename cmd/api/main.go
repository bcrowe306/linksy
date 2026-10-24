package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/gofiber/fiber/v3"

	"github.com/bcrowe306/linksy/internal/api"
	"github.com/bcrowe306/linksy/internal/config"
	"github.com/bcrowe306/linksy/internal/db"
	"github.com/bcrowe306/linksy/internal/repository"
	"github.com/bcrowe306/linksy/internal/service"
	"github.com/bcrowe306/linksy/internal/storage"
)

func main() {
	cfg := config.LoadConfig()
	log.Printf("[Linksy] Starting server in %s mode (DB: %s)...", cfg.LogLevel, cfg.DBType)

	// Initialize Database
	database, err := db.Connect(cfg.DBType, cfg.DBDSN)
	if err != nil {
		log.Fatalf("[Linksy] Database connection failed: %v", err)
	}
	defer database.Close()
	log.Printf("[Linksy] Database connected successfully via %s", database.DriverName)

	// Ensure default team / seed
	repo := repository.New(database)
	if err := repo.EnsureDefaultTeam("team_default", "Main Workspace", "usr_demo123"); err != nil {
		log.Printf("[Linksy] Note ensuring default team: %v", err)
	}

	// Initialize Document Storage
	store, err := storage.NewLocalStorage(cfg.StoragePath, cfg.AppURL)
	if err != nil {
		log.Fatalf("[Linksy] Failed to initialize storage: %v", err)
	}

	// Initialize Services
	svc := service.New(repo, store, cfg.AppURL)

	// Initialize Fiber v3 App
	app := fiber.New(fiber.Config{
		AppName: "Linksy Dynamic QR & Link Manager",
	})

	api.SetupRouter(app, svc, cfg.AppURL)

	// Graceful shutdown handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("[Linksy] Shutting down gracefully...")
		_ = app.Shutdown()
	}()

	addr := fmt.Sprintf(":%s", cfg.Port)
	log.Printf("[Linksy] Server listening on %s (App URL: %s)", addr, cfg.AppURL)
	if err := app.Listen(addr); err != nil {
		log.Printf("[Linksy] Server stopped: %v", err)
	}
}
