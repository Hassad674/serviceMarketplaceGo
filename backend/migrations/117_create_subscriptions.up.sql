-- subscriptions holds Premium plan state per user. One active row per user
-- at a time (enforced by the partial unique index below). Auto-renewal is
-- OFF by default — cancel_at_period_end defaults to TRUE, so a new
-- subscription expires naturally unless the user explicitly flips the
-- toggle during checkout or from the management modal later.
--
-- No cross-feature foreign keys: the table references only users(id),
-- matching the project rule. Payment records stay independent and
-- consume the status via the SubscriptionReader port interface.
CREATE TABLE IF NOT EXISTS subscriptions (
    id                     UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id                UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    -- Product dimensions
    plan                   TEXT NOT NULL CHECK (plan IN ('freelance', 'agency')),
    billing_cycle          TEXT NOT NULL CHECK (billing_cycle IN ('monthly', 'annual')),

    -- Lifecycle status — mirrors Stripe's subscription.status vocabulary
    status                 TEXT NOT NULL CHECK (status IN (
        'incomplete',    -- checkout started, not yet paid
        'active',        -- paid, Premium granted
        'past_due',      -- renewal payment failed, grace period
        'canceled',      -- fully stopped (cancel_at_period_end fired or manual)
        'unpaid'         -- Stripe-terminal after grace period lapses
    )),

    -- Stripe identifiers. stripe_customer_id is per-user and reused across
    -- subscriptions; stripe_subscription_id is per active/past subscription.
    stripe_customer_id     TEXT NOT NULL,
    stripe_subscription_id TEXT NOT NULL UNIQUE,
    stripe_price_id        TEXT NOT NULL,

    -- Current billing window. Stripe is the source of truth; webhooks keep
    -- these in sync. The UI renders "expire le {current_period_end}" when
    -- cancel_at_period_end is TRUE.
    current_period_start   TIMESTAMPTZ NOT NULL,
    current_period_end     TIMESTAMPTZ NOT NULL,

    -- Auto-renewal flag. DEFAULT TRUE means "WILL cancel at period end"
    -- i.e. auto-renew OFF — the product choice is to never charge a user
    -- again unless they opt in.
    cancel_at_period_end   BOOLEAN NOT NULL DEFAULT TRUE,

    -- Grace period window when status becomes past_due. Set by the
    -- invoice.payment_failed webhook handler; cleared on recovery.
    grace_period_ends_at   TIMESTAMPTZ,

    -- Set when the subscription transitions to canceled (either by
    -- natural expiration or explicit cancel). Enables queries like "sub
    -- that was active on date X" for fee-saving stats over periods.
    canceled_at            TIMESTAMPTZ,

    -- started_at is the first time the subscription became active. Used
    -- as the lower bound for "fees saved since subscribing" stats.
    started_at             TIMESTAMPTZ NOT NULL DEFAULT now(),

    created_at             TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TRIGGER subscriptions_updated_at
    BEFORE UPDATE ON subscriptions
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();

-- One "open" subscription per user. Prevents accidental double-charge if
-- the subscribe endpoint is hit twice before the first webhook lands.
-- Canceled/unpaid rows stay for history without tripping the constraint.
CREATE UNIQUE INDEX idx_subscriptions_user_open ON subscriptions(user_id)
    WHERE status IN ('incomplete', 'active', 'past_due');

-- Fast lookup paths used by the app service + webhook handler.
CREATE INDEX idx_subscriptions_user ON subscriptions(user_id);
CREATE INDEX idx_subscriptions_status ON subscriptions(status);
CREATE INDEX idx_subscriptions_stripe_customer ON subscriptions(stripe_customer_id);
