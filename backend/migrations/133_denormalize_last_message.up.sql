-- 133_denormalize_last_message.up.sql
--
-- P6 (perf — F.2 HIGH #3): denormalize the conversation's "last
-- message" preview onto the conversations row to eliminate the
-- correlated LATERAL subquery in /api/v1/messaging/conversations.
--
-- Decision (locked): maintenance applicatif — every INSERT into
-- messages is paired with an UPDATE of conversations.last_message_*
-- in the SAME transaction (see adapter/postgres/conversation_repository.go
-- ::createMessageInTx). Rejected alternative: PG TRIGGER. We want the
-- write to be visible in code, debuggable, and easy to test — magic
-- triggers complicate every postmortem.
--
-- The four columns are nullable so the schema admits empty
-- conversations (created via FindOrCreate without an initial message)
-- and so the symmetric down.sql is trivially safe.
--
--   last_message_seq             INT          — monotonic per-conv seq from messages
--   last_message_content_preview TEXT (≤100)  — UI preview, truncated server-side
--   last_message_at              TIMESTAMPTZ  — message created_at (NOT updated_at)
--   last_message_sender_id       UUID         — nullable: system messages bind NULL
--                                                (mirrors messages.sender_id post-mig 130)
--
-- Backfill: pull the latest message per conversation via DISTINCT ON
-- (conversation_id, seq DESC). Empty conversations are simply absent
-- from the join and remain NULL — no special handling required.

BEGIN;

ALTER TABLE conversations
    ADD COLUMN IF NOT EXISTS last_message_seq             INT,
    ADD COLUMN IF NOT EXISTS last_message_content_preview TEXT,
    ADD COLUMN IF NOT EXISTS last_message_at              TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS last_message_sender_id       UUID;

-- Backfill from existing messages. DISTINCT ON returns the row with
-- the highest seq per conversation (descending sort + DISTINCT ON
-- keeps the first row per group). The LEFT JOIN equivalent via the
-- UPDATE ... FROM (subquery) only touches rows that have at least
-- one message — empty conversations are left untouched (NULL).
--
-- LEFT(content, 100) bounds the preview to 100 chars matching the
-- ReplyPreview truncation already in domain/message. Keeping the
-- truncation in SQL avoids shipping the entire content from PG to
-- Go for trimming.
UPDATE conversations c
SET last_message_seq             = m.seq,
    last_message_content_preview = LEFT(m.content, 100),
    last_message_at              = m.created_at,
    last_message_sender_id       = m.sender_id
FROM (
    SELECT DISTINCT ON (conversation_id)
        conversation_id, seq, content, created_at, sender_id
    FROM messages
    ORDER BY conversation_id, seq DESC
) m
WHERE c.id = m.conversation_id;

COMMIT;
