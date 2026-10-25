package e2e_test

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gofiber/fiber/v3"

	"github.com/bcrowe306/linksy/internal/api"
	"github.com/bcrowe306/linksy/internal/config"
	"github.com/bcrowe306/linksy/internal/db"
	"github.com/bcrowe306/linksy/internal/models"
	"github.com/bcrowe306/linksy/internal/repository"
	"github.com/bcrowe306/linksy/internal/service"
	"github.com/bcrowe306/linksy/internal/storage"
)

func setupTestApp(t *testing.T) (*fiber.App, func()) {
	tmpDir, err := os.MkdirTemp("", "linksy-e2e-*")
	if err != nil {
		t.Fatalf("failed creating temp dir: %v", err)
	}

	dbPath := filepath.Join(tmpDir, "test.db")
	database, err := db.Connect("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed connecting sqlite db: %v", err)
	}

	repo := repository.New(database)
	_ = repo.EnsureDefaultTeam("team_default", "Default Team", "usr_demo123")

	store, err := storage.NewLocalStorage(filepath.Join(tmpDir, "storage"), "http://example.com")
	if err != nil {
		t.Fatalf("failed initializing storage: %v", err)
	}

	svc := service.New(repo, store, "http://example.com")

	cfg := &config.Config{
		AppURL: "http://example.com",
	}

	app := fiber.New()
	api.SetupRouter(app, svc, cfg)

	cleanup := func() {
		database.Close()
		os.RemoveAll(tmpDir)
	}

	return app, cleanup
}

func TestHealthEndpoint(t *testing.T) {
	app, cleanup := setupTestApp(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/health", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("health check failed: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", resp.StatusCode)
	}
}

func TestLinksAndRedirectE2E(t *testing.T) {
	app, cleanup := setupTestApp(t)
	defer cleanup()

	// 1. Create short link
	body, _ := json.Marshal(map[string]interface{}{
		"target_url":  "https://google.com",
		"custom_slug": "my-goog",
		"title":       "Google Search",
	})

	req := httptest.NewRequest("POST", "/api/v1/links", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer sess_user_demo123_test")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("failed creating link: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d", resp.StatusCode)
	}

	var link models.Link
	_ = json.NewDecoder(resp.Body).Decode(&link)

	// 2. Perform redirect
	redReq := httptest.NewRequest("GET", "/s/my-goog", nil)
	redResp, err := app.Test(redReq)
	if err != nil {
		t.Fatalf("redirect failed: %v", err)
	}
	if redResp.StatusCode != http.StatusFound {
		t.Errorf("expected 302 Found, got %d", redResp.StatusCode)
	}
	if loc := redResp.Header.Get("Location"); loc != "https://google.com" {
		t.Errorf("expected redirect to https://google.com, got %s", loc)
	}

	// 3. Update destination URL dynamically
	updateBody, _ := json.Marshal(map[string]interface{}{
		"target_url": "https://duckduckgo.com",
	})
	upReq := httptest.NewRequest("PUT", "/api/v1/links/"+link.ID, bytes.NewReader(updateBody))
	upReq.Header.Set("Content-Type", "application/json")
	upReq.Header.Set("Authorization", "Bearer sess_user_demo123_test")
	upResp, err := app.Test(upReq)
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}
	if upResp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK for update, got %d", upResp.StatusCode)
	}

	// 4. Verify changed redirection
	redResp2, _ := app.Test(redReq)
	if loc := redResp2.Header.Get("Location"); loc != "https://duckduckgo.com" {
		t.Errorf("expected updated redirect to https://duckduckgo.com, got %s", loc)
	}
}

func TestQRCodeAndExportE2E(t *testing.T) {
	app, cleanup := setupTestApp(t)
	defer cleanup()

	// 1. Create dynamic QR code
	body, _ := json.Marshal(map[string]interface{}{
		"title":      "Office Website",
		"type":       "website",
		"target_url": "https://example.com",
		"is_dynamic": true,
	})

	req := httptest.NewRequest("POST", "/api/v1/qr", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer sess_user_demo123_test")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("create QR failed: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d", resp.StatusCode)
	}

	var qr models.QRCode
	_ = json.NewDecoder(resp.Body).Decode(&qr)

	// 2. Export QR as PNG
	exportReq := httptest.NewRequest("GET", "/api/v1/qr/"+qr.ID+"/export?format=png", nil)
	exportReq.Header.Set("Authorization", "Bearer sess_user_demo123_test")
	exportResp, err := app.Test(exportReq)
	if err != nil {
		t.Fatalf("export failed: %v", err)
	}
	if exportResp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK for export, got %d", exportResp.StatusCode)
	}
	if ct := exportResp.Header.Get("Content-Type"); ct != "image/png" {
		t.Errorf("expected image/png, got %s", ct)
	}

	// 3. Export QR as PDF
	pdfReq := httptest.NewRequest("GET", "/api/v1/qr/"+qr.ID+"/export?format=pdf", nil)
	pdfReq.Header.Set("Authorization", "Bearer sess_user_demo123_test")
	pdfResp, _ := app.Test(pdfReq)
	if pdfResp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK for PDF export, got %d", pdfResp.StatusCode)
	}
}

func TestDocumentUploadAndSharingE2E(t *testing.T) {
	app, cleanup := setupTestApp(t)
	defer cleanup()

	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	part, _ := w.CreateFormFile("file", "report.pdf")
	_, _ = part.Write([]byte("%PDF-1.4 sample pdf content for testing"))
	w.Close()

	req := httptest.NewRequest("POST", "/api/v1/documents/upload", &b)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer sess_user_demo123_test")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("doc upload failed: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d", resp.StatusCode)
	}

	var uploadResp struct {
		Document    models.Document `json:"document"`
		DownloadURL string          `json:"download_url"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&uploadResp)
	if uploadResp.Document.Filename != "report.pdf" {
		t.Errorf("expected filename 'report.pdf', got %s", uploadResp.Document.Filename)
	}
}

func TestOpenAPISpecAndDocsEndpoint(t *testing.T) {
	app, cleanup := setupTestApp(t)
	defer cleanup()

	specReq := httptest.NewRequest("GET", "/api/v1/openapi.json", nil)
	specResp, err := app.Test(specReq)
	if err != nil {
		t.Fatalf("openapi spec request failed: %v", err)
	}
	if specResp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", specResp.StatusCode)
	}

	docsReq := httptest.NewRequest("GET", "/api/docs", nil)
	docsResp, err := app.Test(docsReq)
	if err != nil {
		t.Fatalf("docs request failed: %v", err)
	}
	if docsResp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", docsResp.StatusCode)
	}
}

func TestUnauthenticatedRouteProtection(t *testing.T) {
	app, cleanup := setupTestApp(t)
	defer cleanup()

	// 1. Unauthenticated request to /api/v1/links must return 401
	req := httptest.NewRequest("GET", "/api/v1/links", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized for unauthenticated /api/v1/links, got %d", resp.StatusCode)
	}

	// 2. Unauthenticated request to /api/v1/user/profile must return 401
	reqProfile := httptest.NewRequest("GET", "/api/v1/user/profile", nil)
	respProfile, err := app.Test(reqProfile)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if respProfile.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized for unauthenticated /api/v1/user/profile, got %d", respProfile.StatusCode)
	}
}
