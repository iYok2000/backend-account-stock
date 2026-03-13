-- Ensure all users have company_id and default to YPC (including legacy/null/root)

UPDATE users SET company_id = 'YPC' WHERE company_id IS NULL OR company_id = '';

ALTER TABLE users ALTER COLUMN company_id SET DEFAULT 'YPC';

-- Root (if exists as row) should stay with company_id set; no further change needed
