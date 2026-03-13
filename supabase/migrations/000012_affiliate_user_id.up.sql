-- Add user_id to affiliate_sku_row to bind data to the affiliate user who imported it.

ALTER TABLE affiliate_sku_row
    ADD COLUMN IF NOT EXISTS user_id VARCHAR(36) REFERENCES users(id) ON DELETE CASCADE;

-- Update unique key to include user_id (per-user scope within a company)
DROP INDEX IF EXISTS ux_affiliate_sku_row_company_order_sku;
CREATE UNIQUE INDEX IF NOT EXISTS ux_affiliate_sku_row_company_user_order_sku
    ON affiliate_sku_row (company_id, user_id, order_id, sku_id);

CREATE INDEX IF NOT EXISTS ix_affiliate_sku_row_company_user_orderdate
    ON affiliate_sku_row (company_id, user_id, order_date);
