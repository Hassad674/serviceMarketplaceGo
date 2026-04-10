DROP INDEX IF EXISTS idx_users_account_type;
ALTER TABLE users DROP COLUMN IF EXISTS session_version;
ALTER TABLE users DROP COLUMN IF EXISTS account_type;
