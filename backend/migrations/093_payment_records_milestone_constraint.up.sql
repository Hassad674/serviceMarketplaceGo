-- Phase 3 final step: flip the payment_records constraint from
-- "UNIQUE per proposal" to "UNIQUE per milestone + NOT NULL".
--
-- Safe to run only after 091 (synthetic milestones) and 092 (backfill
-- milestone_id) have succeeded — every row must already have a
-- non-null milestone_id.
--
-- The old UNIQUE(proposal_id) constraint blocked the multi-milestone
-- model: a proposal can now legitimately own N payment_records (one
-- per milestone). The new UNIQUE(milestone_id) enforces 1:1 at the
-- milestone level instead.

-- 1. Make milestone_id NOT NULL now that every row has it populated.
ALTER TABLE payment_records
    ALTER COLUMN milestone_id SET NOT NULL;

-- 2. Drop the legacy UNIQUE(proposal_id) that prevented the N:1 layout.
ALTER TABLE payment_records
    DROP CONSTRAINT IF EXISTS payment_records_proposal_id_key;

-- 3. Add the new UNIQUE(milestone_id) so we keep "one payment_record
--    per milestone" as an invariant for the new model.
ALTER TABLE payment_records
    ADD CONSTRAINT payment_records_milestone_id_key UNIQUE (milestone_id);

-- 4. Keep the existing idx_payment_records_proposal btree index — it
--    still helps admin queries that list payments by proposal.
