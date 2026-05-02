-- 132_users_deleted_at_for_gdpr.up.sql
--
-- P5 (RGPD endpoints): adds soft-delete support on users for the
-- GDPR right-to-erasure flow.
--
-- Decision 3 (locked, see docs/plans/P5_brief.md): we soft-delete
-- the user at T0, lock them out, and keep the row for 30 days. A
-- daily cron purges hard at T+30. This column is the single anchor
-- for that flow.
--
-- Why partial: in steady state the index serves only the cron job
-- which scans rows WHERE deleted_at IS NOT NULL AND deleted_at
-- < NOW() - INTERVAL '30 days'. A full index would carry one
-- entry per row in the table; a partial index keeps the footprint
-- proportional to "users currently in their 30-day cooldown",
-- which is a tiny working set.
--
-- The ALTER TABLE ADD COLUMN here is a fast metadata-only operation
-- (Postgres 11+) because deleted_at is nullable with no default.
-- No backfill needed: existing rows get NULL automatically.

BEGIN;

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ NULL;

CREATE INDEX IF NOT EXISTS idx_users_pending_deletion
    ON users (deleted_at)
    WHERE deleted_at IS NOT NULL;

-- pgcrypto powers the in-SQL sha256 hashing used by the GDPR purge
-- cron when it anonymizes audit_logs metadata. Enabled here once so
-- the cron does not have to call back into Go for per-row hashing.
-- IF NOT EXISTS makes this idempotent across environments where
-- pgcrypto was already loaded for another reason.
CREATE EXTENSION IF NOT EXISTS pgcrypto;

COMMIT;
