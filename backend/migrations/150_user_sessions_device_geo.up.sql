-- SEC-SESSIONS: enrich user_sessions with parsed device + geo columns
--
-- The B.4 user_sessions table only stores a SHA-256 hash of the User-Agent
-- and an anonymized IP (/24), which is correct for forensics but useless
-- for the user-facing "Sécurité" page. To render a Malt-style row like
--   "Ordinateur de bureau (Chrome) — Paris — 11/05/2026 10:48:46 — 1.2.3.x"
-- we persist the parsed UA + geo lookup result at session creation time.
--
-- All new columns default to '' so the migration is safe for existing rows.
-- The down migration drops the columns.

ALTER TABLE user_sessions
    ADD COLUMN IF NOT EXISTS device_label TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS browser      TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS os           TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS city         TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS country_code TEXT NOT NULL DEFAULT '';
