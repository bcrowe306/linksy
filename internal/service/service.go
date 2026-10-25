package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/bcrowe306/linksy/internal/auth"
	"github.com/bcrowe306/linksy/internal/models"
	"github.com/bcrowe306/linksy/internal/qr"
	"github.com/bcrowe306/linksy/internal/repository"
	"github.com/bcrowe306/linksy/internal/storage"
)

var (
	ErrInvalidURL       = errors.New("invalid target URL: only http and https protocols are supported")
	ErrDisallowedTarget = errors.New("target URL cannot point to internal network or loopback addresses")
	ErrLinkNotFound     = errors.New("short link not found or has expired")
	ErrQRNotFound       = errors.New("QR code not found")
	ErrDocNotFound      = errors.New("document not found")
	ErrInvalidSlug      = errors.New("custom slug may only contain letters, numbers, '-' and '_' (3-64 characters)")
	ErrSlugTaken        = errors.New("that custom slug is already in use")

	// Generic classes the HTTP layer maps onto status codes.
	ErrNotFound     = repository.ErrNotFound
	ErrForbidden    = errors.New("you do not have permission to perform this action")
	ErrUnauthorized = errors.New("authentication required")
	ErrConflict     = errors.New("conflict")
)

// Actor identifies who is acting, in which team, with which role.
// Every team-scoped service call takes an Actor so tenancy is never optional.
type Actor struct {
	UserID string
	TeamID string
	Role   models.UserRole
	IP     string
}

func (a Actor) require(p auth.Permission) error {
	if a.UserID == "" || a.TeamID == "" {
		return ErrUnauthorized
	}
	if !auth.Can(a.Role, p) {
		return ErrForbidden
	}
	return nil
}

type Service struct {
	repo    *repository.Repository
	storage storage.StorageProvider
	appURL  string
	ipSalt  string
}

func New(repo *repository.Repository, storage storage.StorageProvider, appURL string) *Service {
	return &Service{
		repo:    repo,
		storage: storage,
		appURL:  strings.TrimRight(appURL, "/"),
	}
}

// SetIPSalt configures the secret mixed into analytics IP hashes.
func (s *Service) SetIPSalt(salt string) { s.ipSalt = salt }

func (s *Service) AppURL() string { return s.appURL }

// ----------------- URL Validation & Security -----------------

var blockedNets = func() []*net.IPNet {
	var out []*net.IPNet
	for _, cidr := range []string{
		"0.0.0.0/8", "10.0.0.0/8", "100.64.0.0/10", "127.0.0.0/8", "169.254.0.0/16",
		"172.16.0.0/12", "192.0.0.0/24", "192.168.0.0/16", "198.18.0.0/15", "224.0.0.0/4", "240.0.0.0/4",
		"::1/128", "::/128", "fc00::/7", "fe80::/10", "ff00::/8",
	} {
		_, n, _ := net.ParseCIDR(cidr)
		out = append(out, n)
	}
	return out
}()

// ValidateTargetURL rejects non-HTTP(S) schemes and any destination that
// resolves syntactically to loopback, private, link-local or reserved space.
func ValidateTargetURL(target string) error {
	trimmed := strings.TrimSpace(target)
	if trimmed == "" || len(trimmed) > 4096 {
		return ErrInvalidURL
	}

	u, err := url.Parse(trimmed)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil {
		return ErrInvalidURL
	}

	host := strings.TrimSuffix(strings.ToLower(u.Hostname()), ".")
	if host == "" {
		return ErrInvalidURL
	}
	if host == "localhost" || strings.HasSuffix(host, ".localhost") || strings.HasSuffix(host, ".local") ||
		strings.HasSuffix(host, ".internal") || strings.HasSuffix(host, ".home.arpa") {
		return ErrDisallowedTarget
	}

	ip := net.ParseIP(host)
	if ip == nil {
		// Reject integer / hex / octal IPv4 encodings such as 2130706433 or 0x7f.1
		if _, err := strconv.ParseUint(strings.ReplaceAll(host, ".", ""), 0, 64); err == nil {
			return ErrDisallowedTarget
		}
		if !strings.Contains(host, ".") {
			return ErrInvalidURL
		}
		return nil
	}
	for _, n := range blockedNets {
		if n.Contains(ip) {
			return ErrDisallowedTarget
		}
	}
	return nil
}

var slugPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{2,63}$`)

// reservedSlugs can't be claimed as custom slugs.
var reservedSlugs = map[string]bool{"api": true, "admin": true, "login": true, "signup": true, "static": true, "assets": true}

// GenerateRandomSlug produces a URL-safe collision resistant random string
func GenerateRandomSlug(length int) (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		b[i] = charset[num.Int64()]
	}
	return string(b), nil
}

func (s *Service) allocateSlug(custom string) (string, error) {
	slug := strings.TrimSpace(custom)
	if slug != "" {
		if !slugPattern.MatchString(slug) || reservedSlugs[strings.ToLower(slug)] {
			return "", ErrInvalidSlug
		}
		if s.repo.SlugExists(slug) {
			return "", ErrSlugTaken
		}
		return slug, nil
	}
	for length := 7; length <= 10; length++ {
		for i := 0; i < 3; i++ {
			gen, err := GenerateRandomSlug(length)
			if err != nil {
				return "", err
			}
			if !s.repo.SlugExists(gen) {
				return gen, nil
			}
		}
	}
	return "", errors.New("could not allocate a unique slug")
}

// ----------------- Links Management -----------------

type CreateLinkInput struct {
	TargetURL   string     `json:"target_url"`
	CustomSlug  string     `json:"custom_slug"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	ExpiresAt   *time.Time `json:"expires_at"`
}

func (s *Service) CreateShortLink(a Actor, input CreateLinkInput) (*models.Link, error) {
	if err := a.require(auth.PermEditAssets); err != nil {
		return nil, err
	}
	if err := ValidateTargetURL(input.TargetURL); err != nil {
		return nil, err
	}
	slug, err := s.allocateSlug(input.CustomSlug)
	if err != nil {
		return nil, err
	}

	title := strings.TrimSpace(input.Title)
	if title == "" {
		title = input.TargetURL
	}

	link := &models.Link{
		TeamID:      a.TeamID,
		UserID:      a.UserID,
		Slug:        slug,
		TargetURL:   strings.TrimSpace(input.TargetURL),
		Title:       title,
		Description: input.Description,
		IsActive:    true,
		ExpiresAt:   input.ExpiresAt,
	}

	if err := s.repo.CreateLink(link); err != nil {
		return nil, fmt.Errorf("failed to save link: %w", err)
	}
	s.audit(a, "link.create", "link", link.ID, map[string]any{"slug": link.Slug})
	return link, nil
}

// ResolveShortLink looks up an active, unexpired link for the public redirector.
func (s *Service) ResolveShortLink(slug string) (*models.Link, error) {
	link, err := s.repo.GetLinkBySlug(slug)
	if err != nil {
		return nil, ErrLinkNotFound
	}
	if !link.IsActive {
		return nil, ErrLinkNotFound
	}
	if link.ExpiresAt != nil && link.ExpiresAt.Before(time.Now().UTC()) {
		return nil, ErrLinkNotFound
	}
	_ = s.repo.IncrementLinkClicks(link.ID)
	return link, nil
}

type UpdateLinkInput struct {
	TargetURL   string
	Title       string
	Description string
	IsActive    bool
	ExpiresAt   *time.Time
}

func (s *Service) UpdateShortLink(a Actor, id string, in UpdateLinkInput) (*models.Link, error) {
	if err := a.require(auth.PermEditAssets); err != nil {
		return nil, err
	}
	link, err := s.repo.GetLink(id, a.TeamID)
	if err != nil {
		return nil, ErrLinkNotFound
	}

	if in.TargetURL != "" && in.TargetURL != link.TargetURL {
		if err := ValidateTargetURL(in.TargetURL); err != nil {
			return nil, err
		}
		link.TargetURL = strings.TrimSpace(in.TargetURL)
	}
	if in.Title != "" {
		link.Title = in.Title
	}
	link.Description = in.Description
	link.IsActive = in.IsActive
	link.ExpiresAt = in.ExpiresAt

	if err := s.repo.UpdateLink(link); err != nil {
		return nil, err
	}
	s.audit(a, "link.update", "link", link.ID, map[string]any{"slug": link.Slug})
	return link, nil
}

func clampLimit(limit int) int {
	if limit <= 0 || limit > 100 {
		return 50
	}
	return limit
}

func (s *Service) ListLinks(a Actor, limit, offset int) ([]models.Link, error) {
	if err := a.require(auth.PermViewAssets); err != nil {
		return nil, err
	}
	return s.repo.ListLinks(a.TeamID, clampLimit(limit), max(offset, 0))
}

func (s *Service) GetLink(a Actor, id string) (*models.Link, error) {
	if err := a.require(auth.PermViewAssets); err != nil {
		return nil, err
	}
	return s.repo.GetLink(id, a.TeamID)
}

func (s *Service) DeleteLink(a Actor, id string) error {
	if err := a.require(auth.PermEditAssets); err != nil {
		return err
	}
	if err := s.repo.DeleteLink(id, a.TeamID); err != nil {
		return err
	}
	s.audit(a, "link.delete", "link", id, nil)
	return nil
}

// ----------------- QR Code Management -----------------

type CreateQRInput struct {
	Type       models.QRType  `json:"type"`
	Title      string         `json:"title"`
	TargetURL  string         `json:"target_url"`
	RawData    string         `json:"raw_data"`
	Style      models.QRStyle `json:"style"`
	IsDynamic  bool           `json:"is_dynamic"` // Default true for website & docs so destination can be updated
	DocumentID string         `json:"document_id"`
}

func (s *Service) CreateQRCode(a Actor, input CreateQRInput) (*models.QRCode, error) {
	if err := a.require(auth.PermEditAssets); err != nil {
		return nil, err
	}
	if input.Type == "" {
		input.Type = models.QRTypeWebsite
	}
	var linkID, docID *string
	payload := ""

	if input.DocumentID != "" {
		if _, err := s.repo.GetDocument(input.DocumentID, a.TeamID); err != nil {
			return nil, ErrDocNotFound
		}
		docID = &input.DocumentID
	}

	// Dynamic routing: if website or document, we route through dynamic short link
	if (input.Type == models.QRTypeWebsite || input.Type == models.QRTypeDocument) && input.IsDynamic {
		link, err := s.CreateShortLink(a, CreateLinkInput{
			TargetURL:   input.TargetURL,
			Title:       input.Title,
			Description: fmt.Sprintf("Dynamic link for QR Code: %s", input.Title),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to generate dynamic link: %w", err)
		}
		linkID = &link.ID
		// Payload encoded into QR is the short redirect URL
		payload = fmt.Sprintf("%s/s/%s", s.appURL, link.Slug)
	} else {
		if input.Type == models.QRTypeWebsite {
			if err := ValidateTargetURL(input.TargetURL); err != nil {
				return nil, err
			}
		}
		builtPayload, err := qr.BuildPayload(input.Type, input.RawData, input.TargetURL)
		if err != nil {
			return nil, fmt.Errorf("failed to format QR payload: %w", err)
		}
		payload = builtPayload
	}

	if input.Style.FgColor == "" {
		input.Style.FgColor = "#000000"
	}
	if input.Style.BgColor == "" {
		input.Style.BgColor = "#FFFFFF"
	}

	qrCode := &models.QRCode{
		TeamID:     a.TeamID,
		UserID:     a.UserID,
		LinkID:     linkID,
		DocumentID: docID,
		Type:       input.Type,
		Title:      input.Title,
		Payload:    payload,
		RawData:    input.RawData,
		Style:      input.Style,
	}

	if err := s.repo.CreateQRCode(qrCode); err != nil {
		return nil, fmt.Errorf("failed to save QR code: %w", err)
	}
	s.audit(a, "qr.create", "qr_code", qrCode.ID, map[string]any{"type": qrCode.Type, "title": qrCode.Title})
	return qrCode, nil
}

func (s *Service) UpdateQRCode(a Actor, id string, title, targetURL, rawData string, style models.QRStyle) (*models.QRCode, error) {
	if err := a.require(auth.PermEditAssets); err != nil {
		return nil, err
	}
	existing, err := s.repo.GetQRCode(id, a.TeamID)
	if err != nil {
		return nil, ErrQRNotFound
	}

	if title != "" {
		existing.Title = title
	}
	if rawData != "" {
		existing.RawData = rawData
	}
	existing.Style = style

	// If this QR code is linked to a dynamic short link, update the short link destination!
	if existing.LinkID != nil && targetURL != "" {
		link, err := s.repo.GetLink(*existing.LinkID, a.TeamID)
		if err != nil {
			return nil, fmt.Errorf("failed updating target link: %w", err)
		}
		if _, err := s.UpdateShortLink(a, link.ID, UpdateLinkInput{
			TargetURL: targetURL, Title: existing.Title, Description: link.Description,
			IsActive: link.IsActive, ExpiresAt: link.ExpiresAt,
		}); err != nil {
			return nil, fmt.Errorf("failed updating target link: %w", err)
		}
	} else if targetURL != "" || rawData != "" {
		if existing.Type == models.QRTypeWebsite && targetURL != "" {
			if err := ValidateTargetURL(targetURL); err != nil {
				return nil, err
			}
		}
		newPayload, err := qr.BuildPayload(existing.Type, existing.RawData, targetURL)
		if err != nil {
			return nil, err
		}
		existing.Payload = newPayload
	}

	if err := s.repo.UpdateQRCode(existing); err != nil {
		return nil, err
	}
	s.audit(a, "qr.update", "qr_code", existing.ID, nil)
	return existing, nil
}

// ExportQRCode renders a team's QR code in the requested format.
func (s *Service) ExportQRCode(a Actor, id string, format string, size int) ([]byte, string, error) {
	if err := a.require(auth.PermExportAssets); err != nil {
		return nil, "", err
	}
	qrCode, err := s.repo.GetQRCode(id, a.TeamID)
	if err != nil {
		return nil, "", ErrQRNotFound
	}
	return RenderQR(qrCode, format, size)
}

// RenderQR encodes a QR code's payload in one of the supported export formats.
func RenderQR(qrCode *models.QRCode, format string, size int) ([]byte, string, error) {
	payload := strings.TrimSpace(qrCode.Payload)
	if payload == "" {
		payload = qrCode.RawData
	}
	if size > 4096 {
		size = 4096
	}

	switch strings.ToLower(strings.TrimSpace(format)) {
	case "jpeg", "jpg":
		data, err := qr.RenderJPEG(payload, size, qrCode.Style)
		return data, "image/jpeg", err
	case "svg":
		data, err := qr.RenderSVG(payload, size, qrCode.Style)
		return data, "image/svg+xml", err
	case "pdf":
		data, err := qr.RenderPDF(payload, qrCode.Title, qrCode.Style)
		return data, "application/pdf", err
	case "png", "":
		data, err := qr.RenderPNG(payload, size, qrCode.Style)
		return data, "image/png", err
	default:
		return nil, "", fmt.Errorf("unsupported export format: %s", format)
	}
}

func (s *Service) ListQRCodes(a Actor, limit, offset int) ([]models.QRCode, error) {
	if err := a.require(auth.PermViewAssets); err != nil {
		return nil, err
	}
	return s.repo.ListQRCodes(a.TeamID, clampLimit(limit), max(offset, 0))
}

func (s *Service) GetQRCode(a Actor, id string) (*models.QRCode, error) {
	if err := a.require(auth.PermViewAssets); err != nil {
		return nil, err
	}
	return s.repo.GetQRCode(id, a.TeamID)
}

// GetPublicQRCode is used by the scan redirector, which is unauthenticated by design.
func (s *Service) GetPublicQRCode(id string) (*models.QRCode, error) {
	return s.repo.GetQRCodeByID(id)
}

func (s *Service) DeleteQRCode(a Actor, id string) error {
	if err := a.require(auth.PermEditAssets); err != nil {
		return err
	}
	if err := s.repo.DeleteQRCode(id, a.TeamID); err != nil {
		return err
	}
	s.audit(a, "qr.delete", "qr_code", id, nil)
	return nil
}

func (s *Service) RecordQRScan(id string) {
	_ = s.repo.IncrementQRScans(id)
}

// ----------------- Document Management & Sharing -----------------

func (s *Service) UploadDocument(a Actor, filename string, r io.Reader) (*models.Document, string, error) {
	if err := a.require(auth.PermEditAssets); err != nil {
		return nil, "", err
	}
	storedPath, size, mimeType, err := s.storage.Save(filename, r)
	if err != nil {
		return nil, "", fmt.Errorf("storage save failed: %w", err)
	}

	doc := &models.Document{
		TeamID:     a.TeamID,
		UserID:     a.UserID,
		Filename:   filename,
		StoredPath: storedPath,
		MimeType:   mimeType,
		FileSize:   size,
		Version:    1,
	}

	if err := s.repo.CreateDocument(doc); err != nil {
		_ = s.storage.Delete(storedPath)
		return nil, "", fmt.Errorf("failed saving document metadata: %w", err)
	}
	s.audit(a, "document.upload", "document", doc.ID, map[string]any{"filename": filename})
	return doc, s.storage.GetURL(storedPath), nil
}

func (s *Service) UpdateDocumentFile(a Actor, docID string, filename string, r io.Reader) (*models.Document, string, error) {
	if err := a.require(auth.PermEditAssets); err != nil {
		return nil, "", err
	}
	existing, err := s.repo.GetDocument(docID, a.TeamID)
	if err != nil {
		return nil, "", ErrDocNotFound
	}

	storedPath, size, mimeType, err := s.storage.Save(filename, r)
	if err != nil {
		return nil, "", fmt.Errorf("storage save failed: %w", err)
	}
	oldPath := existing.StoredPath

	existing.Filename = filename
	existing.StoredPath = storedPath
	existing.FileSize = size
	existing.MimeType = mimeType
	existing.Version++

	if err := s.repo.UpdateDocument(existing); err != nil {
		_ = s.storage.Delete(storedPath)
		return nil, "", err
	}
	_ = s.storage.Delete(oldPath)
	s.audit(a, "document.replace", "document", existing.ID, map[string]any{"filename": filename, "version": existing.Version})
	return existing, s.storage.GetURL(storedPath), nil
}

// OpenPublicDocument streams a document by its unguessable storage key.
func (s *Service) OpenPublicDocument(storedPath string) (*models.Document, io.ReadCloser, error) {
	doc, err := s.repo.GetDocumentByStoredPath(storedPath)
	if err != nil {
		return nil, nil, ErrDocNotFound
	}
	rc, err := s.storage.Open(doc.StoredPath)
	if err != nil {
		return nil, nil, ErrDocNotFound
	}
	return doc, rc, nil
}

func (s *Service) ListDocuments(a Actor) ([]models.Document, error) {
	if err := a.require(auth.PermViewAssets); err != nil {
		return nil, err
	}
	return s.repo.ListDocuments(a.TeamID)
}

func (s *Service) DocumentURL(doc *models.Document) string {
	return s.storage.GetURL(doc.StoredPath)
}

func (s *Service) DeleteDocument(a Actor, docID string, deleteAssociatedQR bool) (int, error) {
	if err := a.require(auth.PermEditAssets); err != nil {
		return 0, err
	}
	doc, err := s.repo.GetDocument(docID, a.TeamID)
	if err != nil {
		return 0, ErrDocNotFound
	}

	deletedQRCount := 0
	if deleteAssociatedQR {
		qrs, err := s.repo.ListQRCodesByDocument(docID, a.TeamID)
		if err == nil {
			for _, q := range qrs {
				if q.LinkID != nil {
					_ = s.repo.DeleteLink(*q.LinkID, a.TeamID)
				}
				if s.repo.DeleteQRCode(q.ID, a.TeamID) == nil {
					deletedQRCount++
				}
			}
		}
	}

	if err := s.repo.DeleteDocument(docID, a.TeamID); err != nil {
		return deletedQRCount, err
	}
	_ = s.storage.Delete(doc.StoredPath)
	s.audit(a, "document.delete", "document", docID, map[string]any{"filename": doc.Filename, "deleted_qr": deletedQRCount})
	return deletedQRCount, nil
}

func (s *Service) BulkDeleteDocuments(a Actor, docIDs []string, deleteAssociatedQR bool) (int, int, error) {
	if err := a.require(auth.PermEditAssets); err != nil {
		return 0, 0, err
	}
	deletedDocs, totalQRsDeleted := 0, 0
	for _, id := range docIDs {
		qrsDeleted, err := s.DeleteDocument(a, id, deleteAssociatedQR)
		totalQRsDeleted += qrsDeleted
		if err == nil {
			deletedDocs++
		}
	}
	return deletedDocs, totalQRsDeleted, nil
}

// ----------------- Analytics Ingestion & Reporting -----------------

func (s *Service) hashIP(ip string) string {
	if ip == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(s.ipSalt + "|" + ip))
	return hex.EncodeToString(sum[:16])
}

func (s *Service) TrackEvent(e *models.AnalyticsEvent) {
	// Parse basic user agent info
	ua := strings.ToLower(e.UserAgent)
	switch {
	case strings.Contains(ua, "ipad") || strings.Contains(ua, "tablet"):
		e.Device = "Tablet"
	case strings.Contains(ua, "mobile") || strings.Contains(ua, "android") || strings.Contains(ua, "iphone"):
		e.Device = "Mobile"
	default:
		e.Device = "Desktop"
	}

	switch {
	case strings.Contains(ua, "edg"):
		e.Browser = "Edge"
	case strings.Contains(ua, "chrome") || strings.Contains(ua, "crios"):
		e.Browser = "Chrome"
	case strings.Contains(ua, "firefox") || strings.Contains(ua, "fxios"):
		e.Browser = "Firefox"
	case strings.Contains(ua, "safari"):
		e.Browser = "Safari"
	default:
		e.Browser = "Other"
	}

	switch {
	case strings.Contains(ua, "windows"):
		e.OS = "Windows"
	case strings.Contains(ua, "iphone") || strings.Contains(ua, "ipad"):
		e.OS = "iOS"
	case strings.Contains(ua, "macintosh") || strings.Contains(ua, "mac os"):
		e.OS = "macOS"
	case strings.Contains(ua, "android"):
		e.OS = "Android"
	case strings.Contains(ua, "linux"):
		e.OS = "Linux"
	default:
		e.OS = "Other"
	}

	e.IPHash = s.hashIP(e.IP)
	_ = s.repo.RecordEvent(e)
}

type AnalyticsQuery struct {
	LinkID string
	QRID   string
	From   *time.Time
	To     *time.Time
}

func (s *Service) GetAnalyticsSummary(a Actor, q AnalyticsQuery) (*models.AnalyticsSummary, error) {
	if err := a.require(auth.PermViewAnalytics); err != nil {
		return nil, err
	}
	return s.repo.GetAnalyticsSummary(repository.AnalyticsFilter{
		TeamID: a.TeamID, LinkID: q.LinkID, QRID: q.QRID, From: q.From, To: q.To,
	})
}

// ----------------- API Keys -----------------

func (s *Service) CreateAPIKey(a Actor, name string, expiresInDays int) (*models.APIKey, error) {
	if err := a.require(auth.PermViewAssets); err != nil {
		return nil, err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		name = "Default API Key"
	}

	rawBytes := make([]byte, 24)
	if _, err := rand.Read(rawBytes); err != nil {
		return nil, fmt.Errorf("failed to generate random key: %w", err)
	}
	fullKey := auth.APIKeyPrefix + hex.EncodeToString(rawBytes)
	keyHash := auth.HashToken(fullKey)

	var expiresAt *time.Time
	if expiresInDays > 0 {
		exp := time.Now().UTC().Add(time.Duration(expiresInDays) * 24 * time.Hour)
		expiresAt = &exp
	}

	apiKey := &models.APIKey{
		ID:        "key_" + uuid.NewString(),
		UserID:    a.UserID,
		TeamID:    a.TeamID,
		Name:      name,
		KeyPrefix: fullKey[:12] + "...",
		Key:       fullKey,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now().UTC(),
	}

	if err := s.repo.CreateAPIKey(apiKey, keyHash); err != nil {
		return nil, fmt.Errorf("failed to store API key: %w", err)
	}
	s.audit(a, "api_key.create", "api_key", apiKey.ID, map[string]any{"name": name})
	return apiKey, nil
}

func (s *Service) ListAPIKeys(a Actor) ([]models.APIKey, error) {
	return s.repo.ListAPIKeys(a.UserID, a.TeamID)
}

func (s *Service) DeleteAPIKey(a Actor, id string) error {
	if err := s.repo.DeleteAPIKey(id, a.UserID); err != nil {
		return err
	}
	s.audit(a, "api_key.revoke", "api_key", id, nil)
	return nil
}

// ValidateAPIKey returns the key record for a raw lk_live_ token.
func (s *Service) ValidateAPIKey(rawKey string) (*models.APIKey, error) {
	if !strings.HasPrefix(rawKey, auth.APIKeyPrefix) {
		return nil, errors.New("invalid API key format")
	}
	key, err := s.repo.GetAPIKeyByHash(auth.HashToken(rawKey))
	if err != nil {
		return nil, errors.New("API key not found")
	}
	if key.ExpiresAt != nil && key.ExpiresAt.Before(time.Now().UTC()) {
		return nil, errors.New("API key has expired")
	}
	go func(id string) {
		_ = s.repo.UpdateAPIKeyLastUsed(id)
	}(key.ID)
	return key, nil
}

// ----------------- User Profile -----------------

func (s *Service) GetUserProfile(userID string) (*models.User, error) {
	return s.repo.GetUserByID(userID)
}

func (s *Service) UpdateUserProfile(user *models.User) error {
	return s.repo.UpsertUser(user)
}
