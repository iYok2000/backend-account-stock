-- Revert: Remove default date value from affiliate_sku_row
ALTER TABLE affiliate_sku_row
  ALTER COLUMN date DROP DEFAULT;
