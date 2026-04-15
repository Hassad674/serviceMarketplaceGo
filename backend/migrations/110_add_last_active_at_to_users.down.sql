-- 110_add_last_active_at_to_users.down.sql
DROP INDEX IF EXISTS idx_users_last_active_at;
ALTER TABLE users DROP COLUMN IF EXISTS last_active_at;
