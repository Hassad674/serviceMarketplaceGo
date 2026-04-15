-- 105_create_referrals.up.sql
--
-- Creates the referrals table — the root aggregate of the apport
-- d'affaires (business referral) feature. One row per introduction
-- between a provider and a client made by a referrer (apporteur).
--
-- Lifecycle: pending_provider → pending_referrer (after provider
-- counter-offer) → pending_client → active → terminal. The referrer
-- and provider negotiate the rate bilaterally (modèle A — provider
-- absorbs the commission); the client only sees Accept/Reject after
-- the rate is locked.
--
-- The unique partial index on (provider_id, client_id) WHERE status is
-- non-terminal enforces "first arrived, first served" exclusivity — a
-- second referral on the same couple is impossible while one is in
-- play. Mapped to ErrCoupleLocked at the application layer.
--
-- intro_snapshot is a JSONB blob frozen at creation time so the
-- counter-party always sees the snapshot that was promised, regardless
-- of subsequent profile edits. Schema versioned via intro_snapshot_version
-- so future shape changes don't break older rows.

BEGIN;

CREATE TABLE referrals (
    id                       UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    referrer_id              UUID        NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    provider_id              UUID        NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    client_id                UUID        NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    rate_pct                 NUMERIC(5,2) NOT NULL CHECK (rate_pct >= 0 AND rate_pct <= 50),
    duration_months          SMALLINT    NOT NULL DEFAULT 6 CHECK (duration_months BETWEEN 1 AND 24),
    intro_snapshot           JSONB       NOT NULL,
    intro_snapshot_version   INTEGER     NOT NULL DEFAULT 1 CHECK (intro_snapshot_version >= 1),
    intro_message_provider   TEXT        NOT NULL DEFAULT '',
    intro_message_client     TEXT        NOT NULL DEFAULT '',
    status                   TEXT        NOT NULL CHECK (status IN (
        'pending_provider', 'pending_referrer', 'pending_client',
        'active', 'rejected', 'expired', 'cancelled', 'terminated'
    )),
    version                  INTEGER     NOT NULL DEFAULT 1 CHECK (version >= 1),
    activated_at             TIMESTAMPTZ,
    expires_at               TIMESTAMPTZ,
    last_action_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    rejection_reason         TEXT        NOT NULL DEFAULT '',
    rejected_by              UUID,
    created_at               TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at               TIMESTAMPTZ NOT NULL DEFAULT now(),

    -- Self-deal guards (mirror domain validation).
    CONSTRAINT referrals_no_self_referral_provider CHECK (referrer_id <> provider_id),
    CONSTRAINT referrals_no_self_referral_client   CHECK (referrer_id <> client_id),
    CONSTRAINT referrals_no_provider_client_overlap CHECK (provider_id <> client_id),

    -- Activation invariants: an active referral must have both stamps.
    CONSTRAINT referrals_active_has_stamps CHECK (
        status <> 'active' OR (activated_at IS NOT NULL AND expires_at IS NOT NULL)
    )
);

-- One non-terminal referral per (provider, client) couple — first arrived
-- wins. Enforced by a partial unique index that lets multiple terminal
-- rows coexist on the same couple (history) while preventing concurrent
-- live referrals.
CREATE UNIQUE INDEX idx_referrals_active_couple_unique
    ON referrals (provider_id, client_id)
    WHERE status IN ('pending_provider', 'pending_referrer', 'pending_client', 'active');

-- Hot-path indexes for the dashboard queries.
CREATE INDEX idx_referrals_referrer_status     ON referrals (referrer_id, status);
CREATE INDEX idx_referrals_provider_status     ON referrals (provider_id, status);
CREATE INDEX idx_referrals_client_status       ON referrals (client_id, status);

-- Cron-friendly indexes used by the daily expirer worker.
CREATE INDEX idx_referrals_active_expiry
    ON referrals (expires_at)
    WHERE status = 'active';

CREATE INDEX idx_referrals_pending_last_action
    ON referrals (last_action_at)
    WHERE status IN ('pending_provider', 'pending_referrer', 'pending_client');

CREATE TRIGGER referrals_updated_at
    BEFORE UPDATE ON referrals
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();

COMMENT ON TABLE referrals IS
    'Business referral (apport d''affaires) introductions. One row per (referrer, provider, client) intro through its full lifecycle from negotiation to activation, with rate, duration window and anonymised intro snapshot.';

COMMIT;
