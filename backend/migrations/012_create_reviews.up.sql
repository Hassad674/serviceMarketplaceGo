CREATE TABLE IF NOT EXISTS reviews (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    proposal_id     UUID NOT NULL REFERENCES proposals(id),
    reviewer_id     UUID NOT NULL REFERENCES users(id),
    reviewed_id     UUID NOT NULL REFERENCES users(id),
    global_rating   SMALLINT NOT NULL CHECK (global_rating BETWEEN 1 AND 5),
    timeliness      SMALLINT CHECK (timeliness BETWEEN 1 AND 5),
    communication   SMALLINT CHECK (communication BETWEEN 1 AND 5),
    quality         SMALLINT CHECK (quality BETWEEN 1 AND 5),
    comment         TEXT DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(proposal_id, reviewer_id)
);

CREATE INDEX IF NOT EXISTS idx_reviews_reviewed ON reviews(reviewed_id);
CREATE INDEX IF NOT EXISTS idx_reviews_reviewer ON reviews(reviewer_id);
CREATE INDEX IF NOT EXISTS idx_reviews_proposal ON reviews(proposal_id);
CREATE INDEX IF NOT EXISTS idx_reviews_created_at ON reviews(created_at DESC, id DESC);
