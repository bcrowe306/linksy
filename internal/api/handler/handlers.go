package handler

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/bcrowe306/linksy/internal/api/middleware"
	"github.com/bcrowe306/linksy/internal/models"
	"github.com/bcrowe306/linksy/internal/service"
)

type Handler struct {
	svc    *service.Service
	appURL string
}

func New(svc *service.Service, appURL string) *Handler {
	return &Handler{
		svc:    svc,
		appURL: appURL,
	}
}

func queryInt(c fiber.Ctx, key string, defaultVal int) int {
	valStr := c.Query(key)
	if valStr == "" {
		return defaultVal
	}
	if val, err := strconv.Atoi(valStr); err == nil {
		return val
	}
	return defaultVal
}

// ----------------- Health -----------------

func (h *Handler) Health(c fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status":  "healthy",
		"service": "linksy",
		"time":    time.Now().UTC(),
	})
}

// ----------------- Short Link Redirection -----------------

func (h *Handler) HandleShortLinkRedirect(c fiber.Ctx) error {
	slug := c.Params("slug")
	if slug == "" {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid link slug")
	}

	link, err := h.svc.ResolveShortLink(slug)
	if err != nil {
		return h.renderBrandedError(c, "Link Not Found or Inactive", fmt.Sprintf("The short link \"/s/%s\" does not exist, has expired, or has been removed.", slug), fiber.StatusNotFound)
	}

	// Capture analytics context safely before returning
	ip := c.IP()
	ua := c.Get("User-Agent")
	ref := c.Get("Referer")
	country := c.Get("CF-IPCountry")

	go func() {
		h.svc.TrackEvent(&models.AnalyticsEvent{
			LinkID:    &link.ID,
			EventType: "click",
			IP:        ip,
			UserAgent: ua,
			Referrer:  ref,
			Country:   country,
		})
	}()

	return c.Redirect().Status(fiber.StatusFound).To(link.TargetURL)
}

// ----------------- QR Dynamic Scan Redirection -----------------

func (h *Handler) HandleQRScanRedirect(c fiber.Ctx) error {
	id := c.Params("id")
	qrCode, err := h.svc.GetQRCode(id)
	if err != nil {
		return h.renderBrandedError(c, "QR Code Not Found", "The QR code you scanned does not exist, is inactive, or has been removed.", fiber.StatusNotFound)
	}

	// Track QR scan
	h.svc.RecordQRScan(id)

	ip := c.IP()
	ua := c.Get("User-Agent")
	ref := c.Get("Referer")
	country := c.Get("CF-IPCountry")

	go func() {
		h.svc.TrackEvent(&models.AnalyticsEvent{
			QRCodeID:  &qrCode.ID,
			LinkID:    qrCode.LinkID,
			EventType: "scan",
			IP:        ip,
			UserAgent: ua,
			Referrer:  ref,
			Country:   country,
		})
	}()

	// If linked to dynamic short link or website, redirect!
	if qrCode.LinkID != nil || qrCode.Type == models.QRTypeWebsite || qrCode.Type == models.QRTypeDocument {
		return c.Redirect().Status(fiber.StatusFound).To(qrCode.Payload)
	}

	// For vCard, WiFi, plain text etc., deliver raw content
	return c.SendString(qrCode.Payload)
}

// ----------------- Links API -----------------

type CreateLinkRequest struct {
	TargetURL   string     `json:"target_url"`
	CustomSlug  string     `json:"custom_slug"`
	Slug        string     `json:"slug"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	ExpiresAt   *time.Time `json:"expires_at"`
}

func (h *Handler) CreateLink(c fiber.Ctx) error {
	auth := middleware.GetAuth(c)
	var req CreateLinkRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if req.CustomSlug == "" && req.Slug != "" {
		req.CustomSlug = req.Slug
	}

	link, err := h.svc.CreateShortLink(service.CreateLinkInput{
		TeamID:      auth.TeamID,
		UserID:      auth.UserID,
		TargetURL:   req.TargetURL,
		CustomSlug:  req.CustomSlug,
		Title:       req.Title,
		Description: req.Description,
		ExpiresAt:   req.ExpiresAt,
	})
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(link)
}

func (h *Handler) ListLinks(c fiber.Ctx) error {
	auth := middleware.GetAuth(c)
	limit := queryInt(c, "limit", 50)
	offset := queryInt(c, "offset", 0)

	links, err := h.svc.ListLinks(auth.TeamID, limit, offset)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"data":   links,
		"limit":  limit,
		"offset": offset,
	})
}

func (h *Handler) GetLink(c fiber.Ctx) error {
	id := c.Params("id")
	auth := middleware.GetAuth(c)
	teamID := ""
	if auth != nil {
		teamID = auth.TeamID
	}
	link, err := h.svc.GetLink(id, teamID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Link not found"})
	}
	return c.JSON(link)
}

type UpdateLinkRequest struct {
	TargetURL   string     `json:"target_url"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	IsActive    *bool      `json:"is_active"`
	ExpiresAt   *time.Time `json:"expires_at"`
}

func (h *Handler) UpdateLink(c fiber.Ctx) error {
	id := c.Params("id")
	auth := middleware.GetAuth(c)
	var req UpdateLinkRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	teamID := ""
	if auth != nil {
		teamID = auth.TeamID
	}
	updated, err := h.svc.UpdateShortLink(id, req.TargetURL, req.Title, req.Description, isActive, req.ExpiresAt, teamID)
	if err != nil {
		if errors.Is(err, service.ErrLinkNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Link not found"})
		}
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(updated)
}

func (h *Handler) DeleteLink(c fiber.Ctx) error {
	id := c.Params("id")
	auth := middleware.GetAuth(c)
	if err := h.svc.DeleteLink(id, auth.TeamID); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Link not found"})
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// ----------------- QR Codes API -----------------

type CreateQRRequest struct {
	Type      models.QRType  `json:"type"`
	Title     string         `json:"title"`
	TargetURL string         `json:"target_url"`
	Payload   string         `json:"payload"`
	RawData   string         `json:"raw_data"`
	Style     models.QRStyle `json:"style"`
	IsDynamic bool           `json:"is_dynamic"`
}

func (h *Handler) CreateQR(c fiber.Ctx) error {
	auth := middleware.GetAuth(c)
	var req CreateQRRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if req.TargetURL == "" && req.Payload != "" {
		req.TargetURL = req.Payload
	}
	if req.RawData == "" && req.Payload != "" && req.Type != models.QRTypeWebsite {
		req.RawData = req.Payload
	}
	if req.TargetURL == "" && req.RawData == "" {
		req.TargetURL = "https://lynqr.app"
	}

	qrCode, err := h.svc.CreateQRCode(service.CreateQRInput{
		TeamID:    auth.TeamID,
		UserID:    auth.UserID,
		Type:      req.Type,
		Title:     req.Title,
		TargetURL: req.TargetURL,
		RawData:   req.RawData,
		Style:     req.Style,
		IsDynamic: req.IsDynamic,
	})
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(qrCode)
}

func (h *Handler) ListQR(c fiber.Ctx) error {
	auth := middleware.GetAuth(c)
	limit := queryInt(c, "limit", 50)
	offset := queryInt(c, "offset", 0)

	list, err := h.svc.ListQRCodes(auth.TeamID, limit, offset)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"data":   list,
		"limit":  limit,
		"offset": offset,
	})
}

func (h *Handler) GetQR(c fiber.Ctx) error {
	id := c.Params("id")
	auth := middleware.GetAuth(c)
	qrCode, err := h.svc.GetQRCode(id)
	if err != nil || (auth != nil && auth.TeamID != "" && qrCode.TeamID != auth.TeamID) {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "QR code not found"})
	}
	return c.JSON(qrCode)
}

type UpdateQRRequest struct {
	Title     string         `json:"title"`
	TargetURL string         `json:"target_url"`
	Payload   string         `json:"payload"`
	RawData   string         `json:"raw_data"`
	Style     models.QRStyle `json:"style"`
}

func (h *Handler) UpdateQR(c fiber.Ctx) error {
	id := c.Params("id")
	auth := middleware.GetAuth(c)
	var req UpdateQRRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if req.TargetURL == "" && req.Payload != "" {
		req.TargetURL = req.Payload
	}

	teamID := ""
	if auth != nil {
		teamID = auth.TeamID
	}
	updated, err := h.svc.UpdateQRCode(id, req.Title, req.TargetURL, req.RawData, req.Style, teamID)
	if err != nil {
		if errors.Is(err, service.ErrQRNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "QR code not found"})
		}
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(updated)
}

func (h *Handler) ExportQR(c fiber.Ctx) error {
	id := c.Params("id")
	format := c.Query("format", "png")
	size := queryInt(c, "size", 512)
	download := c.Query("download", "false") == "true"

	data, mime, err := h.svc.ExportQRCode(id, format, size)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	c.Set("Content-Type", mime)
	if download {
		filename := fmt.Sprintf("linksy-qr-%s.%s", id[:min(len(id), 8)], format)
		c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	}

	return c.Send(data)
}

func (h *Handler) DeleteQR(c fiber.Ctx) error {
	id := c.Params("id")
	auth := middleware.GetAuth(c)
	if err := h.svc.DeleteQRCode(id, auth.TeamID); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "QR code not found"})
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// ----------------- Documents API -----------------

func (h *Handler) UploadDocument(c fiber.Ctx) error {
	auth := middleware.GetAuth(c)
	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Missing file in multipart form"})
	}

	src, err := file.Open()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to read uploaded file"})
	}
	defer src.Close()

	doc, downloadURL, err := h.svc.UploadDocument(auth.TeamID, auth.UserID, file.Filename, src)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	// Auto-create dynamic QR code for this document (Requirement 8)
	qrCode, _ := h.svc.CreateQRCode(service.CreateQRInput{
		TeamID:    auth.TeamID,
		UserID:    auth.UserID,
		Type:      models.QRTypeDocument,
		Title:     "Doc: " + doc.Filename,
		TargetURL: downloadURL,
		IsDynamic: true,
	})

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"document":     doc,
		"download_url": downloadURL,
		"qr_code":      qrCode,
	})
}

func (h *Handler) UpdateDocumentFile(c fiber.Ctx) error {
	id := c.Params("id")
	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Missing file in multipart form"})
	}

	src, err := file.Open()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to read uploaded file"})
	}
	defer src.Close()

	doc, downloadURL, err := h.svc.UpdateDocumentFile(id, file.Filename, src)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"document":     doc,
		"download_url": downloadURL,
	})
}

func (h *Handler) DownloadDocumentFile(c fiber.Ctx) error {
	c.Set("Cache-Control", "no-cache, no-store, must-revalidate")
	filename := c.Params("filename")
	rc, err := h.svc.OpenDocumentFile(filename)
	if err != nil {
		return c.Status(fiber.StatusNotFound).SendString("Document not found")
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Failed reading file stream")
	}

	return c.Send(data)
}

func (h *Handler) ListDocuments(c fiber.Ctx) error {
	auth := middleware.GetAuth(c)
	docs, err := h.svc.ListDocuments(auth.TeamID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"data": docs})
}

// ----------------- Analytics API -----------------

func (h *Handler) GetAnalyticsSummary(c fiber.Ctx) error {
	linkID := c.Query("link_id")
	qrID := c.Query("qr_id")

	var pLinkID, pQRID *string
	if linkID != "" {
		pLinkID = &linkID
	}
	if qrID != "" {
		pQRID = &qrID
	}

	summary, err := h.svc.GetAnalyticsSummary(pLinkID, pQRID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(summary)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}


func (h *Handler) DeleteDocument(c fiber.Ctx) error {
	id := c.Params("id")
	auth := middleware.GetAuth(c)
	deleteQR := c.Query("delete_qr", "false") == "true"

	deletedQRs, err := h.svc.DeleteDocument(id, auth.TeamID, deleteQR)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Document not found or delete failed"})
	}

	return c.JSON(fiber.Map{
		"message":          "Document deleted successfully",
		"deleted_qr_count": deletedQRs,
	})
}

func (h *Handler) BulkDeleteDocuments(c fiber.Ctx) error {
	auth := middleware.GetAuth(c)
	var req models.BulkDeleteDocumentsRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if len(req.DocumentIDs) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "No documents specified for deletion"})
	}

	deletedDocs, deletedQRs, err := h.svc.BulkDeleteDocuments(req.DocumentIDs, auth.TeamID, req.DeleteQR)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"message":            "Documents deleted successfully",
		"deleted_docs_count": deletedDocs,
		"deleted_qr_count":   deletedQRs,
	})
}

func (h *Handler) renderBrandedError(c fiber.Ctx, title, message string, statusCode int) error {
	accept := c.Get("Accept")
	if strings.Contains(accept, "application/json") && !strings.Contains(accept, "text/html") {
		return c.Status(statusCode).JSON(fiber.Map{
			"error": message,
			"code":  statusCode,
		})
	}

	html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>%s | Lynqr</title>
  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
  <link href="https://fonts.googleapis.com/css2?family=Plus+Jakarta+Sans:wght@400;500;600;700;800&display=swap" rel="stylesheet">
  <style>
    * { box-sizing: border-box; margin: 0; padding: 0; }
    body {
      background-color: #030712;
      color: #f3f4f6;
      font-family: 'Plus Jakarta Sans', system-ui, -apple-system, sans-serif;
      min-height: 100vh;
      display: flex;
      flex-direction: column;
      align-items: center;
      justify-content: center;
      padding: 1.5rem;
      position: relative;
      overflow-x: hidden;
    }
    .glow {
      position: absolute;
      width: 450px;
      height: 450px;
      border-radius: 50%%;
      background: radial-gradient(circle, rgba(16, 185, 129, 0.15) 0%%, rgba(14, 165, 233, 0.08) 50%%, transparent 70%%);
      filter: blur(50px);
      z-index: 0;
      pointer-events: none;
    }
    .card {
      position: relative;
      z-index: 1;
      background: rgba(17, 24, 39, 0.75);
      border: 1px solid rgba(55, 65, 81, 0.7);
      backdrop-filter: blur(20px);
      border-radius: 1.5rem;
      padding: 2.5rem;
      max-width: 480px;
      width: 100%%;
      text-align: center;
      box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.6);
    }
    .icon-wrapper {
      width: 4.5rem;
      height: 4.5rem;
      border-radius: 1.25rem;
      background: rgba(239, 68, 68, 0.12);
      border: 1px solid rgba(239, 68, 68, 0.25);
      color: #f87171;
      display: inline-flex;
      align-items: center;
      justify-content: center;
      margin-bottom: 1.5rem;
      box-shadow: 0 0 24px rgba(239, 68, 68, 0.15);
    }
    .brand {
      display: inline-flex;
      align-items: center;
      gap: 0.5rem;
      font-size: 0.875rem;
      font-weight: 700;
      letter-spacing: 0.05em;
      text-transform: uppercase;
      color: #10b981;
      margin-bottom: 0.75rem;
    }
    .brand-dot {
      width: 0.5rem;
      height: 0.5rem;
      border-radius: 50%%;
      background: #10b981;
      box-shadow: 0 0 8px #10b981;
    }
    h1 {
      font-size: 1.5rem;
      font-weight: 800;
      color: #ffffff;
      line-height: 1.3;
      margin-bottom: 0.75rem;
    }
    p {
      color: #9ca3af;
      font-size: 0.875rem;
      line-height: 1.6;
      margin-bottom: 2rem;
    }
    .btn {
      display: inline-flex;
      align-items: center;
      justify-content: center;
      gap: 0.5rem;
      width: 100%%;
      padding: 0.875rem 1.5rem;
      background: #10b981;
      color: #ffffff;
      font-weight: 600;
      font-size: 0.875rem;
      border-radius: 0.75rem;
      text-decoration: none;
      transition: all 0.2s;
      box-shadow: 0 4px 12px rgba(16, 185, 129, 0.25);
    }
    .btn:hover {
      background: #059669;
      transform: translateY(-1px);
      box-shadow: 0 6px 16px rgba(16, 185, 129, 0.35);
    }
    .footer {
      margin-top: 1.5rem;
      font-size: 0.75rem;
      color: #6b7280;
    }
  </style>
</head>
<body>
  <div class="glow"></div>
  <div class="card">
    <div class="brand">
      <div class="brand-dot"></div>
      <span>Lynqr Smart Routing</span>
    </div>
    <div class="icon-wrapper">
      <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"></path>
        <path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"></path>
        <line x1="2" y1="2" x2="22" y2="22"></line>
      </svg>
    </div>
    <h1>%s</h1>
    <p>%s</p>
    <a href="/" class="btn">
      <span>Return to Lynqr Home</span>
      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m9 18 6-6-6-6"/></svg>
    </a>
    <div class="footer">Protected by Lynqr Link Management & Dynamic QR Platform</div>
  </div>
</body>
</html>`, title, title, message)

	return c.Status(statusCode).Type("html").SendString(html)
}
