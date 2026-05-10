-- 140_retention_indexes.down.sql
--
-- Drop the retention-sweep helper index. CONCURRENTLY is used here
-- only when running outside the migration tool — the in-tx version
-- is fine for dev rollbacks and the production runbook documents the
-- manual CONCURRENTLY equivalent if the table is hot.

BEGIN;

DROP INDEX IF EXISTS idx_messages_created_at_retention;

COMMIT;
