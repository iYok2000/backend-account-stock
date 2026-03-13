-- Enrich affiliate_sku_row with company scope + order details for dashboard/analytics

ALTER TABLE affiliate_sku_row
    ADD COLUMN IF NOT EXISTS company_id TEXT NOT NULL DEFAULT 'YPC', -- scope ตาม company (affiliate)
    ADD COLUMN IF NOT EXISTS order_id TEXT,
    ADD COLUMN IF NOT EXISTS settlement_status TEXT,
    ADD COLUMN IF NOT EXISTS commission_amount NUMERIC(18,2) NOT NULL DEFAULT 0, -- Total final earned amount (ได้จริง)
    ADD COLUMN IF NOT EXISTS standard_commission NUMERIC(18,2) NOT NULL DEFAULT 0, -- Est. standard commission
    ADD COLUMN IF NOT EXISTS shop_ads_commission NUMERIC(18,2) NOT NULL DEFAULT 0, -- Est. Shop Ads commission
    ADD COLUMN IF NOT EXISTS commission_base NUMERIC(18,2) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS commission_rate NUMERIC(18,4) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS content_type TEXT,
    ADD COLUMN IF NOT EXISTS order_date DATE,
    ADD COLUMN IF NOT EXISTS settlement_date DATE;

-- Unique key ใหม่: company + order + sku
DROP INDEX IF EXISTS ux_affiliate_sku_row_shop_date_sku;
CREATE UNIQUE INDEX IF NOT EXISTS ux_affiliate_sku_row_company_order_sku
    ON affiliate_sku_row (company_id, order_id, sku_id);

-- Index ตามช่วงวันที่
CREATE INDEX IF NOT EXISTS ix_affiliate_sku_row_company_orderdate
    ON affiliate_sku_row (company_id, order_date);
