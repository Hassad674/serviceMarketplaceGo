-- Phase R2 — Portfolio items belong to the organization
--
-- A provider's portfolio is the ORG's collection of work samples;
-- invited operators collaborate on the same portfolio. Moves the
-- anchor column from user_id to organization_id.

BEGIN;

ALTER TABLE portfolio_items
    ADD COLUMN organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE;

UPDATE portfolio_items pi
SET    organization_id = u.organization_id
FROM   users u
WHERE  pi.user_id = u.id;

DO $$
DECLARE
    orphans integer;
BEGIN
    SELECT COUNT(*) INTO orphans
    FROM   portfolio_items
    WHERE  organization_id IS NULL;
    IF orphans > 0 THEN
        RAISE EXCEPTION 'migration 068 left % portfolio_items without an org', orphans;
    END IF;
END $$;

ALTER TABLE portfolio_items ALTER COLUMN organization_id SET NOT NULL;

DROP INDEX IF EXISTS idx_portfolio_items_user_id;
DROP INDEX IF EXISTS idx_portfolio_items_user_position;

ALTER TABLE portfolio_items DROP CONSTRAINT portfolio_items_user_id_fkey;
ALTER TABLE portfolio_items DROP COLUMN user_id;

CREATE INDEX idx_portfolio_items_org_id      ON portfolio_items (organization_id);
CREATE INDEX idx_portfolio_items_org_position ON portfolio_items (organization_id, position);

COMMIT;
