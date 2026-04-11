BEGIN;

DROP INDEX IF EXISTS idx_reviews_reviewer_organization_id;
DROP INDEX IF EXISTS idx_reviews_reviewed_organization_id;

ALTER TABLE reviews
    DROP COLUMN reviewed_organization_id,
    DROP COLUMN reviewer_organization_id;

COMMIT;
