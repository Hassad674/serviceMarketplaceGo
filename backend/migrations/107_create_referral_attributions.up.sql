-- 107_create_referral_attributions.up.sql
--
-- Attribution links a proposal (signed contract between the provider
-- and the client of an active referral) back to that referral, so
-- commissions on its milestones can be routed to the apporteur.
--
-- proposal_id is stored as a bare UUID with NO FOREIGN KEY: the
-- modularity rule (CLAUDE.md "Modularity above all") forbids
-- cross-feature foreign keys. The attribution row exists independently
-- of whether the proposal still exists in its own table.
--
-- UNIQUE(proposal_id) acts as the idempotency guard for the
-- ReferralAttributor.CreateAttributionIfExists port — calling it twice
-- on the same proposal is a no-op.

BEGIN;

CREATE TABLE referral_attributions (
    id                  UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    referral_id         UUID         NOT NULL REFERENCES referrals(id) ON DELETE RESTRICT,
    proposal_id         UUID         NOT NULL UNIQUE,
    provider_id         UUID         NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    client_id           UUID         NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    rate_pct_snapshot   NUMERIC(5,2) NOT NULL CHECK (rate_pct_snapshot >= 0 AND rate_pct_snapshot <= 50),
    attributed_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),

    CONSTRAINT referral_attributions_no_self CHECK (provider_id <> client_id)
);

CREATE INDEX idx_referral_attributions_referral
    ON referral_attributions (referral_id);

CREATE INDEX idx_referral_attributions_provider_client
    ON referral_attributions (provider_id, client_id);

COMMENT ON TABLE referral_attributions IS
    'Links a proposal (signed contract) back to the referral that introduced its parties. proposal_id has no FK because cross-feature foreign keys are forbidden by the modularity rule.';

COMMIT;
