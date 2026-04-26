-- Phase 7 — drop the per-table moderation_* columns now that
-- moderation_results is the single source of truth. Migration 120
-- backfilled every flagged/hidden/deleted row into the new table, and
-- every reader (admin queue, public profile, search ranking, admin
-- conversation detail, seed-search) now joins moderation_results
-- instead of reading the source-table columns directly.
--
-- Rolling back this migration restores the columns AND repopulates
-- them from moderation_results so the legacy code paths can be
-- re-enabled in an emergency. See the .down.sql for the recovery
-- path.

DROP INDEX IF EXISTS idx_messages_moderation_status;
DROP INDEX IF EXISTS idx_reviews_moderation_status;

ALTER TABLE messages DROP COLUMN IF EXISTS moderation_status;
ALTER TABLE messages DROP COLUMN IF EXISTS moderation_score;
ALTER TABLE messages DROP COLUMN IF EXISTS moderation_labels;

ALTER TABLE reviews DROP COLUMN IF EXISTS moderation_status;
ALTER TABLE reviews DROP COLUMN IF EXISTS moderation_score;
ALTER TABLE reviews DROP COLUMN IF EXISTS moderation_labels;
