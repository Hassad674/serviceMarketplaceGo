-- 153_referral_attributions_ended_at.up.sql
--
-- WALLET-UNIFY item 6: "Terminer l'intro" — proper flow.
--
-- The apporteur ("intro owner") can now end a referral_attribution.
-- After ending:
--   * NEW milestones approved AFTER ended_at MUST NOT generate
--     commissions (gate enforced in commission_distributor).
--   * Milestones approved BEFORE ended_at keep their commission rows
--     untouched — fair to the apporteur for work already delivered.
--
-- A partial index on (referrer_id) WHERE ended_at IS NULL makes the
-- common "list my active intros" query cheap even when the apporteur
-- accumulates many historical attributions over time.
--
-- Idempotent: re-runnable via IF NOT EXISTS / IF EXISTS — safe to
-- replay on a partially-applied state.

BEGIN;

ALTER TABLE referral_attributions
    ADD COLUMN IF NOT EXISTS ended_at TIMESTAMPTZ NULL;

-- Partial index — most rows are active (ended_at IS NULL) so this
-- index stays small AND filters out terminated intros for the wallet's
-- "active attributions" query path.
CREATE INDEX IF NOT EXISTS idx_referral_attributions_active
    ON referral_attributions (referral_id)
    WHERE ended_at IS NULL;

COMMENT ON COLUMN referral_attributions.ended_at IS
    'Set when the apporteur explicitly terminates the intro. Milestones approved on or after this timestamp do not generate commissions. NULL = active.';

COMMIT;
