-- 096_add_organization_shared_profile_columns.down.sql
--
-- Reverts migration 096: drops the indexes and shared-profile columns
-- from organizations. Lossless because the data was backfilled from
-- profiles — the original profiles columns are still in place.

BEGIN;

DROP INDEX IF EXISTS idx_organizations_lang_conv_gin;
DROP INDEX IF EXISTS idx_organizations_lang_pro_gin;
DROP INDEX IF EXISTS idx_organizations_work_mode_gin;
DROP INDEX IF EXISTS idx_organizations_country_city_ne_empty;

ALTER TABLE organizations DROP COLUMN IF EXISTS languages_conversational;
ALTER TABLE organizations DROP COLUMN IF EXISTS languages_professional;
ALTER TABLE organizations DROP COLUMN IF EXISTS travel_radius_km;
ALTER TABLE organizations DROP COLUMN IF EXISTS work_mode;
ALTER TABLE organizations DROP COLUMN IF EXISTS longitude;
ALTER TABLE organizations DROP COLUMN IF EXISTS latitude;
ALTER TABLE organizations DROP COLUMN IF EXISTS country_code;
ALTER TABLE organizations DROP COLUMN IF EXISTS city;
ALTER TABLE organizations DROP COLUMN IF EXISTS photo_url;

COMMIT;
