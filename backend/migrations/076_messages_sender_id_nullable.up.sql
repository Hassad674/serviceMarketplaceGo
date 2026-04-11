BEGIN;

-- Drop NOT NULL from messages.sender_id so the existing ON DELETE SET NULL
-- foreign key on the same column can actually set it to NULL when a user
-- is hard-deleted.
--
-- Before this migration the column was declared `UUID NOT NULL ... ON DELETE
-- SET NULL` (see migration 007_create_messaging.up.sql). That combination
-- is a latent bug: the FK clause promises to SET NULL on user delete, but
-- the NOT NULL constraint rejects the very row update the FK triggers, so
-- Postgres aborts the whole DELETE with
--
--     null value in column "sender_id" of relation "messages"
--     violates not-null constraint
--
-- The practical impact was that an operator who had sent at least one chat
-- message could not be removed from the users table, which in turn left
-- the row in an orphan state (organization_members row gone but users row
-- stuck) after MembershipService.LeaveOrganization /.RemoveMember ran.
-- Dropping the NOT NULL lets the SET NULL cascade succeed; orphaned
-- messages keep their content and timestamps, only the sender pointer
-- disappears — the UI renders "Deleted user" for those rows.
--
-- The ALTER is idempotent. If the column is already nullable (for example
-- because the hotfix was applied manually on an environment before this
-- migration shipped), PostgreSQL treats "DROP NOT NULL" on an already
-- nullable column as a no-op and the migration still succeeds.
ALTER TABLE messages ALTER COLUMN sender_id DROP NOT NULL;

COMMIT;
