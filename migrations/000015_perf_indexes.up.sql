-- Performance indexes for high-traffic query paths (OWASP A04 / query optimisation)
-- import_sku_row: dashboard, inventory list, and analytics all filter by (shop_id, date)
CREATE INDEX IF NOT EXISTS idx_import_sku_row_shop_date
    ON import_sku_row (shop_id, date DESC)
    WHERE deleted_at IS NULL;

-- Covering index for top-SKU queries (GROUP BY sku_id ORDER BY revenue DESC)
CREATE INDEX IF NOT EXISTS idx_import_sku_row_shop_sku
    ON import_sku_row (shop_id, sku_id)
    WHERE deleted_at IS NULL;


