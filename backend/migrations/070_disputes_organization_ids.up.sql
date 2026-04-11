-- Phase R3 extended — Scope disputes to organizations
--
-- A dispute is a conflict between two orgs (the client's team and the
-- provider's team). Adds denormalized client_organization_id and
-- provider_organization_id so any operator of either org can see +
-- handle the dispute, matching the Stripe Dashboard shared-workspace
-- semantics applied to jobs, proposals and conversations in R3 / R4.

BEGIN;

ALTER TABLE disputes
    ADD COLUMN client_organization_id   UUID REFERENCES organizations(id) ON DELETE SET NULL,
    ADD COLUMN provider_organization_id UUID REFERENCES organizations(id) ON DELETE SET NULL;

UPDATE disputes d
SET    client_organization_id = u.organization_id
FROM   users u
WHERE  d.client_id = u.id
  AND  u.organization_id IS NOT NULL;

UPDATE disputes d
SET    provider_organization_id = u.organization_id
FROM   users u
WHERE  d.provider_id = u.id
  AND  u.organization_id IS NOT NULL;

DO $$
DECLARE
    orphans integer;
BEGIN
    SELECT COUNT(*) INTO orphans
    FROM   disputes
    WHERE  client_organization_id IS NULL
       OR  provider_organization_id IS NULL;
    IF orphans > 0 THEN
        RAISE EXCEPTION 'migration 070 left % disputes without one of the org columns', orphans;
    END IF;
END $$;

ALTER TABLE disputes ALTER COLUMN client_organization_id   SET NOT NULL;
ALTER TABLE disputes ALTER COLUMN provider_organization_id SET NOT NULL;

CREATE INDEX idx_disputes_client_organization_id
    ON disputes (client_organization_id, status, last_activity_at DESC);

CREATE INDEX idx_disputes_provider_organization_id
    ON disputes (provider_organization_id, status, last_activity_at DESC);

COMMIT;
