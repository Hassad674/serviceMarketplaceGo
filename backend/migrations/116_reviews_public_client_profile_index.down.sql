-- 116_reviews_public_client_profile_index.down.sql
--
-- Drops the provider→client public profile index added by the up migration.

DROP INDEX IF EXISTS idx_reviews_public_client_profile;
