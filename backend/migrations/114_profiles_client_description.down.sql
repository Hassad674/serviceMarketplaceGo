-- 114_profiles_client_description.down.sql
--
-- Drops the client_description column. Safe to run after rollback — every
-- downstream read path uses COALESCE so missing data defaults to empty.

ALTER TABLE profiles
    DROP COLUMN IF EXISTS client_description;
