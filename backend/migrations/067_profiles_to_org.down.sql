-- Best-effort rollback: recreate the user_id column and backfill from
-- the current organization's owner. Operator rows deleted by the up
-- migration cannot be recovered.
BEGIN;

ALTER TABLE profiles ADD COLUMN user_id UUID;

UPDATE profiles p
SET    user_id = o.owner_user_id
FROM   organizations o
WHERE  p.organization_id = o.id;

ALTER TABLE profiles ALTER COLUMN user_id SET NOT NULL;

ALTER TABLE profiles DROP CONSTRAINT profiles_pkey;
ALTER TABLE profiles ADD PRIMARY KEY (user_id);
ALTER TABLE profiles
    ADD CONSTRAINT profiles_user_id_fkey
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE profiles DROP COLUMN organization_id;

COMMIT;
