-- Reverts migration 139_create_consent_log.up.sql.

BEGIN;

DROP INDEX IF EXISTS idx_consent_log_created_at;
DROP INDEX IF EXISTS idx_consent_log_user_id;
DROP TABLE IF EXISTS consent_log;

COMMIT;
