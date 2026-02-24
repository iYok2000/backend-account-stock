-- Rollback: remove the column added in 000002_add_user_phone.up.sql
ALTER TABLE users DROP COLUMN IF EXISTS phone;
