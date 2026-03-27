package repository

import (
	"context"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/review"
)

// ReviewRepository defines persistence operations for reviews.
type ReviewRepository interface {
	Create(ctx context.Context, r *review.Review) error
	GetByID(ctx context.Context, id uuid.UUID) (*review.Review, error)
	ListByReviewedUser(ctx context.Context, userID uuid.UUID, cursor string, limit int) ([]*review.Review, string, error)
	GetAverageRating(ctx context.Context, userID uuid.UUID) (*review.AverageRating, error)
	HasReviewed(ctx context.Context, proposalID, reviewerID uuid.UUID) (bool, error)
}
