-- Team Management V1: repoint users.organization_id FK from users(id) to organizations(id)
--
-- The users.organization_id column was defined in migration 001 as a self-reference
-- to users(id), reflecting the old marketplace pattern where the "organization" was
-- conflated with the founder's user account. That column has always been NULL
-- (verified: 0 rows populated it) and is now repurposed to point to the new
-- organizations table.
--
-- For Agency/Enterprise users (marketplace_owner), this column will hold their
-- own organization's id. For operators, it will hold the id of the organization
-- they were invited into. For Providers, it stays NULL.
--
-- The backfill of organization rows + user.organization_id values will happen in
-- the Phase 1 data migration (separate migration file, created when Phase 1 runs).

ALTER TABLE users DROP CONSTRAINT IF EXISTS users_organization_id_fkey;

ALTER TABLE users
    ADD CONSTRAINT users_organization_id_fkey
    FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE SET NULL;
