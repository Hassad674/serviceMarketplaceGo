-- Team Management V1: add account_type and session_version to users
--
-- account_type distinguishes between:
--   marketplace_owner — user who self-registered as Agency/Enterprise/Provider
--                       (retains their existing marketplace role agency/enterprise/provider)
--   operator         — user who was invited into an existing organization
--                       Inherits the marketplace role of their org's type (agency or enterprise),
--                       so existing queries that filter by role keep working naturally.
--
-- session_version is incremented whenever a user's permissions change (role changed,
-- removed from org, etc.). The JWT carries the session_version at issue time, and the
-- auth middleware compares against the current value in Redis on every request.
-- A mismatch triggers immediate 401 — this is our immediate revocation mechanism.
-- Default 0 means "never changed" — existing users start fresh.

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS account_type TEXT NOT NULL DEFAULT 'marketplace_owner'
        CHECK (account_type IN ('marketplace_owner', 'operator'));

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS session_version INTEGER NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_users_account_type ON users(account_type);
