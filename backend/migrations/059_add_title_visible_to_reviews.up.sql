-- Clients can opt out of showing the mission title alongside their review
-- on the provider's public project history. Default is TRUE (visible) which
-- matches the previous behaviour where no title was shown at all (the new
-- display will gracefully degrade to "title only when the client consented").
ALTER TABLE reviews
    ADD COLUMN IF NOT EXISTS title_visible BOOLEAN NOT NULL DEFAULT true;
