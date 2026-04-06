-- Add last_state JSONB column to track the last-seen Stripe account state
-- so the embedded notifier can diff webhooks against the prior state and
-- emit notifications only on meaningful transitions.
ALTER TABLE test_embedded_accounts
    ADD COLUMN IF NOT EXISTS last_state JSONB;
