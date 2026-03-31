CREATE TABLE IF NOT EXISTS job_applications (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id          UUID NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    applicant_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    message         TEXT NOT NULL DEFAULT '',
    video_url       TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_job_application UNIQUE (job_id, applicant_id)
);

CREATE INDEX idx_job_applications_job ON job_applications(job_id, created_at DESC);
CREATE INDEX idx_job_applications_applicant ON job_applications(applicant_id, created_at DESC);

CREATE TRIGGER job_applications_updated_at
    BEFORE UPDATE ON job_applications
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();
