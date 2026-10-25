package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"

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
)

type Service struct {
	repo    *repository.Repository
	storage storage.StorageProvider
	appURL  string
}

func New(repo *repository.Repository, storage storage.StorageProvider, appURL string) *Service {
	return &Service{
		repo:    repo,
		storage: storage,
		appURL:  strings.TrimRight(appURL, "/"),
	}
}

// ----------------- URL Validation & Security -----------------

func ValidateTargetURL(target string) error {
	trimmed := strings.TrimSpace(target)
	if trimmed == "" {
		return ErrInvalidURL
	}

	u, err := url.Parse(trimmed)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return ErrInvalidURL
	}

	host := strings.ToLower(u.Hostname())
	if host == "localhost" || host == "127.0.0.1" || host == "::1" || strings.HasPrefix(host, "192.168.") || strings.HasPrefix(host, "10.") {
		return ErrDisallowedTarget
	}

	return nil
}

// GenerateRandomSlug produces a URL-safe collision resistant 7-character string
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

// ----------------- Links Management -----------------

type CreateLinkInput struct {
	TeamID      string     `json:"team_id"`
	UserID      string     `json:"user_id"`
	TargetURL   string     `json:"target_url"`
	CustomSlug  string     `json:"custom_slug"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	ExpiresAt   *time.Time `json:"expires_at"`
}

func (s *Service) CreateShortLink(input CreateLinkInput) (*models.Link, error) {
	if err := ValidateTargetURL(input.TargetURL); err != nil {
		return nil, err
	}

	slug := strings.TrimSpace(input.CustomSlug)
	if slug == "" {
		for i := 0; i < 5; i++ {
			genSlug, err := GenerateRandomSlug(7)
			if err != nil {
				return nil, err
			}
			// check if exists
			if _, err := s.repo.GetLinkBySlug(genSlug); err != nil {
				slug = genSlug
				break
			}
		}
		if slug == "" {
			slug = uuid.New().String()[:8]
		}
	} else {
		// check custom slug availability
		if existing, _ := s.repo.GetLinkBySlug(slug); existing != nil {
			return nil, fmt.Errorf("custom slug '%s' is already in use", slug)
		}
	}

	title := input.Title
	if title == "" {
		title = input.TargetURL
	}

	link := &models.Link{
		TeamID:      input.TeamID,
		UserID:      input.UserID,
		Slug:        slug,
		TargetURL:   input.TargetURL,
		Title:       title,
		Description: input.Description,
		IsActive:    true,
		ExpiresAt:   input.ExpiresAt,
	}

	if err := s.repo.CreateLink(link); err != nil {
		return nil, fmt.Errorf("failed to save link: %w", err)
	}

	return link, nil
}

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

func (s *Service) UpdateShortLink(id string, targetURL, title, description string, isActive bool, expiresAt *time.Time, teamID ...string) (*models.Link, error) {
	var link *models.Link
	var err error
	if len(teamID) > 0 && teamID[0] != "" {
		link, err = s.repo.GetLinkByIDAndTeam(id, teamID[0])
	} else {
		link, err = s.repo.GetLinkByID(id)
	}
	if err != nil {
		return nil, ErrLinkNotFound
	}

	if targetURL != "" && targetURL != link.TargetURL {
		if err := ValidateTargetURL(targetURL); err != nil {
			return nil, err
		}
		link.TargetURL = targetURL
	}

	if title != "" {
		link.Title = title
	}
	link.Description = description
	link.IsActive = isActive
	link.ExpiresAt = expiresAt

	if err := s.repo.UpdateLink(link); err != nil {
		return nil, err
	}

	return link, nil
}

func (s *Service) ListLinks(teamID string, limit, offset int) ([]models.Link, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	return s.repo.ListLinks(teamID, limit, offset)
}

func (s *Service) GetLink(id string, teamID ...string) (*models.Link, error) {
	if len(teamID) > 0 && teamID[0] != "" {
		return s.repo.GetLinkByIDAndTeam(id, teamID[0])
	}
	return s.repo.GetLinkByID(id)
}

func (s *Service) DeleteLink(id string, teamID ...string) error {
	if len(teamID) > 0 && teamID[0] != "" {
		return s.repo.DeleteLinkWithTeam(id, teamID[0])
	}
	return s.repo.DeleteLink(id)
}

// ----------------- QR Code Management -----------------

type CreateQRInput struct {
	TeamID    string         `json:"team_id"`
	UserID    string         `json:"user_id"`
	Type      models.QRType  `json:"type"`
	Title     string         `json:"title"`
	TargetURL string         `json:"target_url"`
	RawData   string         `json:"raw_data"`
	Style     models.QRStyle `json:"style"`
	IsDynamic bool           `json:"is_dynamic"` // Default true for website & docs so destination can be updated
}

func (s *Service) CreateQRCode(input CreateQRInput) (*models.QRCode, error) {
	var linkID *string
	payload := ""

	// Dynamic routing: if website or document, we route through dynamic short link
	if (input.Type == models.QRTypeWebsite || input.Type == models.QRTypeDocument) && input.IsDynamic {
		linkInput := CreateLinkInput{
			TeamID:      input.TeamID,
			UserID:      input.UserID,
			TargetURL:   input.TargetURL,
			Title:       input.Title,
			Description: fmt.Sprintf("Dynamic link for QR Code: %s", input.Title),
		}
		link, err := s.CreateShortLink(linkInput)
		if err != nil {
			return nil, fmt.Errorf("failed to generate dynamic link: %w", err)
		}
		linkID = &link.ID
		// Payload encoded into QR is the short redirect URL
		payload = fmt.Sprintf("%s/s/%s", s.appURL, link.Slug)
	} else {
		// Static or protocol-based payload (WiFi, vCard, Email, SMS, etc.)
		builtPayload, err := qr.BuildPayload(input.Type, input.RawData, input.TargetURL)
		if err != nil {
			return nil, fmt.Errorf("failed to format QR payload: %w", err)
		}
		payload = builtPayload
	}

	// Apply default style attributes
	if input.Style.FgColor == "" {
		input.Style.FgColor = "#000000"
	}
	if input.Style.BgColor == "" {
		input.Style.BgColor = "#FFFFFF"
	}

	qrCode := &models.QRCode{
		TeamID:  input.TeamID,
		UserID:  input.UserID,
		LinkID:  linkID,
		Type:    input.Type,
		Title:   input.Title,
		Payload: payload,
		RawData: input.RawData,
		Style:   input.Style,
	}

	if err := s.repo.CreateQRCode(qrCode); err != nil {
		return nil, fmt.Errorf("failed to save QR code: %w", err)
	}

	return qrCode, nil
}

func (s *Service) UpdateQRCode(id string, title, targetURL, rawData string, style models.QRStyle, teamID ...string) (*models.QRCode, error) {
	var existing *models.QRCode
	var err error
	if len(teamID) > 0 && teamID[0] != "" {
		existing, err = s.repo.GetQRCodeByIDAndTeam(id, teamID[0])
	} else {
		existing, err = s.repo.GetQRCodeByID(id)
	}
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
		_, err := s.UpdateShortLink(*existing.LinkID, targetURL, existing.Title, "", true, nil)
		if err != nil {
			return nil, fmt.Errorf("failed updating target link: %w", err)
		}
	} else if targetURL != "" || rawData != "" {
		// Static payload update
		newPayload, err := qr.BuildPayload(existing.Type, existing.RawData, targetURL)
		if err == nil {
			existing.Payload = newPayload
		}
	}

	if err := s.repo.UpdateQRCode(existing); err != nil {
		return nil, err
	}

	return existing, nil
}

func (s *Service) ExportQRCode(id string, format string, size int) ([]byte, string, error) {
	qrCode, err := s.repo.GetQRCodeByID(id)
	if err != nil {
		return nil, "", ErrQRNotFound
	}

	payload := strings.TrimSpace(qrCode.Payload)
	if payload == "" {
		if qrCode.RawData != "" {
			payload = qrCode.RawData
		} else {
			payload = fmt.Sprintf("%s/qr/r/%s", s.appURL, qrCode.ID)
		}
	}

	format = strings.ToLower(strings.TrimSpace(format))
	switch format {
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

func (s *Service) ListQRCodes(teamID string, limit, offset int) ([]models.QRCode, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	return s.repo.ListQRCodes(teamID, limit, offset)
}

func (s *Service) GetQRCode(id string) (*models.QRCode, error) {
	return s.repo.GetQRCodeByID(id)
}

func (s *Service) DeleteQRCode(id string, teamID ...string) error {
	if len(teamID) > 0 && teamID[0] != "" {
		return s.repo.DeleteQRCodeWithTeam(id, teamID[0])
	}
	return s.repo.DeleteQRCode(id)
}

func (s *Service) RecordQRScan(id string) {
	_ = s.repo.IncrementQRScans(id)
}

// ----------------- Document Management & Sharing -----------------

func (s *Service) UploadDocument(teamID, userID, filename string, r io.Reader) (*models.Document, string, error) {
	storedPath, size, mimeType, err := s.storage.Save(filename, r)
	if err != nil {
		return nil, "", fmt.Errorf("storage save failed: %w", err)
	}

	doc := &models.Document{
		TeamID:     teamID,
		UserID:     userID,
		Filename:   filename,
		StoredPath: storedPath,
		MimeType:   mimeType,
		FileSize:   size,
		Version:    1,
	}

	if err := s.repo.CreateDocument(doc); err != nil {
		return nil, "", fmt.Errorf("failed saving document metadata: %w", err)
	}

	downloadURL := s.storage.GetURL(storedPath)
	return doc, downloadURL, nil
}

func (s *Service) UpdateDocumentFile(docID string, filename string, r io.Reader) (*models.Document, string, error) {
	existing, err := s.repo.GetDocumentByID(docID)
	if err != nil {
		return nil, "", ErrDocNotFound
	}

	// Delete old file
	_ = s.storage.Delete(existing.StoredPath)

	storedPath, size, mimeType, err := s.storage.Save(filename, r)
	if err != nil {
		return nil, "", fmt.Errorf("storage save failed: %w", err)
	}

	existing.Filename = filename
	existing.StoredPath = storedPath
	existing.FileSize = size
	existing.MimeType = mimeType

	if err := s.repo.UpdateDocument(existing); err != nil {
		return nil, "", err
	}

	downloadURL := s.storage.GetURL(storedPath)
	return existing, downloadURL, nil
}

func (s *Service) OpenDocumentFile(storedPath string) (io.ReadCloser, error) {
	return s.storage.Open(storedPath)
}

func (s *Service) ListDocuments(teamID string) ([]models.Document, error) {
	return s.repo.ListDocuments(teamID)
}

// ----------------- Analytics Ingestion & Reporting -----------------

func (s *Service) TrackEvent(e *models.AnalyticsEvent) {
	// Parse basic user agent info
	ua := strings.ToLower(e.UserAgent)
	if strings.Contains(ua, "mobile") || strings.Contains(ua, "android") || strings.Contains(ua, "iphone") {
		e.Device = "Mobile"
	} else if strings.Contains(ua, "tablet") || strings.Contains(ua, "ipad") {
		e.Device = "Tablet"
	} else {
		e.Device = "Desktop"
	}

	if strings.Contains(ua, "edg") {
		e.Browser = "Edge"
	} else if strings.Contains(ua, "chrome") {
		e.Browser = "Chrome"
	} else if strings.Contains(ua, "safari") {
		e.Browser = "Safari"
	} else if strings.Contains(ua, "firefox") {
		e.Browser = "Firefox"
	} else {
		e.Browser = "Other"
	}

	if strings.Contains(ua, "windows") {
		e.OS = "Windows"
	} else if strings.Contains(ua, "macintosh") || strings.Contains(ua, "mac os") {
		e.OS = "macOS"
	} else if strings.Contains(ua, "android") {
		e.OS = "Android"
	} else if strings.Contains(ua, "iphone") || strings.Contains(ua, "ipad") {
		e.OS = "iOS"
	} else if strings.Contains(ua, "linux") {
		e.OS = "Linux"
	} else {
		e.OS = "Other"
	}

	if e.Country == "" {
		e.Country = "US"
	}

	_ = s.repo.RecordEvent(e)
}

func (s *Service) GetAnalyticsSummary(linkID, qrID *string) (*models.AnalyticsSummary, error) {
	return s.repo.GetAnalyticsSummary(linkID, qrID)
}

// ----------------- API Keys -----------------

func (s *Service) CreateAPIKey(userID, teamID, name string, expiresInDays int) (*models.APIKey, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "Default API Key"
	}

	rawBytes := make([]byte, 24)
	if _, err := rand.Read(rawBytes); err != nil {
		return nil, fmt.Errorf("failed to generate random key: %w", err)
	}
	rawHex := hex.EncodeToString(rawBytes)
	fullKey := "lk_live_" + rawHex

	hasher := sha256.New()
	hasher.Write([]byte(fullKey))
	keyHash := hex.EncodeToString(hasher.Sum(nil))

	keyPrefix := fullKey[:12] + "..."

	var expiresAt *time.Time
	if expiresInDays > 0 {
		exp := time.Now().UTC().Add(time.Duration(expiresInDays) * 24 * time.Hour)
		expiresAt = &exp
	}

	apiKey := &models.APIKey{
		ID:        "key_" + uuid.New().String(),
		UserID:    userID,
		TeamID:    teamID,
		Name:      name,
		KeyHash:   keyHash,
		KeyPrefix: keyPrefix,
		Key:       fullKey,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now().UTC(),
	}

	if err := s.repo.CreateAPIKey(apiKey, keyHash); err != nil {
		return nil, fmt.Errorf("failed to store API key: %w", err)
	}

	return apiKey, nil
}

func (s *Service) ListAPIKeys(userID string) ([]models.APIKey, error) {
	return s.repo.ListAPIKeys(userID)
}

func (s *Service) DeleteAPIKey(id, userID string) error {
	return s.repo.DeleteAPIKey(id, userID)
}

func (s *Service) ValidateAPIKey(rawKey string) (*models.APIKey, error) {
	if !strings.HasPrefix(rawKey, "lk_live_") {
		return nil, errors.New("invalid API key format")
	}

	hasher := sha256.New()
	hasher.Write([]byte(rawKey))
	hash := hex.EncodeToString(hasher.Sum(nil))

	key, err := s.repo.GetAPIKeyByHash(hash)
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
	return s.repo.CreateOrUpdateUser(user)
}


func (s *Service) DeleteDocument(docID, teamID string, deleteAssociatedQR bool) (int, error) {
	doc, err := s.repo.GetDocumentByIDAndTeam(docID, teamID)
	if err != nil {
		return 0, ErrDocNotFound
	}

	// 1. Delete physical file from storage
	_ = s.storage.Delete(doc.StoredPath)

	deletedQRCount := 0
	if deleteAssociatedQR {
		// Find associated QR codes by stored_path
		qrs, err := s.repo.FindQRCodesByTargetURLSubstring(doc.StoredPath, teamID)
		if err == nil {
			for _, qr := range qrs {
				_ = s.repo.DeleteQRCodeWithTeam(qr.ID, teamID)
				deletedQRCount++
			}
		}
	}

	// 2. Delete document from database
	if err := s.repo.DeleteDocument(docID, teamID); err != nil {
		return deletedQRCount, err
	}

	// 3. Confirm database entry is gone
	if check, _ := s.repo.GetDocumentByIDAndTeam(docID, teamID); check != nil {
		return deletedQRCount, fmt.Errorf("failed to remove document from database")
	}

	return deletedQRCount, nil
}

func (s *Service) BulkDeleteDocuments(docIDs []string, teamID string, deleteAssociatedQR bool) (int, int, error) {
	deletedDocs := 0
	totalQRsDeleted := 0
	for _, id := range docIDs {
		qrsDeleted, err := s.DeleteDocument(id, teamID, deleteAssociatedQR)
		if err == nil {
			deletedDocs++
		totalQRsDeleted += qrsDeleted
		}
	}
	return deletedDocs, totalQRsDeleted, nil
}

// ----------------- Team & User Management -----------------

func (s *Service) GetUserTeamAndRole(userID string) (string, models.UserRole, error) {
	tm, err := s.repo.GetUserTeam(userID)
	if err == nil && tm != nil && tm.TeamID != "" {
		return tm.TeamID, tm.Role, nil
	}
	// Default team association
	defaultTeamID := "team_default"
	_ = s.repo.EnsureDefaultTeam(defaultTeamID, "Lynqr Workspace", userID)
	_ = s.repo.AddTeamMember(defaultTeamID, userID, models.RoleOwner)
	return defaultTeamID, models.RoleOwner, nil
}

func (s *Service) ListTeamMembers(teamID string) ([]models.TeamMemberDetail, error) {
	return s.repo.ListTeamMembers(teamID)
}

func (s *Service) UpdateTeamMemberRole(teamID, userID string, role models.UserRole) error {
	return s.repo.UpdateTeamMemberRole(teamID, userID, role)
}

func (s *Service) RemoveTeamMember(teamID, userID string) error {
	return s.repo.RemoveTeamMember(teamID, userID)
}

func (s *Service) CreateInvitation(inv *models.Invitation) error {
	return s.repo.CreateInvitation(inv)
}

func (s *Service) ListInvitations(teamID string) ([]models.Invitation, error) {
	return s.repo.ListInvitations(teamID)
}

func (s *Service) RevokeInvitation(id, teamID string) error {
	return s.repo.RevokeInvitation(id, teamID)
}
