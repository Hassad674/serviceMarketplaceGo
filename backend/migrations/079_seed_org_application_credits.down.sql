-- 079_seed_org_application_credits.down.sql
--
-- No-op. Reverting a seed-only migration would be destructive — we
-- would have to set the application_credits column back to 0 for
-- every row we touched, which is impossible to do safely since the
-- up migration did not record which rows it changed, and the target
-- orgs may have earned bonus credits or debited the pool in the
-- meantime.
--
-- The up migration is idempotent-safe (running it twice has no effect
-- once every org is above zero) and loses no state on rollback-and-
-- reapply, so an empty down migration is the correct contract here.

SELECT 1;
