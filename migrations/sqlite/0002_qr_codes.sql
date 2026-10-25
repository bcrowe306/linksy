-- 0002: QR codes get their own short scan code (/q/{code}) so scans are tracked
-- separately from link clicks, plus an explicit dynamic flag.
ALTER TABLE qr_codes ADD COLUMN code TEXT;
ALTER TABLE qr_codes ADD COLUMN is_dynamic BOOLEAN NOT NULL DEFAULT FALSE;
CREATE UNIQUE INDEX idx_qr_codes_code ON qr_codes(code);
UPDATE qr_codes SET is_dynamic = TRUE WHERE link_id IS NOT NULL OR document_id IS NOT NULL;
