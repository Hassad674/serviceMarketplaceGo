-- 103_split_profile_pricing_into_persona_tables.down.sql
--
-- Removes the persona pricing rows that were derived from provider_
-- personal profile_pricing entries. Lossless as long as migration 104
-- (drop legacy rows) has not been applied — the source data still
-- exists in profile_pricing.

BEGIN;

DELETE FROM freelance_pricing
WHERE profile_id IN (
    SELECT fp.id
    FROM freelance_profiles fp
    JOIN organizations o ON o.id = fp.organization_id
    WHERE o.type = 'provider_personal'
);

DELETE FROM referrer_pricing
WHERE profile_id IN (
    SELECT rp.id
    FROM referrer_profiles rp
    JOIN organizations o ON o.id = rp.organization_id
    WHERE o.type = 'provider_personal'
);

COMMIT;
