-- 078_create_audit_logs.down.sql
--
-- Reverts migration 078: drops the audit_logs table.
-- The audit history is lost — this rollback is intended for dev resets
-- only. Production rollbacks should create a corrective migration
-- instead (e.g. 079_archive_audit_logs) to preserve history.

DROP INDEX IF EXISTS idx_audit_logs_resource;
DROP INDEX IF EXISTS idx_audit_logs_created_at;
DROP INDEX IF EXISTS idx_audit_logs_action;
DROP INDEX IF EXISTS idx_audit_logs_user_id;
DROP TABLE IF EXISTS audit_logs;
