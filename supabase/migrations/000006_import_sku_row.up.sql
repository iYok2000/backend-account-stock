-- Import SKU per day (source of truth for inventory & dashboard)

CREATE TABLE IF NOT EXISTS import_sku_row (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    -- shops.id เป็น VARCHAR(36) จึงต้องใช้ชนิดเดียวกันเพื่อทำ FK ได้
    shop_id VARCHAR(36) NOT NULL REFERENCES shops(id) ON DELETE CASCADE,
    date DATE NOT NULL,
    sku_id TEXT NOT NULL,
    seller_sku TEXT,
    product_name TEXT,
    variation TEXT,
    quantity NUMERIC(18,4) NOT NULL DEFAULT 0,
    revenue NUMERIC(18,2) NOT NULL DEFAULT 0,
    deductions NUMERIC(18,2) NOT NULL DEFAULT 0,
    refund NUMERIC(18,2) NOT NULL DEFAULT 0,
    net NUMERIC(18,2) NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Upsert key: shop_id + date + sku_id
CREATE UNIQUE INDEX IF NOT EXISTS ux_import_sku_row_shop_date_sku
    ON import_sku_row (shop_id, date, sku_id);

CREATE INDEX IF NOT EXISTS ix_import_sku_row_shop_date
    ON import_sku_row (shop_id, date);
