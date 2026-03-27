-- Jobs: public announcements created by enterprises and agencies.
CREATE TABLE IF NOT EXISTS jobs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    creator_id      UUID NOT NULL REFERENCES users(id),
    title           TEXT NOT NULL,
    description     TEXT NOT NULL,
    skills          TEXT[] NOT NULL DEFAULT '{}',
    applicant_type  TEXT NOT NULL DEFAULT 'all',
    budget_type     TEXT NOT NULL,
    min_budget      INTEGER NOT NULL,
    max_budget      INTEGER NOT NULL,
    status          TEXT NOT NULL DEFAULT 'open',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    closed_at       TIMESTAMPTZ
);

CREATE TRIGGER jobs_updated_at
    BEFORE UPDATE ON jobs
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();

CREATE INDEX idx_jobs_creator ON jobs(creator_id);
CREATE INDEX idx_jobs_status ON jobs(status);
CREATE INDEX idx_jobs_status_created ON jobs(status, created_at DESC);
