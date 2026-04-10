-- Phase 4 — Scope payment_records to organizations
--
-- Payment records track the money movement between a client (the
-- business side) and a provider (the freelance side). The org is the
-- client's organization, mirroring the proposal migration.
--
-- Finance-permitted members (Owner, Admin with wallet.view) need to
-- see the org's payment history via the dashboard; this backfill
-- makes that possible for historical records.

BEGIN;

ALTER TABLE payment_records
    ADD COLUMN IF NOT EXISTS organization_id UUID REFERENCES organizations(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_payment_records_organization_id
    ON payment_records(organization_id)
    WHERE organization_id IS NOT NULL;

-- Composite index for (org, created_at) list queries with cursor pagination.
CREATE INDEX IF NOT EXISTS idx_payment_records_org_created
    ON payment_records(organization_id, created_at DESC, id DESC)
    WHERE organization_id IS NOT NULL;

UPDATE payment_records p
SET organization_id = u.organization_id
FROM users u
WHERE p.client_id = u.id
  AND u.organization_id IS NOT NULL
  AND p.organization_id IS NULL;

COMMIT;
