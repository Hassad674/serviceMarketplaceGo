-- 142_audit_logs_archive.down.sql
--
-- Drops the cold-storage archive table. Any rows previously moved
-- there are lost on rollback — the rollback only makes sense in dev
-- where the table is empty.

BEGIN;

DROP TABLE IF EXISTS audit_logs_archive;

COMMIT;
