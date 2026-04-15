-- 106_create_referral_negotiations.up.sql
--
-- Append-only audit trail for the bilateral apporteur ↔ provider
-- negotiation. One row per action (proposed, countered, accepted,
-- rejected). The client never produces a row here — they only
-- Accept/Reject the locked terms once the apporteur and provider
-- have agreed (Modèle A).
--
-- Used by the dashboard timeline and by the expirer cron (the
-- referrer's last action timestamp is denormalised on the parent
-- referrals.last_action_at; this table just records the events).

BEGIN;

CREATE TABLE referral_negotiations (
    id            UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    referral_id   UUID         NOT NULL REFERENCES referrals(id) ON DELETE CASCADE,
    version       INTEGER      NOT NULL CHECK (version >= 1),
    actor_id      UUID         NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    actor_role    TEXT         NOT NULL CHECK (actor_role IN ('referrer', 'provider', 'client')),
    action        TEXT         NOT NULL CHECK (action IN ('proposed', 'countered', 'accepted', 'rejected')),
    rate_pct      NUMERIC(5,2) NOT NULL CHECK (rate_pct >= 0 AND rate_pct <= 50),
    message       TEXT         NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX idx_referral_negotiations_referral_created
    ON referral_negotiations (referral_id, created_at DESC);

COMMENT ON TABLE referral_negotiations IS
    'Append-only audit trail of negotiation events for a referral. One row per Accept/Reject/Negotiate action by referrer or provider (clients do not negotiate the rate under Modèle A).';

COMMIT;
