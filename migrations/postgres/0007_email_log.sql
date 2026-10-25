-- 0007: one row per scheduled email so reminders and digests are sent once.
CREATE TABLE email_log (
    kind TEXT NOT NULL,
    ref TEXT NOT NULL,
    sent_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (kind, ref)
);
