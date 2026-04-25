-- Partial indexes for tables created in 000016

-- affiliate_sku_row: dashboard KPIs filter by (user_id, order_date)
CREATE INDEX IF NOT EXISTS idx_affiliate_sku_row_user_date
    ON affiliate_sku_row (user_id, order_date DESC)
    WHERE deleted_at IS NULL;

-- invite_codes: lookup by code is on the hot path (validate + use)
CREATE INDEX IF NOT EXISTS idx_invite_codes_code
    ON invite_codes (code)
    WHERE deleted_at IS NULL;
