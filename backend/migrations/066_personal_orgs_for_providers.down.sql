-- Best-effort rollback: delete the provider_personal orgs created by
-- migration 066 and unlink their owners.
BEGIN;

UPDATE users u
SET    organization_id = NULL,
       updated_at      = now()
FROM   organizations o
WHERE  u.organization_id = o.id
  AND  o.type = 'provider_personal';

DELETE FROM organizations
WHERE  type = 'provider_personal';

COMMIT;
