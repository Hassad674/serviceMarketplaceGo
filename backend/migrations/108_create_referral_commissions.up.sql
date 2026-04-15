-- 108_create_referral_commissions.up.sql
--
-- One commission row per milestone payout that is attributed to a
-- referral. Created by the distributor BEFORE the Stripe transfer call
-- so DB idempotency wins over Stripe idempotency in case of a partial
-- crash.
--
-- Lifecycle:
--   pending      → row inserted, not yet sent to Stripe
--   pending_kyc  → referrer has no Stripe Connect account yet, parked
--                  until embedded.OnStripeAccountReady fires
--   paid         → Stripe transfer succeeded, stripe_transfer_id stored
--   failed       → Stripe call failed, failure_reason stored
--   cancelled   → milestone cancelled before transfer, no money moved
--   clawed_back  → milestone refunded after payout; transfer_reversal
--                  executed with stripe_reversal_id stored
--
-- UNIQUE(attribution_id, milestone_id) is the DB-level idempotency
-- guard for the distributor — calling it twice on the same milestone
-- raises a unique violation that the app layer maps to "skipped".
--
-- milestone_id is stored as a bare UUID with NO FK (modularity rule).

BEGIN;

CREATE TABLE referral_commissions (
    id                  UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    attribution_id      UUID         NOT NULL REFERENCES referral_attributions(id) ON DELETE RESTRICT,
    milestone_id        UUID         NOT NULL,
    gross_amount_cents  BIGINT       NOT NULL CHECK (gross_amount_cents > 0),
    commission_cents    BIGINT       NOT NULL CHECK (commission_cents >= 0),
    currency            TEXT         NOT NULL DEFAULT 'EUR',
    status              TEXT         NOT NULL CHECK (status IN (
        'pending', 'pending_kyc', 'paid', 'failed', 'cancelled', 'clawed_back'
    )),
    stripe_transfer_id  TEXT         NOT NULL DEFAULT '',
    stripe_reversal_id  TEXT         NOT NULL DEFAULT '',
    failure_reason      TEXT         NOT NULL DEFAULT '',
    paid_at             TIMESTAMPTZ,
    clawed_back_at      TIMESTAMPTZ,
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ  NOT NULL DEFAULT now(),

    CONSTRAINT referral_commissions_attribution_milestone_unique
        UNIQUE (attribution_id, milestone_id)
);

CREATE INDEX idx_referral_commissions_attribution
    ON referral_commissions (attribution_id);

CREATE INDEX idx_referral_commissions_status
    ON referral_commissions (status);

CREATE INDEX idx_referral_commissions_pending_kyc
    ON referral_commissions (attribution_id)
    WHERE status = 'pending_kyc';

CREATE TRIGGER referral_commissions_updated_at
    BEFORE UPDATE ON referral_commissions
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();

COMMENT ON TABLE referral_commissions IS
    'Apporteur payouts, one row per milestone payment. Created before the Stripe transfer to act as DB-level idempotency. Supports pending_kyc parking when the referrer has no Stripe Connect account yet.';

COMMIT;
