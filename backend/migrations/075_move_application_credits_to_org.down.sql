-- R12 — Rollback: restore the per-user application_credits table.
--
-- WARNING: this rollback is LOSSY. The up migration summed every
-- member's credits into a single org pool. The per-user distribution
-- that existed before cannot be reconstructed. As a best-effort, this
-- down migration assigns the whole org balance to the org owner
-- (organizations.owner_user_id). Every other member of the org starts
-- back at 0 credits. If you care about the pre-migration distribution,
-- restore from a backup instead of running this down migration.

BEGIN;

-- 1. Recreate the table with the original shape.
CREATE TABLE IF NOT EXISTS application_credits (
    user_id       UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    credits       INTEGER NOT NULL DEFAULT 10,
    last_reset_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- 2. Best-effort restore: dump each org's balance onto its owner.
INSERT INTO application_credits (user_id, credits, last_reset_at, created_at, updated_at)
SELECT o.owner_user_id,
       o.application_credits,
       o.credits_last_reset_at,
       now(),
       now()
FROM   organizations o
WHERE  o.application_credits > 0
ON CONFLICT (user_id) DO UPDATE
SET    credits = EXCLUDED.credits,
       last_reset_at = EXCLUDED.last_reset_at,
       updated_at = now();

-- 3. Drop the org-level columns.
ALTER TABLE organizations DROP COLUMN IF EXISTS application_credits;
ALTER TABLE organizations DROP COLUMN IF EXISTS credits_last_reset_at;

COMMIT;
