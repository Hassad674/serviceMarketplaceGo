-- Rollback for Phase 2 moderation extension. Dropping the table is safe
-- because the legacy columns on messages/reviews still exist (they get
-- removed only by migration 122 in Phase 7). If we rolled back AFTER
-- Phase 7, restoring data would be impossible — that is why the legacy
-- column drop is sequenced last.

DROP INDEX IF EXISTS idx_moderation_results_author;
DROP INDEX IF EXISTS idx_moderation_results_content;
DROP INDEX IF EXISTS idx_moderation_results_pending;
DROP TABLE IF EXISTS moderation_results;
