-- Phase R3 extended — Scope job applications to the applying org
--
-- When an org applies to a job, the whole team owns that application.
-- Every operator of the org sees it in their list and can coordinate
-- a response. Adds applicant_organization_id denormalized from
-- applicant_id → users.organization_id.

BEGIN;

ALTER TABLE job_applications
    ADD COLUMN applicant_organization_id UUID REFERENCES organizations(id) ON DELETE SET NULL;

UPDATE job_applications ja
SET    applicant_organization_id = u.organization_id
FROM   users u
WHERE  ja.applicant_id = u.id
  AND  u.organization_id IS NOT NULL;

DO $$
DECLARE
    orphans integer;
BEGIN
    SELECT COUNT(*) INTO orphans
    FROM   job_applications
    WHERE  applicant_organization_id IS NULL;
    IF orphans > 0 THEN
        RAISE EXCEPTION 'migration 072 left % job_applications without an org', orphans;
    END IF;
END $$;

ALTER TABLE job_applications ALTER COLUMN applicant_organization_id SET NOT NULL;

CREATE INDEX idx_job_applications_applicant_organization_id
    ON job_applications (applicant_organization_id, created_at DESC);

COMMIT;
