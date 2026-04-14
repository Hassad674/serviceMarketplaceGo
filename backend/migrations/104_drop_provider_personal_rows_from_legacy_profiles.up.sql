-- 104_drop_provider_personal_rows_from_legacy_profiles.up.sql
--
-- Final step of the split: remove the provider_personal rows from the
-- legacy profile_pricing and profiles tables. The data has already
-- been copied to the persona-specific tables by migrations 101-103.
--
-- The legacy profiles / profile_pricing tables remain in the schema —
-- they continue to own agency rows until the agency refactor (follow-
-- up). The referrer_* columns on profiles are NOT dropped; they
-- become legacy dead weight for agency orgs and will be cleaned up
-- in a later migration.

BEGIN;

DELETE FROM profile_pricing
WHERE organization_id IN (
    SELECT id FROM organizations WHERE type = 'provider_personal'
);

DELETE FROM profiles
WHERE organization_id IN (
    SELECT id FROM organizations WHERE type = 'provider_personal'
);

COMMIT;
