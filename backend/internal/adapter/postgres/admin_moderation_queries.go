package postgres

import (
	"fmt"
	"strings"

	"marketplace-backend/internal/port/repository"
)

// buildModerationUnionQuery constructs a UNION ALL query across reports,
// flagged messages, flagged reviews, and flagged media.
// Each sub-query returns the same column shape for ModerationItem.
func buildModerationUnionQuery(filters repository.ModerationFilters) (string, []any) {
	var subQueries []string
	var args []any
	paramIdx := 1

	wantSource := filters.Source
	wantType := filters.Type

	if shouldInclude(wantSource, wantType, "human_report", "report") {
		subQueries = append(subQueries, buildReportSubQuery())
	}
	if shouldInclude(wantSource, wantType, "auto_text", "message") {
		subQueries = append(subQueries, buildMessageSubQuery())
	}
	if shouldInclude(wantSource, wantType, "auto_text", "review") {
		subQueries = append(subQueries, buildReviewSubQuery())
	}
	if shouldInclude(wantSource, wantType, "auto_media", "media") {
		subQueries = append(subQueries, buildMediaSubQuery())
	}

	if len(subQueries) == 0 {
		// Return all sources when no filter matches
		subQueries = []string{
			buildReportSubQuery(),
			buildMessageSubQuery(),
			buildReviewSubQuery(),
			buildMediaSubQuery(),
		}
	}

	union := strings.Join(subQueries, "\nUNION ALL\n")

	// Wrap with outer WHERE for status filter
	outer := "SELECT * FROM (\n" + union + "\n) AS moderation"

	var whereParts []string
	if filters.Status != "" {
		whereParts = append(whereParts, fmt.Sprintf("status = $%d", paramIdx))
		args = append(args, filters.Status)
		paramIdx++
	}

	if len(whereParts) > 0 {
		outer += " WHERE " + strings.Join(whereParts, " AND ")
	}

	outer += " " + moderationOrderBy(filters.Sort)

	// Pagination: LIMIT + OFFSET
	limit := filters.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	page := filters.Page
	if page < 1 {
		page = 1
	}

	outer += fmt.Sprintf(" LIMIT $%d OFFSET $%d", paramIdx, paramIdx+1)
	args = append(args, limit, (page-1)*limit)

	return outer, args
}

// buildModerationCountQuery builds a COUNT(*) version of the union query.
func buildModerationCountQuery(filters repository.ModerationFilters) (string, []any) {
	var subQueries []string
	var args []any
	paramIdx := 1

	wantSource := filters.Source
	wantType := filters.Type

	if shouldInclude(wantSource, wantType, "human_report", "report") {
		subQueries = append(subQueries, buildReportSubQuery())
	}
	if shouldInclude(wantSource, wantType, "auto_text", "message") {
		subQueries = append(subQueries, buildMessageSubQuery())
	}
	if shouldInclude(wantSource, wantType, "auto_text", "review") {
		subQueries = append(subQueries, buildReviewSubQuery())
	}
	if shouldInclude(wantSource, wantType, "auto_media", "media") {
		subQueries = append(subQueries, buildMediaSubQuery())
	}

	if len(subQueries) == 0 {
		subQueries = []string{
			buildReportSubQuery(),
			buildMessageSubQuery(),
			buildReviewSubQuery(),
			buildMediaSubQuery(),
		}
	}

	union := strings.Join(subQueries, "\nUNION ALL\n")
	outer := "SELECT COUNT(*) FROM (\n" + union + "\n) AS moderation"

	var whereParts []string
	if filters.Status != "" {
		whereParts = append(whereParts, fmt.Sprintf("status = $%d", paramIdx))
		args = append(args, filters.Status)
		paramIdx++ //nolint:ineffassign
	}

	if len(whereParts) > 0 {
		outer += " WHERE " + strings.Join(whereParts, " AND ")
	}

	return outer, args
}

// buildPendingCountQuery counts all pending items across all sources.
func buildPendingCountQuery() string {
	return `SELECT
		(SELECT COUNT(*) FROM reports WHERE status = 'pending') +
		(SELECT COUNT(*) FROM messages WHERE moderation_status IN ('flagged', 'hidden') AND deleted_at IS NULL) +
		(SELECT COUNT(*) FROM reviews WHERE moderation_status = 'flagged') +
		(SELECT COUNT(*) FROM media WHERE moderation_status = 'flagged')`
}

func shouldInclude(wantSource, wantType, source, contentType string) bool {
	if wantSource != "" && wantSource != source {
		return false
	}
	if wantType != "" && wantType != contentType {
		return false
	}
	return true
}

func buildReportSubQuery() string {
	return `SELECT
		r.id,
		'human_report'::text AS source,
		'report'::text AS content_type,
		r.id AS content_id,
		COALESCE(r.description, '') AS content_preview,
		r.status::text AS status,
		0::real AS moderation_score,
		r.reason::text AS reason,
		r.reporter_id AS user_involved_id,
		COALESCE(u.display_name, u.first_name || ' ' || u.last_name) AS user_involved_name,
		COALESCE(u.role, '') AS user_involved_role,
		r.conversation_id,
		r.created_at
	FROM reports r
	JOIN users u ON u.id = r.reporter_id`
}

func buildMessageSubQuery() string {
	return `SELECT
		m.id,
		'auto_text'::text AS source,
		'message'::text AS content_type,
		m.id AS content_id,
		LEFT(m.content, 200) AS content_preview,
		CASE
			WHEN m.moderation_status = 'hidden' THEN 'hidden'
			ELSE 'pending'
		END AS status,
		m.moderation_score,
		m.moderation_status AS reason,
		m.sender_id AS user_involved_id,
		COALESCE(u.display_name, u.first_name || ' ' || u.last_name) AS user_involved_name,
		COALESCE(u.role, '') AS user_involved_role,
		m.conversation_id,
		m.created_at
	FROM messages m
	JOIN users u ON u.id = m.sender_id
	WHERE m.moderation_status IN ('flagged', 'hidden')
		AND m.deleted_at IS NULL`
}

func buildReviewSubQuery() string {
	return `SELECT
		rv.id,
		'auto_text'::text AS source,
		'review'::text AS content_type,
		rv.id AS content_id,
		LEFT(rv.comment, 200) AS content_preview,
		'pending'::text AS status,
		rv.moderation_score,
		rv.moderation_status AS reason,
		rv.reviewer_id AS user_involved_id,
		COALESCE(u.display_name, u.first_name || ' ' || u.last_name) AS user_involved_name,
		COALESCE(u.role, '') AS user_involved_role,
		NULL::uuid AS conversation_id,
		rv.created_at
	FROM reviews rv
	JOIN users u ON u.id = rv.reviewer_id
	WHERE rv.moderation_status = 'flagged'`
}

func buildMediaSubQuery() string {
	return `SELECT
		md.id,
		'auto_media'::text AS source,
		'media'::text AS content_type,
		md.id AS content_id,
		md.file_name AS content_preview,
		'pending'::text AS status,
		md.moderation_score,
		md.context::text AS reason,
		md.uploader_id AS user_involved_id,
		COALESCE(u.display_name, u.first_name || ' ' || u.last_name) AS user_involved_name,
		COALESCE(u.role, '') AS user_involved_role,
		NULL::uuid AS conversation_id,
		md.created_at
	FROM media md
	JOIN users u ON u.id = md.uploader_id
	WHERE md.moderation_status = 'flagged'`
}

func moderationOrderBy(sort string) string {
	switch sort {
	case "oldest":
		return "ORDER BY created_at ASC, id ASC"
	case "score":
		return "ORDER BY moderation_score DESC, created_at DESC"
	default:
		return "ORDER BY created_at DESC, id DESC"
	}
}
