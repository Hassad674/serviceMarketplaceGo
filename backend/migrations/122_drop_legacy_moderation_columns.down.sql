-- Rollback for Phase 7 cleanup. Recreates the legacy moderation_*
-- columns + indexes and rebuilds them from moderation_results so the
-- pre-Phase-7 code paths can be reactivated without data loss.
--
-- Restoration logic:
--   - reviewed_at IS NULL means "no admin override yet" → use the
--     latest auto verdict (status, score, labels).
--   - reviewed_at IS NOT NULL means "admin actioned the row" → use
--     the post-review status (Approve/Hide/Restore wrote it back).
-- Either way, the column reflects what the admin queue shows.

ALTER TABLE messages
    ADD COLUMN IF NOT EXISTS moderation_status TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS moderation_score  REAL NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS moderation_labels JSONB;

ALTER TABLE reviews
    ADD COLUMN IF NOT EXISTS moderation_status TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS moderation_score  REAL NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS moderation_labels JSONB;

CREATE INDEX IF NOT EXISTS idx_messages_moderation_status
    ON messages(moderation_status) WHERE moderation_status != '';

CREATE INDEX IF NOT EXISTS idx_reviews_moderation_status
    ON reviews(moderation_status) WHERE moderation_status != '';

-- Repopulate from moderation_results.
UPDATE messages m
   SET moderation_status = mr.status,
       moderation_score  = mr.score,
       moderation_labels = mr.labels
  FROM moderation_results mr
 WHERE mr.content_type = 'message'
   AND mr.content_id = m.id;

UPDATE reviews rv
   SET moderation_status = mr.status,
       moderation_score  = mr.score,
       moderation_labels = mr.labels
  FROM moderation_results mr
 WHERE mr.content_type = 'review'
   AND mr.content_id = rv.id;
