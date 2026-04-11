-- Phase R1 — Personal orgs for every user without one
--
-- Before the Stripe Dashboard refactor, provider and admin users
-- existed without an organization record (phase 4 only backfilled
-- agencies and enterprises because V1 assumed providers were strictly
-- solo). The org-primary model requires every marketplace actor to
-- work through an org so that invited operators can join them.
--
-- For each user without an organization this migration:
--   1. Creates a provider_personal org owned by them.
--   2. Copies their Stripe + KYC data onto the org.
--   3. Creates an organization_members row with role='owner'.
--   4. Sets users.organization_id to the new org.
--
-- The whole thing runs in one transaction. Idempotent — re-running
-- finds no users without orgs and does nothing.

BEGIN;

WITH
    users_without_org AS (
        SELECT id,
               first_name,
               last_name,
               stripe_account_id,
               stripe_account_country,
               stripe_last_state,
               kyc_first_earning_at,
               kyc_restriction_notified_at
        FROM   users
        WHERE  organization_id IS NULL
        FOR UPDATE
    ),
    new_orgs AS (
        INSERT INTO organizations (
            id, owner_user_id, type, name,
            stripe_account_id, stripe_account_country, stripe_last_state,
            kyc_first_earning_at, kyc_restriction_notified_at,
            created_at, updated_at
        )
        SELECT
            gen_random_uuid(),
            u.id,
            'provider_personal',
            TRIM(u.first_name || ' ' || u.last_name),
            u.stripe_account_id,
            u.stripe_account_country,
            u.stripe_last_state,
            u.kyc_first_earning_at,
            COALESCE(u.kyc_restriction_notified_at, '{}'::jsonb),
            now(), now()
        FROM users_without_org u
        RETURNING id AS org_id, owner_user_id
    ),
    new_members AS (
        INSERT INTO organization_members (
            id, organization_id, user_id, role, title,
            joined_at, created_at, updated_at
        )
        SELECT gen_random_uuid(), n.org_id, n.owner_user_id,
               'owner', '', now(), now(), now()
        FROM new_orgs n
        RETURNING organization_id, user_id
    )
UPDATE users u
SET    organization_id = nm.organization_id,
       updated_at      = now()
FROM   new_members nm
WHERE  u.id = nm.user_id;

-- Sanity assertion — every user must have an org after this point.
DO $$
DECLARE
    orphan_count integer;
BEGIN
    SELECT COUNT(*) INTO orphan_count
    FROM   users
    WHERE  organization_id IS NULL;

    IF orphan_count > 0 THEN
        RAISE EXCEPTION 'migration 066 left % users without an organization', orphan_count;
    END IF;
END $$;

COMMIT;
