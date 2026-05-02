-- 132_users_deleted_at_for_gdpr.down.sql
--
-- Rollback for migration 132. Drops the partial index first, then
-- the column. IF EXISTS makes the rollback idempotent so re-running
-- a partial migrate-down sequence is safe.

BEGIN;

DROP INDEX IF EXISTS idx_users_pending_deletion;

ALTER TABLE users
    DROP COLUMN IF EXISTS deleted_at;

COMMIT;
