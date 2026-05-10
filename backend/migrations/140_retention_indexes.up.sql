-- 140_retention_indexes.up.sql
--
-- Phase B.1 of the GDPR roadmap: enforce storage-limitation (RGPD art.
-- 5-1-e) by adding a retention scheduler that periodically sweeps stale
-- rows from `messages`, `notifications`, `device_tokens`,
-- `search_queries`, and `audit_logs`.
--
-- This migration adds the indexes the retention scheduler needs to run
-- a fast `WHERE age_column < cutoff LIMIT batch_size` sweep.
--
-- The matching schema additions (`device_tokens.last_seen_at`, the
-- `audit_logs_archive` cold-storage table) live in their own migrations
-- (141, 142) so each migration stays focused on a single concern.
--
-- Indexes already present that the scheduler reuses (no work here):
--   * notifications: idx_notifications_user_created (016)
--   * search_queries: idx_search_queries_created_at (111)
--   * audit_logs: idx_audit_logs_created_at (078)
--
-- Per backend/CLAUDE.md migration conventions: golang-migrate wraps
-- every migration in a single transaction, which is incompatible with
-- CREATE INDEX CONCURRENTLY. We therefore use IF NOT EXISTS in the body
-- and document the production-side CONCURRENTLY pre-step. The messages
-- table is large in production (auditperf.md flags it as a hot table);
-- the operator should run the CONCURRENTLY commands first, then the
-- migration's IF NOT EXISTS turns them into a no-op:
--
--   psql $DATABASE_URL -c "CREATE INDEX CONCURRENTLY IF NOT EXISTS \
--     idx_messages_created_at_retention \
--     ON messages(created_at) WHERE deleted_at IS NULL;"
--   make migrate-up
--
-- For dev / staging the in-tx CREATE INDEX is fine because the table
-- size is small.

BEGIN;

-- messages: scan by created_at, skipping already soft-deleted rows.
-- The existing idx_messages_conversation_created composite is not
-- selective enough for a table-wide retention scan.
CREATE INDEX IF NOT EXISTS idx_messages_created_at_retention
    ON messages(created_at)
 WHERE deleted_at IS NULL;

COMMIT;
