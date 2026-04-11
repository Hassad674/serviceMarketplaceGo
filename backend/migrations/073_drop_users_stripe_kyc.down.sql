-- Reverses 073 — restores the legacy columns on users. Intended for
-- emergency rollback only; the data is not re-copied from organizations.
BEGIN;

ALTER TABLE users
    ADD COLUMN stripe_account_id           TEXT,
    ADD COLUMN stripe_account_country      TEXT,
    ADD COLUMN stripe_last_state           JSONB,
    ADD COLUMN kyc_first_earning_at        TIMESTAMPTZ,
    ADD COLUMN kyc_restriction_notified_at JSONB NOT NULL DEFAULT '{}'::jsonb;

ALTER TABLE users
    ADD CONSTRAINT users_stripe_account_id_key UNIQUE (stripe_account_id);

CREATE INDEX idx_users_stripe_account_id
    ON users (stripe_account_id)
    WHERE stripe_account_id IS NOT NULL;

CREATE INDEX idx_users_kyc_enforcement
    ON users (kyc_first_earning_at)
    WHERE kyc_first_earning_at IS NOT NULL AND stripe_account_id IS NULL;

COMMIT;
