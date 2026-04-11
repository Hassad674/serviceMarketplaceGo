-- R12 — Move application credits from per-user to per-organization
--
-- Security fix: before this migration, job application credits were
-- stored per user in a dedicated `application_credits` table. After the
-- team refactor (phases R1–R9), an agency owner could invite N operators
-- and each of them still owned their own 10-credit weekly pool. An agency
-- with 100 invited operators therefore had 100 * 10 = 1000 application
-- credits — trivially bypassing the weekly application rate limit.
--
-- The fix moves credits to the organization level. Every org has a single
-- shared pool. Any operator of the org debits the same pool. Refills,
-- top-ups and bonus credits all credit the org.
--
-- Steps:
--   1. Add `application_credits` and `credits_last_reset_at` columns to
--      `organizations`.
--   2. Backfill: for every org, set `application_credits` to the SUM of
--      the balances held by its members (via users.organization_id).
--   3. Assert every user row with credits > 0 has an organization_id so
--      no balance is silently lost. The team refactor already made
--      `organization_id` non-null for every user, so this is a paranoia
--      check.
--   4. Drop the whole `application_credits` table. It was entirely owned
--      by this feature — no other table references it.
--
-- The migration runs in a single transaction. If the safety assertion
-- trips, the whole thing rolls back and no data is destroyed.

BEGIN;

-- 1. Add the new columns on organizations.
ALTER TABLE organizations
    ADD COLUMN IF NOT EXISTS application_credits INTEGER NOT NULL DEFAULT 0;

ALTER TABLE organizations
    ADD COLUMN IF NOT EXISTS credits_last_reset_at TIMESTAMPTZ NOT NULL DEFAULT now();

-- 2. Backfill — sum each org's members' credits into the org pool.
--    Uses the existing users.organization_id link (team phase R4).
UPDATE organizations o
SET    application_credits = COALESCE(totals.total, 0),
       credits_last_reset_at = COALESCE(totals.last_reset, o.credits_last_reset_at),
       updated_at = now()
FROM (
    SELECT u.organization_id,
           SUM(ac.credits)            AS total,
           MAX(ac.last_reset_at)      AS last_reset
    FROM   application_credits ac
    JOIN   users u ON u.id = ac.user_id
    WHERE  u.organization_id IS NOT NULL
    GROUP BY u.organization_id
) AS totals
WHERE o.id = totals.organization_id;

-- 3. Safety assertion — no credit row should be left behind. If any
--    application_credits row points to a user without an organization,
--    its balance would be silently dropped by the table drop below. We
--    refuse to proceed in that case.
DO $$
DECLARE
    orphan_count   INTEGER;
    orphan_credits INTEGER;
    total_user_credits INTEGER;
    total_org_credits  INTEGER;
BEGIN
    SELECT COUNT(*), COALESCE(SUM(ac.credits), 0)
    INTO   orphan_count, orphan_credits
    FROM   application_credits ac
    LEFT   JOIN users u ON u.id = ac.user_id
    WHERE  u.id IS NULL OR u.organization_id IS NULL;

    IF orphan_count > 0 THEN
        RAISE EXCEPTION
          'migration 075 found % application_credits rows totalling % credits not attached to any organization — refusing to drop the table',
          orphan_count, orphan_credits;
    END IF;

    -- Integrity check — total credits before backfill must match total
    -- credits after backfill so no balance is silently lost or duplicated.
    SELECT COALESCE(SUM(credits), 0)
    INTO   total_user_credits
    FROM   application_credits;

    SELECT COALESCE(SUM(application_credits), 0)
    INTO   total_org_credits
    FROM   organizations;

    IF total_user_credits <> total_org_credits THEN
        RAISE EXCEPTION
          'migration 075 credit sum mismatch: users had % credits, orgs now hold % credits',
          total_user_credits, total_org_credits;
    END IF;
END $$;

-- 4. Drop the per-user credit table. Feature-local — no external FK.
DROP TABLE IF EXISTS application_credits;

COMMIT;
