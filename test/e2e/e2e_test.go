package e2e_test

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gofiber/fiber/v3"

	"github.com/bcrowe306/linksy/internal/api"
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

	database, err := db.Connect("sqlite", filepath.Join(tmpDir, "e2e.db"))
	if err != nil {
		t.Fatalf("failed initializing db: %v", err)
	}

	repo := repository.New(database)
	_ = repo.EnsureDefaultTeam("team_default", "Default Team", "usr_demo123")

	store, err := storage.NewLocalStorage(filepath.Join(tmpDir, "storage"), "http://example.com")
	if err != nil {
		t.Fatalf("failed initializing storage: %v", err)
	}

	svc := service.New(repo, store, "http://example.com")

	app := fiber.New()
	api.SetupRouter(app, svc, "http://example.com")

	cleanup := func() {
		database.Close()
		os.RemoveAll(tmpDir)
	}

	return app, cleanup
}

func TestHealthEndpoint(t *testing.T) {
	app, cleanup := setupTestApp(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}
}

func TestLinksAndRedirectE2E(t *testing.T) {
	app, cleanup := setupTestApp(t)
	defer cleanup()

	// 1. Create Link
	payload := []byte(`{"target_url":"https://google.com","title":"Google Search","custom_slug":"my-goog"}`)
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/links", bytes.NewReader(payload))
	createReq.Header.Set("Content-Type", "application/json")

	createResp, err := app.Test(createReq)
	if err != nil {
		t.Fatalf("create link request error: %v", err)
	}
	if createResp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(createResp.Body)
		t.Fatalf("expected 201 Created, got %d: %s", createResp.StatusCode, string(body))
	}

	var created models.Link
	_ = json.NewDecoder(createResp.Body).Decode(&created)
	if created.Slug != "my-goog" {
		t.Fatalf("expected slug 'my-goog', got '%s'", created.Slug)
	}

	// 2. Resolve via Short Link redirect
	redirReq := httptest.NewRequest(http.MethodGet, "/s/my-goog", nil)
	redirReq.Header.Set("User-Agent", "Mozilla/5.0 (iPhone; CPU iPhone OS 16_0 like Mac OS X)")
	redirResp, err := app.Test(redirReq)
	if err != nil {
		t.Fatalf("redirect request error: %v", err)
	}

	if redirResp.StatusCode != http.StatusFound {
		t.Fatalf("expected 302 Found, got %d", redirResp.StatusCode)
	}
	if location := redirResp.Header.Get("Location"); location != "https://google.com" {
		t.Fatalf("expected redirect to https://google.com, got %s", location)
	}

	// 3. Update destination dynamically (Requirement 6)
	updatePayload := []byte(`{"target_url":"https://duckduckgo.com","title":"DuckDuckGo","is_active":true}`)
	updateReq := httptest.NewRequest(http.MethodPut, "/api/v1/links/"+created.ID, bytes.NewReader(updatePayload))
	updateReq.Header.Set("Content-Type", "application/json")

	updateResp, err := app.Test(updateReq)
	if err != nil {
		t.Fatalf("update request error: %v", err)
	}
	if updateResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(updateResp.Body)
		t.Fatalf("expected 200 OK, got %d: %s", updateResp.StatusCode, string(body))
	}

	// 4. Resolve again to verify dynamic update
	redirResp2, _ := app.Test(httptest.NewRequest(http.MethodGet, "/s/my-goog", nil))
	if location := redirResp2.Header.Get("Location"); location != "https://duckduckgo.com" {
		t.Fatalf("expected redirect to updated target https://duckduckgo.com, got %s", location)
	}
}

func TestQRCodeAndExportE2E(t *testing.T) {
	app, cleanup := setupTestApp(t)
	defer cleanup()

	// Create styled QR Code
	qrReqBody := []byte(`{
		"type": "website",
		"title": "My Portfolio",
		"target_url": "https://portfolio.example.com",
		"is_dynamic": true,
		"style": {
			"fg_color": "#4F46E5",
			"bg_color": "#F8FAFC",
			"border_width": 2,
			"border_color": "#6366F1",
			"label_text": "VISIT PORTFOLIO"
		}
	}`)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/qr", bytes.NewReader(qrReqBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("create QR request error: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 201 Created, got %d: %s", resp.StatusCode, string(body))
	}

	var qrCode models.QRCode
	_ = json.NewDecoder(resp.Body).Decode(&qrCode)

	// Test Export as PNG
	exportPNGReq := httptest.NewRequest(http.MethodGet, "/api/v1/qr/"+qrCode.ID+"/export?format=png", nil)
	pngResp, _ := app.Test(exportPNGReq)
	if pngResp.StatusCode != http.StatusOK || pngResp.Header.Get("Content-Type") != "image/png" {
		t.Fatalf("PNG export failed: code=%d, type=%s", pngResp.StatusCode, pngResp.Header.Get("Content-Type"))
	}

	// Test Export as PDF
	exportPDFReq := httptest.NewRequest(http.MethodGet, "/api/v1/qr/"+qrCode.ID+"/export?format=pdf", nil)
	pdfResp, _ := app.Test(exportPDFReq)
	if pdfResp.StatusCode != http.StatusOK || pdfResp.Header.Get("Content-Type") != "application/pdf" {
		t.Fatalf("PDF export failed: code=%d, type=%s", pdfResp.StatusCode, pdfResp.Header.Get("Content-Type"))
	}
}

func TestDocumentUploadAndSharingE2E(t *testing.T) {
	app, cleanup := setupTestApp(t)
	defer cleanup()

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("file", "quarterly_report.pdf")
	if err != nil {
		t.Fatalf("failed creating form file: %v", err)
	}
	// Write dummy PDF header and content
	_, _ = part.Write([]byte("%PDF-1.4 sample quarterly report content"))
	_ = writer.Close()

	uploadReq := httptest.NewRequest(http.MethodPost, "/api/v1/documents/upload", &buf)
	uploadReq.Header.Set("Content-Type", writer.FormDataContentType())

	uploadResp, err := app.Test(uploadReq)
	if err != nil {
		t.Fatalf("upload request error: %v", err)
	}
	if uploadResp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(uploadResp.Body)
		t.Fatalf("expected 201 Created for document upload, got %d: %s", uploadResp.StatusCode, string(b))
	}
}

func TestOpenAPISpecAndDocsEndpoint(t *testing.T) {
	app, cleanup := setupTestApp(t)
	defer cleanup()

	// Check OpenAPI JSON
	specReq := httptest.NewRequest(http.MethodGet, "/api/v1/openapi.json", nil)
	specResp, _ := app.Test(specReq)
	if specResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for OpenAPI spec, got %d", specResp.StatusCode)
	}

	// Check interactive Scalar UI docs
	docsReq := httptest.NewRequest(http.MethodGet, "/api/docs", nil)
	docsResp, _ := app.Test(docsReq)
	if docsResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for Scalar docs UI, got %d", docsResp.StatusCode)
	}
}
