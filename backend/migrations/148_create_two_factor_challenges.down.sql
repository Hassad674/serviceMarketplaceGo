-- Reverses 148_create_two_factor_challenges.up.sql.
DROP INDEX IF EXISTS idx_2fa_user_pending;
DROP TABLE IF EXISTS two_factor_challenges;
ALTER TABLE users DROP COLUMN IF EXISTS two_factor_email_enabled;
