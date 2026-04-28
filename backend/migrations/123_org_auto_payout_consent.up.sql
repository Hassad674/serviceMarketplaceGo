-- Once a provider successfully completes their first manual payout
-- via the wallet, we record the timestamp so subsequent milestone
-- releases auto-transfer instead of requiring another explicit click.
-- The first payout serves as the consent + the proof that Stripe
-- onboarding actually works for this org. NULL = consent not given,
-- every transfer stays in TransferPending until the provider clicks
-- "Retirer" themselves. Non-null does NOT bypass KYC + billing
-- completeness — those gates are still re-checked at transfer time.
ALTER TABLE organizations
    ADD COLUMN IF NOT EXISTS auto_payout_enabled_at TIMESTAMPTZ;
