-- 101_migrate_provider_personal_to_freelance_profiles.down.sql
--
-- Deletes the rows previously inserted by the up migration. Lossless
-- when run in combination with the 103_drop_* down, because the
-- source data still lives on profiles (the drop only runs in 103).

BEGIN;

DELETE FROM freelance_profiles
WHERE organization_id IN (
    SELECT id FROM organizations WHERE type = 'provider_personal'
);

COMMIT;
