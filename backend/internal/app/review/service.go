package review

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	domain "marketplace-backend/internal/domain/review"
	"marketplace-backend/internal/port/repository"
)

// ServiceDeps groups the dependencies for the review service.
type ServiceDeps struct {
	Reviews   repository.ReviewRepository
	Proposals repository.ProposalRepository
}

// Service orchestrates review use cases.
type Service struct {
	reviews   repository.ReviewRepository
	proposals repository.ProposalRepository
}

// NewService creates a new review service.
func NewService(deps ServiceDeps) *Service {
	return &Service{
		reviews:   deps.Reviews,
		proposals: deps.Proposals,
	}
}

// CreateReviewInput contains the data needed to create a review.
type CreateReviewInput struct {
	ProposalID    uuid.UUID
	ReviewerID    uuid.UUID
	GlobalRating  int
	Timeliness    *int
	Communication *int
	Quality       *int
	Comment       string
	VideoURL      string
}

// CreateReview validates the context and persists a new review.
func (s *Service) CreateReview(ctx context.Context, in CreateReviewInput) (*domain.Review, error) {
	// Verify proposal exists and is completed
	p, err := s.proposals.GetByID(ctx, in.ProposalID)
	if err != nil {
		return nil, fmt.Errorf("get proposal: %w", err)
	}
	if p.Status != "completed" {
		return nil, domain.ErrNotCompleted
	}

	// Only the client (the party who pays) can leave a review.
	// Enterprise evaluates Freelance/Agency, Agency evaluates Freelance.
	// The provider never evaluates the client.
	if in.ReviewerID != p.ClientID {
		return nil, domain.ErrNotParticipant
	}

	// The reviewed party is always the provider.
	reviewedID := p.ProviderID

	// Check for duplicate review
	already, err := s.reviews.HasReviewed(ctx, in.ProposalID, in.ReviewerID)
	if err != nil {
		return nil, fmt.Errorf("check existing review: %w", err)
	}
	if already {
		return nil, domain.ErrAlreadyReviewed
	}

	// Create domain entity
	r, err := domain.NewReview(domain.NewReviewInput{
		ProposalID:    in.ProposalID,
		ReviewerID:    in.ReviewerID,
		ReviewedID:    reviewedID,
		GlobalRating:  in.GlobalRating,
		Timeliness:    in.Timeliness,
		Communication: in.Communication,
		Quality:       in.Quality,
		Comment:       in.Comment,
		VideoURL:      in.VideoURL,
	})
	if err != nil {
		return nil, err
	}

	if err := s.reviews.Create(ctx, r); err != nil {
		return nil, fmt.Errorf("persist review: %w", err)
	}

	return r, nil
}

// ListByUser returns reviews received by a user (public).
func (s *Service) ListByUser(ctx context.Context, userID uuid.UUID, cursor string, limit int) ([]*domain.Review, string, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	return s.reviews.ListByReviewedUser(ctx, userID, cursor, limit)
}

// GetAverageRating returns the average rating for a user.
func (s *Service) GetAverageRating(ctx context.Context, userID uuid.UUID) (*domain.AverageRating, error) {
	return s.reviews.GetAverageRating(ctx, userID)
}

// CanReview checks if the current user can review a given proposal.
// Only the client (the paying party) is allowed to leave a review.
func (s *Service) CanReview(ctx context.Context, proposalID, userID uuid.UUID) (bool, error) {
	p, err := s.proposals.GetByID(ctx, proposalID)
	if err != nil {
		return false, fmt.Errorf("get proposal: %w", err)
	}
	if p.Status != "completed" {
		return false, nil
	}
	// Only the client can review; the provider never evaluates.
	if userID != p.ClientID {
		return false, nil
	}
	already, err := s.reviews.HasReviewed(ctx, proposalID, userID)
	if err != nil {
		return false, fmt.Errorf("check existing review: %w", err)
	}
	return !already, nil
}
