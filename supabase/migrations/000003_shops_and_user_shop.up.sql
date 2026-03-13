-- Shops and user shop_id / password (SHOPS_AND_ROLES_SPEC).
-- 1 user : 1 shop; email unique; Root has shop_id NULL.

CREATE TABLE IF NOT EXISTS shops (
    id VARCHAR(36) PRIMARY KEY,
    name VARCHAR(256) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_shops_deleted_at ON shops (deleted_at);

-- Add shop_id (nullable for Root) and password_hash to users.
ALTER TABLE users ADD COLUMN IF NOT EXISTS shop_id VARCHAR(36) REFERENCES shops(id);
ALTER TABLE users ADD COLUMN IF NOT EXISTS password_hash VARCHAR(256);

CREATE INDEX IF NOT EXISTS idx_users_shop_id ON users (shop_id);
