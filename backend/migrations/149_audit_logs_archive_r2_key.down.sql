-- 149_audit_logs_archive_r2_key.down.sql
--
-- Reverse migration 149. The column drop is non-destructive for R2 —
-- already-uploaded objects survive in the bucket (their lifecycle is
-- managed by R2 itself + a bucket-level retention rule on prod).
-- Only the Postgres pointer is dropped.

BEGIN;

DROP INDEX IF EXISTS idx_audit_logs_archive_pending_upload;
DROP INDEX IF EXISTS idx_audit_logs_archive_uploaded;

ALTER TABLE audit_logs_archive
    DROP COLUMN IF EXISTS r2_key;

COMMIT;
