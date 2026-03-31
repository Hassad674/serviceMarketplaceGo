ALTER TABLE reports DROP CONSTRAINT IF EXISTS reports_reason_check;
ALTER TABLE reports ADD CONSTRAINT reports_reason_check CHECK (reason IN (
    'harassment', 'fraud', 'spam',
    'inappropriate_content', 'fake_profile', 'unprofessional_behavior', 'other'
));
