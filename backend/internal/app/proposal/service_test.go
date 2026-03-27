package proposal

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domain "marketplace-backend/internal/domain/proposal"
	"marketplace-backend/internal/domain/user"
)

// --- helpers ---

func newTestService(
	proposalRepo *mockProposalRepo,
	userRepo *mockUserRepo,
	msgSender *mockMessageSender,
	storage *mockStorageService,
) *Service {
	if proposalRepo == nil {
		proposalRepo = &mockProposalRepo{}
	}
	if userRepo == nil {
		userRepo = &mockUserRepo{}
	}
	if msgSender == nil {
		msgSender = &mockMessageSender{}
	}
	if storage == nil {
		storage = &mockStorageService{}
	}
	return NewService(ServiceDeps{
		Proposals: proposalRepo,
		Users:     userRepo,
		Messages:  msgSender,
		Storage:   storage,
	})
}

func makeUser(id uuid.UUID, role user.Role) *user.User {
	return &user.User{ID: id, Role: role, DisplayName: "Test " + string(role)}
}

// --- CreateProposal tests ---

func TestCreateProposal_Success(t *testing.T) {
	enterpriseID := uuid.New()
	providerID := uuid.New()
	convID := uuid.New()

	userRepo := &mockUserRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*user.User, error) {
			if id == enterpriseID {
				return makeUser(id, user.RoleEnterprise), nil
			}
			return makeUser(id, user.RoleProvider), nil
		},
	}
	proposalRepo := &mockProposalRepo{}
	msgSender := &mockMessageSender{}

	svc := newTestService(proposalRepo, userRepo, msgSender, nil)

	p, err := svc.CreateProposal(context.Background(), CreateProposalInput{
		ConversationID: convID,
		SenderID:       enterpriseID,
		RecipientID:    providerID,
		Title:          "Build website",
		Description:    "Full redesign of the corporate website",
		Amount:         500000,
	})

	require.NoError(t, err)
	assert.Equal(t, "Build website", p.Title)
	assert.Equal(t, int64(500000), p.Amount)
	assert.Equal(t, enterpriseID, p.ClientID)
	assert.Equal(t, providerID, p.ProviderID)
	assert.Equal(t, domain.StatusPending, p.Status)
	assert.Len(t, msgSender.calls, 1)
	assert.Equal(t, "proposal_sent", msgSender.calls[0].Type)
}

func TestCreateProposal_SameUser(t *testing.T) {
	userID := uuid.New()
	svc := newTestService(nil, nil, nil, nil)

	_, err := svc.CreateProposal(context.Background(), CreateProposalInput{
		ConversationID: uuid.New(),
		SenderID:       userID,
		RecipientID:    userID,
		Title:          "Test",
		Description:    "Test",
		Amount:         1000,
	})

	assert.ErrorIs(t, err, domain.ErrSameUser)
}

func TestCreateProposal_InvalidRoles(t *testing.T) {
	providerA := uuid.New()
	providerB := uuid.New()

	userRepo := &mockUserRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*user.User, error) {
			return makeUser(id, user.RoleProvider), nil
		},
	}

	svc := newTestService(nil, userRepo, nil, nil)

	_, err := svc.CreateProposal(context.Background(), CreateProposalInput{
		ConversationID: uuid.New(),
		SenderID:       providerA,
		RecipientID:    providerB,
		Title:          "Test",
		Description:    "Test",
		Amount:         1000,
	})

	assert.ErrorIs(t, err, domain.ErrInvalidRoleCombination)
}

// --- AcceptProposal tests ---

func TestAcceptProposal_ByRecipient(t *testing.T) {
	senderID := uuid.New()
	recipientID := uuid.New()
	proposalID := uuid.New()

	proposalRepo := &mockProposalRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Proposal, error) {
			return &domain.Proposal{
				ID:             proposalID,
				ConversationID: uuid.New(),
				SenderID:       senderID,
				RecipientID:    recipientID,
				ClientID:       senderID,
				ProviderID:     recipientID,
				Status:         domain.StatusPending,
				Title:          "Test",
				Amount:         1000,
			}, nil
		},
	}
	msgSender := &mockMessageSender{}

	svc := newTestService(proposalRepo, nil, msgSender, nil)

	err := svc.AcceptProposal(context.Background(), AcceptProposalInput{
		ProposalID: proposalID,
		UserID:     recipientID,
	})

	require.NoError(t, err)
	// Recipient is the provider, so we expect 2 messages: accepted + payment_requested
	assert.Len(t, msgSender.calls, 2)
	assert.Equal(t, "proposal_accepted", msgSender.calls[0].Type)
	assert.Equal(t, "proposal_payment_requested", msgSender.calls[1].Type)
}

func TestAcceptProposal_BySender_Fails(t *testing.T) {
	senderID := uuid.New()
	recipientID := uuid.New()

	proposalRepo := &mockProposalRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Proposal, error) {
			return &domain.Proposal{
				ID:          uuid.New(),
				SenderID:    senderID,
				RecipientID: recipientID,
				ClientID:    senderID,
				ProviderID:  recipientID,
				Status:      domain.StatusPending,
			}, nil
		},
	}

	svc := newTestService(proposalRepo, nil, nil, nil)

	err := svc.AcceptProposal(context.Background(), AcceptProposalInput{
		ProposalID: uuid.New(),
		UserID:     senderID,
	})

	assert.ErrorIs(t, err, domain.ErrNotAuthorized)
}

func TestAcceptProposal_ProviderAccepts_SendsPaymentRequest(t *testing.T) {
	clientID := uuid.New()
	providerID := uuid.New()

	proposalRepo := &mockProposalRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Proposal, error) {
			return &domain.Proposal{
				ID:             uuid.New(),
				ConversationID: uuid.New(),
				SenderID:       clientID,
				RecipientID:    providerID,
				ClientID:       clientID,
				ProviderID:     providerID,
				Status:         domain.StatusPending,
				Title:          "Test",
				Amount:         1000,
			}, nil
		},
	}
	msgSender := &mockMessageSender{}

	svc := newTestService(proposalRepo, nil, msgSender, nil)

	err := svc.AcceptProposal(context.Background(), AcceptProposalInput{
		ProposalID: uuid.New(),
		UserID:     providerID,
	})

	require.NoError(t, err)
	assert.Len(t, msgSender.calls, 2)
	assert.Equal(t, "proposal_payment_requested", msgSender.calls[1].Type)
}

// --- DeclineProposal tests ---

func TestDeclineProposal_Success(t *testing.T) {
	senderID := uuid.New()
	recipientID := uuid.New()

	proposalRepo := &mockProposalRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Proposal, error) {
			return &domain.Proposal{
				ID:             uuid.New(),
				ConversationID: uuid.New(),
				SenderID:       senderID,
				RecipientID:    recipientID,
				ClientID:       senderID,
				ProviderID:     recipientID,
				Status:         domain.StatusPending,
				Title:          "Test",
				Amount:         1000,
			}, nil
		},
	}
	msgSender := &mockMessageSender{}

	svc := newTestService(proposalRepo, nil, msgSender, nil)

	err := svc.DeclineProposal(context.Background(), DeclineProposalInput{
		ProposalID: uuid.New(),
		UserID:     recipientID,
	})

	require.NoError(t, err)
	assert.Len(t, msgSender.calls, 1)
	assert.Equal(t, "proposal_declined", msgSender.calls[0].Type)
}

// --- ModifyProposal tests ---

func TestModifyProposal_Success(t *testing.T) {
	senderID := uuid.New()
	recipientID := uuid.New()
	proposalID := uuid.New()

	proposalRepo := &mockProposalRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Proposal, error) {
			return &domain.Proposal{
				ID:             proposalID,
				ConversationID: uuid.New(),
				SenderID:       senderID,
				RecipientID:    recipientID,
				ClientID:       senderID,
				ProviderID:     recipientID,
				Status:         domain.StatusPending,
				Title:          "Original",
				Amount:         1000,
				Version:        1,
			}, nil
		},
	}
	msgSender := &mockMessageSender{}

	svc := newTestService(proposalRepo, nil, msgSender, nil)

	modified, err := svc.ModifyProposal(context.Background(), ModifyProposalInput{
		ProposalID:  proposalID,
		UserID:      recipientID,
		Title:       "Counter proposal",
		Description: "Different terms",
		Amount:      800000,
	})

	require.NoError(t, err)
	assert.Equal(t, "Counter proposal", modified.Title)
	assert.Equal(t, int64(800000), modified.Amount)
	assert.Equal(t, 2, modified.Version)
	assert.Equal(t, &proposalID, modified.ParentID)
	assert.Len(t, msgSender.calls, 1)
	assert.Equal(t, "proposal_modified", msgSender.calls[0].Type)
}

func TestModifyProposal_BySender_Fails(t *testing.T) {
	senderID := uuid.New()
	recipientID := uuid.New()

	proposalRepo := &mockProposalRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Proposal, error) {
			return &domain.Proposal{
				ID:          uuid.New(),
				SenderID:    senderID,
				RecipientID: recipientID,
				Status:      domain.StatusPending,
			}, nil
		},
	}

	svc := newTestService(proposalRepo, nil, nil, nil)

	_, err := svc.ModifyProposal(context.Background(), ModifyProposalInput{
		ProposalID:  uuid.New(),
		UserID:      senderID,
		Title:       "Test",
		Description: "Test",
		Amount:      1000,
	})

	assert.ErrorIs(t, err, domain.ErrCannotModify)
}

func TestModifyProposal_NotPending_Fails(t *testing.T) {
	senderID := uuid.New()
	recipientID := uuid.New()

	proposalRepo := &mockProposalRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Proposal, error) {
			return &domain.Proposal{
				ID:          uuid.New(),
				SenderID:    senderID,
				RecipientID: recipientID,
				Status:      domain.StatusAccepted,
			}, nil
		},
	}

	svc := newTestService(proposalRepo, nil, nil, nil)

	_, err := svc.ModifyProposal(context.Background(), ModifyProposalInput{
		ProposalID:  uuid.New(),
		UserID:      recipientID,
		Title:       "Test",
		Description: "Test",
		Amount:      1000,
	})

	assert.ErrorIs(t, err, domain.ErrCannotModify)
}

// --- SimulatePayment tests ---

func TestSimulatePayment_Success(t *testing.T) {
	clientID := uuid.New()
	providerID := uuid.New()
	now := time.Now()

	proposalRepo := &mockProposalRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Proposal, error) {
			return &domain.Proposal{
				ID:             uuid.New(),
				ConversationID: uuid.New(),
				SenderID:       providerID,
				RecipientID:    clientID,
				ClientID:       clientID,
				ProviderID:     providerID,
				Status:         domain.StatusAccepted,
				AcceptedAt:     &now,
				Title:          "Test",
				Amount:         1000,
			}, nil
		},
	}
	msgSender := &mockMessageSender{}

	svc := newTestService(proposalRepo, nil, msgSender, nil)

	err := svc.SimulatePayment(context.Background(), PayProposalInput{
		ProposalID: uuid.New(),
		UserID:     clientID,
	})

	require.NoError(t, err)
	assert.Len(t, msgSender.calls, 1)
	assert.Equal(t, "proposal_paid", msgSender.calls[0].Type)
}

func TestSimulatePayment_ByProvider_Fails(t *testing.T) {
	clientID := uuid.New()
	providerID := uuid.New()
	now := time.Now()

	proposalRepo := &mockProposalRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Proposal, error) {
			return &domain.Proposal{
				ID:         uuid.New(),
				ClientID:   clientID,
				ProviderID: providerID,
				Status:     domain.StatusAccepted,
				AcceptedAt: &now,
			}, nil
		},
	}

	svc := newTestService(proposalRepo, nil, nil, nil)

	err := svc.SimulatePayment(context.Background(), PayProposalInput{
		ProposalID: uuid.New(),
		UserID:     providerID,
	})

	assert.ErrorIs(t, err, domain.ErrNotAuthorized)
}

func TestSimulatePayment_NotAccepted_Fails(t *testing.T) {
	clientID := uuid.New()

	proposalRepo := &mockProposalRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Proposal, error) {
			return &domain.Proposal{
				ID:       uuid.New(),
				ClientID: clientID,
				Status:   domain.StatusPending,
			}, nil
		},
	}

	svc := newTestService(proposalRepo, nil, nil, nil)

	err := svc.SimulatePayment(context.Background(), PayProposalInput{
		ProposalID: uuid.New(),
		UserID:     clientID,
	})

	assert.ErrorIs(t, err, domain.ErrInvalidStatus)
}

// --- GetProposal tests ---

func TestGetProposal_Authorized(t *testing.T) {
	senderID := uuid.New()
	recipientID := uuid.New()
	proposalID := uuid.New()

	proposalRepo := &mockProposalRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Proposal, error) {
			return &domain.Proposal{
				ID:          proposalID,
				SenderID:    senderID,
				RecipientID: recipientID,
				ClientID:    senderID,
				ProviderID:  recipientID,
				Title:       "Test",
				Amount:      1000,
			}, nil
		},
		getDocumentsFn: func(_ context.Context, _ uuid.UUID) ([]*domain.ProposalDocument, error) {
			return []*domain.ProposalDocument{}, nil
		},
	}

	svc := newTestService(proposalRepo, nil, nil, nil)

	p, docs, err := svc.GetProposal(context.Background(), senderID, proposalID)

	require.NoError(t, err)
	assert.Equal(t, proposalID, p.ID)
	assert.NotNil(t, docs)
}

func TestGetProposal_NotAuthorized(t *testing.T) {
	senderID := uuid.New()
	recipientID := uuid.New()
	outsiderID := uuid.New()

	proposalRepo := &mockProposalRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Proposal, error) {
			return &domain.Proposal{
				ID:          uuid.New(),
				SenderID:    senderID,
				RecipientID: recipientID,
				ClientID:    senderID,
				ProviderID:  recipientID,
			}, nil
		},
	}

	svc := newTestService(proposalRepo, nil, nil, nil)

	_, _, err := svc.GetProposal(context.Background(), outsiderID, uuid.New())

	assert.ErrorIs(t, err, domain.ErrNotAuthorized)
}
