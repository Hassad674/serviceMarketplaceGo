-- 077_org_role_overrides.down.sql
--
-- Reverts migration 077: drops the role_overrides column from organizations.
-- Any customized permissions are lost — there is no other place where
-- this data lives.

ALTER TABLE organizations
    DROP COLUMN IF EXISTS role_overrides;
