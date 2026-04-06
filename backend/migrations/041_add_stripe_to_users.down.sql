-- Rollback: restore test_embedded_accounts from users columns then drop columns.
-- Safe-guard: recreate the table if it was dropped by a later migration.

CREATE TABLE IF NOT EXISTS test_embedded_accounts (
    user_id           UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    stripe_account_id TEXT NOT NULL UNIQUE,
    country           TEXT NOT NULL,
    last_state        JSONB,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO test_embedded_accounts (user_id, stripe_account_id, country, last_state)
SELECT id, stripe_account_id, stripe_account_country, stripe_last_state
FROM users
WHERE stripe_account_id IS NOT NULL
ON CONFLICT (user_id) DO NOTHING;

DROP INDEX IF EXISTS idx_users_stripe_account_id;

ALTER TABLE users
    DROP COLUMN IF EXISTS stripe_account_id,
    DROP COLUMN IF EXISTS stripe_account_country,
    DROP COLUMN IF EXISTS stripe_last_state;
