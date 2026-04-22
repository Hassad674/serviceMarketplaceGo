-- Track a scheduled cycle change on a subscription so the UI can show
-- "Annuel jusqu'au DATE → Mensuel ensuite" after a downgrade request.
--
-- A downgrade (annual → monthly) uses a Stripe Subscription Schedule to
-- preserve the prepaid annual period: phase 1 keeps the annual price
-- until current_period_end, phase 2 switches to monthly. Until phase 2
-- fires, the local row stays with billing_cycle='annual' and carries
-- the pending target + effective date + schedule id so the state is
-- never ambiguous.
--
-- Upgrades (monthly → annual) apply immediately via subscription.update
-- and never use these columns; the pending_* fields stay NULL.

ALTER TABLE subscriptions
    ADD COLUMN pending_billing_cycle TEXT CHECK (pending_billing_cycle IN ('monthly', 'annual')),
    ADD COLUMN pending_cycle_effective_at TIMESTAMPTZ,
    ADD COLUMN stripe_schedule_id TEXT;

-- A subscription can only have one active schedule at a time; the
-- partial unique index keeps the table honest without touching rows
-- that never use scheduling.
CREATE UNIQUE INDEX idx_subscriptions_schedule_id ON subscriptions(stripe_schedule_id)
    WHERE stripe_schedule_id IS NOT NULL;

-- Enforce the full pending tuple is set together or not at all. Half-set
-- rows would confuse the UI ("pending mensuel" with no date) and the
-- webhook handler (schedule id missing → can't release on cancel).
ALTER TABLE subscriptions
    ADD CONSTRAINT subscriptions_pending_tuple_all_or_none CHECK (
        (pending_billing_cycle IS NULL AND pending_cycle_effective_at IS NULL AND stripe_schedule_id IS NULL)
        OR
        (pending_billing_cycle IS NOT NULL AND pending_cycle_effective_at IS NOT NULL AND stripe_schedule_id IS NOT NULL)
    );
