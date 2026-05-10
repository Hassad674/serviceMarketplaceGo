-- 143_create_profile_view_events.down.sql

BEGIN;

DROP INDEX IF EXISTS idx_pve_unique_visitor;
DROP INDEX IF EXISTS idx_pve_org_created;
DROP TABLE IF EXISTS profile_view_events;

COMMIT;
