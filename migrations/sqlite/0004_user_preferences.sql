-- 0004: per-user notification preferences (JSON document).
ALTER TABLE users ADD COLUMN preferences TEXT NOT NULL DEFAULT '{}';
