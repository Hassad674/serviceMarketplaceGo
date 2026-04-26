package postgres

import "fmt"

// queryUpsertModerationResult inserts a moderation verdict OR replaces
// the existing row for the (content_type, content_id) pair. The
// excluded.* references read the values from the conflicting input
// row — the previous row's reviewed_by/reviewed_at are explicitly
// reset because a fresh decision means the admin's previous override
// no longer applies.
const queryUpsertModerationResult = `
INSERT INTO moderation_results (
    id, content_type, content_id, author_user_id,
    status, score, labels, reason, decided_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
ON CONFLICT (content_type, content_id) DO UPDATE SET
    status      = excluded.status,
    score       = excluded.score,
    labels      = excluded.labels,
    reason      = excluded.reason,
    decided_at  = excluded.decided_at,
    reviewed_by = NULL,
    reviewed_at = NULL`

// queryGetModerationResultByContent fetches the single decision row
// keyed by (content_type, content_id). Used by the admin handler when
// it needs the latest verdict for a specific content reference.
const queryGetModerationResultByContent = `
SELECT id, content_type, content_id, author_user_id, status, score,
       labels::text, reason, decided_at, reviewed_by, reviewed_at
  FROM moderation_results
 WHERE content_type = $1 AND content_id = $2`

// queryMarkModerationReviewed records an admin override on an existing
// decision. Updates status, reviewed_by and reviewed_at atomically;
// the previous status is captured by the calling app service via
// audit_logs before this call.
const queryMarkModerationReviewed = `
UPDATE moderation_results
   SET status = $1,
       reviewed_by = $2,
       reviewed_at = now()
 WHERE content_type = $3 AND content_id = $4`

// buildModerationResultsListQuery composes the paginated SELECT used
// by the admin queue. limit/offset placeholders are appended after
// the existing WHERE args (so they get the next two slots). Kept as
// a string-builder rather than a const because the WHERE clause is
// dynamic — the alternative (always-on no-op WHERE) reads worse and
// confuses query planners.
func buildModerationResultsListQuery(whereSQL, orderSQL string, limit, offset, argCount int) string {
	return fmt.Sprintf(`
SELECT id, content_type, content_id, author_user_id, status, score,
       labels::text, reason, decided_at, reviewed_by, reviewed_at
  FROM moderation_results
  %s
  %s
  LIMIT %d OFFSET %d`, whereSQL, orderSQL, limit, offset)
}

// buildModerationResultsCountQuery mirrors the list query without
// limit/offset/order so the admin UI can render a "X results" header.
// Re-uses the same WHERE so a content_type+status filter affects both
// list and count consistently.
func buildModerationResultsCountQuery(whereSQL string) string {
	return fmt.Sprintf(`SELECT COUNT(*) FROM moderation_results %s`, whereSQL)
}
