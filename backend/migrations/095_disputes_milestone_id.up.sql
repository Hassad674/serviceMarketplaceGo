-- Phase 8 of the milestones feature: scope every dispute to a single
-- milestone instead of the whole proposal.
--
-- Step 1: add the column nullable so the backfill can populate it.
-- Step 2: backfill from the synthetic milestone created in 091 — every
--         existing dispute gets linked to its proposal's only milestone
--         (sequence=1) since pre-phase-4 proposals are 1:1 with milestones.
-- Step 3: tighten to NOT NULL once the backfill has succeeded.
--
-- The legacy disputes.proposal_id column stays for the org-scoped
-- listing queries. Going forward, the dispute resolution split is on
-- milestone.amount, not proposal.amount — see the dispute service
-- changes in the same phase.

ALTER TABLE disputes
    ADD COLUMN IF NOT EXISTS milestone_id UUID REFERENCES proposal_milestones(id);

CREATE INDEX IF NOT EXISTS idx_disputes_milestone
    ON disputes(milestone_id)
    WHERE milestone_id IS NOT NULL;

-- Backfill: link each dispute to the synthetic milestone of its
-- proposal. Idempotent via NOT NULL guard so partial reruns are safe.
UPDATE disputes d
SET milestone_id = m.id
FROM proposal_milestones m
WHERE m.proposal_id = d.proposal_id
  AND m.sequence    = 1
  AND d.milestone_id IS NULL;

-- Tighten to NOT NULL now that the backfill has succeeded.
ALTER TABLE disputes
    ALTER COLUMN milestone_id SET NOT NULL;
