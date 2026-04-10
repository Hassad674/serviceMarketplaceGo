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
	Create(ctx context.Context, r *review.Review) error
	GetByID(ctx context.Context, id uuid.UUID) (*review.Review, error)
	ListByReviewedUser(ctx context.Context, userID uuid.UUID, cursor string, limit int) ([]*review.Review, string, error)
	GetAverageRating(ctx context.Context, userID uuid.UUID) (*review.AverageRating, error)
	HasReviewed(ctx context.Context, proposalID, reviewerID uuid.UUID) (bool, error)
	// GetByProposalIDs returns a map of proposalID → review for the given
	// proposal IDs. Missing entries mean the proposal was not yet reviewed.
	// Hidden reviews (moderation_status = 'hidden') are excluded.
	GetByProposalIDs(ctx context.Context, proposalIDs []uuid.UUID) (map[uuid.UUID]*review.Review, error)
	UpdateReviewModeration(ctx context.Context, reviewID uuid.UUID, status string, score float64, labelsJSON []byte) error

	// Admin operations
	ListAdmin(ctx context.Context, filters AdminReviewFilters) ([]AdminReview, error)
	CountAdmin(ctx context.Context, filters AdminReviewFilters) (int, error)
	GetAdminByID(ctx context.Context, id uuid.UUID) (*AdminReview, error)
	DeleteAdmin(ctx context.Context, id uuid.UUID) error
}
