DROP INDEX IF EXISTS idx_media_rekognition_job;
ALTER TABLE media DROP COLUMN IF EXISTS rekognition_job_id;
