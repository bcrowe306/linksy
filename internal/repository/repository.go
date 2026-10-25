package repository

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/bcrowe306/linksy/internal/db"
	"github.com/bcrowe306/linksy/internal/models"
)

type Repository struct {
	db *db.DB
}

func New(database *db.DB) *Repository {
	return &Repository{db: database}
}

// ----------------- Links -----------------

func (r *Repository) CreateLink(l *models.Link) error {
	if l.ID == "" {
		l.ID = uuid.New().String()
	}
	now := time.Now().UTC()
	l.CreatedAt = now
	l.UpdatedAt = now

	query := `INSERT INTO links (id, team_id, user_id, slug, target_url, title, description, is_active, expires_at, password, click_count, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`
	if r.db.DriverName == "sqlite" {
		query = `INSERT INTO links (id, team_id, user_id, slug, target_url, title, description, is_active, expires_at, password, click_count, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	}

	_, err := r.db.Exec(query, l.ID, l.TeamID, l.UserID, l.Slug, l.TargetURL, l.Title, l.Description, l.IsActive, l.ExpiresAt, l.Password, l.ClickCount, l.CreatedAt, l.UpdatedAt)
	return err
}

func (r *Repository) GetLinkBySlug(slug string) (*models.Link, error) {
	query := `SELECT id, team_id, user_id, slug, target_url, title, description, is_active, expires_at, password, click_count, created_at, updated_at
		FROM links WHERE slug = $1`
	if r.db.DriverName == "sqlite" {
		query = `SELECT id, team_id, user_id, slug, target_url, title, description, is_active, expires_at, password, click_count, created_at, updated_at
			FROM links WHERE slug = ?`
	}

	var l models.Link
	err := r.db.QueryRow(query, slug).Scan(
		&l.ID, &l.TeamID, &l.UserID, &l.Slug, &l.TargetURL, &l.Title, &l.Description,
		&l.IsActive, &l.ExpiresAt, &l.Password, &l.ClickCount, &l.CreatedAt, &l.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &l, nil
}

func (r *Repository) GetLinkByID(id string) (*models.Link, error) {
	query := `SELECT id, team_id, user_id, slug, target_url, title, description, is_active, expires_at, password, click_count, created_at, updated_at
		FROM links WHERE id = $1`
	if r.db.DriverName == "sqlite" {
		query = `SELECT id, team_id, user_id, slug, target_url, title, description, is_active, expires_at, password, click_count, created_at, updated_at
			FROM links WHERE id = ?`
	}

	var l models.Link
	err := r.db.QueryRow(query, id).Scan(
		&l.ID, &l.TeamID, &l.UserID, &l.Slug, &l.TargetURL, &l.Title, &l.Description,
		&l.IsActive, &l.ExpiresAt, &l.Password, &l.ClickCount, &l.CreatedAt, &l.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &l, nil
}

func (r *Repository) ListLinks(teamID string, limit, offset int) ([]models.Link, error) {
	query := `SELECT id, team_id, user_id, slug, target_url, title, description, is_active, expires_at, password, click_count, created_at, updated_at
		FROM links WHERE team_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`
	if r.db.DriverName == "sqlite" {
		query = `SELECT id, team_id, user_id, slug, target_url, title, description, is_active, expires_at, password, click_count, created_at, updated_at
			FROM links WHERE team_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?`
	}

	rows, err := r.db.Query(query, teamID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []models.Link
	for rows.Next() {
		var l models.Link
		if err := rows.Scan(
			&l.ID, &l.TeamID, &l.UserID, &l.Slug, &l.TargetURL, &l.Title, &l.Description,
			&l.IsActive, &l.ExpiresAt, &l.Password, &l.ClickCount, &l.CreatedAt, &l.UpdatedAt,
		); err != nil {
			return nil, err
		}
		links = append(links, l)
	}
	return links, nil
}

func (r *Repository) UpdateLink(l *models.Link) error {
	l.UpdatedAt = time.Now().UTC()
	query := `UPDATE links SET target_url = $1, title = $2, description = $3, is_active = $4, expires_at = $5, updated_at = $6
		WHERE id = $7`
	if r.db.DriverName == "sqlite" {
		query = `UPDATE links SET target_url = ?, title = ?, description = ?, is_active = ?, expires_at = ?, updated_at = ?
			WHERE id = ?`
	}

	_, err := r.db.Exec(query, l.TargetURL, l.Title, l.Description, l.IsActive, l.ExpiresAt, l.UpdatedAt, l.ID)
	return err
}

func (r *Repository) IncrementLinkClicks(id string) error {
	query := `UPDATE links SET click_count = click_count + 1 WHERE id = $1`
	if r.db.DriverName == "sqlite" {
		query = `UPDATE links SET click_count = click_count + 1 WHERE id = ?`
	}
	_, err := r.db.Exec(query, id)
	return err
}

func (r *Repository) DeleteLink(id string) error {
	query := `DELETE FROM links WHERE id = $1`
	if r.db.DriverName == "sqlite" {
		query = `DELETE FROM links WHERE id = ?`
	}
	_, err := r.db.Exec(query, id)
	return err
}

// ----------------- QR Codes -----------------

func (r *Repository) CreateQRCode(q *models.QRCode) error {
	if q.ID == "" {
		q.ID = uuid.New().String()
	}
	now := time.Now().UTC()
	q.CreatedAt = now
	q.UpdatedAt = now

	styleJSON, err := json.Marshal(q.Style)
	if err != nil {
		return fmt.Errorf("failed to marshal style: %w", err)
	}

	query := `INSERT INTO qr_codes (id, team_id, user_id, link_id, type, title, payload, raw_data, style, scan_count, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`
	if r.db.DriverName == "sqlite" {
		query = `INSERT INTO qr_codes (id, team_id, user_id, link_id, type, title, payload, raw_data, style, scan_count, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	}

	_, err = r.db.Exec(query, q.ID, q.TeamID, q.UserID, q.LinkID, q.Type, q.Title, q.Payload, q.RawData, string(styleJSON), q.ScanCount, q.CreatedAt, q.UpdatedAt)
	return err
}

func (r *Repository) GetQRCodeByID(id string) (*models.QRCode, error) {
	query := `SELECT id, team_id, user_id, link_id, type, title, payload, raw_data, style, scan_count, created_at, updated_at
		FROM qr_codes WHERE id = $1`
	if r.db.DriverName == "sqlite" {
		query = `SELECT id, team_id, user_id, link_id, type, title, payload, raw_data, style, scan_count, created_at, updated_at
			FROM qr_codes WHERE id = ?`
	}

	var q models.QRCode
	var styleJSON string
	err := r.db.QueryRow(query, id).Scan(
		&q.ID, &q.TeamID, &q.UserID, &q.LinkID, &q.Type, &q.Title, &q.Payload, &q.RawData,
		&styleJSON, &q.ScanCount, &q.CreatedAt, &q.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal([]byte(styleJSON), &q.Style)
	return &q, nil
}

func (r *Repository) ListQRCodes(teamID string, limit, offset int) ([]models.QRCode, error) {
	query := `SELECT id, team_id, user_id, link_id, type, title, payload, raw_data, style, scan_count, created_at, updated_at
		FROM qr_codes WHERE team_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`
	if r.db.DriverName == "sqlite" {
		query = `SELECT id, team_id, user_id, link_id, type, title, payload, raw_data, style, scan_count, created_at, updated_at
			FROM qr_codes WHERE team_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?`
	}

	rows, err := r.db.Query(query, teamID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []models.QRCode
	for rows.Next() {
		var q models.QRCode
		var styleJSON string
		if err := rows.Scan(
			&q.ID, &q.TeamID, &q.UserID, &q.LinkID, &q.Type, &q.Title, &q.Payload, &q.RawData,
			&styleJSON, &q.ScanCount, &q.CreatedAt, &q.UpdatedAt,
		); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(styleJSON), &q.Style)
		list = append(list, q)
	}
	return list, nil
}

func (r *Repository) UpdateQRCode(q *models.QRCode) error {
	q.UpdatedAt = time.Now().UTC()
	styleJSON, err := json.Marshal(q.Style)
	if err != nil {
		return fmt.Errorf("failed to marshal style: %w", err)
	}

	query := `UPDATE qr_codes SET title = $1, payload = $2, raw_data = $3, style = $4, updated_at = $5
		WHERE id = $6`
	if r.db.DriverName == "sqlite" {
		query = `UPDATE qr_codes SET title = ?, payload = ?, raw_data = ?, style = ?, updated_at = ?
			WHERE id = ?`
	}

	_, err = r.db.Exec(query, q.Title, q.Payload, q.RawData, string(styleJSON), q.UpdatedAt, q.ID)
	return err
}

func (r *Repository) IncrementQRScans(id string) error {
	query := `UPDATE qr_codes SET scan_count = scan_count + 1 WHERE id = $1`
	if r.db.DriverName == "sqlite" {
		query = `UPDATE qr_codes SET scan_count = scan_count + 1 WHERE id = ?`
	}
	_, err := r.db.Exec(query, id)
	return err
}

func (r *Repository) DeleteQRCode(id string) error {
	query := `DELETE FROM qr_codes WHERE id = $1`
	if r.db.DriverName == "sqlite" {
		query = `DELETE FROM qr_codes WHERE id = ?`
	}
	_, err := r.db.Exec(query, id)
	return err
}

// ----------------- Analytics -----------------

func (r *Repository) RecordEvent(e *models.AnalyticsEvent) error {
	if e.ID == "" {
		e.ID = uuid.New().String()
	}
	if e.CreatedAt.IsZero() {
		e.CreatedAt = time.Now().UTC()
	}

	query := `INSERT INTO analytics_events (id, link_id, qr_code_id, event_type, ip, user_agent, browser, os, device, referrer, country, city, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`
	if r.db.DriverName == "sqlite" {
		query = `INSERT INTO analytics_events (id, link_id, qr_code_id, event_type, ip, user_agent, browser, os, device, referrer, country, city, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	}

	_, err := r.db.Exec(query, e.ID, e.LinkID, e.QRCodeID, e.EventType, e.IP, e.UserAgent, e.Browser, e.OS, e.Device, e.Referrer, e.Country, e.City, e.CreatedAt)
	return err
}

func (r *Repository) GetAnalyticsSummary(linkID, qrID *string) (*models.AnalyticsSummary, error) {
	summary := &models.AnalyticsSummary{
		ByDate:     []models.DateMetric{},
		ByCountry:  []models.NameCount{},
		ByDevice:   []models.NameCount{},
		ByBrowser:  []models.NameCount{},
		ByReferrer: []models.NameCount{},
	}

	whereClause := "1=1"
	var args []interface{}
	idx := 1

	if linkID != nil && *linkID != "" {
		if r.db.DriverName == "sqlite" {
			whereClause += " AND link_id = ?"
		} else {
			whereClause += fmt.Sprintf(" AND link_id = $%d", idx)
			idx++
		}
		args = append(args, *linkID)
	}

	if qrID != nil && *qrID != "" {
		if r.db.DriverName == "sqlite" {
			whereClause += " AND qr_code_id = ?"
		} else {
			whereClause += fmt.Sprintf(" AND qr_code_id = $%d", idx)
			idx++
		}
		args = append(args, *qrID)
	}

	// Totals
	totalQuery := fmt.Sprintf(`SELECT
		COALESCE(SUM(CASE WHEN event_type = 'click' THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN event_type = 'scan' THEN 1 ELSE 0 END), 0)
		FROM analytics_events WHERE %s`, whereClause)

	_ = r.db.QueryRow(totalQuery, args...).Scan(&summary.TotalClicks, &summary.TotalScans)

	// By Country
	countryQuery := fmt.Sprintf(`SELECT COALESCE(NULLIF(country, ''), 'Direct / Unknown') as name, COUNT(*) as cnt
		FROM analytics_events WHERE %s GROUP BY name ORDER BY cnt DESC LIMIT 10`, whereClause)
	if rows, err := r.db.Query(countryQuery, args...); err == nil {
		defer rows.Close()
		for rows.Next() {
			var nc models.NameCount
			if err := rows.Scan(&nc.Name, &nc.Count); err == nil {
				summary.ByCountry = append(summary.ByCountry, nc)
			}
		}
	}

	// By Device
	deviceQuery := fmt.Sprintf(`SELECT COALESCE(NULLIF(device, ''), 'Unknown') as name, COUNT(*) as cnt
		FROM analytics_events WHERE %s GROUP BY name ORDER BY cnt DESC LIMIT 10`, whereClause)
	if rows, err := r.db.Query(deviceQuery, args...); err == nil {
		defer rows.Close()
		for rows.Next() {
			var nc models.NameCount
			if err := rows.Scan(&nc.Name, &nc.Count); err == nil {
				summary.ByDevice = append(summary.ByDevice, nc)
			}
		}
	}

	// By Browser
	browserQuery := fmt.Sprintf(`SELECT COALESCE(NULLIF(browser, ''), 'Unknown') as name, COUNT(*) as cnt
		FROM analytics_events WHERE %s GROUP BY name ORDER BY cnt DESC LIMIT 10`, whereClause)
	if rows, err := r.db.Query(browserQuery, args...); err == nil {
		defer rows.Close()
		for rows.Next() {
			var nc models.NameCount
			if err := rows.Scan(&nc.Name, &nc.Count); err == nil {
				summary.ByBrowser = append(summary.ByBrowser, nc)
			}
		}
	}

	// By Referrer
	refQuery := fmt.Sprintf(`SELECT COALESCE(NULLIF(referrer, ''), 'Direct / None') as name, COUNT(*) as cnt
		FROM analytics_events WHERE %s GROUP BY name ORDER BY cnt DESC LIMIT 10`, whereClause)
	if rows, err := r.db.Query(refQuery, args...); err == nil {
		defer rows.Close()
		for rows.Next() {
			var nc models.NameCount
			if err := rows.Scan(&nc.Name, &nc.Count); err == nil {
				summary.ByReferrer = append(summary.ByReferrer, nc)
			}
		}
	}

	return summary, nil
}

// ----------------- Documents -----------------

func (r *Repository) CreateDocument(doc *models.Document) error {
	if doc.ID == "" {
		doc.ID = uuid.New().String()
	}
	now := time.Now().UTC()
	doc.CreatedAt = now
	doc.UpdatedAt = now

	query := `INSERT INTO documents (id, team_id, user_id, filename, stored_path, mime_type, file_size, version, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`
	if r.db.DriverName == "sqlite" {
		query = `INSERT INTO documents (id, team_id, user_id, filename, stored_path, mime_type, file_size, version, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	}

	_, err := r.db.Exec(query, doc.ID, doc.TeamID, doc.UserID, doc.Filename, doc.StoredPath, doc.MimeType, doc.FileSize, doc.Version, doc.CreatedAt, doc.UpdatedAt)
	return err
}

func (r *Repository) GetDocumentByID(id string) (*models.Document, error) {
	query := `SELECT id, team_id, user_id, filename, stored_path, mime_type, file_size, version, created_at, updated_at
		FROM documents WHERE id = $1`
	if r.db.DriverName == "sqlite" {
		query = `SELECT id, team_id, user_id, filename, stored_path, mime_type, file_size, version, created_at, updated_at
			FROM documents WHERE id = ?`
	}

	var d models.Document
	err := r.db.QueryRow(query, id).Scan(
		&d.ID, &d.TeamID, &d.UserID, &d.Filename, &d.StoredPath, &d.MimeType, &d.FileSize, &d.Version, &d.CreatedAt, &d.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *Repository) UpdateDocument(doc *models.Document) error {
	doc.UpdatedAt = time.Now().UTC()
	query := `UPDATE documents SET filename = $1, stored_path = $2, mime_type = $3, file_size = $4, version = version + 1, updated_at = $5
		WHERE id = $6`
	if r.db.DriverName == "sqlite" {
		query = `UPDATE documents SET filename = ?, stored_path = ?, mime_type = ?, file_size = ?, version = version + 1, updated_at = ?
			WHERE id = ?`
	}

	_, err := r.db.Exec(query, doc.Filename, doc.StoredPath, doc.MimeType, doc.FileSize, doc.UpdatedAt, doc.ID)
	return err
}

func (r *Repository) ListDocuments(teamID string) ([]models.Document, error) {
	query := `SELECT id, team_id, user_id, filename, stored_path, mime_type, file_size, version, created_at, updated_at
		FROM documents WHERE team_id = $1 ORDER BY updated_at DESC`
	if r.db.DriverName == "sqlite" {
		query = `SELECT id, team_id, user_id, filename, stored_path, mime_type, file_size, version, created_at, updated_at
			FROM documents WHERE team_id = ? ORDER BY updated_at DESC`
	}

	rows, err := r.db.Query(query, teamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var docs []models.Document
	for rows.Next() {
		var d models.Document
		if err := rows.Scan(
			&d.ID, &d.TeamID, &d.UserID, &d.Filename, &d.StoredPath, &d.MimeType, &d.FileSize, &d.Version, &d.CreatedAt, &d.UpdatedAt,
		); err != nil {
			return nil, err
		}
		docs = append(docs, d)
	}
	return docs, nil
}

// ----------------- Teams -----------------

func (r *Repository) EnsureDefaultTeam(teamID, teamName, userID string) error {
	var count int
	query := `SELECT COUNT(*) FROM teams WHERE id = $1`
	if r.db.DriverName == "sqlite" {
		query = `SELECT COUNT(*) FROM teams WHERE id = ?`
	}
	_ = r.db.QueryRow(query, teamID).Scan(&count)
	if count > 0 {
		return nil
	}

	now := time.Now().UTC()
	insertTeam := `INSERT INTO teams (id, name, slug, created_at, updated_at) VALUES ($1, $2, $3, $4, $5)`
	if r.db.DriverName == "sqlite" {
		insertTeam = `INSERT INTO teams (id, name, slug, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`
	}
	if _, err := r.db.Exec(insertTeam, teamID, teamName, "default-team", now, now); err != nil {
		return err
	}

	insertUser := `INSERT INTO users (id, email, first_name, last_name, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6)`
	if r.db.DriverName == "sqlite" {
		insertUser = `INSERT INTO users (id, email, first_name, last_name, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`
	}
	_, _ = r.db.Exec(insertUser, userID, "user@linksy.local", "Default", "User", now, now)

	insertMember := `INSERT INTO team_members (id, team_id, user_id, role, created_at) VALUES ($1, $2, $3, $4, $5)`
	if r.db.DriverName == "sqlite" {
		insertMember = `INSERT INTO team_members (id, team_id, user_id, role, created_at) VALUES (?, ?, ?, ?, ?)`
	}
	_, _ = r.db.Exec(insertMember, uuid.New().String(), teamID, userID, string(models.RoleOwner), now)

	return nil
}

// ----------------- Users & Profile -----------------

func (r *Repository) CreateOrUpdateUser(u *models.User) error {
	now := time.Now().UTC()
	if u.CreatedAt.IsZero() {
		u.CreatedAt = now
	}
	u.UpdatedAt = now

	// Check if exists
	var count int
	checkQuery := `SELECT COUNT(*) FROM users WHERE id = $1`
	if r.db.DriverName == "sqlite" {
		checkQuery = `SELECT COUNT(*) FROM users WHERE id = ?`
	}
	_ = r.db.QueryRow(checkQuery, u.ID).Scan(&count)

	if count > 0 {
		updateQuery := `UPDATE users SET first_name = $1, last_name = $2, avatar_url = $3, updated_at = $4 WHERE id = $5`
		if r.db.DriverName == "sqlite" {
			updateQuery = `UPDATE users SET first_name = ?, last_name = ?, avatar_url = ?, updated_at = ? WHERE id = ?`
		}
		_, err := r.db.Exec(updateQuery, u.FirstName, u.LastName, u.AvatarURL, u.UpdatedAt, u.ID)
		return err
	}

	insertQuery := `INSERT INTO users (id, email, first_name, last_name, avatar_url, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7)`
	if r.db.DriverName == "sqlite" {
		insertQuery = `INSERT INTO users (id, email, first_name, last_name, avatar_url, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`
	}
	_, err := r.db.Exec(insertQuery, u.ID, u.Email, u.FirstName, u.LastName, u.AvatarURL, u.CreatedAt, u.UpdatedAt)
	return err
}

func (r *Repository) GetUserByID(id string) (*models.User, error) {
	query := `SELECT id, email, first_name, last_name, avatar_url, created_at, updated_at FROM users WHERE id = $1`
	if r.db.DriverName == "sqlite" {
		query = `SELECT id, email, first_name, last_name, avatar_url, created_at, updated_at FROM users WHERE id = ?`
	}
	var u models.User
	var avatar sql.NullString
	err := r.db.QueryRow(query, id).Scan(&u.ID, &u.Email, &u.FirstName, &u.LastName, &avatar, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if avatar.Valid {
		u.AvatarURL = avatar.String
	}
	return &u, nil
}

func (r *Repository) GetUserByEmail(email string) (*models.User, error) {
	query := `SELECT id, email, first_name, last_name, avatar_url, created_at, updated_at FROM users WHERE email = $1`
	if r.db.DriverName == "sqlite" {
		query = `SELECT id, email, first_name, last_name, avatar_url, created_at, updated_at FROM users WHERE email = ?`
	}
	var u models.User
	var avatar sql.NullString
	err := r.db.QueryRow(query, email).Scan(&u.ID, &u.Email, &u.FirstName, &u.LastName, &avatar, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if avatar.Valid {
		u.AvatarURL = avatar.String
	}
	return &u, nil
}

// ----------------- API Keys -----------------

func (r *Repository) CreateAPIKey(k *models.APIKey, hash string) error {
	if k.ID == "" {
		k.ID = "key_" + uuid.New().String()
	}
	now := time.Now().UTC()
	k.CreatedAt = now

	query := `INSERT INTO api_keys (id, user_id, team_id, name, key_hash, key_prefix, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	if r.db.DriverName == "sqlite" {
		query = `INSERT INTO api_keys (id, user_id, team_id, name, key_hash, key_prefix, expires_at, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	}
	_, err := r.db.Exec(query, k.ID, k.UserID, k.TeamID, k.Name, hash, k.KeyPrefix, k.ExpiresAt, k.CreatedAt)
	return err
}

func (r *Repository) ListAPIKeys(userID string) ([]models.APIKey, error) {
	query := `SELECT id, user_id, team_id, name, key_prefix, last_used_at, expires_at, created_at
		FROM api_keys WHERE user_id = $1 ORDER BY created_at DESC`
	if r.db.DriverName == "sqlite" {
		query = `SELECT id, user_id, team_id, name, key_prefix, last_used_at, expires_at, created_at
			FROM api_keys WHERE user_id = ? ORDER BY created_at DESC`
	}
	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []models.APIKey
	for rows.Next() {
		var k models.APIKey
		var teamID sql.NullString
		if err := rows.Scan(&k.ID, &k.UserID, &teamID, &k.Name, &k.KeyPrefix, &k.LastUsedAt, &k.ExpiresAt, &k.CreatedAt); err != nil {
			return nil, err
		}
		if teamID.Valid {
			k.TeamID = teamID.String
		}
		keys = append(keys, k)
	}
	return keys, nil
}

func (r *Repository) DeleteAPIKey(id, userID string) error {
	query := `DELETE FROM api_keys WHERE id = $1 AND user_id = $2`
	if r.db.DriverName == "sqlite" {
		query = `DELETE FROM api_keys WHERE id = ? AND user_id = ?`
	}
	_, err := r.db.Exec(query, id, userID)
	return err
}

func (r *Repository) GetAPIKeyByHash(hash string) (*models.APIKey, error) {
	query := `SELECT id, user_id, team_id, name, key_prefix, last_used_at, expires_at, created_at
		FROM api_keys WHERE key_hash = $1`
	if r.db.DriverName == "sqlite" {
		query = `SELECT id, user_id, team_id, name, key_prefix, last_used_at, expires_at, created_at
			FROM api_keys WHERE key_hash = ?`
	}
	var k models.APIKey
	var teamID sql.NullString
	err := r.db.QueryRow(query, hash).Scan(&k.ID, &k.UserID, &teamID, &k.Name, &k.KeyPrefix, &k.LastUsedAt, &k.ExpiresAt, &k.CreatedAt)
	if err != nil {
		return nil, err
	}
	if teamID.Valid {
		k.TeamID = teamID.String
	}
	return &k, nil
}

func (r *Repository) UpdateAPIKeyLastUsed(id string) error {
	now := time.Now().UTC()
	query := `UPDATE api_keys SET last_used_at = $1 WHERE id = $2`
	if r.db.DriverName == "sqlite" {
		query = `UPDATE api_keys SET last_used_at = ? WHERE id = ?`
	}
	_, err := r.db.Exec(query, now, id)
	return err
}
