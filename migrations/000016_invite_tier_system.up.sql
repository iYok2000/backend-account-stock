-- Create invite_codes, tier_history, system_config tables and add user tier columns.

-- 1. Add tier-tracking columns to users
ALTER TABLE users ADD COLUMN IF NOT EXISTS tier_started_at  TIMESTAMPTZ;
ALTER TABLE users ADD COLUMN IF NOT EXISTS tier_expires_at  TIMESTAMPTZ;
ALTER TABLE users ADD COLUMN IF NOT EXISTS invite_code_used VARCHAR(36);
ALTER TABLE users ADD COLUMN IF NOT EXISTS invite_slots     INT NOT NULL DEFAULT 0;

-- 2. Add deleted_at to affiliate_sku_row (needed for partial indexes in 000017)
ALTER TABLE affiliate_sku_row ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;
CREATE INDEX IF NOT EXISTS ix_affiliate_sku_row_deleted_at ON affiliate_sku_row (deleted_at);

-- 3. invite_codes table
CREATE TABLE IF NOT EXISTS invite_codes (
    id               VARCHAR(36)  PRIMARY KEY,
    code             VARCHAR(32)  NOT NULL,
    grant_tier       VARCHAR(16)  NOT NULL DEFAULT 'free',
    tier_duration_days INT,
    max_uses         INT          NOT NULL DEFAULT 1,
    used_count       INT          NOT NULL DEFAULT 0,
    is_active        BOOLEAN      NOT NULL DEFAULT TRUE,
    expires_at       TIMESTAMPTZ,
    note             TEXT,
    created_by       VARCHAR(36),
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at       TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS ux_invite_codes_code ON invite_codes (code) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS ix_invite_codes_deleted_at ON invite_codes (deleted_at);

-- 4. tier_history table
CREATE TABLE IF NOT EXISTS tier_history (
    id             VARCHAR(36) PRIMARY KEY,
    user_id        VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    old_tier       VARCHAR(16),
    new_tier       VARCHAR(16) NOT NULL,
    reason         VARCHAR(64),
    changed_by     VARCHAR(36),
    invite_code_id VARCHAR(36),
    started_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at     TIMESTAMPTZ,
    note           TEXT,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS ix_tier_history_user_id ON tier_history (user_id);

-- 5. system_config table
CREATE TABLE IF NOT EXISTS system_config (
    id          VARCHAR(36) PRIMARY KEY,
    key         VARCHAR(64) NOT NULL,
    value       TEXT        NOT NULL,
    description TEXT,
    updated_by  VARCHAR(36),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS ux_system_config_key ON system_config (key);
