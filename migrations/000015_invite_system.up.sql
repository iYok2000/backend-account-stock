-- Invite system for Tier management (inspired by Congrats-seller but adapted for account-stock)
-- Supports FREE/STARTER/PRO/ENTERPRISE tiers with invite codes

-- Invite codes table
CREATE TABLE IF NOT EXISTS invite_codes (
    id VARCHAR(36) PRIMARY KEY,
    code VARCHAR(32) NOT NULL UNIQUE,
    grant_tier VARCHAR(16) NOT NULL, -- Tier to grant when used (FREE/STARTER/PRO/ENTERPRISE)
    tier_duration_days INT, -- NULL = unlimited, else duration in days
    max_uses INT NOT NULL DEFAULT 1,
    used_count INT NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    expires_at TIMESTAMPTZ, -- Code expiry (NULL = never expires)
    note TEXT, -- Admin note about this code
    created_by VARCHAR(36), -- User ID who created this code (Root or SuperAdmin)
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_invite_codes_code ON invite_codes (code) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_invite_codes_is_active ON invite_codes (is_active) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_invite_codes_deleted_at ON invite_codes (deleted_at);

-- Tier change history
CREATE TABLE IF NOT EXISTS tier_history (
    id VARCHAR(36) PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL REFERENCES users(id),
    old_tier VARCHAR(16),
    new_tier VARCHAR(16) NOT NULL,
    reason VARCHAR(64), -- 'invite_code', 'admin_grant', 'upgrade', 'downgrade', 'expired'
    changed_by VARCHAR(36), -- User ID who made the change (admin or system)
    invite_code_id VARCHAR(36), -- FK to invite_codes if reason = 'invite_code'
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ, -- NULL = never expires
    note TEXT, -- Optional note
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_tier_history_user_id ON tier_history (user_id);
CREATE INDEX IF NOT EXISTS idx_tier_history_created_at ON tier_history (created_at);

-- Add tier tracking fields to users table (if not exists)
ALTER TABLE users ADD COLUMN IF NOT EXISTS tier_started_at TIMESTAMPTZ;
ALTER TABLE users ADD COLUMN IF NOT EXISTS tier_expires_at TIMESTAMPTZ;
ALTER TABLE users ADD COLUMN IF NOT EXISTS invite_code_used VARCHAR(32); -- Track which code was used
ALTER TABLE users ADD COLUMN IF NOT EXISTS invite_slots INT NOT NULL DEFAULT 0; -- Future: user can invite others

-- System config table for global settings (e.g., require_invite_code toggle)
CREATE TABLE IF NOT EXISTS system_config (
    id VARCHAR(36) PRIMARY KEY,
    key VARCHAR(64) NOT NULL UNIQUE,
    value TEXT NOT NULL,
    description TEXT,
    updated_by VARCHAR(36), -- User ID who last updated
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_system_config_key ON system_config (key);

-- Insert default config: require invite code (default: false)
INSERT INTO system_config (id, key, value, description)
VALUES (
    gen_random_uuid()::VARCHAR,
    'require_invite_code',
    'false',
    'Whether new user registration requires an invite code'
)
ON CONFLICT (key) DO NOTHING;
