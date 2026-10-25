package handler

import (
	"errors"
	"fmt"
	"html"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/bcrowe306/linksy/internal/api/middleware"
	"github.com/bcrowe306/linksy/internal/models"
	"github.com/bcrowe306/linksy/internal/service"
)

// MaxUploadBytes caps document uploads (CONVENTIONS §3.4).
const MaxUploadBytes = 15 << 20

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
	val, err := strconv.Atoi(valStr)
	if err != nil {
		return defaultVal
	}
	return val
}

// fail maps service errors onto HTTP status codes with a JSON body.
func fail(c fiber.Ctx, err error) error {
	status := fiber.StatusBadRequest
	switch {
	case errors.Is(err, service.ErrUnauthorized):
		status = fiber.StatusUnauthorized
	case errors.Is(err, service.ErrForbidden), errors.Is(err, service.ErrOwnerProtected),
		errors.Is(err, service.ErrSuspended), errors.Is(err, service.ErrNotMember),
		errors.Is(err, service.ErrInvitationMismatch):
		status = fiber.StatusForbidden
	case service.IsNotFound(err), errors.Is(err, service.ErrInvitationInvalid):
		status = fiber.StatusNotFound
	case errors.Is(err, service.ErrConflict), errors.Is(err, service.ErrAlreadyMember), errors.Is(err, service.ErrSlugTaken):
		status = fiber.StatusConflict
	}
	return c.Status(status).JSON(fiber.Map{"error": err.Error()})
}

// ----------------- Health -----------------

func (h *Handler) Health(c fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status":  "healthy",
		"service": "lynqr",
		"time":    time.Now().UTC(),
	})
}

// ----------------- Public Redirection -----------------

// visitorEvent captures request metadata for analytics before the handler returns.
func visitorEvent(c fiber.Ctx, eventType, teamID string, linkID, qrID *string) *models.AnalyticsEvent {
	return &models.AnalyticsEvent{
		TeamID:    teamID,
		LinkID:    linkID,
		QRCodeID:  qrID,
		EventType: eventType,
		IP:        middleware.ClientIP(c),
		UserAgent: c.Get("User-Agent"),
		Referrer:  c.Get("Referer"),
		Country:   strings.ToUpper(c.Get("CF-IPCountry")),
		Region:    c.Get("CF-Region"),
		City:      c.Get("CF-IPCity"),
	}
}

func (h *Handler) HandleShortLinkRedirect(c fiber.Ctx) error {
	slug := c.Params("slug")
	link, err := h.svc.ResolveShortLink(slug)
	if err != nil {
		return h.renderBrandedError(c, "Link Not Found or Inactive", fmt.Sprintf("The short link \"/s/%s\" does not exist, has expired, or has been removed.", slug), fiber.StatusNotFound)
	}

	ev := visitorEvent(c, "click", link.TeamID, &link.ID, nil)
	go h.svc.TrackEvent(ev)

	c.Set("Cache-Control", "no-store")
	return c.Redirect().Status(fiber.StatusFound).To(link.TargetURL)
}

func (h *Handler) HandleQRScanRedirect(c fiber.Ctx) error {
	qrCode, err := h.svc.GetPublicQRCode(c.Params("id"))
	if err != nil {
		return h.renderBrandedError(c, "QR Code Not Found", "The QR code you scanned does not exist, is inactive, or has been removed.", fiber.StatusNotFound)
	}

	h.svc.RecordQRScan(qrCode.ID)
	ev := visitorEvent(c, "scan", qrCode.TeamID, qrCode.LinkID, &qrCode.ID)
	go h.svc.TrackEvent(ev)

	c.Set("Cache-Control", "no-store")
	if qrCode.LinkID != nil || qrCode.Type == models.QRTypeWebsite || qrCode.Type == models.QRTypeDocument {
		return c.Redirect().Status(fiber.StatusFound).To(qrCode.Payload)
	}
	// For vCard, WiFi, plain text etc., deliver raw content
	c.Set("Content-Type", "text/plain; charset=utf-8")
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
	var req CreateLinkRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	if req.CustomSlug == "" && req.Slug != "" {
		req.CustomSlug = req.Slug
	}

	link, err := h.svc.CreateShortLink(middleware.Actor(c), service.CreateLinkInput{
		TargetURL:   req.TargetURL,
		CustomSlug:  req.CustomSlug,
		Title:       req.Title,
		Description: req.Description,
		ExpiresAt:   req.ExpiresAt,
	})
	if err != nil {
		return fail(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(link)
}

func (h *Handler) ListLinks(c fiber.Ctx) error {
	limit := queryInt(c, "limit", 50)
	offset := queryInt(c, "offset", 0)
	links, err := h.svc.ListLinks(middleware.Actor(c), limit, offset)
	if err != nil {
		return fail(c, err)
	}
	return c.JSON(fiber.Map{"data": links, "limit": limit, "offset": offset})
}

func (h *Handler) GetLink(c fiber.Ctx) error {
	link, err := h.svc.GetLink(middleware.Actor(c), c.Params("id"))
	if err != nil {
		return fail(c, err)
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
	var req UpdateLinkRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	updated, err := h.svc.UpdateShortLink(middleware.Actor(c), c.Params("id"), service.UpdateLinkInput{
		TargetURL: req.TargetURL, Title: req.Title, Description: req.Description, IsActive: isActive, ExpiresAt: req.ExpiresAt,
	})
	if err != nil {
		return fail(c, err)
	}
	return c.JSON(updated)
}

func (h *Handler) DeleteLink(c fiber.Ctx) error {
	if err := h.svc.DeleteLink(middleware.Actor(c), c.Params("id")); err != nil {
		return fail(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// ----------------- QR Codes API -----------------

type CreateQRRequest struct {
	Type       models.QRType  `json:"type"`
	Title      string         `json:"title"`
	TargetURL  string         `json:"target_url"`
	Payload    string         `json:"payload"`
	RawData    string         `json:"raw_data"`
	Style      models.QRStyle `json:"style"`
	IsDynamic  bool           `json:"is_dynamic"`
	DocumentID string         `json:"document_id"`
}

func (h *Handler) CreateQR(c fiber.Ctx) error {
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

	qrCode, err := h.svc.CreateQRCode(middleware.Actor(c), service.CreateQRInput{
		Type:       req.Type,
		Title:      req.Title,
		TargetURL:  req.TargetURL,
		RawData:    req.RawData,
		Style:      req.Style,
		IsDynamic:  req.IsDynamic,
		DocumentID: req.DocumentID,
	})
	if err != nil {
		return fail(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(qrCode)
}

func (h *Handler) ListQR(c fiber.Ctx) error {
	limit := queryInt(c, "limit", 50)
	offset := queryInt(c, "offset", 0)
	list, err := h.svc.ListQRCodes(middleware.Actor(c), limit, offset)
	if err != nil {
		return fail(c, err)
	}
	return c.JSON(fiber.Map{"data": list, "limit": limit, "offset": offset})
}

func (h *Handler) GetQR(c fiber.Ctx) error {
	qrCode, err := h.svc.GetQRCode(middleware.Actor(c), c.Params("id"))
	if err != nil {
		return fail(c, err)
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
	var req UpdateQRRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	if req.TargetURL == "" && req.Payload != "" {
		req.TargetURL = req.Payload
	}
	updated, err := h.svc.UpdateQRCode(middleware.Actor(c), c.Params("id"), req.Title, req.TargetURL, req.RawData, req.Style)
	if err != nil {
		return fail(c, err)
	}
	return c.JSON(updated)
}

func (h *Handler) ExportQR(c fiber.Ctx) error {
	id := c.Params("id")
	format := strings.ToLower(c.Query("format", "png"))
	data, mime, err := h.svc.ExportQRCode(middleware.Actor(c), id, format, queryInt(c, "size", 512))
	if err != nil {
		return fail(c, err)
	}

	c.Set("Content-Type", mime)
	c.Set("Cache-Control", "private, no-cache")
	if c.Query("download") == "true" {
		c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"lynqr-qr-%s.%s\"", id[:min(len(id), 8)], format))
	}
	return c.Send(data)
}

func (h *Handler) DeleteQR(c fiber.Ctx) error {
	if err := h.svc.DeleteQRCode(middleware.Actor(c), c.Params("id")); err != nil {
		return fail(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// ----------------- Documents API -----------------

func (h *Handler) readUpload(c fiber.Ctx) (string, io.ReadCloser, error) {
	file, err := c.FormFile("file")
	if err != nil {
		return "", nil, errors.New("missing file in multipart form")
	}
	if file.Size > MaxUploadBytes {
		return "", nil, errors.New("file exceeds the 15MB upload limit")
	}
	src, err := file.Open()
	if err != nil {
		return "", nil, errors.New("failed to read uploaded file")
	}
	return file.Filename, src, nil
}

func (h *Handler) UploadDocument(c fiber.Ctx) error {
	a := middleware.Actor(c)
	name, src, err := h.readUpload(c)
	if err != nil {
		return fail(c, err)
	}
	defer src.Close()

	doc, downloadURL, err := h.svc.UploadDocument(a, name, src)
	if err != nil {
		return fail(c, err)
	}

	// Auto-create dynamic QR code for this document (Requirement 8)
	qrCode, _ := h.svc.CreateQRCode(a, service.CreateQRInput{
		Type:       models.QRTypeDocument,
		Title:      "Doc: " + doc.Filename,
		TargetURL:  downloadURL,
		IsDynamic:  true,
		DocumentID: doc.ID,
	})

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"document":     doc,
		"download_url": downloadURL,
		"qr_code":      qrCode,
	})
}

func (h *Handler) UpdateDocumentFile(c fiber.Ctx) error {
	name, src, err := h.readUpload(c)
	if err != nil {
		return fail(c, err)
	}
	defer src.Close()

	doc, downloadURL, err := h.svc.UpdateDocumentFile(middleware.Actor(c), c.Params("id"), name, src)
	if err != nil {
		return fail(c, err)
	}
	return c.JSON(fiber.Map{"document": doc, "download_url": downloadURL})
}

// DownloadDocumentFile serves a document by its random storage key (public by design:
// it is the destination behind document QR codes).
func (h *Handler) DownloadDocumentFile(c fiber.Ctx) error {
	doc, rc, err := h.svc.OpenPublicDocument(c.Params("filename"))
	if err != nil {
		return h.renderBrandedError(c, "Document Not Found", "This document has been removed or replaced.", fiber.StatusNotFound)
	}
	// SendStream closes rc once the body has been written.

	c.Set("Cache-Control", "no-cache, no-store, must-revalidate")
	// The SPA's CSP would break browser PDF viewers, so documents carry none.
	// Active content (HTML, SVG, XML) is never rendered inline in our origin.
	c.Response().Header.Del("Content-Security-Policy")
	disposition, mime := "inline", doc.MimeType
	if lower := strings.ToLower(mime); strings.Contains(lower, "html") || strings.Contains(lower, "xml") || strings.Contains(lower, "svg") || strings.Contains(lower, "javascript") {
		disposition, mime = "attachment", "application/octet-stream"
	}
	c.Set("Content-Type", mime)
	c.Set("Content-Disposition", fmt.Sprintf("%s; filename=%q", disposition, strings.ReplaceAll(doc.Filename, "\"", "")))
	return c.SendStream(rc)
}

func (h *Handler) ListDocuments(c fiber.Ctx) error {
	docs, err := h.svc.ListDocuments(middleware.Actor(c))
	if err != nil {
		return fail(c, err)
	}
	return c.JSON(fiber.Map{"data": docs})
}

func (h *Handler) DeleteDocument(c fiber.Ctx) error {
	deletedQRs, err := h.svc.DeleteDocument(middleware.Actor(c), c.Params("id"), c.Query("delete_qr") == "true")
	if err != nil {
		return fail(c, err)
	}
	return c.JSON(fiber.Map{"message": "Document deleted successfully", "deleted_qr_count": deletedQRs})
}

func (h *Handler) BulkDeleteDocuments(c fiber.Ctx) error {
	var req models.BulkDeleteDocumentsRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	if len(req.DocumentIDs) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "No documents specified for deletion"})
	}
	deletedDocs, deletedQRs, err := h.svc.BulkDeleteDocuments(middleware.Actor(c), req.DocumentIDs, req.DeleteQR)
	if err != nil {
		return fail(c, err)
	}
	return c.JSON(fiber.Map{
		"message":            "Documents deleted successfully",
		"deleted_docs_count": deletedDocs,
		"deleted_qr_count":   deletedQRs,
	})
}

// ----------------- Analytics API -----------------

func parseTimeParam(v string) (*time.Time, error) {
	if v == "" {
		return nil, nil
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02"} {
		if t, err := time.Parse(layout, v); err == nil {
			return &t, nil
		}
	}
	return nil, fmt.Errorf("invalid date %q: use RFC3339 or YYYY-MM-DD", v)
}

func (h *Handler) GetAnalyticsSummary(c fiber.Ctx) error {
	from, err := parseTimeParam(c.Query("from"))
	if err != nil {
		return fail(c, err)
	}
	to, err := parseTimeParam(c.Query("to"))
	if err != nil {
		return fail(c, err)
	}
	summary, err := h.svc.GetAnalyticsSummary(middleware.Actor(c), service.AnalyticsQuery{
		LinkID: c.Query("link_id"), QRID: c.Query("qr_id"), From: from, To: to,
	})
	if err != nil {
		return fail(c, err)
	}
	return c.JSON(summary)
}

func (h *Handler) renderBrandedError(c fiber.Ctx, title, message string, statusCode int) error {
	accept := c.Get("Accept")
	if strings.Contains(accept, "application/json") && !strings.Contains(accept, "text/html") {
		return c.Status(statusCode).JSON(fiber.Map{
			"error": message,
			"code":  statusCode,
		})
	}

	page := fmt.Sprintf(`<!DOCTYPE html>
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
</html>`, html.EscapeString(title), html.EscapeString(title), html.EscapeString(message))

	return c.Status(statusCode).Type("html").SendString(page)
}
