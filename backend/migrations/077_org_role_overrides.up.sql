-- 077_org_role_overrides.up.sql
--
-- Per-organization customization of role permissions.
--
-- The static defaults in backend/internal/domain/organization/permissions.go
-- remain the single source of truth for "out of the box" behavior. This
-- column stores the Owner's edits on top of those defaults:
--
--   {
--     "admin":  {"billing.manage": true},
--     "member": {"jobs.delete": true, "team.invite": true},
--     "viewer": {"messaging.send": true}
--   }
--
-- A key set to true grants a permission that is NOT in the defaults.
-- A key set to false revokes a permission that IS in the defaults.
-- Missing keys follow the default. The Owner row is never customized.
--
-- NOT NULL DEFAULT '{}'::jsonb so existing rows get an empty object and
-- the repository code can unmarshal without null checks. Adding a NOT
-- NULL column with a default is a fast operation in PostgreSQL 11+
-- (no table rewrite) thanks to the metadata-only default optimization.
--
-- No index needed: this column is always read together with the rest of
-- the organization row via FindByID / FindByOwnerUserID. Querying "all
-- orgs where admin can withdraw" is not a supported use case.

ALTER TABLE organizations
    ADD COLUMN IF NOT EXISTS role_overrides JSONB NOT NULL DEFAULT '{}'::jsonb;

COMMENT ON COLUMN organizations.role_overrides IS
    'Per-org role permission overrides on top of static defaults. Owner row is never customized.';
