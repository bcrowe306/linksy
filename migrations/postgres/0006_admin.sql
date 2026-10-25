-- 0006: platform administration. Platform admins are separate from workspace
-- roles; admin actions are recorded in an append-only platform audit log.
ALTER TABLE users ADD COLUMN is_platform_admin BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE users ADD COLUMN suspended_reason TEXT NOT NULL DEFAULT '';
ALTER TABLE sessions ADD COLUMN admin_until TIMESTAMPTZ;
ALTER TABLE teams ADD COLUMN suspended_reason TEXT NOT NULL DEFAULT '';
ALTER TABLE links ADD COLUMN moderation_reason TEXT NOT NULL DEFAULT '';
ALTER TABLE qr_codes ADD COLUMN moderation_reason TEXT NOT NULL DEFAULT '';

-- One pending email code per session for admin re-authentication.
CREATE TABLE admin_challenges (
    session_id TEXT PRIMARY KEY REFERENCES sessions(id) ON DELETE CASCADE,
    code_hash TEXT NOT NULL,
    attempts INTEGER NOT NULL DEFAULT 0,
    expires_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE platform_audit (
    id TEXT PRIMARY KEY,
    actor_id TEXT NOT NULL,
    actor_email TEXT NOT NULL DEFAULT '',
    action TEXT NOT NULL,
    target_type TEXT NOT NULL DEFAULT '',
    target_id TEXT NOT NULL DEFAULT '',
    target_label TEXT NOT NULL DEFAULT '',
    metadata TEXT NOT NULL DEFAULT '{}',
    ip TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX idx_platform_audit_created ON platform_audit(created_at);

CREATE TABLE blocked_domains (
    domain TEXT PRIMARY KEY,
    reason TEXT NOT NULL DEFAULT '',
    created_by TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE abuse_reports (
    id TEXT PRIMARY KEY,
    target_type TEXT NOT NULL,
    target_id TEXT NOT NULL,
    target_url TEXT NOT NULL DEFAULT '',
    reason TEXT NOT NULL,
    details TEXT NOT NULL DEFAULT '',
    reporter_email TEXT NOT NULL DEFAULT '',
    reporter_ip TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'open',
    resolved_by TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL,
    resolved_at TIMESTAMPTZ
);
CREATE INDEX idx_abuse_reports_status ON abuse_reports(status, created_at);

-- Lynqr-side codes that lengthen a workspace's free trial.
CREATE TABLE trial_codes (
    code TEXT PRIMARY KEY,
    days INTEGER NOT NULL,
    max_redemptions INTEGER NOT NULL DEFAULT 0,
    redemptions INTEGER NOT NULL DEFAULT 0,
    expires_at TIMESTAMPTZ,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    note TEXT NOT NULL DEFAULT '',
    created_by TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE trial_code_redemptions (
    code TEXT NOT NULL REFERENCES trial_codes(code) ON DELETE CASCADE,
    team_id TEXT NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    redeemed_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (code, team_id)
);
