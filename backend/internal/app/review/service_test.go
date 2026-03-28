package review

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	domain "marketplace-backend/internal/domain/review"
)

func TestService_CreateReview_Success(t *testing.T) {
	proposalID := uuid.New()
	clientID := uuid.New()
	providerID := uuid.New()

	svc := NewService(ServiceDeps{
		Reviews: &mockReviewRepo{
			hasReviewedFn: func(_ context.Context, _, _ uuid.UUID) (bool, error) {
				return false, nil
			},
			createFn: func(_ context.Context, _ *domain.Review) error {
				return nil
			},
		},
		Proposals: &mockProposalRepo{
			getByIDFn: func(_ context.Context, id uuid.UUID) (*mockProposal, error) {
				return &mockProposal{
					ID:         proposalID,
					ClientID:   clientID,
					ProviderID: providerID,
					Status:     "completed",
				}, nil
			},
		},
	})

	r, err := svc.CreateReview(context.Background(), CreateReviewInput{
		ProposalID:   proposalID,
		ReviewerID:   clientID,
		GlobalRating: 5,
		Comment:      "Excellent work!",
	})

	assert.NoError(t, err)
	assert.NotNil(t, r)
	assert.Equal(t, 5, r.GlobalRating)
	assert.Equal(t, providerID, r.ReviewedID)
}

func TestService_CreateReview_NotCompleted(t *testing.T) {
	svc := NewService(ServiceDeps{
		Reviews: &mockReviewRepo{},
		Proposals: &mockProposalRepo{
			getByIDFn: func(_ context.Context, _ uuid.UUID) (*mockProposal, error) {
				return &mockProposal{Status: "active"}, nil
			},
		},
	})

	_, err := svc.CreateReview(context.Background(), CreateReviewInput{
		ProposalID:   uuid.New(),
		ReviewerID:   uuid.New(),
		GlobalRating: 4,
	})

	assert.ErrorIs(t, err, domain.ErrNotCompleted)
}

func TestService_CreateReview_NotParticipant(t *testing.T) {
	svc := NewService(ServiceDeps{
		Reviews: &mockReviewRepo{},
		Proposals: &mockProposalRepo{
			getByIDFn: func(_ context.Context, _ uuid.UUID) (*mockProposal, error) {
				return &mockProposal{
					ClientID:   uuid.New(),
					ProviderID: uuid.New(),
					Status:     "completed",
				}, nil
			},
		},
	})

	_, err := svc.CreateReview(context.Background(), CreateReviewInput{
		ProposalID:   uuid.New(),
		ReviewerID:   uuid.New(), // not a participant
		GlobalRating: 4,
	})

	assert.ErrorIs(t, err, domain.ErrNotParticipant)
}

func TestService_CreateReview_AlreadyReviewed(t *testing.T) {
	clientID := uuid.New()

	svc := NewService(ServiceDeps{
		Reviews: &mockReviewRepo{
			hasReviewedFn: func(_ context.Context, _, _ uuid.UUID) (bool, error) {
				return true, nil
			},
		},
		Proposals: &mockProposalRepo{
			getByIDFn: func(_ context.Context, _ uuid.UUID) (*mockProposal, error) {
				return &mockProposal{
					ClientID:   clientID,
					ProviderID: uuid.New(),
					Status:     "completed",
				}, nil
			},
		},
	})

	_, err := svc.CreateReview(context.Background(), CreateReviewInput{
		ProposalID:   uuid.New(),
		ReviewerID:   clientID,
		GlobalRating: 4,
	})

	assert.ErrorIs(t, err, domain.ErrAlreadyReviewed)
}

func TestService_CanReview(t *testing.T) {
	proposalID := uuid.New()
	clientID := uuid.New()
	providerID := uuid.New()

	svc := NewService(ServiceDeps{
		Reviews: &mockReviewRepo{
			hasReviewedFn: func(_ context.Context, _, _ uuid.UUID) (bool, error) {
				return false, nil
			},
		},
		Proposals: &mockProposalRepo{
			getByIDFn: func(_ context.Context, _ uuid.UUID) (*mockProposal, error) {
				return &mockProposal{
					ID:         proposalID,
					ClientID:   clientID,
					ProviderID: providerID,
					Status:     "completed",
				}, nil
			},
		},
	})

	can, err := svc.CanReview(context.Background(), proposalID, clientID)
	assert.NoError(t, err)
	assert.True(t, can)
}

func TestService_CreateReview_ProviderCannotReview(t *testing.T) {
	clientID := uuid.New()
	providerID := uuid.New()

	svc := NewService(ServiceDeps{
		Reviews: &mockReviewRepo{},
		Proposals: &mockProposalRepo{
			getByIDFn: func(_ context.Context, _ uuid.UUID) (*mockProposal, error) {
				return &mockProposal{
					ClientID:   clientID,
					ProviderID: providerID,
					Status:     "completed",
				}, nil
			},
		},
	})

	_, err := svc.CreateReview(context.Background(), CreateReviewInput{
		ProposalID:   uuid.New(),
		ReviewerID:   providerID, // provider tries to review
		GlobalRating: 5,
		Comment:      "Great client",
	})

	assert.ErrorIs(t, err, domain.ErrNotParticipant)
}

func TestService_CanReview_ProviderCannotReview(t *testing.T) {
	clientID := uuid.New()
	providerID := uuid.New()

	svc := NewService(ServiceDeps{
		Reviews: &mockReviewRepo{},
		Proposals: &mockProposalRepo{
			getByIDFn: func(_ context.Context, _ uuid.UUID) (*mockProposal, error) {
				return &mockProposal{
					ClientID:   clientID,
					ProviderID: providerID,
					Status:     "completed",
				}, nil
			},
		},
	})

	can, err := svc.CanReview(context.Background(), uuid.New(), providerID)
	assert.NoError(t, err)
	assert.False(t, can, "provider must not be able to review")
}

func TestService_CanReview_NotCompleted(t *testing.T) {
	svc := NewService(ServiceDeps{
		Reviews: &mockReviewRepo{},
		Proposals: &mockProposalRepo{
			getByIDFn: func(_ context.Context, _ uuid.UUID) (*mockProposal, error) {
				return &mockProposal{Status: "active"}, nil
			},
		},
	})

	can, err := svc.CanReview(context.Background(), uuid.New(), uuid.New())
	assert.NoError(t, err)
	assert.False(t, can)
}
