-- Stripe Connect account information lives directly on the users table
-- since a user has at most one Stripe account (1-1 relation). This replaces
-- the test_embedded_accounts table and the custom KYC payment_info table,
-- both of which mixed this identifier with other concerns.
--
-- Columns:
--   stripe_account_id      — Stripe Custom connected account ID (acct_*)
--   stripe_account_country — ISO 3166-1 alpha-2 country code at creation
--   stripe_last_state      — JSONB snapshot used by the embedded Notifier
--                            to diff against incoming webhooks

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS stripe_account_id      TEXT UNIQUE,
    ADD COLUMN IF NOT EXISTS stripe_account_country TEXT,
    ADD COLUMN IF NOT EXISTS stripe_last_state      JSONB;

-- Partial index: only users who actually have a Stripe account
CREATE INDEX IF NOT EXISTS idx_users_stripe_account_id
    ON users(stripe_account_id)
    WHERE stripe_account_id IS NOT NULL;

-- Migrate existing data from test_embedded_accounts (if any rows exist)
UPDATE users u
SET
    stripe_account_id      = tea.stripe_account_id,
    stripe_account_country = tea.country,
    stripe_last_state      = tea.last_state
FROM test_embedded_accounts tea
WHERE u.id = tea.user_id
  AND u.stripe_account_id IS NULL;
