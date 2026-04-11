BEGIN;

DROP INDEX IF EXISTS idx_disputes_provider_organization_id;
DROP INDEX IF EXISTS idx_disputes_client_organization_id;

ALTER TABLE disputes
    DROP COLUMN provider_organization_id,
    DROP COLUMN client_organization_id;

COMMIT;
