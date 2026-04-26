package postgres

import (
	"fmt"
	"strings"

	"marketplace-backend/internal/port/repository"
)

// buildModerationUnionQuery constructs a UNION ALL query across reports,
// flagged messages, flagged reviews, flagged media, AND every new
// content type added in Phase 2 (profile_*, job_*, proposal_*,
// job_application_*, user_display_name).
//
// Each sub-query returns the same column shape so the outer SELECT
// can sort + filter without caring about the source. Phase 2 added a
// generic sub-query (buildGenericModerationSubQuery) that reads from
// moderation_results without requiring a per-content-type JOIN — it
// covers every new type with one query, and gracefully includes
// blocked attempts that have no source row to JOIN to.
func buildModerationUnionQuery(filters repository.ModerationFilters) (string, []any) {
	var subQueries []string
	var args []any
	paramIdx := 1

	wantSource := filters.Source
	wantType := filters.Type

	if shouldIncludeReports(wantSource, wantType) {
		subQueries = append(subQueries, buildReportSubQuery(wantType))
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
	// Generic Phase 2 types — flagged/hidden/deleted/blocked rows for
	// any content_type not covered by the sub-queries above. The
	// helper accepts the wantType filter so a UI request scoped to
	// "job_title" only returns matching rows.
	if shouldIncludeGeneric(wantSource, wantType) {
		subQueries = append(subQueries, buildGenericModerationSubQuery(wantType))
	}

	if len(subQueries) == 0 {
		// Return all sources when no filter matches
		subQueries = []string{
			buildReportSubQuery(""),
			buildMessageSubQuery(),
			buildReviewSubQuery(),
			buildMediaSubQuery(),
			buildGenericModerationSubQuery(""),
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

	if shouldIncludeReports(wantSource, wantType) {
		subQueries = append(subQueries, buildReportSubQuery(wantType))
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
	if shouldIncludeGeneric(wantSource, wantType) {
		subQueries = append(subQueries, buildGenericModerationSubQuery(wantType))
	}

	if len(subQueries) == 0 {
		subQueries = []string{
			buildReportSubQuery(""),
			buildMessageSubQuery(),
			buildReviewSubQuery(),
			buildMediaSubQuery(),
			buildGenericModerationSubQuery(""),
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
// Phase 3: messages + reviews now read from moderation_results so the
// counter reflects the new central table. reports + media keep their
// own tables for now (media moderation is unchanged in this phase).
// reviewed_at IS NULL excludes items the admin has already actioned.
func buildPendingCountQuery() string {
	return `SELECT
		(SELECT COUNT(*) FROM reports WHERE status = 'pending') +
		(SELECT COUNT(*) FROM moderation_results
			WHERE content_type IN ('message', 'review')
			  AND status IN ('flagged', 'hidden', 'deleted')
			  AND reviewed_at IS NULL) +
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

func shouldIncludeReports(wantSource, wantType string) bool {
	if wantSource != "" && wantSource != "human_report" {
		return false
	}
	// Reports have dynamic target_type, so we always include them
	// and let the SQL WHERE clause filter by target_type if needed
	return true
}

func buildReportSubQuery(wantType string) string {
	where := "WHERE 1=1"
	if wantType != "" {
		where += " AND r.target_type = '" + wantType + "'"
	}
	return `SELECT
		r.id,
		'human_report'::text AS source,
		r.target_type::text AS content_type,
		r.target_id AS content_id,
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
	JOIN users u ON u.id = r.reporter_id ` + where
}

// buildMessageSubQuery — Phase 3: now reads from moderation_results
// joined to messages (for the content preview + conversation_id) and
// users (for the author display name). Only rows the admin still
// needs to look at are returned: reviewed_at IS NULL filters out
// messages an admin has already approved/restored.
//
// The status passed to the admin UI maps the four storage statuses
// to the three display states it knows about (pending, hidden,
// deleted). 'flagged' surfaces as 'pending' so the UI's existing
// MessageActions component (Approve/Hide) keeps working unchanged.
func buildMessageSubQuery() string {
	return `SELECT
		mr.id,
		'auto_text'::text AS source,
		'message'::text AS content_type,
		mr.content_id,
		LEFT(m.content, 200) AS content_preview,
		CASE
			WHEN mr.status = 'deleted' THEN 'deleted'
			WHEN mr.status = 'hidden' THEN 'hidden'
			ELSE 'pending'
		END AS status,
		mr.score AS moderation_score,
		mr.reason AS reason,
		COALESCE(mr.author_user_id, m.sender_id) AS user_involved_id,
		COALESCE(u.display_name, u.first_name || ' ' || u.last_name) AS user_involved_name,
		COALESCE(u.role, '') AS user_involved_role,
		m.conversation_id,
		mr.decided_at AS created_at
	FROM moderation_results mr
	JOIN messages m ON m.id = mr.content_id
	LEFT JOIN users u ON u.id = COALESCE(mr.author_user_id, m.sender_id)
	WHERE mr.content_type = 'message'
		AND mr.status IN ('flagged', 'hidden', 'deleted')
		AND mr.reviewed_at IS NULL
		AND m.deleted_at IS NULL`
}

func buildReviewSubQuery() string {
	return `SELECT
		mr.id,
		'auto_text'::text AS source,
		'review'::text AS content_type,
		mr.content_id,
		LEFT(rv.comment, 200) AS content_preview,
		CASE
			WHEN mr.status = 'deleted' THEN 'deleted'
			WHEN mr.status = 'hidden' THEN 'hidden'
			ELSE 'pending'
		END AS status,
		mr.score AS moderation_score,
		mr.reason AS reason,
		COALESCE(mr.author_user_id, rv.reviewer_id) AS user_involved_id,
		COALESCE(u.display_name, u.first_name || ' ' || u.last_name) AS user_involved_name,
		COALESCE(u.role, '') AS user_involved_role,
		NULL::uuid AS conversation_id,
		mr.decided_at AS created_at
	FROM moderation_results mr
	JOIN reviews rv ON rv.id = mr.content_id
	LEFT JOIN users u ON u.id = COALESCE(mr.author_user_id, rv.reviewer_id)
	WHERE mr.content_type = 'review'
		AND mr.status IN ('flagged', 'hidden', 'deleted')
		AND mr.reviewed_at IS NULL`
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

// genericContentTypes is the closed set of Phase 2 content_types the
// generic sub-query covers. Reports + message + review + media have
// dedicated sub-queries above and are intentionally excluded from
// this list (they would double-count if included).
var genericContentTypes = []string{
	"profile_about", "profile_title",
	"job_title", "job_description",
	"proposal_description",
	"job_application_message",
	"user_display_name",
}

// shouldIncludeGeneric decides whether the generic sub-query should be
// part of the UNION for this filter combination. Anything outside the
// dedicated four (report / message / review / media) routes here.
func shouldIncludeGeneric(wantSource, wantType string) bool {
	// auto_media is exclusive to the media sub-query.
	if wantSource == "auto_media" || wantSource == "human_report" {
		return false
	}
	if wantType == "" {
		return true
	}
	for _, ct := range genericContentTypes {
		if ct == wantType {
			return true
		}
	}
	return false
}

// buildGenericModerationSubQuery returns flagged/hidden/deleted/blocked
// rows for every Phase 2 content type. Reads moderation_results +
// users; the source-table content preview is left empty so the admin
// UI relies on the content_type label + the click-through "Voir"
// button. This deliberate trade-off keeps the query simple — fetching
// the per-type preview would require seven LEFT JOINs and an SQL
// CASE on content_type, which crosses the readability bar for marginal
// UX gain.
//
// wantType (optional) narrows the WHERE to a single content type.
// reviewed_at IS NULL filters out rows the admin already actioned —
// the queue should never repeat work.
func buildGenericModerationSubQuery(wantType string) string {
	where := `mr.content_type = ANY(ARRAY['profile_about','profile_title',
		'job_title','job_description','proposal_description',
		'job_application_message','user_display_name'])
		AND mr.status IN ('flagged', 'hidden', 'deleted', 'blocked')
		AND mr.reviewed_at IS NULL`
	if wantType != "" {
		where = "mr.content_type = '" + wantType + "'" +
			" AND mr.status IN ('flagged', 'hidden', 'deleted', 'blocked')" +
			" AND mr.reviewed_at IS NULL"
	}
	return `SELECT
		mr.id,
		'auto_text'::text AS source,
		mr.content_type,
		mr.content_id,
		'' AS content_preview,
		mr.status,
		mr.score AS moderation_score,
		mr.reason AS reason,
		mr.author_user_id AS user_involved_id,
		COALESCE(u.display_name, COALESCE(u.first_name, '') || ' ' || COALESCE(u.last_name, '')) AS user_involved_name,
		COALESCE(u.role, '') AS user_involved_role,
		NULL::uuid AS conversation_id,
		mr.decided_at AS created_at
	FROM moderation_results mr
	LEFT JOIN users u ON u.id = mr.author_user_id
	WHERE ` + where
}
