-- Isolated table for the /test-embedded Stripe Connect test page.
-- Maps a user to a Stripe Custom account ID for the embedded onboarding demo.
-- This table is NOT related to the production payment_info table — it exists
-- solely so the test page can reuse the same Stripe account across reloads.
CREATE TABLE IF NOT EXISTS test_embedded_accounts (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id           UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    stripe_account_id TEXT NOT NULL,
    country           TEXT NOT NULL DEFAULT 'FR',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_test_embedded_accounts_user ON test_embedded_accounts(user_id);
