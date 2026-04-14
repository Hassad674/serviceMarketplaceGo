-- Down: delete every synthetic milestone that was auto-created from
-- backfill. We target only single-milestone rows at sequence = 1 where
-- title/description/amount match the parent proposal (i.e. the shape a
-- backfill would produce) to avoid clobbering legitimate milestones
-- created by the app after the backfill ran.
--
-- This is best-effort: if the down path is invoked after proposal
-- amendments have changed the milestone contents, it may leave orphan
-- rows. That is acceptable because the down path is only used in local
-- development to test round-trippability.
DELETE FROM proposal_milestones m
USING proposals p
WHERE m.proposal_id = p.id
  AND m.sequence    = 1
  AND m.title       = p.title
  AND m.description = p.description
  AND m.amount      = p.amount;
