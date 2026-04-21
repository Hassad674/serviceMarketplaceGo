-- 114_profiles_client_description.up.sql
--
-- Adds the client-facing facet of the organization's profile: a free-form
-- "client_description" TEXT column on the legacy agency/enterprise profiles
-- row. Mirror of the existing "about" / "referrer_about" columns but
-- scoped to the organization's client persona — the text an enterprise (or
-- an agency acting as a client) puts on its client-facing public page.
--
-- Default is the empty string so every existing row stays valid without a
-- backfill, and so the app layer can load the column with a plain SELECT
-- without COALESCE gymnastics.

ALTER TABLE profiles
    ADD COLUMN IF NOT EXISTS client_description TEXT NOT NULL DEFAULT '';
