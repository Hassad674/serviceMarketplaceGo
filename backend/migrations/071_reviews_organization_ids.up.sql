-- Phase R3 extended — Scope reviews to organizations
--
-- A review is left by one org on another (the client reviews the
-- provider after a mission completes). Adds reviewer_organization_id
-- and reviewed_organization_id denormalized from reviewer_id and
-- reviewed_id so the org's aggregate rating and public profile can be
-- computed without chasing users every time.

BEGIN;

ALTER TABLE reviews
    ADD COLUMN reviewer_organization_id UUID REFERENCES organizations(id) ON DELETE SET NULL,
    ADD COLUMN reviewed_organization_id UUID REFERENCES organizations(id) ON DELETE SET NULL;

UPDATE reviews r
SET    reviewer_organization_id = u.organization_id
FROM   users u
WHERE  r.reviewer_id = u.id
  AND  u.organization_id IS NOT NULL;

UPDATE reviews r
SET    reviewed_organization_id = u.organization_id
FROM   users u
WHERE  r.reviewed_id = u.id
  AND  u.organization_id IS NOT NULL;

DO $$
DECLARE
    orphans integer;
BEGIN
    SELECT COUNT(*) INTO orphans
    FROM   reviews
    WHERE  reviewer_organization_id IS NULL
       OR  reviewed_organization_id IS NULL;
    IF orphans > 0 THEN
        RAISE EXCEPTION 'migration 071 left % reviews without one of the org columns', orphans;
    END IF;
END $$;

ALTER TABLE reviews ALTER COLUMN reviewer_organization_id SET NOT NULL;
ALTER TABLE reviews ALTER COLUMN reviewed_organization_id SET NOT NULL;

CREATE INDEX idx_reviews_reviewed_organization_id
    ON reviews (reviewed_organization_id, created_at DESC, id DESC)
    WHERE moderation_status <> 'hidden';

CREATE INDEX idx_reviews_reviewer_organization_id
    ON reviews (reviewer_organization_id);

COMMIT;
