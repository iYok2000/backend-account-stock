DROP TABLE IF EXISTS system_config;
DROP TABLE IF EXISTS tier_history;
DROP TABLE IF EXISTS invite_codes;

ALTER TABLE affiliate_sku_row DROP COLUMN IF EXISTS deleted_at;

ALTER TABLE users DROP COLUMN IF EXISTS invite_slots;
ALTER TABLE users DROP COLUMN IF EXISTS invite_code_used;
ALTER TABLE users DROP COLUMN IF EXISTS tier_expires_at;
ALTER TABLE users DROP COLUMN IF EXISTS tier_started_at;
