package repository

import (
	"context"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/review"
)

// AdminReviewFilters holds query filters for admin review listing.
type AdminReviewFilters struct {
	Search string
	Rating int
	Sort   string
	Filter string
	Page   int
	Limit  int
}

// AdminReview extends Review with reviewer and reviewed user info for admin views.
type AdminReview struct {
	review.Review
	ReviewerDisplayName string
	ReviewerEmail       string
	ReviewerRole        string
	ReviewedDisplayName string
	ReviewedEmail       string
	ReviewedRole        string
	PendingReportCount  int
}

// ReviewRepository defines persistence operations for reviews.
type ReviewRepository interface {
	// Create persists a new review with published_at = NULL (hidden until
	// its counterpart is submitted or the 14-day window elapses).
	// Prefer CreateAndMaybeReveal from the app layer so that the reveal
	// logic stays atomic with the insert.
	Create(ctx context.Context, r *review.Review) error
	// CreateAndMaybeReveal atomically inserts the new review and, in the
	// same transaction, reveals every pending review for the proposal
	// whenever the submission completes the pair. Concretely:
	//
	//   - INSERT the new review (published_at = NULL)
	//   - COUNT existing reviews for the proposal
	//   - If this brings the total to 2+ OR a previously-published review
	//     already exists on the proposal (backfilled historic row), UPDATE
	//     every pending review on that proposal to published_at = NOW().
	//   - Return the newly-created review with its final published_at.
	//
	// Both the INSERT and the conditional UPDATE run inside a single
	// transaction — Postgres serializes concurrent submissions naturally,
	// so no explicit locking is needed.
	CreateAndMaybeReveal(ctx context.Context, r *review.Review) (*review.Review, error)
	GetByID(ctx context.Context, id uuid.UUID) (*review.Review, error)
	// ListByReviewedOrganization returns the reviews received by the
	// given org (provider side), ordered by created_at DESC. Hidden
	// reviews (moderation_status='hidden' OR published_at IS NULL) are
	// excluded at the SQL level. Also runs a lazy auto-publish sweep
	// against proposals whose 14-day window has elapsed so deadline
	// reveals do not need a cron worker.
	ListByReviewedOrganization(ctx context.Context, orgID uuid.UUID, cursor string, limit int) ([]*review.Review, string, error)
	// GetAverageRatingByOrganization returns the aggregated rating for
	// an organization. Only published, non-hidden client→provider
	// reviews are counted.
	GetAverageRatingByOrganization(ctx context.Context, orgID uuid.UUID) (*review.AverageRating, error)
	HasReviewed(ctx context.Context, proposalID, reviewerID uuid.UUID) (bool, error)
	// GetByProposalIDs returns a map of proposalID → review for the given
	// proposal IDs, filtered to the requested side. Missing entries mean
	// the proposal was not yet reviewed on that side.
	//
	// side must be one of:
	//   - "client_to_provider": return the client→provider review per proposal
	//   - "provider_to_client": return the provider→client review per proposal
	//   - ""                   : return ANY one review per proposal (legacy)
	//
	// Each proposal has up to two reviews (double-blind: one per side).
	// The map is keyed by proposal_id, so without the side filter the two
	// sides collide and whichever row scans last wins. New callers MUST
	// pass an explicit side — the "any-side" mode only exists for legacy
	// integrations and must not be used in user-facing flows.
	//
	// Hidden (moderation_status='hidden') and unpublished (published_at
	// IS NULL) reviews are excluded so the project history surface never
	// leaks blind submissions.
	GetByProposalIDs(ctx context.Context, proposalIDs []uuid.UUID, side string) (map[uuid.UUID]*review.Review, error)
	UpdateReviewModeration(ctx context.Context, reviewID uuid.UUID, status string, score float64, labelsJSON []byte) error

	// Admin operations
	ListAdmin(ctx context.Context, filters AdminReviewFilters) ([]AdminReview, error)
	CountAdmin(ctx context.Context, filters AdminReviewFilters) (int, error)
	GetAdminByID(ctx context.Context, id uuid.UUID) (*AdminReview, error)
	DeleteAdmin(ctx context.Context, id uuid.UUID) error
}
