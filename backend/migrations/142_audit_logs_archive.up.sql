-- 142_audit_logs_archive.up.sql
--
-- Phase B.1 of the GDPR roadmap: introduce a cold-storage archive
-- table for `audit_logs`. Rows older than the configured hot-tier
-- retention (24 months by default) are moved into `audit_logs_archive`
-- by the retention scheduler. The archive table preserves every
-- column verbatim so the legal/compliance read path remains unchanged
-- — `SELECT … FROM audit_logs_archive` returns rows in the same shape
-- as the live table.
--
-- Why a secondary table rather than a hard DELETE:
--   * RGPD art. 5-1-e tolerates retention proportional to purpose. A
--     security/legal audit trail has a multi-year purpose; deleting it
--     after 24 months would break incident investigations.
--   * Moving rows out of the hot table caps the live audit_logs size
--     so day-to-day queries stay fast and CONCURRENTLY index builds
--     remain quick.
--   * R2 / S3 cold storage was considered (see CLAUDE.md Section
--     "Audit logs are kept indefinitely … archive to cold storage")
--     but a same-database secondary table is the simpler first step:
--     no new infrastructure, queryable with plain SQL, and the move
--     to R2 is a future Phase C migration that re-reads
--     `audit_logs_archive` and writes JSONL.gz to R2 before TRUNCATE.
--
-- The archive table mirrors `audit_logs` exactly. The only schema
-- diffs are:
--   * `archived_at TIMESTAMPTZ NOT NULL DEFAULT now()` — bookkeeping
--     column so the retention sweep that produced the row is visible.
--   * No RLS policy on archive: cross-org queries are admin-only
--     (the retention sweep itself is a system actor; user-facing
--     reads continue to use the hot `audit_logs` table). Keeping RLS
--     off the archive avoids the WITH CHECK gymnastics from migration
--     129 — the sweep INSERTs as a system actor, and any future read
--     path is admin-only.
--
-- `audit_logs` permissions remain INSERT/SELECT-only for the application
-- role; the retention sweep runs as the migration owner today (same as
-- the GDPR purge cron) so DELETE on the hot table is permitted. When
-- the dedicated `marketplace_archiver` role lands (B.7 in the roadmap),
-- the GRANT setup will move there.

BEGIN;

CREATE TABLE IF NOT EXISTS audit_logs_archive (
    id            UUID PRIMARY KEY,
    user_id       UUID,
    action        TEXT NOT NULL,
    resource_type TEXT,
    resource_id   UUID,
    metadata      JSONB NOT NULL DEFAULT '{}'::jsonb,
    ip_address    INET,
    created_at    TIMESTAMPTZ NOT NULL,
    archived_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Indexes mirror the live table's read patterns: per-user trail,
-- per-action audit, per-resource history, plus the archived_at column
-- for "what did the latest sweep move out" diagnostics.
CREATE INDEX IF NOT EXISTS idx_audit_logs_archive_user_id
    ON audit_logs_archive(user_id) WHERE user_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_audit_logs_archive_action
    ON audit_logs_archive(action);
CREATE INDEX IF NOT EXISTS idx_audit_logs_archive_created_at
    ON audit_logs_archive(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_archive_resource
    ON audit_logs_archive(resource_type, resource_id) WHERE resource_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_audit_logs_archive_archived_at
    ON audit_logs_archive(archived_at DESC);

COMMENT ON TABLE audit_logs_archive IS
    'Cold-storage of audit_logs older than the configured hot retention. Append-only. See migration 142.';

COMMIT;
