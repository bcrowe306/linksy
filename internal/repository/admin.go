package repository

import (
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/bcrowe306/linksy/internal/models"
)

// ErrConflict is returned when a unique record already exists.
var ErrConflict = errors.New("record already exists")

func likeArg(q string) string {
	r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return "%" + strings.ToLower(r.Replace(strings.TrimSpace(q))) + "%"
}

// ----------------- Platform admins & suspension -----------------

func (r *Repository) SetPlatformAdmin(userID string, admin bool) error {
	return requireAffected(r.exec(`UPDATE users SET is_platform_admin = ?, updated_at = ? WHERE id = ?`, admin, now(), userID))
}

// GrantAdminsByEmail flags existing users whose email is in the list.
func (r *Repository) GrantAdminsByEmail(emails []string) (int64, error) {
	var total int64
	for _, e := range emails {
		res, err := r.exec(`UPDATE users SET is_platform_admin = ? WHERE email = ? AND is_platform_admin = ?`, true, strings.ToLower(e), false)
		if err != nil {
			return total, err
		}
		n, _ := res.RowsAffected()
		total += n
	}
	return total, nil
}

func (r *Repository) SetUserSuspended(userID, reason string) error {
	return requireAffected(r.exec(`UPDATE users SET suspended_reason = ?, updated_at = ? WHERE id = ?`, reason, now(), userID))
}

func (r *Repository) SetTeamSuspended(teamID, reason string) error {
	return requireAffected(r.exec(`UPDATE teams SET suspended_reason = ?, updated_at = ? WHERE id = ?`, reason, now(), teamID))
}

func (r *Repository) SetLinkModeration(id, reason string) error {
	return requireAffected(r.exec(`UPDATE links SET moderation_reason = ? WHERE id = ?`, reason, id))
}

func (r *Repository) SetQRModeration(id, reason string) error {
	return requireAffected(r.exec(`UPDATE qr_codes SET moderation_reason = ? WHERE id = ?`, reason, id))
}

// ----------------- Admin sessions -----------------

func (r *Repository) SetSessionAdminUntil(sessionID string, until *time.Time) error {
	return requireAffected(r.exec(`UPDATE sessions SET admin_until = ? WHERE id = ?`, until, sessionID))
}

func (r *Repository) GetSessionAdminUntil(sessionID string) (*time.Time, error) {
	var t sql.NullTime
	if err := r.queryRow(`SELECT admin_until FROM sessions WHERE id = ?`, sessionID).Scan(&t); err != nil {
		return nil, notFound(err)
	}
	if !t.Valid {
		return nil, nil
	}
	return &t.Time, nil
}

func (r *Repository) PutAdminChallenge(sessionID, codeHash string, expires time.Time) error {
	_, _ = r.exec(`DELETE FROM admin_challenges WHERE session_id = ?`, sessionID)
	_, err := r.exec(`INSERT INTO admin_challenges (session_id, code_hash, attempts, expires_at) VALUES (?, ?, 0, ?)`, sessionID, codeHash, expires)
	return err
}

// TakeAdminChallengeAttempt counts an attempt and returns the stored challenge.
func (r *Repository) TakeAdminChallengeAttempt(sessionID string) (codeHash string, attempts int, expires time.Time, err error) {
	if _, err = r.exec(`UPDATE admin_challenges SET attempts = attempts + 1 WHERE session_id = ?`, sessionID); err != nil {
		return
	}
	err = notFound(r.queryRow(`SELECT code_hash, attempts, expires_at FROM admin_challenges WHERE session_id = ?`, sessionID).Scan(&codeHash, &attempts, &expires))
	return
}

func (r *Repository) DeleteAdminChallenge(sessionID string) {
	_, _ = r.exec(`DELETE FROM admin_challenges WHERE session_id = ?`, sessionID)
}

// ----------------- Platform audit -----------------

func (r *Repository) RecordPlatformAudit(e *models.PlatformAuditEvent) error {
	if e.ID == "" {
		e.ID = uuid.NewString()
	}
	if e.CreatedAt.IsZero() {
		e.CreatedAt = now()
	}
	if e.Metadata == "" {
		e.Metadata = "{}"
	}
	_, err := r.exec(`INSERT INTO platform_audit (id, actor_id, actor_email, action, target_type, target_id, target_label, metadata, ip, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		e.ID, e.ActorID, e.ActorEmail, e.Action, e.TargetType, e.TargetID, e.TargetLabel, e.Metadata, e.IP, e.CreatedAt)
	return err
}

func (r *Repository) ListPlatformAudit(q string, limit, offset int) ([]models.PlatformAuditEvent, error) {
	where, args := "1 = 1", []any{}
	if strings.TrimSpace(q) != "" {
		where = `(LOWER(action) LIKE ? ESCAPE '\' OR LOWER(actor_email) LIKE ? ESCAPE '\' OR LOWER(target_label) LIKE ? ESCAPE '\' OR target_id = ?)`
		l := likeArg(q)
		args = append(args, l, l, l, strings.TrimSpace(q))
	}
	args = append(args, limit, offset)
	rows, err := r.query(`SELECT id, actor_id, actor_email, action, target_type, target_id, target_label, metadata, ip, created_at
		FROM platform_audit WHERE `+where+` ORDER BY created_at DESC LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.PlatformAuditEvent{}
	for rows.Next() {
		var e models.PlatformAuditEvent
		if err := rows.Scan(&e.ID, &e.ActorID, &e.ActorEmail, &e.Action, &e.TargetType, &e.TargetID, &e.TargetLabel, &e.Metadata, &e.IP, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// ----------------- Domain blocklist -----------------

func (r *Repository) ListBlockedDomains() ([]models.BlockedDomain, error) {
	rows, err := r.query(`SELECT domain, reason, created_by, created_at FROM blocked_domains ORDER BY domain`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.BlockedDomain{}
	for rows.Next() {
		var d models.BlockedDomain
		if err := rows.Scan(&d.Domain, &d.Reason, &d.CreatedBy, &d.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (r *Repository) AddBlockedDomain(d *models.BlockedDomain) error {
	d.CreatedAt = now()
	var n int
	_ = r.queryRow(`SELECT COUNT(*) FROM blocked_domains WHERE domain = ?`, d.Domain).Scan(&n)
	if n > 0 {
		return ErrConflict
	}
	_, err := r.exec(`INSERT INTO blocked_domains (domain, reason, created_by, created_at) VALUES (?, ?, ?, ?)`, d.Domain, d.Reason, d.CreatedBy, d.CreatedAt)
	return err
}

func (r *Repository) RemoveBlockedDomain(domain string) error {
	return requireAffected(r.exec(`DELETE FROM blocked_domains WHERE domain = ?`, domain))
}

// ----------------- Abuse reports -----------------

func (r *Repository) CreateAbuseReport(a *models.AbuseReport) error {
	a.ID = "rpt_" + uuid.NewString()
	a.CreatedAt = now()
	a.Status = "open"
	_, err := r.exec(`INSERT INTO abuse_reports (id, target_type, target_id, target_url, reason, details, reporter_email, reporter_ip, status, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		a.ID, a.TargetType, a.TargetID, a.TargetURL, a.Reason, a.Details, a.ReporterEmail, a.ReporterIP, a.Status, a.CreatedAt)
	return err
}

// CountRecentReports limits how many reports one network can file.
func (r *Repository) CountRecentReports(ip string, since time.Time) int {
	var n int
	_ = r.queryRow(`SELECT COUNT(*) FROM abuse_reports WHERE reporter_ip = ? AND created_at > ?`, ip, since).Scan(&n)
	return n
}

func (r *Repository) ListAbuseReports(status string, limit int) ([]models.AbuseReport, error) {
	where, args := "1 = 1", []any{}
	if status != "" {
		where, args = "status = ?", append(args, status)
	}
	args = append(args, limit)
	rows, err := r.query(`SELECT id, target_type, target_id, target_url, reason, details, reporter_email, status, resolved_by, created_at, resolved_at
		FROM abuse_reports WHERE `+where+` ORDER BY created_at DESC LIMIT ?`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.AbuseReport{}
	for rows.Next() {
		var a models.AbuseReport
		if err := rows.Scan(&a.ID, &a.TargetType, &a.TargetID, &a.TargetURL, &a.Reason, &a.Details, &a.ReporterEmail, &a.Status, &a.ResolvedBy, &a.CreatedAt, &a.ResolvedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *Repository) GetAbuseReport(id string) (*models.AbuseReport, error) {
	var a models.AbuseReport
	err := r.queryRow(`SELECT id, target_type, target_id, target_url, reason, details, reporter_email, status, resolved_by, created_at, resolved_at
		FROM abuse_reports WHERE id = ?`, id).Scan(&a.ID, &a.TargetType, &a.TargetID, &a.TargetURL, &a.Reason, &a.Details, &a.ReporterEmail, &a.Status, &a.ResolvedBy, &a.CreatedAt, &a.ResolvedAt)
	if err != nil {
		return nil, notFound(err)
	}
	return &a, nil
}

func (r *Repository) ResolveAbuseReport(id, status, by string) error {
	return requireAffected(r.exec(`UPDATE abuse_reports SET status = ?, resolved_by = ?, resolved_at = ? WHERE id = ?`, status, by, now(), id))
}

// ----------------- Trial codes -----------------

const trialCodeCols = `code, days, max_redemptions, redemptions, expires_at, active, note, created_by, created_at`

func scanTrialCode(row interface{ Scan(...any) error }) (*models.TrialCode, error) {
	var c models.TrialCode
	if err := row.Scan(&c.Code, &c.Days, &c.MaxRedemptions, &c.Redemptions, &c.ExpiresAt, &c.Active, &c.Note, &c.CreatedBy, &c.CreatedAt); err != nil {
		return nil, notFound(err)
	}
	return &c, nil
}

func (r *Repository) CreateTrialCode(c *models.TrialCode) error {
	if _, err := r.GetTrialCode(c.Code); err == nil {
		return ErrConflict
	}
	c.CreatedAt, c.Active = now(), true
	_, err := r.exec(`INSERT INTO trial_codes (`+trialCodeCols+`) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		c.Code, c.Days, c.MaxRedemptions, 0, c.ExpiresAt, c.Active, c.Note, c.CreatedBy, c.CreatedAt)
	return err
}

// GetTrialCode looks a code up case-insensitively (codes are stored upper-case).
func (r *Repository) GetTrialCode(code string) (*models.TrialCode, error) {
	return scanTrialCode(r.queryRow(`SELECT `+trialCodeCols+` FROM trial_codes WHERE code = ?`, strings.ToUpper(strings.TrimSpace(code))))
}

func (r *Repository) ListTrialCodes() ([]models.TrialCode, error) {
	rows, err := r.query(`SELECT ` + trialCodeCols + ` FROM trial_codes ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.TrialCode{}
	for rows.Next() {
		c, err := scanTrialCode(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *c)
	}
	return out, rows.Err()
}

func (r *Repository) SetTrialCodeActive(code string, active bool) error {
	return requireAffected(r.exec(`UPDATE trial_codes SET active = ? WHERE code = ?`, active, strings.ToUpper(code)))
}

// RedeemTrialCode records a redemption and moves the trial end, atomically.
// It returns ErrConflict if the workspace already used the code and
// ErrNotFound if the code ran out between lookup and redemption.
func (r *Repository) RedeemTrialCode(code, teamID string, trialEnd time.Time) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var n int
	_ = tx.QueryRow(r.db.Rebind(`SELECT COUNT(*) FROM trial_code_redemptions WHERE code = ? AND team_id = ?`), code, teamID).Scan(&n)
	if n > 0 {
		return ErrConflict
	}
	res, err := tx.Exec(r.db.Rebind(`UPDATE trial_codes SET redemptions = redemptions + 1
		WHERE code = ? AND active = ? AND (max_redemptions = 0 OR redemptions < max_redemptions)`), code, true)
	if err := requireAffected(res, err); err != nil {
		return err
	}
	if _, err := tx.Exec(r.db.Rebind(`INSERT INTO trial_code_redemptions (code, team_id, redeemed_at) VALUES (?, ?, ?)`), code, teamID, now()); err != nil {
		return err
	}
	if _, err := tx.Exec(r.db.Rebind(`UPDATE teams SET trial_ends_at = ?, updated_at = ? WHERE id = ?`), trialEnd, now(), teamID); err != nil {
		return err
	}
	return tx.Commit()
}

// ----------------- Admin search -----------------

const adminTeamSelect = `SELECT t.id, t.name, t.owner_id, COALESCE(u.email, ''), t.created_at, t.trial_ends_at, t.subscription_status,
	t.billing_interval, t.current_period_end, t.cancel_at_period_end, t.comped, t.discount_label, t.stripe_customer_id, t.suspended_reason,
	(SELECT COUNT(*) FROM team_members m WHERE m.team_id = t.id),
	(SELECT COUNT(*) FROM links l WHERE l.team_id = t.id),
	(SELECT COUNT(*) FROM qr_codes q WHERE q.team_id = t.id)
	FROM teams t LEFT JOIN users u ON u.id = t.owner_id`

func scanAdminTeam(row interface{ Scan(...any) error }) (*models.AdminTeam, error) {
	var t models.AdminTeam
	var b models.TeamBilling
	if err := row.Scan(&t.ID, &t.Name, &t.OwnerID, &t.OwnerEmail, &t.CreatedAt, &b.TrialEndsAt, &b.SubscriptionStatus, &b.Interval,
		&b.CurrentPeriodEnd, &b.CancelAtPeriodEnd, &b.Comped, &t.DiscountLabel, &t.CustomerID, &t.SuspendedReason,
		&t.Members, &t.Links, &t.QRCodes); err != nil {
		return nil, notFound(err)
	}
	t.State, t.Interval, t.TrialEndsAt, t.PeriodEnd, t.Comped = b.State(time.Now()), b.Interval, b.TrialEndsAt, b.CurrentPeriodEnd, b.Comped
	return &t, nil
}

// AdminListTeams searches workspaces by name, id, owner email or Stripe customer.
func (r *Repository) AdminListTeams(q string, limit, offset int) ([]models.AdminTeam, error) {
	where, args := "1 = 1", []any{}
	if q = strings.TrimSpace(q); q != "" {
		l := likeArg(q)
		where = `(LOWER(t.name) LIKE ? ESCAPE '\' OR LOWER(u.email) LIKE ? ESCAPE '\' OR t.id = ? OR t.stripe_customer_id = ?)`
		args = append(args, l, l, q, q)
	}
	args = append(args, limit, offset)
	rows, err := r.query(adminTeamSelect+` WHERE `+where+` ORDER BY t.created_at DESC LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.AdminTeam{}
	for rows.Next() {
		t, err := scanAdminTeam(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *t)
	}
	return out, rows.Err()
}

func (r *Repository) AdminGetTeam(id string) (*models.AdminTeam, error) {
	return scanAdminTeam(r.queryRow(adminTeamSelect+` WHERE t.id = ?`, id))
}

const adminUserSelect = `SELECT u.id, u.email, TRIM(u.first_name || ' ' || u.last_name), u.created_at,
	(SELECT MAX(s.last_seen_at) FROM sessions s WHERE s.user_id = u.id), u.is_platform_admin, u.suspended_reason,
	(SELECT COUNT(*) FROM team_members m WHERE m.user_id = u.id)
	FROM users u`

func scanAdminUser(row interface{ Scan(...any) error }, driver string) (*models.AdminUser, error) {
	var u models.AdminUser
	var last any
	if err := row.Scan(&u.ID, &u.Email, &u.Name, &u.CreatedAt, &last, &u.IsPlatformAdmin, &u.SuspendedReason, &u.Workspaces); err != nil {
		return nil, notFound(err)
	}
	u.LastSeenAt = parseAnyTime(last)
	return &u, nil
}

// parseAnyTime handles aggregate results, which SQLite returns as text.
func parseAnyTime(v any) *time.Time {
	switch t := v.(type) {
	case time.Time:
		return &t
	case string:
		return parseTimeString(t)
	case []byte:
		return parseTimeString(string(t))
	}
	return nil
}

func parseTimeString(s string) *time.Time {
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02 15:04:05.999999999-07:00", "2006-01-02 15:04:05.999999999Z07:00", "2006-01-02 15:04:05"} {
		if t, err := time.Parse(layout, s); err == nil {
			return &t
		}
	}
	return nil
}

func (r *Repository) AdminListUsers(q string, limit, offset int) ([]models.AdminUser, error) {
	where, args := "1 = 1", []any{}
	if q = strings.TrimSpace(q); q != "" {
		l := likeArg(q)
		where = `(LOWER(u.email) LIKE ? ESCAPE '\' OR LOWER(u.first_name || ' ' || u.last_name) LIKE ? ESCAPE '\' OR u.id = ?)`
		args = append(args, l, l, q)
	}
	args = append(args, limit, offset)
	rows, err := r.query(adminUserSelect+` WHERE `+where+` ORDER BY u.created_at DESC LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.AdminUser{}
	for rows.Next() {
		u, err := scanAdminUser(rows, r.db.DriverName)
		if err != nil {
			return nil, err
		}
		out = append(out, *u)
	}
	return out, rows.Err()
}

func (r *Repository) AdminGetUser(id string) (*models.AdminUser, error) {
	return scanAdminUser(r.queryRow(adminUserSelect+` WHERE u.id = ?`, id), r.db.DriverName)
}

// AdminSearchAssets finds links and QR codes by destination, domain, slug or code.
func (r *Repository) AdminSearchAssets(q string, limit int) ([]models.AdminAsset, error) {
	q = strings.TrimSpace(q)
	l := likeArg(q)
	out := []models.AdminAsset{}
	rows, err := r.query(`SELECT l.id, l.team_id, COALESCE(t.name, ''), l.slug, l.target_url, l.moderation_reason, l.click_count, l.created_at
		FROM links l LEFT JOIN teams t ON t.id = l.team_id
		WHERE LOWER(l.target_url) LIKE ? ESCAPE '\' OR l.slug = ? OR l.id = ? OR l.team_id = ?
		ORDER BY l.created_at DESC LIMIT ?`, l, q, q, q, limit)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		a := models.AdminAsset{Type: "link"}
		if err := rows.Scan(&a.ID, &a.TeamID, &a.TeamName, &a.Label, &a.Destination, &a.ModerationReason, &a.Count, &a.CreatedAt); err != nil {
			rows.Close()
			return nil, err
		}
		out = append(out, a)
	}
	rows.Close()
	// QR codes that point at their own short link are covered by the link row.
	rows, err = r.query(`SELECT q.id, q.team_id, COALESCE(t.name, ''), q.title, COALESCE(q.code, ''), q.payload, q.moderation_reason, q.scan_count, q.created_at
		FROM qr_codes q LEFT JOIN teams t ON t.id = q.team_id
		WHERE (q.link_id IS NULL AND LOWER(q.payload) LIKE ? ESCAPE '\') OR q.code = ? OR q.id = ? OR q.team_id = ?
		ORDER BY q.created_at DESC LIMIT ?`, l, q, q, q, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		a := models.AdminAsset{Type: "qr"}
		var code string
		if err := rows.Scan(&a.ID, &a.TeamID, &a.TeamName, &a.Label, &code, &a.Destination, &a.ModerationReason, &a.Count, &a.CreatedAt); err != nil {
			return nil, err
		}
		a.PublicURL = code // the service turns this into a full scan URL
		out = append(out, a)
	}
	return out, rows.Err()
}

// ----------------- Platform metrics -----------------

// PlatformCounts holds point-in-time totals.
type PlatformCounts struct {
	Users, Teams, Links, QRCodes int64
}

func (r *Repository) PlatformCounts() (PlatformCounts, error) {
	var c PlatformCounts
	err := r.queryRow(`SELECT (SELECT COUNT(*) FROM users), (SELECT COUNT(*) FROM teams), (SELECT COUNT(*) FROM links), (SELECT COUNT(*) FROM qr_codes)`).
		Scan(&c.Users, &c.Teams, &c.Links, &c.QRCodes)
	return c, err
}

// AllTeamBilling returns every workspace's billing state with its creation time.
func (r *Repository) AllTeamBilling() ([]models.Team, error) {
	rows, err := r.query(`SELECT ` + teamCols + ` FROM teams`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.Team{}
	for rows.Next() {
		t, err := scanTeam(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *t)
	}
	return out, rows.Err()
}

// DailyCounts buckets rows of a table by UTC day since a time.
func (r *Repository) DailyCounts(table, where string, since time.Time) ([]models.DayCount, error) {
	if where == "" {
		where = "1 = 1"
	}
	rows, err := r.query(`SELECT `+r.dayExpr()+` AS d, COUNT(*) FROM `+table+` WHERE created_at >= ? AND `+where+` GROUP BY 1 ORDER BY 1`, since.UTC())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.DayCount{}
	for rows.Next() {
		var d models.DayCount
		if err := rows.Scan(&d.Day, &d.Count); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// ----------------- Email log -----------------

// MarkEmailSent records a scheduled email; false means it was already sent.
func (r *Repository) MarkEmailSent(kind, ref string) (bool, error) {
	res, err := r.exec(`INSERT INTO email_log (kind, ref, sent_at) VALUES (?, ?, ?) ON CONFLICT (kind, ref) DO NOTHING`, kind, ref, now())
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

// UsersWithPreference lists users who turned on a boolean notification preference.
func (r *Repository) UsersWithPreference(key string) ([]models.User, error) {
	rows, err := r.query(`SELECT `+userSelectCols+` FROM users WHERE suspended_reason = '' AND preferences LIKE ?`, `%"`+key+`":true%`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.User{}
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *u)
	}
	return out, rows.Err()
}

func (r *Repository) ListPlatformAdmins() ([]models.User, error) {
	rows, err := r.query(`SELECT `+userSelectCols+` FROM users WHERE is_platform_admin = ? AND suspended_reason = ''`, true)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.User{}
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *u)
	}
	return out, rows.Err()
}
