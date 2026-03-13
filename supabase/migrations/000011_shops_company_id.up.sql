-- Add company_id to shops for tenant scoping (1 company : N shops).
-- Backfill existing rows with default root company (YPC) so Root and other handlers work.

-- Ensure default company exists
INSERT INTO companies (id, name)
VALUES ('YPC', 'YPC Affiliate')
ON CONFLICT (id) DO NOTHING;

ALTER TABLE shops ADD COLUMN IF NOT EXISTS company_id VARCHAR(36);
UPDATE shops SET company_id = 'YPC' WHERE company_id IS NULL OR company_id = '';
ALTER TABLE shops ALTER COLUMN company_id SET NOT NULL;

CREATE INDEX IF NOT EXISTS idx_shops_company_id ON shops (company_id);

-- Optional FK (skip if not supported by existing data)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints
        WHERE constraint_name = 'fk_shops_company'
          AND table_name = 'shops'
    ) THEN
        ALTER TABLE shops
            ADD CONSTRAINT fk_shops_company
            FOREIGN KEY (company_id) REFERENCES companies(id) ON DELETE CASCADE;
    END IF;
END $$;
