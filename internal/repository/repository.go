package repository

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/bcrowe306/linksy/internal/db"
	"github.com/bcrowe306/linksy/internal/models"
)

// ErrNotFound is returned when a record does not exist or is outside the caller's team.
var ErrNotFound = errors.New("record not found")

type Repository struct {
	db *db.DB
}

func New(database *db.DB) *Repository {
	return &Repository{db: database}
}

// All queries are written with '?' placeholders and rebound for the active driver.
func (r *Repository) exec(q string, args ...any) (sql.Result, error) {
	return r.db.Exec(r.db.Rebind(q), args...)
}

func (r *Repository) query(q string, args ...any) (*sql.Rows, error) {
	return r.db.Query(r.db.Rebind(q), args...)
}

func (r *Repository) queryRow(q string, args ...any) *sql.Row {
	return r.db.QueryRow(r.db.Rebind(q), args...)
}

func notFound(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

// requireAffected converts a zero-row write into ErrNotFound so callers can't
// silently "succeed" against records belonging to another team.
func requireAffected(res sql.Result, err error) error {
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func now() time.Time { return time.Now().UTC() }

// ----------------- Users -----------------

const userCols = `id, email, first_name, last_name, avatar_url, created_at, updated_at`

func scanUser(row interface{ Scan(...any) error }) (*models.User, error) {
	var u models.User
	if err := row.Scan(&u.ID, &u.Email, &u.FirstName, &u.LastName, &u.AvatarURL, &u.CreatedAt, &u.UpdatedAt); err != nil {
		return nil, notFound(err)
	}
	return &u, nil
}

// UpsertUser inserts a user or refreshes their identity fields from the identity provider.
func (r *Repository) UpsertUser(u *models.User) error {
	t := now()
	u.UpdatedAt = t
	existing, err := r.GetUserByID(u.ID)
	if err == nil {
		u.CreatedAt = existing.CreatedAt
		_, err = r.exec(`UPDATE users SET email = ?, first_name = ?, last_name = ?, avatar_url = ?, updated_at = ? WHERE id = ?`,
			u.Email, u.FirstName, u.LastName, u.AvatarURL, t, u.ID)
		return err
	}
	if !errors.Is(err, ErrNotFound) {
		return err
	}
	u.CreatedAt = t
	_, err = r.exec(`INSERT INTO users (`+userCols+`) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		u.ID, u.Email, u.FirstName, u.LastName, u.AvatarURL, u.CreatedAt, u.UpdatedAt)
	return err
}

func (r *Repository) GetUserByID(id string) (*models.User, error) {
	return scanUser(r.queryRow(`SELECT `+userCols+` FROM users WHERE id = ?`, id))
}

func (r *Repository) GetUserByEmail(email string) (*models.User, error) {
	return scanUser(r.queryRow(`SELECT `+userCols+` FROM users WHERE email = ?`, strings.ToLower(email)))
}

// ----------------- Teams -----------------

// CreateTeam creates a team and its owner membership atomically.
func (r *Repository) CreateTeam(t *models.Team) error {
	if t.ID == "" {
		t.ID = "team_" + uuid.NewString()
	}
	ts := now()
	t.CreatedAt, t.UpdatedAt = ts, ts

	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(r.db.Rebind(`INSERT INTO teams (id, name, slug, owner_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`),
		t.ID, t.Name, t.Slug, t.OwnerID, ts, ts); err != nil {
		return err
	}
	if _, err := tx.Exec(r.db.Rebind(`INSERT INTO team_members (id, team_id, user_id, role, status, created_at) VALUES (?, ?, ?, ?, ?, ?)`),
		uuid.NewString(), t.ID, t.OwnerID, string(models.RoleOwner), models.MemberActive, ts); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *Repository) GetTeam(id string) (*models.Team, error) {
	var t models.Team
	err := r.queryRow(`SELECT id, name, slug, owner_id, created_at, updated_at FROM teams WHERE id = ?`, id).
		Scan(&t.ID, &t.Name, &t.Slug, &t.OwnerID, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, notFound(err)
	}
	return &t, nil
}

func (r *Repository) TeamSlugExists(slug string) bool {
	var n int
	_ = r.queryRow(`SELECT COUNT(*) FROM teams WHERE slug = ?`, slug).Scan(&n)
	return n > 0
}

func (r *Repository) RenameTeam(id, name string) error {
	return requireAffected(r.exec(`UPDATE teams SET name = ?, updated_at = ? WHERE id = ?`, name, now(), id))
}

func (r *Repository) DeleteTeam(id string) error {
	// Child rows without FK cascades are removed explicitly.
	for _, q := range []string{
		`DELETE FROM analytics_events WHERE team_id = ?`,
	} {
		if _, err := r.exec(q, id); err != nil {
			return err
		}
	}
	return requireAffected(r.exec(`DELETE FROM teams WHERE id = ?`, id))
}

// ListUserTeams returns every team the user belongs to, including suspended memberships.
func (r *Repository) ListUserTeams(userID string) ([]models.TeamMembership, error) {
	rows, err := r.query(`SELECT t.id, t.name, t.slug, t.owner_id, tm.role, tm.status, t.created_at,
		(SELECT COUNT(*) FROM team_members m2 WHERE m2.team_id = t.id)
		FROM team_members tm JOIN teams t ON t.id = tm.team_id
		WHERE tm.user_id = ? ORDER BY tm.created_at ASC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.TeamMembership{}
	for rows.Next() {
		var m models.TeamMembership
		if err := rows.Scan(&m.ID, &m.Name, &m.Slug, &m.OwnerID, &m.Role, &m.Status, &m.CreatedAt, &m.MemberCount); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// TransferOwnership makes newOwnerID the owner and demotes the previous owner to admin.
func (r *Repository) TransferOwnership(teamID, oldOwnerID, newOwnerID string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	steps := []struct {
		q    string
		args []any
	}{
		{`UPDATE teams SET owner_id = ?, updated_at = ? WHERE id = ? AND owner_id = ?`, []any{newOwnerID, now(), teamID, oldOwnerID}},
		{`UPDATE team_members SET role = ?, status = ? WHERE team_id = ? AND user_id = ?`, []any{string(models.RoleOwner), models.MemberActive, teamID, newOwnerID}},
		{`UPDATE team_members SET role = ? WHERE team_id = ? AND user_id = ?`, []any{string(models.RoleAdmin), teamID, oldOwnerID}},
	}
	for _, s := range steps {
		res, err := tx.Exec(r.db.Rebind(s.q), s.args...)
		if err := requireAffected(res, err); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// ----------------- Team Members -----------------

func (r *Repository) GetMembership(teamID, userID string) (*models.TeamMember, error) {
	var tm models.TeamMember
	err := r.queryRow(`SELECT id, team_id, user_id, role, status, created_at FROM team_members WHERE team_id = ? AND user_id = ?`, teamID, userID).
		Scan(&tm.ID, &tm.TeamID, &tm.UserID, &tm.Role, &tm.Status, &tm.CreatedAt)
	if err != nil {
		return nil, notFound(err)
	}
	return &tm, nil
}

func (r *Repository) ListTeamMembers(teamID string) ([]models.TeamMemberDetail, error) {
	rows, err := r.query(`SELECT tm.id, tm.team_id, tm.user_id, tm.role, tm.status, tm.created_at,
		u.email, u.first_name, u.last_name, u.avatar_url
		FROM team_members tm JOIN users u ON tm.user_id = u.id
		WHERE tm.team_id = ? ORDER BY tm.created_at ASC`, teamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	members := []models.TeamMemberDetail{}
	for rows.Next() {
		var m models.TeamMemberDetail
		if err := rows.Scan(&m.ID, &m.TeamID, &m.UserID, &m.Role, &m.Status, &m.CreatedAt,
			&m.Email, &m.FirstName, &m.LastName, &m.AvatarURL); err != nil {
			return nil, err
		}
		m.Name = strings.TrimSpace(m.FirstName + " " + m.LastName)
		if m.Name == "" {
			m.Name = strings.Split(m.Email, "@")[0]
		}
		members = append(members, m)
	}
	return members, rows.Err()
}

func (r *Repository) AddTeamMember(teamID, userID string, role models.UserRole) error {
	_, err := r.exec(`INSERT INTO team_members (id, team_id, user_id, role, status, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		uuid.NewString(), teamID, userID, string(role), models.MemberActive, now())
	return err
}

func (r *Repository) UpdateMemberRole(teamID, userID string, role models.UserRole) error {
	return requireAffected(r.exec(`UPDATE team_members SET role = ? WHERE team_id = ? AND user_id = ?`, string(role), teamID, userID))
}

func (r *Repository) UpdateMemberStatus(teamID, userID, status string) error {
	return requireAffected(r.exec(`UPDATE team_members SET status = ? WHERE team_id = ? AND user_id = ?`, status, teamID, userID))
}

// RemoveTeamMember removes only the membership; the user's account and other teams are untouched.
func (r *Repository) RemoveTeamMember(teamID, userID string) error {
	if err := requireAffected(r.exec(`DELETE FROM team_members WHERE team_id = ? AND user_id = ?`, teamID, userID)); err != nil {
		return err
	}
	// Keys issued for this team stop working once the member leaves.
	_, _ = r.exec(`DELETE FROM api_keys WHERE team_id = ? AND user_id = ?`, teamID, userID)
	_, _ = r.exec(`UPDATE sessions SET active_team_id = NULL WHERE user_id = ? AND active_team_id = ?`, userID, teamID)
	return nil
}

// ----------------- Sessions -----------------

func (r *Repository) CreateSession(s *models.Session, tokenHash string) error {
	if s.ID == "" {
		s.ID = "ses_" + uuid.NewString()
	}
	var team any
	if s.ActiveTeamID != "" {
		team = s.ActiveTeamID
	}
	_, err := r.exec(`INSERT INTO sessions (id, user_id, token_hash, active_team_id, ip, user_agent, created_at, last_seen_at, expires_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		s.ID, s.UserID, tokenHash, team, s.IP, s.UserAgent, s.CreatedAt, s.LastSeenAt, s.ExpiresAt)
	return err
}

const sessionCols = `id, user_id, COALESCE(active_team_id, ''), ip, user_agent, created_at, last_seen_at, expires_at`

func scanSession(row interface{ Scan(...any) error }) (*models.Session, error) {
	var s models.Session
	if err := row.Scan(&s.ID, &s.UserID, &s.ActiveTeamID, &s.IP, &s.UserAgent, &s.CreatedAt, &s.LastSeenAt, &s.ExpiresAt); err != nil {
		return nil, notFound(err)
	}
	return &s, nil
}

func (r *Repository) GetSessionByTokenHash(hash string) (*models.Session, error) {
	return scanSession(r.queryRow(`SELECT `+sessionCols+` FROM sessions WHERE token_hash = ?`, hash))
}

func (r *Repository) ListUserSessions(userID string) ([]models.Session, error) {
	rows, err := r.query(`SELECT `+sessionCols+` FROM sessions WHERE user_id = ? AND expires_at > ? ORDER BY last_seen_at DESC`, userID, now())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.Session{}
	for rows.Next() {
		s, err := scanSession(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *s)
	}
	return out, rows.Err()
}

func (r *Repository) TouchSession(id string, lastSeen, expiresAt time.Time) error {
	_, err := r.exec(`UPDATE sessions SET last_seen_at = ?, expires_at = ? WHERE id = ?`, lastSeen, expiresAt, id)
	return err
}

func (r *Repository) SetSessionTeam(id, teamID string) error {
	return requireAffected(r.exec(`UPDATE sessions SET active_team_id = ? WHERE id = ?`, teamID, id))
}

func (r *Repository) DeleteSession(id, userID string) error {
	return requireAffected(r.exec(`DELETE FROM sessions WHERE id = ? AND user_id = ?`, id, userID))
}

func (r *Repository) DeleteUserSessionsExcept(userID, keepID string) error {
	_, err := r.exec(`DELETE FROM sessions WHERE user_id = ? AND id <> ?`, userID, keepID)
	return err
}

func (r *Repository) DeleteExpiredSessions() error {
	_, err := r.exec(`DELETE FROM sessions WHERE expires_at < ?`, now())
	return err
}

// ----------------- Invitations -----------------

func (r *Repository) CreateInvitation(inv *models.Invitation, tokenHash string) error {
	if inv.ID == "" {
		inv.ID = "inv_" + uuid.NewString()
	}
	if inv.CreatedAt.IsZero() {
		inv.CreatedAt = now()
	}
	inv.State = "pending"
	_, err := r.exec(`INSERT INTO invitations (id, team_id, email, role, token_hash, invited_by, state, created_at, expires_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		inv.ID, inv.TeamID, inv.Email, string(inv.Role), tokenHash, inv.InvitedBy, inv.State, inv.CreatedAt, inv.ExpiresAt)
	return err
}

const invitationCols = `i.id, i.team_id, t.name, i.email, i.role, i.invited_by, i.state, i.created_at, i.expires_at`

func scanInvitation(row interface{ Scan(...any) error }) (*models.Invitation, error) {
	var inv models.Invitation
	if err := row.Scan(&inv.ID, &inv.TeamID, &inv.TeamName, &inv.Email, &inv.Role, &inv.InvitedBy, &inv.State, &inv.CreatedAt, &inv.ExpiresAt); err != nil {
		return nil, notFound(err)
	}
	return &inv, nil
}

func (r *Repository) GetInvitationByTokenHash(hash string) (*models.Invitation, error) {
	return scanInvitation(r.queryRow(`SELECT `+invitationCols+` FROM invitations i JOIN teams t ON t.id = i.team_id WHERE i.token_hash = ?`, hash))
}

func (r *Repository) listInvitations(where string, arg any) ([]models.Invitation, error) {
	rows, err := r.query(`SELECT `+invitationCols+` FROM invitations i JOIN teams t ON t.id = i.team_id
		WHERE `+where+` AND i.state = 'pending' AND i.expires_at > ? ORDER BY i.created_at DESC`, arg, now())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.Invitation{}
	for rows.Next() {
		inv, err := scanInvitation(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *inv)
	}
	return out, rows.Err()
}

func (r *Repository) ListPendingInvitations(teamID string) ([]models.Invitation, error) {
	return r.listInvitations("i.team_id = ?", teamID)
}

func (r *Repository) ListPendingInvitationsForEmail(email string) ([]models.Invitation, error) {
	return r.listInvitations("i.email = ?", strings.ToLower(email))
}

func (r *Repository) HasPendingInvitation(teamID, email string) bool {
	var n int
	_ = r.queryRow(`SELECT COUNT(*) FROM invitations WHERE team_id = ? AND email = ? AND state = 'pending' AND expires_at > ?`, teamID, email, now()).Scan(&n)
	return n > 0
}

func (r *Repository) SetInvitationState(id, teamID, state string) error {
	return requireAffected(r.exec(`UPDATE invitations SET state = ? WHERE id = ? AND team_id = ? AND state = 'pending'`, state, id, teamID))
}

// ----------------- Links -----------------

const linkCols = `id, team_id, user_id, slug, target_url, title, description, is_active, expires_at, password_hash,
	utm_source, utm_medium, utm_campaign, utm_term, utm_content, click_count, created_at, updated_at`

func scanLink(row interface{ Scan(...any) error }) (*models.Link, error) {
	var l models.Link
	if err := row.Scan(&l.ID, &l.TeamID, &l.UserID, &l.Slug, &l.TargetURL, &l.Title, &l.Description,
		&l.IsActive, &l.ExpiresAt, &l.PasswordHash,
		&l.UTMSource, &l.UTMMedium, &l.UTMCampaign, &l.UTMTerm, &l.UTMContent,
		&l.ClickCount, &l.CreatedAt, &l.UpdatedAt); err != nil {
		return nil, notFound(err)
	}
	l.HasPassword = l.PasswordHash != ""
	return &l, nil
}

func (r *Repository) CreateLink(l *models.Link) error {
	if l.ID == "" {
		l.ID = uuid.NewString()
	}
	ts := now()
	l.CreatedAt, l.UpdatedAt = ts, ts
	_, err := r.exec(`INSERT INTO links (`+linkCols+`) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		l.ID, l.TeamID, l.UserID, l.Slug, l.TargetURL, l.Title, l.Description, l.IsActive, l.ExpiresAt, l.PasswordHash,
		l.UTMSource, l.UTMMedium, l.UTMCampaign, l.UTMTerm, l.UTMContent, l.ClickCount, l.CreatedAt, l.UpdatedAt)
	l.HasPassword = l.PasswordHash != ""
	return err
}

func (r *Repository) SlugExists(slug string) bool {
	var n int
	_ = r.queryRow(`SELECT COUNT(*) FROM links WHERE slug = ?`, slug).Scan(&n)
	return n > 0
}

// GetLinkBySlug is used by the public redirector and is intentionally not team-scoped.
func (r *Repository) GetLinkBySlug(slug string) (*models.Link, error) {
	return scanLink(r.queryRow(`SELECT `+linkCols+` FROM links WHERE slug = ?`, slug))
}

// GetLinkByID is used by internal flows (QR redirection) and is not team-scoped.
func (r *Repository) GetLinkByID(id string) (*models.Link, error) {
	return scanLink(r.queryRow(`SELECT `+linkCols+` FROM links WHERE id = ?`, id))
}

func (r *Repository) GetLink(id, teamID string) (*models.Link, error) {
	return scanLink(r.queryRow(`SELECT `+linkCols+` FROM links WHERE id = ? AND team_id = ?`, id, teamID))
}

func (r *Repository) ListLinks(teamID string, limit, offset int) ([]models.Link, error) {
	rows, err := r.query(`SELECT `+linkCols+` FROM links WHERE team_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?`, teamID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	links := []models.Link{}
	for rows.Next() {
		l, err := scanLink(rows)
		if err != nil {
			return nil, err
		}
		links = append(links, *l)
	}
	return links, rows.Err()
}

func (r *Repository) UpdateLink(l *models.Link) error {
	l.UpdatedAt = now()
	l.HasPassword = l.PasswordHash != ""
	return requireAffected(r.exec(`UPDATE links SET target_url = ?, title = ?, description = ?, is_active = ?, expires_at = ?, password_hash = ?,
		utm_source = ?, utm_medium = ?, utm_campaign = ?, utm_term = ?, utm_content = ?, updated_at = ?
		WHERE id = ? AND team_id = ?`,
		l.TargetURL, l.Title, l.Description, l.IsActive, l.ExpiresAt, l.PasswordHash,
		l.UTMSource, l.UTMMedium, l.UTMCampaign, l.UTMTerm, l.UTMContent, l.UpdatedAt, l.ID, l.TeamID))
}

func (r *Repository) IncrementLinkClicks(id string) error {
	_, err := r.exec(`UPDATE links SET click_count = click_count + 1 WHERE id = ?`, id)
	return err
}

func (r *Repository) DeleteLink(id, teamID string) error {
	if err := requireAffected(r.exec(`DELETE FROM links WHERE id = ? AND team_id = ?`, id, teamID)); err != nil {
		return err
	}
	// Dynamic QR codes keep their own payload; detach the dangling reference.
	_, _ = r.exec(`UPDATE qr_codes SET link_id = NULL WHERE link_id = ? AND team_id = ?`, id, teamID)
	return nil
}

// ----------------- QR Codes -----------------

const qrCols = `id, team_id, user_id, link_id, document_id, type, title, payload, raw_data, style, scan_count, created_at, updated_at`

func scanQR(row interface{ Scan(...any) error }) (*models.QRCode, error) {
	var q models.QRCode
	var styleJSON string
	if err := row.Scan(&q.ID, &q.TeamID, &q.UserID, &q.LinkID, &q.DocumentID, &q.Type, &q.Title, &q.Payload, &q.RawData,
		&styleJSON, &q.ScanCount, &q.CreatedAt, &q.UpdatedAt); err != nil {
		return nil, notFound(err)
	}
	_ = json.Unmarshal([]byte(styleJSON), &q.Style)
	return &q, nil
}

func (r *Repository) CreateQRCode(q *models.QRCode) error {
	if q.ID == "" {
		q.ID = uuid.NewString()
	}
	ts := now()
	q.CreatedAt, q.UpdatedAt = ts, ts
	styleJSON, err := json.Marshal(q.Style)
	if err != nil {
		return err
	}
	_, err = r.exec(`INSERT INTO qr_codes (`+qrCols+`) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		q.ID, q.TeamID, q.UserID, q.LinkID, q.DocumentID, string(q.Type), q.Title, q.Payload, q.RawData, string(styleJSON), q.ScanCount, q.CreatedAt, q.UpdatedAt)
	return err
}

// GetQRCodeByID is used by the public scan redirector and is not team-scoped.
func (r *Repository) GetQRCodeByID(id string) (*models.QRCode, error) {
	return scanQR(r.queryRow(`SELECT `+qrCols+` FROM qr_codes WHERE id = ?`, id))
}

func (r *Repository) GetQRCode(id, teamID string) (*models.QRCode, error) {
	return scanQR(r.queryRow(`SELECT `+qrCols+` FROM qr_codes WHERE id = ? AND team_id = ?`, id, teamID))
}

func (r *Repository) listQR(where string, args ...any) ([]models.QRCode, error) {
	rows, err := r.query(`SELECT `+qrCols+` FROM qr_codes WHERE `+where, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := []models.QRCode{}
	for rows.Next() {
		q, err := scanQR(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, *q)
	}
	return list, rows.Err()
}

func (r *Repository) ListQRCodes(teamID string, limit, offset int) ([]models.QRCode, error) {
	return r.listQR(`team_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?`, teamID, limit, offset)
}

func (r *Repository) ListQRCodesByDocument(docID, teamID string) ([]models.QRCode, error) {
	return r.listQR(`document_id = ? AND team_id = ?`, docID, teamID)
}

func (r *Repository) UpdateQRCode(q *models.QRCode) error {
	q.UpdatedAt = now()
	styleJSON, err := json.Marshal(q.Style)
	if err != nil {
		return err
	}
	return requireAffected(r.exec(`UPDATE qr_codes SET title = ?, payload = ?, raw_data = ?, style = ?, link_id = ?, document_id = ?, updated_at = ?
		WHERE id = ? AND team_id = ?`,
		q.Title, q.Payload, q.RawData, string(styleJSON), q.LinkID, q.DocumentID, q.UpdatedAt, q.ID, q.TeamID))
}

func (r *Repository) IncrementQRScans(id string) error {
	_, err := r.exec(`UPDATE qr_codes SET scan_count = scan_count + 1 WHERE id = ?`, id)
	return err
}

func (r *Repository) DeleteQRCode(id, teamID string) error {
	return requireAffected(r.exec(`DELETE FROM qr_codes WHERE id = ? AND team_id = ?`, id, teamID))
}

// ----------------- Documents -----------------

const docCols = `id, team_id, user_id, filename, stored_path, mime_type, file_size, version, created_at, updated_at`

func scanDoc(row interface{ Scan(...any) error }) (*models.Document, error) {
	var d models.Document
	if err := row.Scan(&d.ID, &d.TeamID, &d.UserID, &d.Filename, &d.StoredPath, &d.MimeType, &d.FileSize, &d.Version, &d.CreatedAt, &d.UpdatedAt); err != nil {
		return nil, notFound(err)
	}
	return &d, nil
}

func (r *Repository) CreateDocument(d *models.Document) error {
	if d.ID == "" {
		d.ID = uuid.NewString()
	}
	ts := now()
	d.CreatedAt, d.UpdatedAt = ts, ts
	if d.Version == 0 {
		d.Version = 1
	}
	_, err := r.exec(`INSERT INTO documents (`+docCols+`) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		d.ID, d.TeamID, d.UserID, d.Filename, d.StoredPath, d.MimeType, d.FileSize, d.Version, d.CreatedAt, d.UpdatedAt)
	return err
}

// GetDocumentByID is used by the public document viewer and is not team-scoped.
func (r *Repository) GetDocumentByID(id string) (*models.Document, error) {
	return scanDoc(r.queryRow(`SELECT `+docCols+` FROM documents WHERE id = ?`, id))
}

func (r *Repository) GetDocument(id, teamID string) (*models.Document, error) {
	return scanDoc(r.queryRow(`SELECT `+docCols+` FROM documents WHERE id = ? AND team_id = ?`, id, teamID))
}

// GetDocumentByStoredPath resolves a public file download by its random storage key.
func (r *Repository) GetDocumentByStoredPath(path string) (*models.Document, error) {
	return scanDoc(r.queryRow(`SELECT `+docCols+` FROM documents WHERE stored_path = ?`, path))
}

func (r *Repository) UpdateDocument(d *models.Document) error {
	d.UpdatedAt = now()
	return requireAffected(r.exec(`UPDATE documents SET filename = ?, stored_path = ?, mime_type = ?, file_size = ?, version = ?, updated_at = ?
		WHERE id = ? AND team_id = ?`,
		d.Filename, d.StoredPath, d.MimeType, d.FileSize, d.Version, d.UpdatedAt, d.ID, d.TeamID))
}

func (r *Repository) ListDocuments(teamID string) ([]models.Document, error) {
	rows, err := r.query(`SELECT `+docCols+` FROM documents WHERE team_id = ? ORDER BY created_at DESC`, teamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	docs := []models.Document{}
	for rows.Next() {
		d, err := scanDoc(rows)
		if err != nil {
			return nil, err
		}
		docs = append(docs, *d)
	}
	return docs, rows.Err()
}

func (r *Repository) DeleteDocument(id, teamID string) error {
	return requireAffected(r.exec(`DELETE FROM documents WHERE id = ? AND team_id = ?`, id, teamID))
}

// ----------------- Analytics -----------------

func (r *Repository) RecordEvent(e *models.AnalyticsEvent) error {
	if e.ID == "" {
		e.ID = uuid.NewString()
	}
	if e.CreatedAt.IsZero() {
		e.CreatedAt = now()
	}
	_, err := r.exec(`INSERT INTO analytics_events (id, team_id, link_id, qr_code_id, event_type, ip_hash, user_agent, browser, os, device, referrer, channel, country, region, city, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		e.ID, e.TeamID, e.LinkID, e.QRCodeID, e.EventType, e.IPHash, e.UserAgent, e.Browser, e.OS, e.Device, e.Referrer, e.Channel, e.Country, e.Region, e.City, e.CreatedAt)
	return err
}

// AnalyticsFilter scopes analytics queries. TeamID is mandatory.
type AnalyticsFilter struct {
	TeamID string
	LinkID string
	QRID   string
	From   *time.Time
	To     *time.Time
}

func (f AnalyticsFilter) where() (string, []any) {
	clauses := []string{"team_id = ?"}
	args := []any{f.TeamID}
	if f.LinkID != "" {
		clauses = append(clauses, "link_id = ?")
		args = append(args, f.LinkID)
	}
	if f.QRID != "" {
		clauses = append(clauses, "qr_code_id = ?")
		args = append(args, f.QRID)
	}
	if f.From != nil {
		clauses = append(clauses, "created_at >= ?")
		args = append(args, f.From.UTC())
	}
	if f.To != nil {
		clauses = append(clauses, "created_at < ?")
		args = append(args, f.To.UTC())
	}
	return strings.Join(clauses, " AND "), args
}

func (r *Repository) dayExpr() string {
	if r.db.DriverName == "postgres" {
		return `to_char(created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD')`
	}
	return `substr(created_at, 1, 10)`
}

func (r *Repository) nameCounts(column, fallback, where string, args []any, limit int) ([]models.NameCount, error) {
	q := fmt.Sprintf(`SELECT COALESCE(NULLIF(%s, ''), '%s') AS name, COUNT(*) AS cnt FROM analytics_events
		WHERE %s GROUP BY 1 ORDER BY cnt DESC LIMIT %d`, column, fallback, where, limit)
	rows, err := r.query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.NameCount{}
	for rows.Next() {
		var nc models.NameCount
		if err := rows.Scan(&nc.Name, &nc.Count); err != nil {
			return nil, err
		}
		out = append(out, nc)
	}
	return out, rows.Err()
}

func (r *Repository) GetAnalyticsSummary(f AnalyticsFilter) (*models.AnalyticsSummary, error) {
	if f.TeamID == "" {
		return nil, errors.New("analytics query requires a team")
	}
	where, args := f.where()
	s := &models.AnalyticsSummary{}

	if err := r.queryRow(`SELECT
		COALESCE(SUM(CASE WHEN event_type = 'click' THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN event_type = 'scan' THEN 1 ELSE 0 END), 0)
		FROM analytics_events WHERE `+where, args...).Scan(&s.TotalClicks, &s.TotalScans); err != nil {
		return nil, err
	}

	rows, err := r.query(`SELECT `+r.dayExpr()+` AS day,
		SUM(CASE WHEN event_type = 'click' THEN 1 ELSE 0 END),
		SUM(CASE WHEN event_type = 'scan' THEN 1 ELSE 0 END)
		FROM analytics_events WHERE `+where+` GROUP BY 1 ORDER BY 1 ASC`, args...)
	if err != nil {
		return nil, err
	}
	s.ByDate = []models.DateMetric{}
	for rows.Next() {
		var d models.DateMetric
		if err := rows.Scan(&d.Date, &d.Clicks, &d.Scans); err != nil {
			rows.Close()
			return nil, err
		}
		s.ByDate = append(s.ByDate, d)
	}
	rows.Close()

	breakdowns := []struct {
		dest     *[]models.NameCount
		column   string
		fallback string
	}{
		{&s.ByCountry, "country", "Unknown"},
		{&s.ByDevice, "device", "Unknown"},
		{&s.ByBrowser, "browser", "Unknown"},
		{&s.ByReferrer, "referrer", "Direct"},
	}
	for _, b := range breakdowns {
		nc, err := r.nameCounts(b.column, b.fallback, where, args, 10)
		if err != nil {
			return nil, err
		}
		*b.dest = nc
	}
	return s, nil
}

// ----------------- API Keys -----------------

const apiKeyCols = `id, user_id, team_id, name, key_prefix, last_used_at, expires_at, created_at`

func scanAPIKey(row interface{ Scan(...any) error }) (*models.APIKey, error) {
	var k models.APIKey
	if err := row.Scan(&k.ID, &k.UserID, &k.TeamID, &k.Name, &k.KeyPrefix, &k.LastUsedAt, &k.ExpiresAt, &k.CreatedAt); err != nil {
		return nil, notFound(err)
	}
	return &k, nil
}

func (r *Repository) CreateAPIKey(k *models.APIKey, hash string) error {
	if k.ID == "" {
		k.ID = "key_" + uuid.NewString()
	}
	if k.CreatedAt.IsZero() {
		k.CreatedAt = now()
	}
	_, err := r.exec(`INSERT INTO api_keys (id, user_id, team_id, name, key_hash, key_prefix, expires_at, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		k.ID, k.UserID, k.TeamID, k.Name, hash, k.KeyPrefix, k.ExpiresAt, k.CreatedAt)
	return err
}

func (r *Repository) ListAPIKeys(userID, teamID string) ([]models.APIKey, error) {
	rows, err := r.query(`SELECT `+apiKeyCols+` FROM api_keys WHERE user_id = ? AND team_id = ? ORDER BY created_at DESC`, userID, teamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	keys := []models.APIKey{}
	for rows.Next() {
		k, err := scanAPIKey(rows)
		if err != nil {
			return nil, err
		}
		keys = append(keys, *k)
	}
	return keys, rows.Err()
}

func (r *Repository) DeleteAPIKey(id, userID string) error {
	return requireAffected(r.exec(`DELETE FROM api_keys WHERE id = ? AND user_id = ?`, id, userID))
}

func (r *Repository) GetAPIKeyByHash(hash string) (*models.APIKey, error) {
	return scanAPIKey(r.queryRow(`SELECT `+apiKeyCols+` FROM api_keys WHERE key_hash = ?`, hash))
}

func (r *Repository) UpdateAPIKeyLastUsed(id string) error {
	_, err := r.exec(`UPDATE api_keys SET last_used_at = ? WHERE id = ?`, now(), id)
	return err
}

// ----------------- Audit Log -----------------

func (r *Repository) RecordAudit(e *models.AuditEvent) error {
	if e.ID == "" {
		e.ID = uuid.NewString()
	}
	if e.CreatedAt.IsZero() {
		e.CreatedAt = now()
	}
	if e.Metadata == "" {
		e.Metadata = "{}"
	}
	_, err := r.exec(`INSERT INTO audit_events (id, team_id, actor_id, action, target_type, target_id, metadata, ip, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		e.ID, e.TeamID, e.ActorID, e.Action, e.TargetType, e.TargetID, e.Metadata, e.IP, e.CreatedAt)
	return err
}

func (r *Repository) ListAudit(teamID string, limit, offset int) ([]models.AuditEvent, error) {
	rows, err := r.query(`SELECT a.id, a.team_id, a.actor_id, COALESCE(TRIM(u.first_name || ' ' || u.last_name), ''), COALESCE(u.email, ''),
		a.action, a.target_type, a.target_id, a.metadata, a.ip, a.created_at
		FROM audit_events a LEFT JOIN users u ON u.id = a.actor_id
		WHERE a.team_id = ? ORDER BY a.created_at DESC LIMIT ? OFFSET ?`, teamID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.AuditEvent{}
	for rows.Next() {
		var e models.AuditEvent
		if err := rows.Scan(&e.ID, &e.TeamID, &e.ActorID, &e.ActorName, &e.ActorEmail, &e.Action, &e.TargetType, &e.TargetID, &e.Metadata, &e.IP, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
