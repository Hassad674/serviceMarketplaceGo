-- 099_create_freelance_pricing.down.sql

BEGIN;

DROP TRIGGER IF EXISTS freelance_pricing_updated_at ON freelance_pricing;
DROP INDEX IF EXISTS idx_freelance_pricing_type;
DROP TABLE IF EXISTS freelance_pricing;

COMMIT;
