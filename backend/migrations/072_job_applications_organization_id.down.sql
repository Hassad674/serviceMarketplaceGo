BEGIN;

DROP INDEX IF EXISTS idx_job_applications_applicant_organization_id;

ALTER TABLE job_applications
    DROP COLUMN applicant_organization_id;

COMMIT;
