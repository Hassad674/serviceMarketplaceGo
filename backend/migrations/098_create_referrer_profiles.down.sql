-- 098_create_referrer_profiles.down.sql

BEGIN;

DROP TRIGGER IF EXISTS referrer_profiles_updated_at ON referrer_profiles;
DROP INDEX IF EXISTS idx_referrer_profiles_expertise_domains_gin;
DROP INDEX IF EXISTS idx_referrer_profiles_availability;
DROP TABLE IF EXISTS referrer_profiles;

COMMIT;
