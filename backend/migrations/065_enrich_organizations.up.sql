-- Phase R1 — Enrich organizations with data that belongs to the team
--
-- The Stripe Dashboard model requires every resource that a team of
-- operators collectively owns to live on the organization, not the
-- individual user. This migration moves three families of fields onto
-- organizations:
--
--   1. Display name                — organizations.name
--   2. Stripe Connect account      — stripe_account_id, country, state
--   3. KYC enforcement bookkeeping — kyc_first_earning_at,
--                                    kyc_restriction_notified_at
--
-- Data is COPIED (not moved) in this migration. The matching columns
-- on users are dropped later in phase R5 once all read paths have
-- switched to the org columns.
--
-- A new org type `provider_personal` is also introduced — it is the
-- auto-created org for every solo user (providers, admins) so that
-- invited operators can join them under the same Stripe Dashboard
-- semantics as agencies and enterprises.

BEGIN;

-- 1. Add the new columns. NOT NULL with DEFAULT so the ALTER is
--    instant on small tables. Empty-string / '{}' defaults are
--    overwritten by the backfill below.
ALTER TABLE organizations
    ADD COLUMN name                         TEXT        NOT NULL DEFAULT '',
    ADD COLUMN stripe_account_id            TEXT,
    ADD COLUMN stripe_account_country       TEXT,
    ADD COLUMN stripe_last_state            JSONB,
    ADD COLUMN kyc_first_earning_at         TIMESTAMPTZ,
    ADD COLUMN kyc_restriction_notified_at  JSONB       NOT NULL DEFAULT '{}'::jsonb;

-- 2. Widen the type check to include provider_personal.
ALTER TABLE organizations DROP CONSTRAINT organizations_type_check;
ALTER TABLE organizations
    ADD CONSTRAINT organizations_type_check
    CHECK (type IN ('agency', 'enterprise', 'provider_personal'));

-- 3. Backfill name from the current owner's first/last name. For
--    agencies/enterprises the operator will usually rename the org
--    to their company afterwards; for provider_personal orgs the
--    personal name is already the right display name.
UPDATE organizations o
SET    name       = TRIM(u.first_name || ' ' || u.last_name),
       updated_at = now()
FROM   users u
WHERE  o.owner_user_id = u.id
  AND  o.name = '';

-- 4. Copy Stripe + KYC data from the owner onto the org. Only owners
--    that already started KYC have a non-NULL stripe_account_id, so
--    this touches a subset of rows.
UPDATE organizations o
SET    stripe_account_id           = u.stripe_account_id,
       stripe_account_country      = u.stripe_account_country,
       stripe_last_state           = u.stripe_last_state,
       kyc_first_earning_at        = u.kyc_first_earning_at,
       kyc_restriction_notified_at = COALESCE(u.kyc_restriction_notified_at, '{}'::jsonb),
       updated_at                  = now()
FROM   users u
WHERE  o.owner_user_id = u.id
  AND  u.stripe_account_id IS NOT NULL;

-- 5. Unique + query indexes on the moved columns.
CREATE UNIQUE INDEX idx_organizations_stripe_account_id
    ON organizations (stripe_account_id)
    WHERE stripe_account_id IS NOT NULL;

CREATE INDEX idx_organizations_kyc_enforcement
    ON organizations (kyc_first_earning_at)
    WHERE kyc_first_earning_at IS NOT NULL
      AND stripe_account_id IS NULL;

-- 6. Index on name for org search (admin panel + future marketplace search).
CREATE INDEX idx_organizations_name ON organizations (lower(name));

COMMIT;
