-- 128_messages_nullable_sender.down.sql
--
-- Rollback to NOT NULL sender_id. Because system-actor messages
-- inserted with NULL sender after this migration was applied would
-- block the rollback (NOT NULL violations on existing data), the
-- rollback first deletes any NULL-sender rows and then re-adds the
-- constraint. This is a one-way data drop — only run if you know what
-- you are doing.

DELETE FROM messages WHERE sender_id IS NULL;

ALTER TABLE messages
    ALTER COLUMN sender_id SET NOT NULL;
