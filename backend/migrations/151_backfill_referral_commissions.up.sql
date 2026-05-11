-- 151_backfill_referral_commissions.up.sql
--
-- One-shot backfill of referral_commissions for milestones that were
-- already APPROVED or RELEASED by clients BEFORE the
-- PrepareCommissionForMilestone hook landed (commit 62784222).
--
-- WITHOUT this migration: every commission that should have been
-- earned on an existing in-flight or completed mission is invisible to
-- the apporteur. The hook only fires going FORWARD on new approvals.
--
-- After the rows land in 'pending', the existing scheduler
-- (DrainPendingCommissions in app/referral/pending_sweeper.go) is
-- responsible for either:
--   - transferring them to Stripe (if the referrer has Connect KYC),
--   - or parking them as 'pending_kyc' for the KYC listener to pick up.
--
-- The migration is IDEMPOTENT: re-running it after partial application
-- (or running it twice) is a no-op thanks to the
-- (attribution_id, milestone_id) unique guard from migration 108.
--
-- Commission cents are computed using the same basis-point formula as
-- the domain (referral.computeCommissionCents):
--   amount_cents * (rate_pct * 100)::int / 10_000
-- truncated DOWN — never owe the apporteur more than one centime of
-- rounding error.
--
-- Currency defaults to 'EUR' because milestones do not carry a
-- currency column (the platform is single-currency today).
--
-- Eligible proposal statuses: any state that means the proposal was
-- ACCEPTED and at least one milestone could have been worked on:
--   'active', 'completion_requested', 'completed', 'paid'
-- ('accepted' is excluded because no milestone funding has happened yet.)
--
-- Eligible milestone statuses: 'approved' and 'released'. Both indicate
-- the client validated the work and a commission was earned. Disputed,
-- refunded and cancelled milestones are correctly excluded — no
-- commission is owed when the milestone never paid out cleanly.

BEGIN;

INSERT INTO referral_commissions (
    id,
    attribution_id,
    milestone_id,
    gross_amount_cents,
    commission_cents,
    currency,
    status,
    stripe_transfer_id,
    stripe_reversal_id,
    failure_reason,
    created_at,
    updated_at
)
SELECT
    gen_random_uuid(),
    a.id,
    m.id,
    m.amount,
    -- basis-point math, truncated DOWN to match domain.computeCommissionCents
    (m.amount * (a.rate_pct_snapshot * 100)::bigint / 10000)::bigint,
    'EUR',
    'pending',
    '',
    '',
    '',
    now(),
    now()
FROM referral_attributions a
JOIN proposals p              ON p.id = a.proposal_id
JOIN proposal_milestones m    ON m.proposal_id = p.id
WHERE p.status IN ('active', 'completion_requested', 'completed', 'paid')
  AND m.status IN ('approved', 'released')
  AND m.amount > 0
  AND NOT EXISTS (
      SELECT 1 FROM referral_commissions c
      WHERE c.attribution_id = a.id
        AND c.milestone_id   = m.id
  );

COMMIT;
