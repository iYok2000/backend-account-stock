-- Affiliate SKU metrics (แยกจาก inventory ร้านค้า)
-- ใช้บันทึกผลหลังประมวลผลไฟล์ affiliate

CREATE TABLE IF NOT EXISTS affiliate_sku_row (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    affiliate_shop TEXT NOT NULL, -- ชื่อร้าน/แอคเคานต์ affiliate
    date DATE NOT NULL,
    sku_id TEXT NOT NULL,
    product_name TEXT,
    items_sold NUMERIC(18,4) NOT NULL DEFAULT 0,
    gmv NUMERIC(18,2) NOT NULL DEFAULT 0,
    commission NUMERIC(18,2) NOT NULL DEFAULT 0,
    ineligible_amount NUMERIC(18,2) NOT NULL DEFAULT 0, -- ค่าคอมที่หายไป (Ineligible)
    commission_rate NUMERIC(18,4) NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Dedup per affiliate_shop + date + sku_id
CREATE UNIQUE INDEX IF NOT EXISTS ux_affiliate_sku_row_shop_date_sku
    ON affiliate_sku_row (affiliate_shop, date, sku_id);

CREATE INDEX IF NOT EXISTS ix_affiliate_sku_row_shop_date
    ON affiliate_sku_row (affiliate_shop, date);
