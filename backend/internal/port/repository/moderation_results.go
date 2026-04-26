package repository

import (
	"context"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/moderation"
)

// ModerationResultsFilters drives the admin queue listing. Defaults
// (zero values) mean "no filter". Limit must be enforced by the
// repository to a sane cap to prevent unbounded queries.
type ModerationResultsFilters struct {
	ContentType  string // empty = any
	Status       string // empty = any (incl. flagged | hidden | deleted | blocked)
	AuthorUserID *uuid.UUID
	Sort         string // "newest" (default), "oldest", "score"
	Limit        int    // capped at 100
	Offset       int
}

// ModerationResultsRepository persists moderation decisions in the
// generic moderation_results table. There is one row per
// (content_type, content_id) pair — Upsert overwrites the previous
// verdict (the full transition history lives in audit_logs).
type ModerationResultsRepository interface {
	// Upsert inserts a new decision or replaces the existing one for
	// the (content_type, content_id) pair. Idempotent.
	Upsert(ctx context.Context, r *moderation.Result) error

	// GetByContent returns the latest decision for a specific content
	// reference, or nil + moderation.ErrResultNotFound when none exists.
	GetByContent(ctx context.Context, contentType moderation.ContentType, contentID uuid.UUID) (*moderation.Result, error)

	// List returns admin-queue rows with the given filters and a total
	// count (for pagination UI). Total is computed without limit/offset.
	List(ctx context.Context, filters ModerationResultsFilters) ([]*moderation.Result, int, error)

	// MarkReviewed records an admin override (Approve, Hide, Restore).
	// Updates status, reviewed_by and reviewed_at in a single SQL
	// statement. The previous status is the responsibility of the
	// caller (audit_logs holds the trail).
	MarkReviewed(ctx context.Context, contentType moderation.ContentType, contentID uuid.UUID, reviewerID uuid.UUID, newStatus moderation.Status) error
}
