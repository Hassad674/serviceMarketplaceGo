-- 141_device_tokens_last_seen_at.up.sql
--
-- Phase B.1 of the GDPR roadmap: add `last_seen_at` to `device_tokens`
-- so the retention scheduler can prune push tokens that have not been
-- delivered to in 60 days. FCM marks tokens stale after ~60 days of
-- app inactivity (see audit gdpr-audit.md Section 7) — keeping them in
-- the database past that window is dead weight and a minor security
-- risk.
--
-- The column defaults to NOW() so existing rows get a fresh window
-- before the first sweep. Newly inserted rows also default to NOW()
-- via the queryInsertDeviceToken path (the DB default carries the
-- value when the INSERT does not specify the column).
--
-- The companion application-side update (UPDATE device_tokens SET
-- last_seen_at = NOW() on every successful push delivery) lives in the
-- notification adapter alongside the FCM send call — it is wired in
-- the same commit that introduces this migration.
--
-- Production note: ALTER TABLE ADD COLUMN with a non-volatile DEFAULT
-- is a metadata-only operation in Postgres 11+ — no table rewrite, so
-- this is safe to run during business hours. The CONCURRENTLY index
-- is documented as the manual pre-step for the same reason as 140:
--
--   psql $DATABASE_URL -c "CREATE INDEX CONCURRENTLY IF NOT EXISTS \
--     idx_device_tokens_last_seen_at ON device_tokens(last_seen_at);"
--   make migrate-up

BEGIN;

ALTER TABLE device_tokens
    ADD COLUMN IF NOT EXISTS last_seen_at TIMESTAMPTZ NOT NULL DEFAULT now();

CREATE INDEX IF NOT EXISTS idx_device_tokens_last_seen_at
    ON device_tokens(last_seen_at);

COMMIT;
