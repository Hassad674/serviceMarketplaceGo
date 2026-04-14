-- 097_create_freelance_profiles.down.sql

BEGIN;

DROP TRIGGER IF EXISTS freelance_profiles_updated_at ON freelance_profiles;
DROP INDEX IF EXISTS idx_freelance_profiles_expertise_domains_gin;
DROP INDEX IF EXISTS idx_freelance_profiles_availability;
DROP TABLE IF EXISTS freelance_profiles;

COMMIT;
