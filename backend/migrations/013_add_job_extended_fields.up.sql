-- Add long-term budget fields, description type, and video URL to jobs table.
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS payment_frequency TEXT;
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS duration_weeks INTEGER;
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS is_indefinite BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS description_type TEXT NOT NULL DEFAULT 'text';
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS video_url TEXT;

-- Allow description to be nullable for video-only jobs.
ALTER TABLE jobs ALTER COLUMN description DROP NOT NULL;
