-- Revert: make description NOT NULL again, drop new columns.
ALTER TABLE jobs ALTER COLUMN description SET NOT NULL;

ALTER TABLE jobs DROP COLUMN IF EXISTS video_url;
ALTER TABLE jobs DROP COLUMN IF EXISTS description_type;
ALTER TABLE jobs DROP COLUMN IF EXISTS is_indefinite;
ALTER TABLE jobs DROP COLUMN IF EXISTS duration_weeks;
ALTER TABLE jobs DROP COLUMN IF EXISTS payment_frequency;
