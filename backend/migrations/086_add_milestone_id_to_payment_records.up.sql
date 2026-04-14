-- Step 1 of migrating payment_records from 1:1-with-proposal to
-- 1:1-with-milestone: add the new column as nullable, without
-- touching the existing UNIQUE(proposal_id) constraint yet.
--
-- Phase 3's backfill migration populates this column for every existing
-- row (one synthetic milestone per proposal), then a subsequent
-- migration flips the constraint and sets NOT NULL.
ALTER TABLE payment_records
    ADD COLUMN IF NOT EXISTS milestone_id UUID REFERENCES proposal_milestones(id);

CREATE INDEX IF NOT EXISTS idx_payment_records_milestone
    ON payment_records(milestone_id)
    WHERE milestone_id IS NOT NULL;
