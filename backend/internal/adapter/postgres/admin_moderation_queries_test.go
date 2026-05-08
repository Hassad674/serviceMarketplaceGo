package postgres

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"marketplace-backend/internal/port/repository"
)

// TestBuildMediaSubQuery_IncludesFlaggedAndRejected guards the fix for
// the production regression observed on 2026-05-08: auto-rejected
// uploads disappeared silently from /admin/moderation because the
// media sub-query filtered on `moderation_status = 'flagged'` only.
// The reviewer could not audit false positives. The corrected query
// surfaces both `flagged` (awaiting human verdict) and `rejected`
// (auto-rejected at score >= AutoRejectThreshold) so the queue is the
// honest source of truth for "every automated moderation outcome".
func TestBuildMediaSubQuery_IncludesFlaggedAndRejected(t *testing.T) {
	q := buildMediaSubQuery()

	assert.Contains(t, q, "moderation_status IN ('flagged', 'rejected')",
		"media sub-query must surface auto-rejected media so admins can audit false positives")
	assert.NotContains(t, q, "moderation_status = 'flagged'",
		"the legacy single-status filter must be gone — replaced by IN (...)")
}

// TestBuildMediaSubQuery_MapsRejectedToDisplayDeleted documents the UI
// contract: 'rejected' rows surface with display status 'deleted' so
// the existing message/review queues' vocabulary applies. Keeps the
// admin filter ('deleted') usable for "see auto-rejected media".
func TestBuildMediaSubQuery_MapsRejectedToDisplayDeleted(t *testing.T) {
	q := buildMediaSubQuery()

	assert.Contains(t, q, "WHEN md.moderation_status = 'rejected' THEN 'deleted'",
		"auto-rejected media must surface as display status 'deleted' to align with messages/reviews")
}

// TestBuildModerationUnionQuery_DefaultIncludesMedia confirms a default
// queue read (no source/type filter, no status filter) routes through
// buildMediaSubQuery via the UNION. Without it the entire fix is moot.
func TestBuildModerationUnionQuery_DefaultIncludesMedia(t *testing.T) {
	query, _ := buildModerationUnionQuery(repository.ModerationFilters{})

	// The auto_media subquery shape comes from buildMediaSubQuery —
	// we look for its discriminator marker.
	assert.Contains(t, query, "'auto_media'::text AS source",
		"default UNION must include the media sub-query")
	assert.Contains(t, query, "moderation_status IN ('flagged', 'rejected')",
		"default UNION must surface both flagged AND rejected media")
}

// TestBuildModerationUnionQuery_AutoMediaSourceFilter — when the admin
// scopes the filter to source=auto_media, the resulting SQL must still
// surface both statuses (flagged + rejected). This is the path the UI
// uses to drive its "Détection média" tab.
func TestBuildModerationUnionQuery_AutoMediaSourceFilter(t *testing.T) {
	query, _ := buildModerationUnionQuery(repository.ModerationFilters{
		Source: "auto_media",
	})

	assert.Contains(t, query, "'auto_media'::text AS source")
	assert.Contains(t, query, "moderation_status IN ('flagged', 'rejected')")
	// Other sub-queries must NOT be in the UNION when source is
	// auto_media — otherwise the count is wrong.
	assert.NotContains(t, query, "FROM moderation_results mr\n\t\tJOIN messages m",
		"auto_media filter must exclude the message sub-query")
}

// TestBuildPendingCountQuery_StillCountsFlaggedMediaOnly — auto-rejected
// items are NOT "in attente" (they're already actioned by the system),
// so the pending count must keep reading only flagged media. The badge
// would otherwise inflate after every auto-reject and never drain.
func TestBuildPendingCountQuery_StillCountsFlaggedMediaOnly(t *testing.T) {
	q := buildPendingCountQuery()

	assert.Contains(t, q, "FROM media WHERE moderation_status = 'flagged'",
		"pending count must keep counting flagged media only — auto-rejected items don't need attention")
	// No 'rejected' state in the count.
	if strings.Contains(q, "moderation_status = 'rejected'") {
		t.Error("pending count must NOT include auto-rejected media (they're already actioned)")
	}
}
