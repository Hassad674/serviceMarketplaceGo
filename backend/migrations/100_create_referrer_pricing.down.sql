-- 100_create_referrer_pricing.down.sql

BEGIN;

DROP TRIGGER IF EXISTS referrer_pricing_updated_at ON referrer_pricing;
DROP INDEX IF EXISTS idx_referrer_pricing_type;
DROP TABLE IF EXISTS referrer_pricing;

COMMIT;
