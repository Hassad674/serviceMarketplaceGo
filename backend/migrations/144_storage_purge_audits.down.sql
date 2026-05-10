-- 144_storage_purge_audits.down.sql
-- Rolls back the storage purge audit trail. Any compliance evidence
-- already collected is lost — only run this in dev or before the
-- table has been written to in production.

BEGIN;

DROP INDEX IF EXISTS idx_storage_purge_audits_created_at;
DROP INDEX IF EXISTS idx_storage_purge_audits_user_id;
DROP TABLE IF EXISTS storage_purge_audits;

COMMIT;
