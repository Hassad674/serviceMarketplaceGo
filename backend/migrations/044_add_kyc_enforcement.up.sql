-- KYC enforcement tracking.
-- kyc_first_earning_at: set once when the first mission completes with funds
--   available for payout (transfer_status = 'pending'). Never cleared.
-- kyc_restriction_notified_at: JSONB tracking which notification tiers have
--   been sent (day0/day3/day7/day14) to avoid duplicate sends.

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS kyc_first_earning_at       TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS kyc_restriction_notified_at JSONB DEFAULT '{}';

-- Partial index for the scheduler: only users with earnings but no KYC
CREATE INDEX IF NOT EXISTS idx_users_kyc_enforcement
    ON users(kyc_first_earning_at)
    WHERE kyc_first_earning_at IS NOT NULL
      AND stripe_account_id IS NULL;
