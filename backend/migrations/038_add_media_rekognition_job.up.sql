ALTER TABLE media ADD COLUMN rekognition_job_id TEXT;
CREATE INDEX idx_media_rekognition_job ON media(rekognition_job_id) WHERE rekognition_job_id IS NOT NULL;
