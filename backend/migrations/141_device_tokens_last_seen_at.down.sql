-- 141_device_tokens_last_seen_at.down.sql

BEGIN;

DROP INDEX IF EXISTS idx_device_tokens_last_seen_at;

ALTER TABLE device_tokens
    DROP COLUMN IF EXISTS last_seen_at;

COMMIT;
