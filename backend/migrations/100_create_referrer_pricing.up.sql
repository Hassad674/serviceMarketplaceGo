-- 100_create_referrer_pricing.up.sql
--
-- Creates the referrer_pricing table — pricing rows for the referrer
-- persona of a provider_personal organization. At most one row per
-- profile (PK on profile_id), allowing the two referrer pricing types:
-- commission_pct, commission_flat.
--
-- Commission_pct min/max are basis points (0..10000) with currency
-- set to the literal 'pct'. Commission_flat uses cents like any other
-- currency pricing. The Go domain layer enforces these correlations.

BEGIN;

CREATE TABLE referrer_pricing (
    profile_id    UUID        PRIMARY KEY REFERENCES referrer_profiles(id) ON DELETE CASCADE,
    pricing_type  TEXT        NOT NULL CHECK (pricing_type IN (
        'commission_pct',
        'commission_flat'
    )),
    min_amount    BIGINT      NOT NULL,
    max_amount    BIGINT,
    currency      TEXT        NOT NULL DEFAULT 'EUR',
    pricing_note  TEXT        NOT NULL DEFAULT '',
    negotiable    BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_referrer_pricing_type ON referrer_pricing (pricing_type);

CREATE TRIGGER referrer_pricing_updated_at
    BEFORE UPDATE ON referrer_pricing
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();

COMMENT ON TABLE referrer_pricing IS
    'Pricing row for the referrer persona of a provider_personal organization with referrer_enabled=true. One row per referrer_profiles row. Allowed types: commission_pct (min/max in basis points, currency pct) and commission_flat (min in cents, currency ISO 4217).';

COMMIT;
