-- 094_add_pricing_negotiable.up.sql
--
-- Adds an explicit yes/no "negotiable" flag to the profile_pricing
-- table. Prior to this migration the concept only existed as free
-- text in the pricing_note field ("Négociable selon scope..."),
-- which is not machine-readable and cannot be rendered as a
-- distinct "négociable" badge on the profile card.
--
-- Default FALSE keeps existing rows behaviourally identical: a
-- provider that did not declare negotiability stays explicitly
-- non-negotiable until they open the editor and choose.
--
-- IF NOT EXISTS guards against re-application after a prior run
-- registered under version 084 on environments that were migrated
-- before the file was renumbered to 094.

BEGIN;

ALTER TABLE profile_pricing
    ADD COLUMN IF NOT EXISTS negotiable BOOLEAN NOT NULL DEFAULT FALSE;

COMMENT ON COLUMN profile_pricing.negotiable IS
    'Explicit yes/no flag surfaced as a "négociable" badge on the public profile card. Distinct from pricing_note which describes constraints.';

COMMIT;
