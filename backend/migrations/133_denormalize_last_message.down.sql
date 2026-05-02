-- 133_denormalize_last_message.down.sql
--
-- Rollback for migration 133. Drops the four denormalized last_message_*
-- columns symmetrically to the up.sql. IF EXISTS keeps the rollback
-- idempotent so partial migrate-down sequences (rare) stay safe.
--
-- After this rollback, /api/v1/messaging/conversations falls back to
-- the LATERAL subquery shape — readers MUST re-deploy the previous
-- conversation_queries.go alongside the migration.

BEGIN;

ALTER TABLE conversations
    DROP COLUMN IF EXISTS last_message_sender_id,
    DROP COLUMN IF EXISTS last_message_at,
    DROP COLUMN IF EXISTS last_message_content_preview,
    DROP COLUMN IF EXISTS last_message_seq;

COMMIT;
