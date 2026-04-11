package projecthistory_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/app/projecthistory"
	proposaldomain "marketplace-backend/internal/domain/proposal"
	reviewdomain "marketplace-backend/internal/domain/review"
)

func newTestService(propRepo *mockProposalRepo, revRepo *mockReviewRepo) *projecthistory.Service {
	return projecthistory.NewService(projecthistory.ServiceDeps{
		Proposals: propRepo,
		Reviews:   revRepo,
	})
}

func completedProposal(id uuid.UUID, providerID uuid.UUID, amount int64, completed time.Time) *proposaldomain.Proposal {
	return &proposaldomain.Proposal{
		ID:          id,
		ProviderID:  providerID,
		Amount:      amount,
		Status:      proposaldomain.StatusCompleted,
		CompletedAt: &completed,
	}
}

func TestListByProvider_HappyPath_MixedReviews(t *testing.T) {
	provider := uuid.New()
	p1ID := uuid.New()
	p2ID := uuid.New()
	p3ID := uuid.New()

	p1 := completedProposal(p1ID, provider, 50000, time.Now().Add(-72*time.Hour))
	p2 := completedProposal(p2ID, provider, 120000, time.Now().Add(-48*time.Hour))
	p3 := completedProposal(p3ID, provider, 8000, time.Now().Add(-24*time.Hour))

	rev1 := &reviewdomain.Review{
		ID:           uuid.New(),
		ProposalID:   p1ID,
		GlobalRating: 5,
		Comment:      "Excellent",
	}
	rev3 := &reviewdomain.Review{
		ID:           uuid.New(),
		ProposalID:   p3ID,
		GlobalRating: 4,
		Comment:      "Solid",
	}

	propRepo := &mockProposalRepo{
		ListCompletedByOrganizationFunc: func(_ context.Context, pid uuid.UUID, cursor string, limit int) ([]*proposaldomain.Proposal, string, error) {
			assert.Equal(t, provider, pid)
			return []*proposaldomain.Proposal{p1, p2, p3}, "next-cursor", nil
		},
	}
	revRepo := &mockReviewRepo{
		GetByProposalIDsFunc: func(_ context.Context, ids []uuid.UUID) (map[uuid.UUID]*reviewdomain.Review, error) {
			assert.Len(t, ids, 3)
			return map[uuid.UUID]*reviewdomain.Review{
				p1ID: rev1,
				p3ID: rev3,
			}, nil
		},
	}
	svc := newTestService(propRepo, revRepo)

	entries, next, err := svc.ListByOrganization(context.Background(), provider, "", 20)
	require.NoError(t, err)
	assert.Equal(t, "next-cursor", next)
	require.Len(t, entries, 3)

	// p1 has a review
	assert.Equal(t, p1ID, entries[0].ProposalID)
	require.NotNil(t, entries[0].Review)
	assert.Equal(t, 5, entries[0].Review.GlobalRating)
	assert.Equal(t, int64(50000), entries[0].Amount)
	assert.Equal(t, "EUR", entries[0].Currency)

	// p2 has no review
	assert.Equal(t, p2ID, entries[1].ProposalID)
	assert.Nil(t, entries[1].Review)
	assert.Equal(t, int64(120000), entries[1].Amount)

	// p3 has a review
	assert.Equal(t, p3ID, entries[2].ProposalID)
	require.NotNil(t, entries[2].Review)
	assert.Equal(t, 4, entries[2].Review.GlobalRating)
}

func TestListByProvider_Empty(t *testing.T) {
	propRepo := &mockProposalRepo{
		ListCompletedByOrganizationFunc: func(context.Context, uuid.UUID, string, int) ([]*proposaldomain.Proposal, string, error) {
			return []*proposaldomain.Proposal{}, "", nil
		},
	}
	revRepo := &mockReviewRepo{}
	svc := newTestService(propRepo, revRepo)

	entries, next, err := svc.ListByOrganization(context.Background(), uuid.New(), "", 20)
	require.NoError(t, err)
	assert.Empty(t, entries)
	assert.Empty(t, next)
}

func TestListByProvider_ProposalRepoError(t *testing.T) {
	propRepo := &mockProposalRepo{
		ListCompletedByOrganizationFunc: func(context.Context, uuid.UUID, string, int) ([]*proposaldomain.Proposal, string, error) {
			return nil, "", errors.New("db down")
		},
	}
	revRepo := &mockReviewRepo{}
	svc := newTestService(propRepo, revRepo)

	_, _, err := svc.ListByOrganization(context.Background(), uuid.New(), "", 20)
	assert.Error(t, err)
}

func TestListByProvider_ReviewRepoError(t *testing.T) {
	p1ID := uuid.New()
	propRepo := &mockProposalRepo{
		ListCompletedByOrganizationFunc: func(context.Context, uuid.UUID, string, int) ([]*proposaldomain.Proposal, string, error) {
			return []*proposaldomain.Proposal{
				completedProposal(p1ID, uuid.New(), 1000, time.Now()),
			}, "", nil
		},
	}
	revRepo := &mockReviewRepo{
		GetByProposalIDsFunc: func(context.Context, []uuid.UUID) (map[uuid.UUID]*reviewdomain.Review, error) {
			return nil, errors.New("reviews down")
		},
	}
	svc := newTestService(propRepo, revRepo)

	_, _, err := svc.ListByOrganization(context.Background(), uuid.New(), "", 20)
	assert.Error(t, err)
}

func TestListByProvider_DefaultLimit(t *testing.T) {
	var capturedLimit int
	propRepo := &mockProposalRepo{
		ListCompletedByOrganizationFunc: func(_ context.Context, _ uuid.UUID, _ string, limit int) ([]*proposaldomain.Proposal, string, error) {
			capturedLimit = limit
			return []*proposaldomain.Proposal{}, "", nil
		},
	}
	svc := newTestService(propRepo, &mockReviewRepo{})

	_, _, _ = svc.ListByOrganization(context.Background(), uuid.New(), "", 0)
	assert.Equal(t, 20, capturedLimit)

	_, _, _ = svc.ListByOrganization(context.Background(), uuid.New(), "", 999)
	assert.Equal(t, 20, capturedLimit)
}
