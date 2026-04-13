-- 082_double_blind_reviews.up.sql
--
-- Introduces double-blind reviews: both parties of a completed proposal can
-- review each other, but neither side sees the other's review until both have
-- submitted OR 14 days have elapsed since the proposal was completed. Once
-- revealed, reviews become immutable (enforced at the service layer).
--
-- Columns added:
--   - side          — which direction the review goes (client→provider or
--                     provider→client). Historically all rows were the former,
--                     so the backfill sets them to 'client_to_provider'.
--   - published_at  — NULL while the review is hidden (awaiting the
--                     counterpart or the 14-day deadline), set to the reveal
--                     moment otherwise. Historical rows are backfilled to
--                     created_at so they remain visible on public profiles.
--
-- Constraints:
--   - side must be one of the two known values
--   - provider→client reviews cannot carry provider-side sub-criteria
--     (timeliness / communication / quality) — those are specific to evaluating
--     the delivery work, not the client relationship

BEGIN;

ALTER TABLE reviews
    ADD COLUMN side TEXT NOT NULL DEFAULT 'client_to_provider'
        CHECK (side IN ('client_to_provider', 'provider_to_client')),
    ADD COLUMN published_at TIMESTAMPTZ NULL;

-- Provider→client reviews cannot carry provider-specific sub-criteria.
ALTER TABLE reviews
    ADD CONSTRAINT reviews_provider_side_no_subcriteria
    CHECK (
        side = 'client_to_provider'
        OR (timeliness IS NULL AND communication IS NULL AND quality IS NULL)
    );

-- Backfill existing rows: they were all client→provider and always visible.
UPDATE reviews SET published_at = created_at WHERE published_at IS NULL;

-- Drop the default so new rows must specify side explicitly.
ALTER TABLE reviews ALTER COLUMN side DROP DEFAULT;

-- Hot index for public profile reads (index-only scan target): the list
-- endpoint filters on reviewed_id + side='client_to_provider' + visible.
CREATE INDEX idx_reviews_public_profile
    ON reviews(reviewed_id, published_at DESC)
    WHERE side = 'client_to_provider' AND published_at IS NOT NULL;

-- Tiny index for pending reviews (auto-publish candidates + dashboards).
CREATE INDEX idx_reviews_pending
    ON reviews(proposal_id)
    WHERE published_at IS NULL;

COMMIT;
