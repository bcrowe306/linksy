package service_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bcrowe306/linksy/internal/db"
	"github.com/bcrowe306/linksy/internal/models"
	"github.com/bcrowe306/linksy/internal/repository"
	"github.com/bcrowe306/linksy/internal/service"
	"github.com/bcrowe306/linksy/internal/storage"
)

func setupTestService(t *testing.T) (*service.Service, func()) {
	tmpDir, err := os.MkdirTemp("", "linksy-test-*")
	if err != nil {
		t.Fatalf("failed creating temp dir: %v", err)
	}

	dbPath := filepath.Join(tmpDir, "test.db")
	database, err := db.Connect("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed connecting to test db: %v", err)
	}

	repo := repository.New(database)
	_ = repo.EnsureDefaultTeam("team-1", "Default Team", "user-1")

	store, err := storage.NewLocalStorage(filepath.Join(tmpDir, "storage"), "http://localhost:8080")
	if err != nil {
		t.Fatalf("failed creating storage: %v", err)
	}

	svc := service.New(repo, store, "http://localhost:8080")

	cleanup := func() {
		database.Close()
		os.RemoveAll(tmpDir)
	}

	return svc, cleanup
}

func TestValidateTargetURL(t *testing.T) {
	valid := []string{
		"https://example.com",
		"http://example.com/some/path?query=1",
		"https://sub.domain.org/app#section",
	}
	for _, v := range valid {
		if err := service.ValidateTargetURL(v); err != nil {
			t.Errorf("expected '%s' to be valid, got: %v", v, err)
		}
	}

	invalid := []string{
		"ftp://example.com",
		"javascript:alert(1)",
		"http://localhost:3000",
		"http://127.0.0.1:8080",
		"http://192.168.1.1/admin",
		"not-a-url",
	}
	for _, inv := range invalid {
		if err := service.ValidateTargetURL(inv); err == nil {
			t.Errorf("expected '%s' to be rejected", inv)
		}
	}
}

func TestShortLinkLifecycle(t *testing.T) {
	svc, cleanup := setupTestService(t)
	defer cleanup()

	// 1. Create short link
	link, err := svc.CreateShortLink(service.CreateLinkInput{
		TeamID:    "team-1",
		UserID:    "user-1",
		TargetURL: "https://example.com/original-page",
		Title:     "Original Link Title",
	})
	if err != nil {
		t.Fatalf("failed creating link: %v", err)
	}

	if link.Slug == "" {
		t.Fatal("expected generated slug, got empty")
	}

	// 2. Resolve short link
	resolved, err := svc.ResolveShortLink(link.Slug)
	if err != nil {
		t.Fatalf("failed resolving short link: %v", err)
	}
	if resolved.TargetURL != "https://example.com/original-page" {
		t.Fatalf("expected target URL 'https://example.com/original-page', got: %s", resolved.TargetURL)
	}

	// 3. Update target URL dynamically (Requirement 6)
	updated, err := svc.UpdateShortLink(link.ID, "https://example.com/updated-destination", "New Title", "New Desc", true, nil)
	if err != nil {
		t.Fatalf("failed updating short link: %v", err)
	}
	if updated.TargetURL != "https://example.com/updated-destination" {
		t.Fatalf("updated link has wrong target: %s", updated.TargetURL)
	}

	// 4. Resolve again to verify dynamic routing works without changing slug
	resolvedAfterUpdate, err := svc.ResolveShortLink(link.Slug)
	if err != nil {
		t.Fatalf("failed resolving updated short link: %v", err)
	}
	if resolvedAfterUpdate.TargetURL != "https://example.com/updated-destination" {
		t.Fatalf("expected new destination, got: %s", resolvedAfterUpdate.TargetURL)
	}
}

func TestDynamicQRCodeAndExport(t *testing.T) {
	svc, cleanup := setupTestService(t)
	defer cleanup()

	// Create dynamic QR code
	qrCode, err := svc.CreateQRCode(service.CreateQRInput{
		TeamID:    "team-1",
		UserID:    "user-1",
		Type:      models.QRTypeWebsite,
		Title:     "My Company Website",
		TargetURL: "https://example.com/company",
		IsDynamic: true,
		Style: models.QRStyle{
			FgColor: "#0055AA",
			BgColor: "#FFFFFF",
		},
	})
	if err != nil {
		t.Fatalf("failed creating QR code: %v", err)
	}

	if qrCode.LinkID == nil {
		t.Fatal("expected dynamic QR code to have associated short link ID")
	}

	// Test exports in all formats (Requirement 5)
	formats := []string{"png", "jpeg", "svg", "pdf"}
	for _, fmtName := range formats {
		bytes, mime, err := svc.ExportQRCode(qrCode.ID, fmtName, 256)
		if err != nil {
			t.Fatalf("failed exporting QR as %s: %v", fmtName, err)
		}
		if len(bytes) == 0 {
			t.Fatalf("exported %s is empty", fmtName)
		}
		if mime == "" {
			t.Fatalf("missing mime type for %s", fmtName)
		}
	}
}
