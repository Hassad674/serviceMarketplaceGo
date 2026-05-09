BEGIN;

DROP INDEX IF EXISTS idx_job_applications_job_kind;

ALTER TABLE job_applications DROP COLUMN IF EXISTS applicant_kind;

COMMIT;
