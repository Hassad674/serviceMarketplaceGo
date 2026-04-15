-- 110_add_last_active_at_to_users.up.sql
--
-- Adds a last_active_at column to users so the search engine can rank
-- "recently active" profiles higher. Populated on login + message-sent
-- (with a 5-minute debounce to avoid reindex storms).
--
-- The column is also used by the Typesense indexer (`signalsFromUser`)
-- to compute the `last_active_at` signal field on each SearchDocument.

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS last_active_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

-- Backfill: every existing user starts "active" at their updated_at (or
-- created_at, or NOW() as a last resort) so the initial bulk reindex has
-- meaningful data instead of every user sharing the exact same timestamp.
UPDATE users
   SET last_active_at = COALESCE(updated_at, created_at, NOW())
 WHERE last_active_at IS NULL
    OR last_active_at = NOW();

CREATE INDEX IF NOT EXISTS idx_users_last_active_at
    ON users(last_active_at DESC);
