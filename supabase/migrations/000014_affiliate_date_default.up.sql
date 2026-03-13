-- Ensure affiliate_sku_row.date always has a value (prevents NOT NULL errors on import)
ALTER TABLE affiliate_sku_row
  ALTER COLUMN date SET DEFAULT CURRENT_DATE;

UPDATE affiliate_sku_row
  SET date = CURRENT_DATE
  WHERE date IS NULL;
