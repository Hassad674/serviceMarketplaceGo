-- Revert: point users.organization_id FK back to users(id)
-- Note: any rows populated after migration 057 was applied will become orphaned
-- references if the target organization no longer exists when this is run.
-- In practice, this down migration should only be used when no data exists yet.

ALTER TABLE users DROP CONSTRAINT IF EXISTS users_organization_id_fkey;

ALTER TABLE users
    ADD CONSTRAINT users_organization_id_fkey
    FOREIGN KEY (organization_id) REFERENCES users(id) ON DELETE SET NULL;
