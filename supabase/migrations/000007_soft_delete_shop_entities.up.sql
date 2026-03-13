-- Add deleted_at for soft delete on shops and import_sku_row

ALTER TABLE shops ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;
CREATE INDEX IF NOT EXISTS ix_shops_deleted_at ON shops (deleted_at);

ALTER TABLE import_sku_row ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;
CREATE INDEX IF NOT EXISTS ix_import_sku_row_deleted_at ON import_sku_row (deleted_at);
