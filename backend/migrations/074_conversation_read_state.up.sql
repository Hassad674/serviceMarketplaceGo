-- Phase R11 — Per-user conversation read state
--
-- Before this migration, `conversation_participants` carried both
-- "is this user a direct endpoint of this conversation" (membership)
-- and "how many unread does this user have" (read state). That made
-- it impossible to track read state for org operators who joined the
-- team AFTER the conversation was opened — they have no participant
-- row, so there was nowhere to persist their unread count and the
-- authorization check in the messaging service rejected them with
-- `not_participant` even though the org-scoped list query correctly
-- surfaced the conversation to them.
--
-- R11 splits read state off into its own table. Any user (direct
-- participant OR org-mediated operator) gets a row lazily when they
-- first read or receive a message. The authorization check becomes
-- org-level everywhere (see migration 074 callers in the app/handler
-- layer) and `conversation_participants` reverts to being just a
-- membership table — the "who are the two endpoints of this
-- conversation" that proposals, calls and the conversation list's
-- LATERAL join still need.

BEGIN;

-- 1. New table: per-user read state
CREATE TABLE conversation_read_state (
    user_id          UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    conversation_id  UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    last_read_seq    INTEGER NOT NULL DEFAULT 0,
    unread_count     INTEGER NOT NULL DEFAULT 0,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, conversation_id),
    CHECK (last_read_seq >= 0),
    CHECK (unread_count >= 0)
);

-- 2. Partial index for the "total unread across all conversations" query
--    (GetTotalUnread / GetTotalUnreadBatch). Only indexes rows with a
--    positive unread count so the hot path skips fully-read rows.
CREATE INDEX idx_conversation_read_state_user_unread
    ON conversation_read_state(user_id)
    WHERE unread_count > 0;

-- 3. Secondary index for the conversation list JOIN (by conversation_id).
CREATE INDEX idx_conversation_read_state_conversation
    ON conversation_read_state(conversation_id);

-- 4. Sanity: count the rows we are about to backfill so we can assert
--    after the INSERT that none were silently dropped.
DO $$
DECLARE
    source_rows integer;
    inserted_rows integer;
    nonzero_source integer;
BEGIN
    SELECT COUNT(*) INTO source_rows FROM conversation_participants;
    SELECT COUNT(*) INTO nonzero_source
    FROM conversation_participants
    WHERE unread_count > 0 OR last_read_seq > 0;

    INSERT INTO conversation_read_state
        (user_id, conversation_id, last_read_seq, unread_count, created_at, updated_at)
    SELECT cp.user_id,
           cp.conversation_id,
           COALESCE(cp.last_read_seq, 0),
           COALESCE(cp.unread_count, 0),
           now(), now()
    FROM   conversation_participants cp
    ON CONFLICT (user_id, conversation_id) DO NOTHING;

    GET DIAGNOSTICS inserted_rows = ROW_COUNT;

    IF inserted_rows <> source_rows THEN
        RAISE EXCEPTION 'migration 074 backfill mismatch: expected % rows, inserted % (nonzero source rows: %)',
            source_rows, inserted_rows, nonzero_source;
    END IF;
END $$;

-- 5. Drop the read-state columns from conversation_participants. The
--    table now only records membership (joined_at).
ALTER TABLE conversation_participants DROP COLUMN IF EXISTS unread_count;
ALTER TABLE conversation_participants DROP COLUMN IF EXISTS last_read_seq;

COMMIT;
