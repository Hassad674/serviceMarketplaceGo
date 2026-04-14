-- 083_profile_tier1_completion.down.sql
--
-- Reverts migration 083: drops profile_pricing and its index/trigger,
-- then strips the new columns (location, languages, availability) from
-- profiles along with every index that depended on them.
--
-- Any declared pricing / location / language / availability data is
-- lost on downgrade — there is no other place where this data lives.

BEGIN;

-- ---- profile_pricing (self-contained, drop first) ----
DROP TABLE IF EXISTS profile_pricing;

-- ---- profiles indexes ----
DROP INDEX IF EXISTS idx_profiles_availability;
DROP INDEX IF EXISTS idx_profiles_lang_conv_gin;
DROP INDEX IF EXISTS idx_profiles_lang_pro_gin;
DROP INDEX IF EXISTS idx_profiles_work_mode_gin;
DROP INDEX IF EXISTS idx_profiles_country_city_ne_empty;

-- ---- availability columns ----
ALTER TABLE profiles DROP COLUMN IF EXISTS referrer_availability_status;
ALTER TABLE profiles DROP COLUMN IF EXISTS availability_status;

-- ---- languages columns ----
ALTER TABLE profiles DROP COLUMN IF EXISTS languages_conversational;
ALTER TABLE profiles DROP COLUMN IF EXISTS languages_professional;

-- ---- location columns ----
ALTER TABLE profiles DROP COLUMN IF EXISTS travel_radius_km;
ALTER TABLE profiles DROP COLUMN IF EXISTS work_mode;
ALTER TABLE profiles DROP COLUMN IF EXISTS longitude;
ALTER TABLE profiles DROP COLUMN IF EXISTS latitude;
ALTER TABLE profiles DROP COLUMN IF EXISTS country_code;
ALTER TABLE profiles DROP COLUMN IF EXISTS city;

COMMIT;
