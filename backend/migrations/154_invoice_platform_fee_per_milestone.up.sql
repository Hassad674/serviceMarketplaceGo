-- 154_invoice_platform_fee_per_milestone.up.sql
--
-- Adds per-milestone invoice support:
--   1. invoice.milestone_id (nullable — subscription/monthly rows leave it NULL)
--   2. Extends invoice_source_type_check to accept 'platform_fee'
--   3. Partial UNIQUE index so a given milestone gets at most ONE
--      platform_fee invoice (idempotence at the DB layer, defense in
--      depth on top of the app-level FindPlatformFeeByMilestoneID probe).
--
-- Idempotent (IF NOT EXISTS / IF EXISTS) so re-runs on a partially-
-- applied state are safe — required for the multi-agent shared-DB
-- workflow described in backend/CLAUDE.md.

BEGIN;

-- 1. Add the milestone_id column. Nullable: subscription and
-- monthly_commission rows do not reference a single milestone.
ALTER TABLE invoice
    ADD COLUMN IF NOT EXISTS milestone_id UUID NULL;

-- 2. Extend the source_type CHECK constraint.
ALTER TABLE invoice DROP CONSTRAINT IF EXISTS invoice_source_type_check;
ALTER TABLE invoice ADD CONSTRAINT invoice_source_type_check
    CHECK (source_type IN ('subscription', 'monthly_commission', 'platform_fee'));

-- 3. UNIQUE partial index — at most one platform_fee invoice per
-- milestone. Subscription / monthly_commission rows are excluded by the
-- WHERE clause so they never participate in the uniqueness check.
CREATE UNIQUE INDEX IF NOT EXISTS idx_invoice_milestone_platform_fee_unique
    ON invoice (milestone_id)
    WHERE source_type = 'platform_fee' AND milestone_id IS NOT NULL;

COMMIT;
