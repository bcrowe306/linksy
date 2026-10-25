-- 0005: per-workspace subscriptions. The 7-day trial is tracked by Lynqr (no card
-- needed); paid state is synced from Stripe webhooks.
ALTER TABLE teams ADD COLUMN trial_ends_at TIMESTAMPTZ;
ALTER TABLE teams ADD COLUMN stripe_customer_id TEXT NOT NULL DEFAULT '';
ALTER TABLE teams ADD COLUMN stripe_subscription_id TEXT NOT NULL DEFAULT '';
ALTER TABLE teams ADD COLUMN subscription_status TEXT NOT NULL DEFAULT '';
ALTER TABLE teams ADD COLUMN billing_interval TEXT NOT NULL DEFAULT '';
ALTER TABLE teams ADD COLUMN current_period_end TIMESTAMPTZ;
ALTER TABLE teams ADD COLUMN cancel_at_period_end BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE teams ADD COLUMN discount_code TEXT NOT NULL DEFAULT '';
ALTER TABLE teams ADD COLUMN discount_label TEXT NOT NULL DEFAULT '';
ALTER TABLE teams ADD COLUMN comped BOOLEAN NOT NULL DEFAULT FALSE;
UPDATE teams SET trial_ends_at = created_at + INTERVAL '7 days';
CREATE INDEX idx_teams_stripe_customer ON teams(stripe_customer_id);

CREATE TABLE stripe_events (
    id TEXT PRIMARY KEY,
    type TEXT NOT NULL,
    received_at TIMESTAMPTZ NOT NULL
);
