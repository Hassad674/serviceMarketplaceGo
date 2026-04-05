-- Revert target_type CHECK constraint to original values.
ALTER TABLE reports DROP CONSTRAINT IF EXISTS reports_target_type_check;

ALTER TABLE reports ADD CONSTRAINT reports_target_type_check
    CHECK (target_type IN ('message', 'user'));

-- Revert reason CHECK constraint to original values.
ALTER TABLE reports DROP CONSTRAINT IF EXISTS reports_reason_check;

ALTER TABLE reports ADD CONSTRAINT reports_reason_check
    CHECK (reason IN (
        'harassment', 'fraud', 'off_platform_payment', 'spam',
        'inappropriate_content', 'fake_profile', 'unprofessional_behavior', 'other'
    ));
