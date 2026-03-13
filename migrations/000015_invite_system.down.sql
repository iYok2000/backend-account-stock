-- Rollback invite system

DROP TABLE IF EXISTS tier_history;
DROP TABLE IF EXISTS invite_codes;
DROP TABLE IF EXISTS system_config;

-- Remove tier tracking columns from users
ALTER TABLE users DROP COLUMN IF EXISTS tier_started_at;
ALTER TABLE users DROP COLUMN IF EXISTS tier_expires_at;
ALTER TABLE users DROP COLUMN IF EXISTS invite_code_used;
ALTER TABLE users DROP COLUMN IF EXISTS invite_slots;
