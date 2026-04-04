-- Add 'job' and 'job_application' to the reports target_type CHECK constraint.
-- Also add new report reasons needed for job/application reports.

-- Step 1: drop the old constraint
ALTER TABLE reports DROP CONSTRAINT IF EXISTS reports_target_type_check;

-- Step 2: add the new constraint with expanded values
ALTER TABLE reports ADD CONSTRAINT reports_target_type_check
    CHECK (target_type IN ('message', 'user', 'job', 'job_application'));

-- Step 3: expand the reason constraint to include new reasons
ALTER TABLE reports DROP CONSTRAINT IF EXISTS reports_reason_check;

ALTER TABLE reports ADD CONSTRAINT reports_reason_check
    CHECK (reason IN (
        'harassment', 'fraud', 'off_platform_payment', 'spam',
        'inappropriate_content', 'fake_profile', 'unprofessional_behavior',
        'misleading_description', 'fraud_or_scam', 'other'
    ));
