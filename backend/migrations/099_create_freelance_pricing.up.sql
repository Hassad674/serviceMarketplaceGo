-- 099_create_freelance_pricing.up.sql
--
-- Creates the freelance_pricing table — pricing rows for the freelance
-- persona of a provider_personal organization. At most one row per
-- profile (PK on profile_id), allowing the four freelance pricing
-- types: daily, hourly, project_from, project_range.
--
-- Separated from the legacy profile_pricing table so the new split
-- aggregate owns its own pricing cadence. CHECK constraints enforce
-- a coarse type shape; the Go domain layer enforces the richer
-- min/max/currency invariants and the type-vs-profile compatibility.

BEGIN;

CREATE TABLE freelance_pricing (
    profile_id    UUID        PRIMARY KEY REFERENCES freelance_profiles(id) ON DELETE CASCADE,
    pricing_type  TEXT        NOT NULL CHECK (pricing_type IN (
        'daily',
        'hourly',
        'project_from',
        'project_range'
    )),
    min_amount    BIGINT      NOT NULL,
    max_amount    BIGINT,
    currency      TEXT        NOT NULL DEFAULT 'EUR',
    pricing_note  TEXT        NOT NULL DEFAULT '',
    negotiable    BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_freelance_pricing_type ON freelance_pricing (pricing_type);

CREATE TRIGGER freelance_pricing_updated_at
    BEFORE UPDATE ON freelance_pricing
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();

COMMENT ON TABLE freelance_pricing IS
    'Pricing row for the freelance persona of a provider_personal organization. One row per freelance_profiles row. Allowed types: daily, hourly, project_from, project_range. min_amount is cents, currency is ISO 4217.';

COMMIT;
