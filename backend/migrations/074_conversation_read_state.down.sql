-- Phase R11 — Rollback per-user conversation read state
--
-- This restores the pre-R11 layout: read state columns live back on
-- conversation_participants and are populated from the dedicated table
-- where possible. Any read state belonging to an operator who did not
-- have a direct participant row is lost (by definition — that state
-- cannot be represented in the old model).

BEGIN;

ALTER TABLE conversation_participants
    ADD COLUMN IF NOT EXISTS unread_count  INTEGER NOT NULL DEFAULT 0;
ALTER TABLE conversation_participants
    ADD COLUMN IF NOT EXISTS last_read_seq INTEGER NOT NULL DEFAULT 0;

UPDATE conversation_participants cp
SET unread_count  = crs.unread_count,
    last_read_seq = crs.last_read_seq
FROM conversation_read_state crs
WHERE crs.user_id = cp.user_id
  AND crs.conversation_id = cp.conversation_id;

DROP INDEX IF EXISTS idx_conversation_read_state_user_unread;
DROP INDEX IF EXISTS idx_conversation_read_state_conversation;
DROP TABLE IF EXISTS conversation_read_state;

COMMIT;
