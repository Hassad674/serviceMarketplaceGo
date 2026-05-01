-- 130_messages_nullable_sender.up.sql
--
-- System messages (proposal_completed, evaluation_request,
-- milestone_auto_approved, proposal_auto_closed, dispute_auto_resolved,
-- etc.) are emitted by background paths (end-of-project effects, the
-- scheduler worker, the dispute auto-resolver) that have no human
-- author. The application carries a uuid.Nil sender for these and the
-- adapter layer rewrites it to SQL NULL on insert (senderForInsert /
-- senderForRead helpers).
--
-- The original messages table forced sender_id NOT NULL, which silently
-- dropped every system-actor send: the proposal service ignores the
-- error from SendSystemMessage and the FK/NOT-NULL violation rolled
-- back the row without surfacing. Net effect: a completed mission never
-- got a "proposal_completed" or "evaluation_request" card in the
-- conversation, blocking the post-mission review flow end-to-end.
--
-- Drop the NOT NULL constraint so system-actor messages persist as
-- expected. The existing FK to users(id) already allows NULL (ON DELETE
-- SET NULL) and the read path already handles NULL via senderForRead.

ALTER TABLE messages
    ALTER COLUMN sender_id DROP NOT NULL;
