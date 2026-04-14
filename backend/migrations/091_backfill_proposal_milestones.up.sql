-- Phase 3 of the milestones migration: backfill a synthetic milestone
-- for every existing proposal so the unified "proposal = at least one
-- milestone" invariant holds on legacy data.
--
-- Each synthetic milestone takes the full amount of its proposal at
-- sequence = 1, with a status derived from the proposal status so the
-- milestone state machine is consistent with the legacy macro state.
-- Dispute references are carried over so an open dispute keeps its
-- scoping on the synthetic milestone.
--
-- Timestamps that we can map directly (funded_at from paid_at,
-- released_at from completed_at, disputed_at from updated_at when
-- status = 'disputed') are set. submitted_at and approved_at are left
-- NULL for historical data — the UI renders those gracefully.
--
-- This migration is idempotent: the NOT EXISTS clause prevents
-- re-inserting a synthetic milestone if the table was partially
-- backfilled from a previous attempt.
INSERT INTO proposal_milestones (
    id,
    proposal_id,
    sequence,
    title,
    description,
    amount,
    deadline,
    status,
    version,
    funded_at,
    submitted_at,
    approved_at,
    released_at,
    disputed_at,
    cancelled_at,
    active_dispute_id,
    last_dispute_id,
    created_at,
    updated_at
)
SELECT
    gen_random_uuid(),
    p.id,
    1,
    p.title,
    p.description,
    p.amount,
    p.deadline,
    CASE p.status
        WHEN 'pending'              THEN 'pending_funding'
        WHEN 'accepted'             THEN 'pending_funding'
        WHEN 'declined'             THEN 'cancelled'
        WHEN 'withdrawn'            THEN 'cancelled'
        WHEN 'paid'                 THEN 'funded'
        WHEN 'active'               THEN 'funded'
        WHEN 'completion_requested' THEN 'submitted'
        WHEN 'completed'            THEN 'released'
        WHEN 'disputed'             THEN 'disputed'
        ELSE                             'pending_funding'
    END,
    0,
    -- funded_at: map from paid_at for any status that has received funding.
    CASE WHEN p.status IN ('paid', 'active', 'completion_requested', 'completed', 'disputed')
         THEN p.paid_at ELSE NULL END,
    -- submitted_at: no direct source on proposal, left NULL for legacy data.
    NULL,
    -- approved_at: no direct source on proposal, left NULL for legacy data.
    NULL,
    -- released_at: map from completed_at when the proposal is completed.
    CASE WHEN p.status = 'completed' THEN p.completed_at ELSE NULL END,
    -- disputed_at: use updated_at as a best-effort proxy for disputed proposals.
    CASE WHEN p.status = 'disputed' THEN p.updated_at ELSE NULL END,
    -- cancelled_at: use updated_at as proxy for declined/withdrawn proposals.
    CASE WHEN p.status IN ('declined', 'withdrawn') THEN p.updated_at ELSE NULL END,
    p.active_dispute_id,
    p.last_dispute_id,
    p.created_at,
    p.updated_at
FROM proposals p
WHERE NOT EXISTS (
    SELECT 1 FROM proposal_milestones m
    WHERE m.proposal_id = p.id AND m.sequence = 1
);

-- Also mark all backfilled proposals as payment_mode = 'one_time' so the
-- frontend keeps rendering the legacy simple-amount UX for them. New
-- proposals created after this migration via the updated API can set
-- payment_mode = 'milestone' if they want the multi-step editor.
-- (No UPDATE needed because the column default is 'one_time'.)
