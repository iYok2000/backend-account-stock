-- Example: add optional column to existing table (table already has data).
-- Use ALTER TABLE ADD COLUMN; nullable or with DEFAULT so existing rows are valid.

ALTER TABLE users ADD COLUMN IF NOT EXISTS phone VARCHAR(32);
