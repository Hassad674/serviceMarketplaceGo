-- 081_create_skills_tables.down.sql
--
-- Reverts migration 081: drops the profile_skills and skills_catalog
-- tables and all their indexes and triggers. Any declared skill data
-- is lost — there is no other place where this data lives.
--
-- pg_trgm is intentionally NOT dropped: the extension may be used by
-- other features (present or future), and dropping it here would break
-- them unrelated to the skills rollback.

BEGIN;

DROP TABLE IF EXISTS profile_skills;
DROP TABLE IF EXISTS skills_catalog;
-- pg_trgm extension NOT dropped — may be used by other features.

COMMIT;
