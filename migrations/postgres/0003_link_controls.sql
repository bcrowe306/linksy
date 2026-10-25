-- 0003: link fallback destinations, and whether a QR code owns its destination link
-- (QR codes can now attach to an existing short link without taking ownership of it).
ALTER TABLE links ADD COLUMN fallback_url TEXT NOT NULL DEFAULT '';
ALTER TABLE qr_codes ADD COLUMN owns_link BOOLEAN NOT NULL DEFAULT TRUE;
