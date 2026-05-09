-- Job applications · applicant_kind column
--
-- A provider with referrer_enabled=true can apply to a job either as a
-- freelance (do the work themselves) or as an apporteur d'affaires
-- (refer a freelance/agency for a commission). Pure agencies always
-- apply as 'agency'. We persist the chosen kind at apply time so the
-- employer's candidates list can filter (web) and the kind shows in the
-- candidate row pill — without having to recompute it from the
-- applicant's role + referrer_enabled state at read time.
--
-- 'freelance' is the safe default that mirrors today's behaviour — every
-- existing application was submitted as freelance/agency from the
-- applicant's role; this default keeps the historical rows consistent.
-- The CHECK constraint pins the enum surface so a typo never lands.

BEGIN;

ALTER TABLE job_applications
    ADD COLUMN IF NOT EXISTS applicant_kind TEXT NOT NULL DEFAULT 'freelance'
        CHECK (applicant_kind IN ('freelance', 'agency', 'referrer'));

-- Backfill existing rows from the applicant's role: agencies → 'agency',
-- everything else → 'freelance' (the column default already covered the
-- rest, but this UPDATE makes the historical data correct rather than
-- merely defaulted).
UPDATE job_applications ja
SET    applicant_kind = 'agency'
FROM   users u
WHERE  ja.applicant_id = u.id
  AND  u.role = 'agency'
  AND  ja.applicant_kind = 'freelance';

-- Index supports the candidates list filter (`?kind=...`). The composite
-- (job_id, applicant_kind, created_at DESC) keeps the existing
-- (job_id, created_at DESC) ordering free for non-filtered reads.
CREATE INDEX IF NOT EXISTS idx_job_applications_job_kind
    ON job_applications (job_id, applicant_kind, created_at DESC);

COMMIT;
