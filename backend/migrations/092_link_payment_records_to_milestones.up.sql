-- Phase 3 of the milestones migration: link every existing
-- payment_record to the synthetic milestone created in migration 091.
--
-- Because every proposal now has exactly one milestone at sequence=1
-- (created by 091) and every payment_record references exactly one
-- proposal, this UPDATE produces a perfect 1:1 mapping.
--
-- The milestone_id column was added as nullable in migration 086. After
-- this backfill runs, every payment_record has a non-null milestone_id
-- and the next migration (093) can flip the constraint to NOT NULL +
-- UNIQUE(milestone_id).
UPDATE payment_records pr
SET milestone_id = m.id
FROM proposal_milestones m
WHERE m.proposal_id = pr.proposal_id
  AND m.sequence    = 1
  AND pr.milestone_id IS NULL;
