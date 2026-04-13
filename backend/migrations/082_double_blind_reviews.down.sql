-- 082_double_blind_reviews.down.sql
--
-- Drops the double-blind review columns. Historic rows lose their reveal
-- timestamp, but the column itself simply goes away — we do NOT attempt to
-- restore the previous state of published_at.

BEGIN;

DROP INDEX IF EXISTS idx_reviews_pending;
DROP INDEX IF EXISTS idx_reviews_public_profile;

ALTER TABLE reviews DROP CONSTRAINT IF EXISTS reviews_provider_side_no_subcriteria;

ALTER TABLE reviews DROP COLUMN IF EXISTS published_at;
ALTER TABLE reviews DROP COLUMN IF EXISTS side;

COMMIT;
