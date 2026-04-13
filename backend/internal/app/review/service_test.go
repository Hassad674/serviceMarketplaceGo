package review

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	domain "marketplace-backend/internal/domain/review"
)

// recentCompletion returns a CompletedAt timestamp safely within the
// 14-day review window so happy-path tests do not trip the deadline.
func recentCompletion() *time.Time {
	t := time.Now().Add(-1 * time.Hour)
	return &t
}

func TestService_CreateReview_Success(t *testing.T) {
	proposalID := uuid.New()
	clientID := uuid.New()
	providerID := uuid.New()

	svc := NewService(ServiceDeps{
		Notifications: &mockNotificationSender{}, Users: &mockUserRepo{},
		Reviews: &mockReviewRepo{
			hasReviewedFn: func(_ context.Context, _, _ uuid.UUID) (bool, error) {
				return false, nil
			},
			createAndRevealFn: func(_ context.Context, r *domain.Review) (*domain.Review, error) {
				// Echo back — no reveal (the pair is incomplete).
				return r, nil
			},
		},
		Proposals: &mockProposalRepo{
			getByIDFn: func(_ context.Context, id uuid.UUID) (*mockProposal, error) {
				return &mockProposal{
					ID:          proposalID,
					ClientID:    clientID,
					ProviderID:  providerID,
					Status:      "completed",
					CompletedAt: recentCompletion(),
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
	assert.Equal(t, domain.SideClientToProvider, r.Side)
	assert.Nil(t, r.PublishedAt, "first submission must remain hidden")
}

func TestService_CreateReview_ProviderSide(t *testing.T) {
	proposalID := uuid.New()
	clientID := uuid.New()
	providerID := uuid.New()

	svc := NewService(ServiceDeps{
		Notifications: &mockNotificationSender{}, Users: &mockUserRepo{},
		Reviews: &mockReviewRepo{
			hasReviewedFn: func(_ context.Context, _, _ uuid.UUID) (bool, error) {
				return false, nil
			},
			createAndRevealFn: func(_ context.Context, r *domain.Review) (*domain.Review, error) {
				return r, nil
			},
		},
		Proposals: &mockProposalRepo{
			getByIDFn: func(_ context.Context, _ uuid.UUID) (*mockProposal, error) {
				return &mockProposal{
					ID:          proposalID,
					ClientID:    clientID,
					ProviderID:  providerID,
					Status:      "completed",
					CompletedAt: recentCompletion(),
				}, nil
			},
		},
	})

	r, err := svc.CreateReview(context.Background(), CreateReviewInput{
		ProposalID:   proposalID,
		ReviewerID:   providerID,
		GlobalRating: 4,
		Comment:      "Clear brief, paid on time.",
	})

	assert.NoError(t, err)
	assert.NotNil(t, r)
	assert.Equal(t, domain.SideProviderToClient, r.Side)
	assert.Equal(t, clientID, r.ReviewedID)
}

func TestService_CreateReview_ProviderSide_RejectsSubCriteria(t *testing.T) {
	clientID := uuid.New()
	providerID := uuid.New()
	timeliness := 5

	svc := NewService(ServiceDeps{
		Notifications: &mockNotificationSender{}, Users: &mockUserRepo{},
		Reviews: &mockReviewRepo{
			hasReviewedFn: func(_ context.Context, _, _ uuid.UUID) (bool, error) {
				return false, nil
			},
		},
		Proposals: &mockProposalRepo{
			getByIDFn: func(_ context.Context, _ uuid.UUID) (*mockProposal, error) {
				return &mockProposal{
					ClientID:    clientID,
					ProviderID:  providerID,
					Status:      "completed",
					CompletedAt: recentCompletion(),
				}, nil
			},
		},
	})

	_, err := svc.CreateReview(context.Background(), CreateReviewInput{
		ProposalID:   uuid.New(),
		ReviewerID:   providerID,
		GlobalRating: 4,
		Timeliness:   &timeliness,
	})

	assert.ErrorIs(t, err, domain.ErrInvalidSubCriteriaForSide)
}

func TestService_CreateReview_Reveal_OnSecondSubmission(t *testing.T) {
	proposalID := uuid.New()
	clientID := uuid.New()
	providerID := uuid.New()

	svc := NewService(ServiceDeps{
		Notifications: &mockNotificationSender{}, Users: &mockUserRepo{},
		Reviews: &mockReviewRepo{
			hasReviewedFn: func(_ context.Context, _, _ uuid.UUID) (bool, error) {
				return false, nil
			},
			createAndRevealFn: func(_ context.Context, r *domain.Review) (*domain.Review, error) {
				// Simulate the repo atomically flipping the review to
				// published because the pair is now complete.
				now := time.Now()
				r.PublishedAt = &now
				return r, nil
			},
		},
		Proposals: &mockProposalRepo{
			getByIDFn: func(_ context.Context, _ uuid.UUID) (*mockProposal, error) {
				return &mockProposal{
					ID:          proposalID,
					ClientID:    clientID,
					ProviderID:  providerID,
					Status:      "completed",
					CompletedAt: recentCompletion(),
				}, nil
			},
		},
	})

	r, err := svc.CreateReview(context.Background(), CreateReviewInput{
		ProposalID:   proposalID,
		ReviewerID:   providerID,
		GlobalRating: 5,
		Comment:      "Great client!",
	})

	assert.NoError(t, err)
	assert.NotNil(t, r)
	assert.NotNil(t, r.PublishedAt, "reveal must populate published_at")
}

func TestService_CreateReview_WindowClosed(t *testing.T) {
	proposalID := uuid.New()
	clientID := uuid.New()
	providerID := uuid.New()

	oldCompletion := time.Now().Add(-15 * 24 * time.Hour)

	svc := NewService(ServiceDeps{
		Notifications: &mockNotificationSender{}, Users: &mockUserRepo{},
		Reviews:       &mockReviewRepo{},
		Proposals: &mockProposalRepo{
			getByIDFn: func(_ context.Context, _ uuid.UUID) (*mockProposal, error) {
				return &mockProposal{
					ID:          proposalID,
					ClientID:    clientID,
					ProviderID:  providerID,
					Status:      "completed",
					CompletedAt: &oldCompletion,
				}, nil
			},
		},
	})

	_, err := svc.CreateReview(context.Background(), CreateReviewInput{
		ProposalID:   proposalID,
		ReviewerID:   clientID,
		GlobalRating: 5,
	})

	assert.ErrorIs(t, err, domain.ErrReviewWindowClosed)
}

func TestService_CreateReview_WindowMissingCompletedAt(t *testing.T) {
	clientID := uuid.New()
	providerID := uuid.New()

	svc := NewService(ServiceDeps{
		Notifications: &mockNotificationSender{}, Users: &mockUserRepo{},
		Reviews:       &mockReviewRepo{},
		Proposals: &mockProposalRepo{
			getByIDFn: func(_ context.Context, _ uuid.UUID) (*mockProposal, error) {
				return &mockProposal{
					ClientID:   clientID,
					ProviderID: providerID,
					Status:     "completed",
					// CompletedAt nil — treat as closed (defensive).
				}, nil
			},
		},
	})

	_, err := svc.CreateReview(context.Background(), CreateReviewInput{
		ProposalID:   uuid.New(),
		ReviewerID:   clientID,
		GlobalRating: 4,
	})

	assert.ErrorIs(t, err, domain.ErrReviewWindowClosed)
}

func TestService_CreateReview_NotCompleted(t *testing.T) {
	svc := NewService(ServiceDeps{
		Notifications: &mockNotificationSender{}, Users: &mockUserRepo{},
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
		Notifications: &mockNotificationSender{}, Users: &mockUserRepo{},
		Reviews: &mockReviewRepo{},
		Proposals: &mockProposalRepo{
			getByIDFn: func(_ context.Context, _ uuid.UUID) (*mockProposal, error) {
				return &mockProposal{
					ClientID:    uuid.New(),
					ProviderID:  uuid.New(),
					Status:      "completed",
					CompletedAt: recentCompletion(),
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
		Notifications: &mockNotificationSender{}, Users: &mockUserRepo{},
		Reviews: &mockReviewRepo{
			hasReviewedFn: func(_ context.Context, _, _ uuid.UUID) (bool, error) {
				return true, nil
			},
		},
		Proposals: &mockProposalRepo{
			getByIDFn: func(_ context.Context, _ uuid.UUID) (*mockProposal, error) {
				return &mockProposal{
					ClientID:    clientID,
					ProviderID:  uuid.New(),
					Status:      "completed",
					CompletedAt: recentCompletion(),
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

func TestService_CreateReview_NotifiesCounterpart(t *testing.T) {
	clientID := uuid.New()
	providerID := uuid.New()
	notif := &mockNotificationSender{}

	svc := NewService(ServiceDeps{
		Notifications: notif,
		Users:         &mockUserRepo{},
		Reviews: &mockReviewRepo{
			hasReviewedFn: func(_ context.Context, _, _ uuid.UUID) (bool, error) {
				return false, nil
			},
			createAndRevealFn: func(_ context.Context, r *domain.Review) (*domain.Review, error) {
				return r, nil // still hidden, no reveal
			},
		},
		Proposals: &mockProposalRepo{
			getByIDFn: func(_ context.Context, _ uuid.UUID) (*mockProposal, error) {
				return &mockProposal{
					ClientID:    clientID,
					ProviderID:  providerID,
					Status:      "completed",
					CompletedAt: recentCompletion(),
				}, nil
			},
		},
	})

	_, err := svc.CreateReview(context.Background(), CreateReviewInput{
		ProposalID:   uuid.New(),
		ReviewerID:   clientID,
		GlobalRating: 5,
	})

	assert.NoError(t, err)
	// One notification: counterpart. No reveal ⇒ no second notif to
	// the reviewer.
	assert.Len(t, notif.calls, 1)
	assert.Equal(t, providerID, notif.calls[0].UserID)
}

func TestService_CreateReview_Reveal_NotifiesBothParties(t *testing.T) {
	clientID := uuid.New()
	providerID := uuid.New()
	notif := &mockNotificationSender{}

	svc := NewService(ServiceDeps{
		Notifications: notif,
		Users:         &mockUserRepo{},
		Reviews: &mockReviewRepo{
			hasReviewedFn: func(_ context.Context, _, _ uuid.UUID) (bool, error) {
				return false, nil
			},
			createAndRevealFn: func(_ context.Context, r *domain.Review) (*domain.Review, error) {
				now := time.Now()
				r.PublishedAt = &now
				return r, nil
			},
		},
		Proposals: &mockProposalRepo{
			getByIDFn: func(_ context.Context, _ uuid.UUID) (*mockProposal, error) {
				return &mockProposal{
					ClientID:    clientID,
					ProviderID:  providerID,
					Status:      "completed",
					CompletedAt: recentCompletion(),
				}, nil
			},
		},
	})

	_, err := svc.CreateReview(context.Background(), CreateReviewInput{
		ProposalID:   uuid.New(),
		ReviewerID:   providerID,
		GlobalRating: 4,
	})

	assert.NoError(t, err)
	// Two notifications: counterpart (client) + reviewer (provider).
	assert.Len(t, notif.calls, 2)
}

func TestService_CanReview_Client(t *testing.T) {
	proposalID := uuid.New()
	clientID := uuid.New()
	providerID := uuid.New()

	svc := NewService(ServiceDeps{
		Notifications: &mockNotificationSender{}, Users: &mockUserRepo{},
		Reviews: &mockReviewRepo{
			hasReviewedFn: func(_ context.Context, _, _ uuid.UUID) (bool, error) {
				return false, nil
			},
		},
		Proposals: &mockProposalRepo{
			getByIDFn: func(_ context.Context, _ uuid.UUID) (*mockProposal, error) {
				return &mockProposal{
					ID:          proposalID,
					ClientID:    clientID,
					ProviderID:  providerID,
					Status:      "completed",
					CompletedAt: recentCompletion(),
				}, nil
			},
		},
	})

	can, err := svc.CanReview(context.Background(), proposalID, clientID)
	assert.NoError(t, err)
	assert.True(t, can)
}

func TestService_CanReview_ProviderNowAllowed(t *testing.T) {
	clientID := uuid.New()
	providerID := uuid.New()

	svc := NewService(ServiceDeps{
		Notifications: &mockNotificationSender{}, Users: &mockUserRepo{},
		Reviews: &mockReviewRepo{
			hasReviewedFn: func(_ context.Context, _, _ uuid.UUID) (bool, error) {
				return false, nil
			},
		},
		Proposals: &mockProposalRepo{
			getByIDFn: func(_ context.Context, _ uuid.UUID) (*mockProposal, error) {
				return &mockProposal{
					ClientID:    clientID,
					ProviderID:  providerID,
					Status:      "completed",
					CompletedAt: recentCompletion(),
				}, nil
			},
		},
	})

	can, err := svc.CanReview(context.Background(), uuid.New(), providerID)
	assert.NoError(t, err)
	assert.True(t, can, "provider must now be able to review the client")
}

func TestService_CanReview_WindowClosed(t *testing.T) {
	clientID := uuid.New()
	providerID := uuid.New()
	old := time.Now().Add(-30 * 24 * time.Hour)

	svc := NewService(ServiceDeps{
		Notifications: &mockNotificationSender{}, Users: &mockUserRepo{},
		Reviews:       &mockReviewRepo{},
		Proposals: &mockProposalRepo{
			getByIDFn: func(_ context.Context, _ uuid.UUID) (*mockProposal, error) {
				return &mockProposal{
					ClientID:    clientID,
					ProviderID:  providerID,
					Status:      "completed",
					CompletedAt: &old,
				}, nil
			},
		},
	})

	can, err := svc.CanReview(context.Background(), uuid.New(), clientID)
	assert.NoError(t, err)
	assert.False(t, can, "window closed: CanReview must return false")
}

func TestService_CanReview_NotCompleted(t *testing.T) {
	svc := NewService(ServiceDeps{
		Notifications: &mockNotificationSender{}, Users: &mockUserRepo{},
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
