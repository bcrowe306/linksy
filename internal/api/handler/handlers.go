package handler

import (
	"errors"
	"fmt"
	"html"
	"io"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/bcrowe306/linksy/internal/api/middleware"
	"github.com/bcrowe306/linksy/internal/models"
	"github.com/bcrowe306/linksy/internal/qr"
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
	case errors.Is(err, service.ErrWorkspaceSuspended):
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": err.Error(), "code": "workspace_suspended"})
	case errors.Is(err, service.ErrDomainBlocked):
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error(), "code": "domain_blocked"})
	case errors.Is(err, service.ErrAdminReauth):
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": err.Error(), "code": "admin_reauth"})
	case errors.Is(err, service.ErrForbidden), errors.Is(err, service.ErrOwnerProtected), errors.Is(err, service.ErrNotPlatformAdmin),
		errors.Is(err, service.ErrSelfAdminAction), errors.Is(err, service.ErrAccountSuspended),
		errors.Is(err, service.ErrSuspended), errors.Is(err, service.ErrNotMember),
		errors.Is(err, service.ErrInvitationMismatch):
		status = fiber.StatusForbidden
	case service.IsNotFound(err), errors.Is(err, service.ErrInvitationInvalid):
		status = fiber.StatusNotFound
	case errors.Is(err, service.ErrConflict), errors.Is(err, service.ErrAlreadyMember), errors.Is(err, service.ErrSlugTaken),
		errors.Is(err, service.ErrAlreadySubscribed):
		status = fiber.StatusConflict
	case errors.Is(err, service.ErrPaymentRequired):
		return c.Status(fiber.StatusPaymentRequired).JSON(fiber.Map{"error": err.Error(), "code": "subscription_required"})
	case errors.Is(err, service.ErrBillingDisabled), errors.Is(err, service.ErrEmailDisabled):
		status = fiber.StatusServiceUnavailable
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

// visitorEvent captures request metadata for analytics. Values are copied
// because the event is recorded asynchronously after the request buffer is reused.
func visitorEvent(c fiber.Ctx, eventType, teamID string, linkID, qrID *string) *models.AnalyticsEvent {
	return &models.AnalyticsEvent{
		TeamID:    teamID,
		LinkID:    linkID,
		QRCodeID:  qrID,
		EventType: eventType,
		IP:        strings.Clone(middleware.ClientIP(c)),
		UserAgent: strings.Clone(c.Get("User-Agent")),
		Referrer:  strings.Clone(c.Get("Referer")),
		Country:   strings.ToUpper(c.Get("CF-IPCountry")),
		Region:    strings.Clone(c.Get("CF-Region")),
		City:      strings.Clone(c.Get("CF-IPCity")),
	}
}

const unlockCookiePrefix = "lq_unlock_"

// HandleShortLinkRedirect serves /s/{slug}: fallback for paused/expired links,
// a password page for protected links, otherwise a redirect with UTM tags.
func (h *Handler) HandleShortLinkRedirect(c fiber.Ctx) error {
	slug := c.Params("slug")
	link, state, err := h.svc.LookupLink(slug)
	if err != nil {
		return h.renderBrandedError(c, "Link Not Found", fmt.Sprintf("The short link \"/s/%s\" does not exist or has been removed.", slug), fiber.StatusNotFound)
	}
	c.Set("Cache-Control", "no-store")
	c.Set("X-Robots-Tag", "noindex, nofollow")
	if blocked, err := h.gate(c, link.TeamID); blocked {
		return err
	}
	if state == service.LinkDisabled {
		return h.renderDisabled(c)
	}
	if state != service.LinkOK {
		if link.FallbackURL != "" {
			return c.Redirect().Status(fiber.StatusFound).To(link.FallbackURL)
		}
		title, msg := "Link Paused", "The owner of this link has temporarily paused it."
		if state == service.LinkExpired {
			title, msg = "Link Expired", "This link has expired and no longer redirects."
		}
		return h.renderBrandedError(c, title, msg, fiber.StatusGone)
	}
	if link.PasswordHash != "" && !h.svc.ValidUnlock(link, c.Cookies(unlockCookiePrefix+link.ID), time.Now()) {
		return h.renderPasswordPage(c, link, "", fiber.StatusUnauthorized)
	}
	return h.redirectLink(c, link)
}

// HandleShortLinkUnlock checks the password posted from the unlock page.
func (h *Handler) HandleShortLinkUnlock(c fiber.Ctx) error {
	link, state, err := h.svc.LookupLink(c.Params("slug"))
	if err != nil || state != service.LinkOK || link.PasswordHash == "" || h.svc.TeamLapsed(link.TeamID) {
		return c.Redirect().Status(fiber.StatusSeeOther).To("/s/" + c.Params("slug"))
	}
	if !h.svc.CheckLinkPassword(link, c.FormValue("password")) {
		return h.renderPasswordPage(c, link, "That password is incorrect. Please try again.", fiber.StatusUnauthorized)
	}
	c.Cookie(&fiber.Cookie{
		Name: unlockCookiePrefix + link.ID, Value: h.svc.UnlockToken(link, time.Now()),
		Path: "/s/" + link.Slug, MaxAge: int((12 * time.Hour).Seconds()), HTTPOnly: true,
		Secure: strings.HasPrefix(h.appURL, "https://"), SameSite: fiber.CookieSameSiteLaxMode,
	})
	return h.redirectLink(c, link)
}

func (h *Handler) redirectLink(c fiber.Ctx, link *models.Link) error {
	// Scans of a protected QR code arrive here after being counted as scans.
	if c.Query("src") != "qr" {
		h.svc.CountClick(link.ID)
		ev := visitorEvent(c, "click", link.TeamID, &link.ID, nil)
		go h.svc.TrackEvent(ev)
	}
	c.Set("Cache-Control", "no-store")
	return c.Redirect().Status(fiber.StatusFound).To(service.Destination(link))
}

// HandleQRScan serves /q/{code}: records the scan and sends the visitor to
// the code's current destination.
func (h *Handler) HandleQRScan(c fiber.Ctx) error {
	t, err := h.svc.ResolveScan(c.Params("code"), c.Get("User-Agent"))
	if err != nil {
		return h.renderBrandedError(c, "QR Code Not Found", "The QR code you scanned does not exist or has been removed.", fiber.StatusNotFound)
	}
	return h.serveScan(c, t)
}

// HandleLegacyQRScan keeps codes printed with /qr/r/{id} working.
func (h *Handler) HandleLegacyQRScan(c fiber.Ctx) error {
	t, err := h.svc.ResolveLegacyScan(c.Params("id"), c.Get("User-Agent"))
	if err != nil {
		return h.renderBrandedError(c, "QR Code Not Found", "The QR code you scanned does not exist or has been removed.", fiber.StatusNotFound)
	}
	return h.serveScan(c, t)
}

func (h *Handler) serveScan(c fiber.Ctx, t *service.ScanTarget) error {
	q := t.QR
	c.Set("Cache-Control", "no-store")
	if blocked, err := h.gate(c, q.TeamID); blocked {
		return err
	}
	if t.Disabled {
		return h.renderDisabled(c)
	}
	if t.Inactive {
		return h.renderBrandedError(c, "This QR Code Is Inactive", "The owner of this QR code has paused it or its destination has expired.", fiber.StatusGone)
	}

	h.svc.RecordQRScan(q.ID)
	ev := visitorEvent(c, "scan", q.TeamID, q.LinkID, &q.ID)
	go h.svc.TrackEvent(ev)

	switch {
	case t.VCard != "":
		c.Set("Content-Type", "text/vcard; charset=utf-8")
		c.Set("Content-Disposition", `inline; filename="contact.vcf"`)
		return c.SendString(t.VCard)
	case t.Redirect != "":
		return c.Redirect().Status(fiber.StatusFound).To(t.Redirect)
	case t.App != nil:
		return h.renderAppChooser(c, q.Title, t.App.AppStoreURL, t.App.PlayStoreURL)
	}
	// Static payloads scanned through the legacy route: show the raw content.
	c.Set("Content-Type", "text/plain; charset=utf-8")
	return c.SendString(q.Payload)
}

// HandleDocumentLink serves /d/{id}, a stable link to a document's current version.
func (h *Handler) HandleDocumentLink(c fiber.Ctx) error {
	target, teamID, err := h.svc.DocumentRedirect(c.Params("id"))
	if err == nil {
		if blocked, err := h.gate(c, teamID); blocked {
			return err
		}
	}
	if err != nil {
		return h.renderBrandedError(c, "Document Not Found", "This document has been removed.", fiber.StatusNotFound)
	}
	c.Set("Cache-Control", "no-store")
	return c.Redirect().Status(fiber.StatusFound).To(target)
}

// ----------------- Links API -----------------

type CreateLinkRequest struct {
	TargetURL   string     `json:"target_url"`
	CustomSlug  string     `json:"custom_slug"`
	Slug        string     `json:"slug"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	ExpiresAt   *time.Time `json:"expires_at"`
	Password    string     `json:"password"`
	FallbackURL string     `json:"fallback_url"`
	service.UTM
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
		Password:    req.Password,
		UTM:         req.UTM,
		FallbackURL: req.FallbackURL,
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

// UpdateLinkRequest: omitted optional fields are left unchanged; password ""
// removes protection.
type UpdateLinkRequest struct {
	TargetURL   string     `json:"target_url"`
	Slug        string     `json:"slug"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	IsActive    *bool      `json:"is_active"`
	ExpiresAt   *time.Time `json:"expires_at"`
	Password    *string    `json:"password"`
	FallbackURL *string    `json:"fallback_url"`
	UTMSource   *string    `json:"utm_source"`
	UTMMedium   *string    `json:"utm_medium"`
	UTMCampaign *string    `json:"utm_campaign"`
	UTMTerm     *string    `json:"utm_term"`
	UTMContent  *string    `json:"utm_content"`
}

func (r UpdateLinkRequest) utm(current *models.Link) *service.UTM {
	if r.UTMSource == nil && r.UTMMedium == nil && r.UTMCampaign == nil && r.UTMTerm == nil && r.UTMContent == nil {
		return nil
	}
	pick := func(p *string, cur string) string {
		if p == nil {
			return cur
		}
		return *p
	}
	return &service.UTM{
		Source: pick(r.UTMSource, current.UTMSource), Medium: pick(r.UTMMedium, current.UTMMedium),
		Campaign: pick(r.UTMCampaign, current.UTMCampaign), Term: pick(r.UTMTerm, current.UTMTerm),
		Content: pick(r.UTMContent, current.UTMContent),
	}
}

func (h *Handler) UpdateLink(c fiber.Ctx) error {
	var req UpdateLinkRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	a := middleware.Actor(c)
	current, err := h.svc.GetLink(a, c.Params("id"))
	if err != nil {
		return fail(c, err)
	}
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	updated, err := h.svc.UpdateShortLink(a, current.ID, service.UpdateLinkInput{
		TargetURL: req.TargetURL, Slug: req.Slug, Title: req.Title, Description: req.Description, IsActive: isActive,
		ExpiresAt: req.ExpiresAt, Password: req.Password, FallbackURL: req.FallbackURL, UTM: req.utm(current),
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
	LinkID     string         `json:"link_id"`
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
		LinkID:     req.LinkID,
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
	Title      string          `json:"title"`
	TargetURL  string          `json:"target_url"`
	Payload    string          `json:"payload"`
	RawData    string          `json:"raw_data"`
	Style      *models.QRStyle `json:"style"`
	DocumentID string          `json:"document_id"`
}

func (h *Handler) UpdateQR(c fiber.Ctx) error {
	var req UpdateQRRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	if req.TargetURL == "" && req.Payload != "" {
		req.TargetURL = req.Payload
	}
	updated, err := h.svc.UpdateQRCode(middleware.Actor(c), c.Params("id"), service.UpdateQRInput{
		Title: req.Title, TargetURL: req.TargetURL, RawData: req.RawData, Style: req.Style, DocumentID: req.DocumentID,
	})
	if err != nil {
		return fail(c, err)
	}
	return c.JSON(updated)
}

func (h *Handler) ExportQR(c fiber.Ctx) error {
	id := c.Params("id")
	format := strings.ToLower(c.Query("format", "png"))
	data, mime, err := h.svc.ExportQRCode(middleware.Actor(c), id, service.ExportOptions{
		Format: format, Size: queryInt(c, "size", 1024), Layout: qr.PDFLayout(c.Query("layout", "sheet")),
	})
	if err != nil {
		return fail(c, err)
	}

	c.Set("Content-Type", mime)
	c.Set("Cache-Control", "private, no-cache")
	if format == "jpg" {
		format = "jpeg"
	}
	if c.Query("download") == "true" {
		c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"lynqr-qr-%s.%s\"", id[:min(len(id), 8)], format))
	}
	return c.Send(data)
}

// PreviewQR renders an unsaved design for the studio (SVG + scannability check).
func (h *Handler) PreviewQR(c fiber.Ctx) error {
	var req service.PreviewInput
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	res, err := h.svc.Preview(middleware.Actor(c), req)
	if err != nil {
		return fail(c, err)
	}
	return c.JSON(res)
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
		Title:      truncateRunes("Doc: "+doc.Filename, 120),
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
	if blocked, err := h.gate(c, doc.TeamID); blocked {
		rc.Close()
		return err
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

// renderNotice shows a neutral confirmation page in the branded layout.
func (h *Handler) renderNotice(c fiber.Ctx, title, message string) error {
	c.Set("Cache-Control", "no-store")
	page := fmt.Sprintf(`<!DOCTYPE html><html lang="en"><head><meta charset="UTF-8"><meta name="viewport" content="width=device-width, initial-scale=1">
<meta name="robots" content="noindex"><title>%s | Lynqr</title><style>
*{box-sizing:border-box}body{margin:0;min-height:100vh;display:flex;align-items:center;justify-content:center;font-family:system-ui,-apple-system,'Segoe UI',sans-serif;background:#0c0c0e;color:#f4f4f6;padding:24px}
.card{max-width:420px;width:100%%;background:#16161a;border:1px solid #2c2c32;border-radius:20px;padding:32px;text-align:center}
.icon{width:48px;height:48px;margin:0 auto 16px;border-radius:14px;background:rgba(16,185,129,.15);display:flex;align-items:center;justify-content:center;color:#34d399}
h1{font-size:20px;margin:0 0 8px}p{color:#acacb4;font-size:14px;line-height:1.55;margin:0 0 20px}
a{display:block;padding:12px;border-radius:12px;background:linear-gradient(#6366f1,#4f46e5);color:#fff;font-weight:600;text-decoration:none}
</style></head><body><div class="card"><div class="icon"><svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><path d="M20 6 9 17l-5-5"/></svg></div>
<h1>%s</h1><p>%s</p><a href="/profile?tab=notifications">Email settings</a></div></body></html>`,
		html.EscapeString(title), html.EscapeString(title), html.EscapeString(message))
	return c.Type("html").SendString(page)
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
    <div class="footer">Lynqr short links &amp; QR codes · <a href="/report" style="color:#9ca3af">Report abuse</a></div>
  </div>
</body>
</html>`, html.EscapeString(title), html.EscapeString(title), html.EscapeString(message))

	return c.Status(statusCode).Type("html").SendString(page)
}

// renderAppChooser shows store buttons when the scanning device can't be matched.
func (h *Handler) renderAppChooser(c fiber.Ctx, title, ios, android string) error {
	btn := func(href, label string) string {
		if href == "" {
			return ""
		}
		return fmt.Sprintf(`<a class="btn" href="%s" rel="noopener">%s</a>`, html.EscapeString(href), label)
	}
	page := fmt.Sprintf(`<!DOCTYPE html><html lang="en"><head><meta charset="UTF-8"><meta name="viewport" content="width=device-width, initial-scale=1">
<title>%s | Lynqr</title><style>
body{margin:0;min-height:100vh;display:flex;align-items:center;justify-content:center;font-family:system-ui,-apple-system,sans-serif;background:#0c0c0e;color:#f4f4f6;padding:24px}
.card{max-width:380px;width:100%%;text-align:center;background:#16161a;border:1px solid #2c2c32;border-radius:20px;padding:32px}
h1{font-size:20px;margin:0 0 8px}p{color:#acacb4;font-size:14px;margin:0 0 24px}
.btn{display:block;margin-top:12px;padding:12px 16px;border-radius:12px;background:#4f46e5;color:#fff;text-decoration:none;font-weight:600}
</style></head><body><div class="card"><h1>%s</h1><p>Choose your app store to continue.</p>%s%s</div></body></html>`,
		html.EscapeString(title), html.EscapeString(title), btn(ios, "Download on the App Store"), btn(android, "Get it on Google Play"))
	return c.Type("html").SendString(page)
}

func truncateRunes(s string, n int) string {
	r := []rune(s)
	if len(r) > n {
		return string(r[:n])
	}
	return s
}

// renderPasswordPage asks for a protected link's password.
func (h *Handler) renderPasswordPage(c fiber.Ctx, link *models.Link, errMsg string, status int) error {
	errHTML := ""
	if errMsg != "" {
		errHTML = `<p class="err" role="alert">` + html.EscapeString(errMsg) + `</p>`
	}
	action := "/s/" + url.PathEscape(link.Slug)
	if c.Query("src") == "qr" {
		action += "?src=qr"
	}
	page := fmt.Sprintf(`<!DOCTYPE html><html lang="en"><head><meta charset="UTF-8"><meta name="viewport" content="width=device-width, initial-scale=1">
<meta name="robots" content="noindex, nofollow"><title>Protected link | Lynqr</title><style>
*{box-sizing:border-box}body{margin:0;min-height:100vh;display:flex;align-items:center;justify-content:center;font-family:system-ui,-apple-system,'Segoe UI',sans-serif;background:#0c0c0e;color:#f4f4f6;padding:24px}
.card{max-width:380px;width:100%%;background:#16161a;border:1px solid #2c2c32;border-radius:20px;padding:32px;text-align:center}
.icon{width:48px;height:48px;margin:0 auto 16px;border-radius:14px;background:rgba(99,102,241,.15);display:flex;align-items:center;justify-content:center;color:#a5b4fc}
h1{font-size:20px;margin:0 0 6px}p{color:#acacb4;font-size:14px;margin:0 0 20px}
input{width:100%%;padding:12px 14px;border-radius:12px;border:1px solid #3c3c44;background:#0c0c0e;color:#f4f4f6;font-size:15px;outline:none}
input:focus{border-color:#6366f1;box-shadow:0 0 0 4px rgba(99,102,241,.2)}
button{margin-top:12px;width:100%%;padding:12px;border:0;border-radius:12px;background:linear-gradient(#6366f1,#4f46e5);color:#fff;font-weight:600;font-size:15px;cursor:pointer}
.err{color:#fb7185;font-size:13px;margin:0 0 12px}.foot{margin-top:20px;font-size:12px;color:#7c7c86}
</style></head><body><form class="card" method="POST" action="%s">
<div class="icon"><svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="11" width="18" height="11" rx="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg></div>
<h1>This link is protected</h1><p>Enter the password to continue.</p>%s
<input type="password" name="password" placeholder="Password" autocomplete="current-password" autofocus required aria-label="Password">
<button type="submit">Continue</button><div class="foot">Secured by Lynqr</div></form></body></html>`,
		html.EscapeString(action), errHTML)
	// form-action also governs the redirect that follows a correct password, and
	// that redirect goes to the link's (external) destination.
	c.Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; form-action 'self' https: http:; frame-ancestors 'none'; base-uri 'none'")
	return c.Status(status).Type("html").SendString(page)
}

// renderInactiveWorkspace is shown to visitors when the owning workspace's subscription has lapsed.
func (h *Handler) renderInactiveWorkspace(c fiber.Ctx) error {
	c.Set("Cache-Control", "no-store")
	c.Set("X-Robots-Tag", "noindex, nofollow")
	return h.renderBrandedError(c, "Temporarily Unavailable",
		"This link or QR code is currently inactive. If it belongs to you, sign in to Lynqr to reactivate it.", fiber.StatusGone)
}
