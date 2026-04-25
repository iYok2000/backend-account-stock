-- Reverse: restore affiliate_sku_row.date to no default
ALTER TABLE affiliate_sku_row
    ALTER COLUMN date DROP DEFAULT;
