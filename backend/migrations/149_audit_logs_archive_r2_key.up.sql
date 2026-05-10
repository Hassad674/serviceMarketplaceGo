-- 149_audit_logs_archive_r2_key.up.sql
--
-- Phase B.2 of the GDPR roadmap: extend audit_logs_archive (created in
-- migration 142) with a `r2_key` column that records the R2 object key
-- of the gzipped JSONL bundle holding the row's payload AFTER the
-- cold-tier sweep dumped it to Cloudflare R2.
--
-- Why a column rather than a side table:
--   * The lifecycle is strict: a row is "uploaded" when r2_key IS NOT
--     NULL, and "still in Postgres only" when r2_key IS NULL. A side
--     table would force a JOIN on every cold-tier query for zero
--     extra information.
--   * No new infrastructure: the cold sweep can SELECT FOR UPDATE
--     SKIP LOCKED on (r2_key IS NULL) and have an idempotent advance
--     condition without grabbing extra locks.
--   * Reversibility: dropping the column on the down migration leaves
--     audit_logs_archive untouched in shape. R2 objects survive the
--     down migration; that is intentional — once dumped to R2, the
--     payload is canonical.
--
-- After this migration, the cold sweep flow is:
--   1. SELECT a batch of audit_logs_archive WHERE archived_at < cutoff
--      AND r2_key IS NULL.
--   2. Build gzipped JSONL bundle, upload to R2 under
--      `audit-cold/<year>/<month>/<batch_id>.jsonl.gz`.
--   3. UPDATE the same rows with r2_key = '<key>' (single statement,
--      same tx).
--   4. On the *next* sweep tick, DELETE those rows (r2_key IS NOT NULL
--      AND archived_at < cutoff). The two-phase split is intentional:
--      a crash between the upload and the UPDATE leaves the row in
--      Postgres (we will retry next tick); a crash after the UPDATE
--      leaves the row marked for deletion, which is also safe.
--
-- The column is NULL by default — every row inserted by migration 142's
-- archive sweep predates B.2 and starts in the "not uploaded" state.

BEGIN;

ALTER TABLE audit_logs_archive
    ADD COLUMN IF NOT EXISTS r2_key TEXT NULL;

-- Partial index to make the cold-sweep SELECT fast even on a large
-- archive: only rows that still need uploading match the predicate.
-- Without it, a scan on archived_at would have to filter every row
-- (most of which already have r2_key set).
CREATE INDEX IF NOT EXISTS idx_audit_logs_archive_pending_upload
    ON audit_logs_archive(archived_at)
    WHERE r2_key IS NULL;

-- Symmetric partial index for the DELETE phase: rows whose payload
-- has been uploaded and that are now eligible for hard removal.
CREATE INDEX IF NOT EXISTS idx_audit_logs_archive_uploaded
    ON audit_logs_archive(archived_at)
    WHERE r2_key IS NOT NULL;

COMMENT ON COLUMN audit_logs_archive.r2_key IS
    'Cloudflare R2 object key holding the row''s JSONL payload, set by the B.2 cold-tier sweep. NULL means the row is still Postgres-canonical.';

COMMIT;
