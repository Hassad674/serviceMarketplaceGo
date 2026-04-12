-- 079_seed_org_application_credits.up.sql
--
-- One-shot backfill that restores the starter application-credit quota
-- for every organization created between migration 075 (which shipped
-- the column with a `DEFAULT 0`) and the auto-seed fix in
-- CreateWithOwnerMembership.
--
-- Background: before R12 the job feature stored credits per user in a
-- dedicated table, and `JobCreditRepository.GetOrCreate(userID)`
-- lazily inserted a row with 10 credits on first read — so every new
-- account could apply to jobs immediately. R12 moved the pool to
-- `organizations.application_credits` but did not seed it, and the
-- `DEFAULT 0` on the column meant every org born after R12 came into
-- the world with zero credits.
--
-- This migration patches those rows so existing teams can apply to
-- jobs again. Future orgs are handled by the auto-seed in
-- `OrganizationRepository.CreateWithOwnerMembership`, and the weekly
-- refill (lazy, on read, inside `JobCreditRepository.GetOrCreate`)
-- keeps everything flowing after that.
--
-- The literal `10` matches the `job.WeeklyQuota` constant in code.
-- Migrations are immutable once applied, so the value is duplicated
-- here by design — if we ever bump the quota we will create a new
-- migration rather than editing this one.
--
-- We also advance `credits_last_reset_at` to `now()` so the weekly
-- refill clock starts from the moment of the backfill, not from
-- whatever stale timestamp the row was carrying.

UPDATE organizations
SET    application_credits   = 10,
       credits_last_reset_at = now(),
       updated_at            = now()
WHERE  application_credits = 0;
