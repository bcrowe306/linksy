package handler

import (
	"fmt"
	"io"
	"strconv"
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
		return c.Status(fiber.StatusNotFound).SendString("Link not found or expired")
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
		return c.Status(fiber.StatusNotFound).SendString("QR Code not found")
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

type UpdateLinkRequest struct {
	TargetURL   string     `json:"target_url"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	IsActive    *bool      `json:"is_active"`
	ExpiresAt   *time.Time `json:"expires_at"`
}

func (h *Handler) UpdateLink(c fiber.Ctx) error {
	id := c.Params("id")
	var req UpdateLinkRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	updated, err := h.svc.UpdateShortLink(id, req.TargetURL, req.Title, req.Description, isActive, req.ExpiresAt)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(updated)
}

func (h *Handler) DeleteLink(c fiber.Ctx) error {
	id := c.Params("id")
	if err := h.svc.DeleteLink(id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// ----------------- QR Codes API -----------------

type CreateQRRequest struct {
	Type      models.QRType  `json:"type"`
	Title     string         `json:"title"`
	TargetURL string         `json:"target_url"`
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
	qrCode, err := h.svc.GetQRCode(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "QR code not found"})
	}
	return c.JSON(qrCode)
}

type UpdateQRRequest struct {
	Title     string         `json:"title"`
	TargetURL string         `json:"target_url"`
	RawData   string         `json:"raw_data"`
	Style     models.QRStyle `json:"style"`
}

func (h *Handler) UpdateQR(c fiber.Ctx) error {
	id := c.Params("id")
	var req UpdateQRRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	updated, err := h.svc.UpdateQRCode(id, req.Title, req.TargetURL, req.RawData, req.Style)
	if err != nil {
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
	if err := h.svc.DeleteQRCode(id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
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
