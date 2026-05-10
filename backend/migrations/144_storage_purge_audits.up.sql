-- 144_storage_purge_audits.up.sql
--
-- GDPR right-to-erasure compliance evidence: every time the cron
-- purge of a soft-deleted user runs, the scheduler records which
-- R2/MinIO object keys it attempted to delete and which succeeded.
--
-- This is the operator's audit trail when a regulator asks "did you
-- actually erase the user's media?" — without this row we only have
-- "trust us, we ran the job".
--
-- Schema choices
-- --------------
--   * user_id is FK ON DELETE SET NULL: the cron purge eventually
--     anonymizes/wipes the users row in place, so a hard cascade
--     would also delete this audit. Keeping the row with NULL
--     user_id preserves the evidence post-purge — the keys_count
--     and the timestamp are what compliance needs.
--   * organization_id is plain UUID nullable: orgs come and go
--     independently of users; we don't FK it on purpose.
--   * keys_count is denormalized so list/aggregation queries don't
--     need to load the TEXT[] every time.
--   * purged_keys / failed_keys are TEXT[] and stay queryable
--     without a join. Sample size is small (a heavy user has tens
--     of objects, not thousands) so column TOAST overhead is fine.
--   * created_at is the audit creation time; the application always
--     writes UTC.
--
-- This table is APPEND-ONLY by policy: same convention as audit_logs
-- (no UPDATE, no DELETE from the application path). The retention
-- scheduler may archive after 24 months but should never overwrite.

BEGIN;

CREATE TABLE IF NOT EXISTS storage_purge_audits (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID REFERENCES users(id) ON DELETE SET NULL,
    organization_id UUID,
    keys_count      INT NOT NULL DEFAULT 0,
    purged_keys     TEXT[] NOT NULL DEFAULT '{}',
    failed_keys     TEXT[] NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_storage_purge_audits_user_id
    ON storage_purge_audits(user_id)
    WHERE user_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_storage_purge_audits_created_at
    ON storage_purge_audits(created_at DESC);

COMMIT;
