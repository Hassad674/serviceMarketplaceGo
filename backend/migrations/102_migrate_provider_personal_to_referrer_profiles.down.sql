-- 102_migrate_provider_personal_to_referrer_profiles.down.sql

BEGIN;

DELETE FROM referrer_profiles
WHERE organization_id IN (
    SELECT o.id
    FROM organizations o
    JOIN users u ON u.id = o.owner_user_id
    WHERE o.type = 'provider_personal'
      AND u.referrer_enabled = TRUE
);

COMMIT;
