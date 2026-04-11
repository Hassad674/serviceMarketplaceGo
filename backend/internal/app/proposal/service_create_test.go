package proposal

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/domain/organization"
	domain "marketplace-backend/internal/domain/proposal"
	"marketplace-backend/internal/domain/user"
)

// --- CreateProposal validation tests ---

func TestCreateProposal_EmptyTitle(t *testing.T) {
	enterpriseID := uuid.New()
	providerID := uuid.New()

	userRepo := &mockUserRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*user.User, error) {
			if id == enterpriseID {
				return makeUser(id, user.RoleEnterprise), nil
			}
			return makeUser(id, user.RoleProvider), nil
		},
	}
	svc := newTestService(nil, userRepo, nil, nil)

	_, err := svc.CreateProposal(context.Background(), CreateProposalInput{
		ConversationID: uuid.New(),
		SenderID:       enterpriseID,
		RecipientID:    providerID,
		Title:          "",
		Description:    "Some description",
		Amount:         5000,
	})

	assert.ErrorIs(t, err, domain.ErrEmptyTitle)
}

func TestCreateProposal_EmptyDescription(t *testing.T) {
	enterpriseID := uuid.New()
	providerID := uuid.New()

	userRepo := &mockUserRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*user.User, error) {
			if id == enterpriseID {
				return makeUser(id, user.RoleEnterprise), nil
			}
			return makeUser(id, user.RoleProvider), nil
		},
	}
	svc := newTestService(nil, userRepo, nil, nil)

	_, err := svc.CreateProposal(context.Background(), CreateProposalInput{
		ConversationID: uuid.New(),
		SenderID:       enterpriseID,
		RecipientID:    providerID,
		Title:          "Valid title",
		Description:    "",
		Amount:         5000,
	})

	assert.ErrorIs(t, err, domain.ErrEmptyDescription)
}

func TestCreateProposal_InvalidAmount(t *testing.T) {
	enterpriseID := uuid.New()
	providerID := uuid.New()

	userRepo := &mockUserRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*user.User, error) {
			if id == enterpriseID {
				return makeUser(id, user.RoleEnterprise), nil
			}
			return makeUser(id, user.RoleProvider), nil
		},
	}
	svc := newTestService(nil, userRepo, nil, nil)

	tests := []struct {
		name   string
		amount int64
	}{
		{"zero amount", 0},
		{"negative amount", -500},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.CreateProposal(context.Background(), CreateProposalInput{
				ConversationID: uuid.New(),
				SenderID:       enterpriseID,
				RecipientID:    providerID,
				Title:          "Valid title",
				Description:    "Valid desc",
				Amount:         tt.amount,
			})

			assert.ErrorIs(t, err, domain.ErrInvalidAmount)
		})
	}
}

func TestCreateProposal_BelowMinimumAmount(t *testing.T) {
	enterpriseID := uuid.New()
	providerID := uuid.New()

	userRepo := &mockUserRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*user.User, error) {
			if id == enterpriseID {
				return makeUser(id, user.RoleEnterprise), nil
			}
			return makeUser(id, user.RoleProvider), nil
		},
	}
	svc := newTestService(nil, userRepo, nil, nil)

	tests := []struct {
		name   string
		amount int64
	}{
		{"below minimum 1000 centimes", 1000},
		{"below minimum 2999 centimes", 2999},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.CreateProposal(context.Background(), CreateProposalInput{
				ConversationID: uuid.New(),
				SenderID:       enterpriseID,
				RecipientID:    providerID,
				Title:          "Valid title",
				Description:    "Valid desc",
				Amount:         tt.amount,
			})

			assert.ErrorIs(t, err, domain.ErrBelowMinimumAmount)
		})
	}
}

func TestCreateProposal_ExactMinimumAmountSucceeds(t *testing.T) {
	enterpriseID := uuid.New()
	providerID := uuid.New()

	userRepo := &mockUserRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*user.User, error) {
			if id == enterpriseID {
				return makeUser(id, user.RoleEnterprise), nil
			}
			return makeUser(id, user.RoleProvider), nil
		},
	}
	svc := newTestService(nil, userRepo, &mockMessageSender{}, nil)

	p, err := svc.CreateProposal(context.Background(), CreateProposalInput{
		ConversationID: uuid.New(),
		SenderID:       enterpriseID,
		RecipientID:    providerID,
		Title:          "Minimum amount proposal",
		Description:    "Exactly 30 EUR",
		Amount:         3000,
	})

	require.NoError(t, err)
	assert.Equal(t, int64(3000), p.Amount)
}

// --- CreateProposal user lookup errors ---

func TestCreateProposal_SenderNotFound(t *testing.T) {
	senderID := uuid.New()
	recipientID := uuid.New()

	userRepo := &mockUserRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*user.User, error) {
			if id == senderID {
				return nil, errors.New("user not found")
			}
			return makeUser(id, user.RoleProvider), nil
		},
	}
	svc := newTestService(nil, userRepo, nil, nil)

	_, err := svc.CreateProposal(context.Background(), CreateProposalInput{
		ConversationID: uuid.New(),
		SenderID:       senderID,
		RecipientID:    recipientID,
		Title:          "Test",
		Description:    "Test desc",
		Amount:         5000,
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "get sender")
}

func TestCreateProposal_RecipientNotFound(t *testing.T) {
	senderID := uuid.New()
	recipientID := uuid.New()

	userRepo := &mockUserRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*user.User, error) {
			if id == senderID {
				return makeUser(id, user.RoleEnterprise), nil
			}
			return nil, errors.New("user not found")
		},
	}
	svc := newTestService(nil, userRepo, nil, nil)

	_, err := svc.CreateProposal(context.Background(), CreateProposalInput{
		ConversationID: uuid.New(),
		SenderID:       senderID,
		RecipientID:    recipientID,
		Title:          "Test",
		Description:    "Test desc",
		Amount:         5000,
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "get recipient")
}

// --- CreateProposal repo persistence error ---

func TestCreateProposal_PersistError(t *testing.T) {
	enterpriseID := uuid.New()
	providerID := uuid.New()

	userRepo := &mockUserRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*user.User, error) {
			if id == enterpriseID {
				return makeUser(id, user.RoleEnterprise), nil
			}
			return makeUser(id, user.RoleProvider), nil
		},
	}
	proposalRepo := &mockProposalRepo{
		createWithDocsFn: func(_ context.Context, _ *domain.Proposal, _ []*domain.ProposalDocument) error {
			return errors.New("unique constraint violated")
		},
	}
	svc := newTestService(proposalRepo, userRepo, nil, nil)

	_, err := svc.CreateProposal(context.Background(), CreateProposalInput{
		ConversationID: uuid.New(),
		SenderID:       enterpriseID,
		RecipientID:    providerID,
		Title:          "Build website",
		Description:    "Full redesign",
		Amount:         500000,
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "persist proposal")
}

// --- CreateProposal role combinations ---

func TestCreateProposal_RoleCombinations(t *testing.T) {
	tests := []struct {
		name       string
		senderRole user.Role
		recipRole  user.Role
		wantErr    error
	}{
		{
			name:       "enterprise sends to provider",
			senderRole: user.RoleEnterprise,
			recipRole:  user.RoleProvider,
		},
		{
			name:       "provider sends to enterprise",
			senderRole: user.RoleProvider,
			recipRole:  user.RoleEnterprise,
		},
		{
			name:       "enterprise sends to agency",
			senderRole: user.RoleEnterprise,
			recipRole:  user.RoleAgency,
		},
		{
			name:       "agency sends to enterprise",
			senderRole: user.RoleAgency,
			recipRole:  user.RoleEnterprise,
		},
		{
			name:       "agency sends to provider",
			senderRole: user.RoleAgency,
			recipRole:  user.RoleProvider,
		},
		{
			name:       "provider sends to agency",
			senderRole: user.RoleProvider,
			recipRole:  user.RoleAgency,
		},
		{
			name:       "provider to provider is invalid",
			senderRole: user.RoleProvider,
			recipRole:  user.RoleProvider,
			wantErr:    domain.ErrInvalidRoleCombination,
		},
		{
			name:       "enterprise to enterprise is invalid",
			senderRole: user.RoleEnterprise,
			recipRole:  user.RoleEnterprise,
			wantErr:    domain.ErrInvalidRoleCombination,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			senderID := uuid.New()
			recipientID := uuid.New()

			userRepo := &mockUserRepo{
				getByIDFn: func(_ context.Context, id uuid.UUID) (*user.User, error) {
					if id == senderID {
						return makeUser(id, tt.senderRole), nil
					}
					return makeUser(id, tt.recipRole), nil
				},
			}
			msgs := &mockMessageSender{}
			svc := newTestService(nil, userRepo, msgs, nil)

			p, err := svc.CreateProposal(context.Background(), CreateProposalInput{
				ConversationID: uuid.New(),
				SenderID:       senderID,
				RecipientID:    recipientID,
				Title:          "Test proposal",
				Description:    "Test description",
				Amount:         10000,
			})

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, p)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, p)
			assert.Equal(t, domain.StatusPending, p.Status)
			assert.Equal(t, 1, p.Version)
		})
	}
}

// --- CreateProposal verifies client/provider assignment ---

func TestCreateProposal_ClientProviderAssignment(t *testing.T) {
	enterpriseID := uuid.New()
	providerID := uuid.New()

	userRepo := &mockUserRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*user.User, error) {
			if id == enterpriseID {
				return makeUser(id, user.RoleEnterprise), nil
			}
			return makeUser(id, user.RoleProvider), nil
		},
	}
	msgs := &mockMessageSender{}

	t.Run("enterprise sender is client", func(t *testing.T) {
		svc := newTestService(nil, userRepo, msgs, nil)

		p, err := svc.CreateProposal(context.Background(), CreateProposalInput{
			ConversationID: uuid.New(),
			SenderID:       enterpriseID,
			RecipientID:    providerID,
			Title:          "Build API",
			Description:    "REST API development",
			Amount:         100000,
		})

		require.NoError(t, err)
		assert.Equal(t, enterpriseID, p.ClientID)
		assert.Equal(t, providerID, p.ProviderID)
	})

	t.Run("provider sender means enterprise is still client", func(t *testing.T) {
		msgs2 := &mockMessageSender{}
		svc := newTestService(nil, userRepo, msgs2, nil)

		p, err := svc.CreateProposal(context.Background(), CreateProposalInput{
			ConversationID: uuid.New(),
			SenderID:       providerID,
			RecipientID:    enterpriseID,
			Title:          "Build API",
			Description:    "REST API development",
			Amount:         100000,
		})

		require.NoError(t, err)
		assert.Equal(t, enterpriseID, p.ClientID)
		assert.Equal(t, providerID, p.ProviderID)
	})
}

// --- ModifyProposal additional edge cases ---

func TestModifyProposal_PersistError(t *testing.T) {
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
				Title:          "Original",
				Amount:         5000,
				Version:        1,
			}, nil
		},
		createWithDocsFn: func(_ context.Context, _ *domain.Proposal, _ []*domain.ProposalDocument) error {
			return errors.New("write failed")
		},
	}
	svc := newTestService(proposalRepo, nil, nil, nil)

	_, err := svc.ModifyProposal(context.Background(), ModifyProposalInput{
		ProposalID:  uuid.New(),
		UserID:      recipientID,
		Title:       "Counter",
		Description: "Counter desc",
		Amount:      5000,
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "persist modified proposal")
}

func TestModifyProposal_WithDocuments(t *testing.T) {
	senderID := uuid.New()
	recipientID := uuid.New()
	deadline := time.Now().Add(60 * 24 * time.Hour)

	var capturedDocs []*domain.ProposalDocument
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
				Title:          "Original",
				Amount:         5000,
				Version:        1,
			}, nil
		},
		createWithDocsFn: func(_ context.Context, _ *domain.Proposal, docs []*domain.ProposalDocument) error {
			capturedDocs = docs
			return nil
		},
	}
	msgs := &mockMessageSender{}
	svc := newTestService(proposalRepo, nil, msgs, nil)

	modified, err := svc.ModifyProposal(context.Background(), ModifyProposalInput{
		ProposalID:  uuid.New(),
		UserID:      recipientID,
		Title:       "Counter with docs",
		Description: "With attached spec",
		Amount:      5000,
		Deadline:    &deadline,
		Documents: []DocumentInput{
			{Filename: "contract.pdf", URL: "https://s3/contract.pdf", Size: 1024, MimeType: "application/pdf"},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, "Counter with docs", modified.Title)
	assert.Equal(t, 2, modified.Version)
	require.Len(t, capturedDocs, 1)
	assert.Equal(t, "contract.pdf", capturedDocs[0].Filename)
	assert.Equal(t, modified.ID, capturedDocs[0].ProposalID)
}

func TestModifyProposal_GetError(t *testing.T) {
	repo := &mockProposalRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Proposal, error) {
			return nil, errors.New("db error")
		},
	}
	svc := newTestService(repo, nil, nil, nil)

	_, err := svc.ModifyProposal(context.Background(), ModifyProposalInput{
		ProposalID:  uuid.New(),
		UserID:      uuid.New(),
		Title:       "Test",
		Description: "Test",
		Amount:      5000,
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "get proposal")
}

// --- KYC enforcement tests ---

func TestCreateProposal_KYCBlocked(t *testing.T) {
	enterpriseID := uuid.New()
	providerID := uuid.New()
	past15 := time.Now().Add(-15 * 24 * time.Hour)

	userRepo := &mockUserRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*user.User, error) {
			if id == enterpriseID {
				return makeUser(id, user.RoleEnterprise), nil
			}
			return makeUser(id, user.RoleProvider), nil
		},
	}
	orgRepo := &mockOrgRepo{
		findByUserIDFn: func(_ context.Context, uid uuid.UUID) (*organization.Organization, error) {
			// The provider's org is blocked (15 days elapsed without Stripe).
			if uid == providerID {
				return &organization.Organization{
					ID:                uuid.New(),
					Type:              organization.OrgTypeProviderPersonal,
					KYCFirstEarningAt: &past15,
				}, nil
			}
			return &organization.Organization{ID: uuid.New(), Type: organization.OrgTypeEnterprise}, nil
		},
	}
	svc := newTestServiceWithCreditsAndOrgs(nil, userRepo, orgRepo, nil, nil, nil)

	_, err := svc.CreateProposal(context.Background(), CreateProposalInput{
		ConversationID: uuid.New(),
		SenderID:       providerID,
		RecipientID:    enterpriseID,
		Title:          "Test proposal",
		Description:    "Test description",
		Amount:         5000,
	})

	assert.ErrorIs(t, err, user.ErrKYCRestricted)
}

func TestCreateProposal_KYCCompleted_OK(t *testing.T) {
	enterpriseID := uuid.New()
	providerID := uuid.New()
	past15 := time.Now().Add(-15 * 24 * time.Hour)
	stripeID := "acct_123"

	userRepo := &mockUserRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*user.User, error) {
			if id == enterpriseID {
				return makeUser(id, user.RoleEnterprise), nil
			}
			return makeUser(id, user.RoleProvider), nil
		},
	}
	orgRepo := &mockOrgRepo{
		findByUserIDFn: func(_ context.Context, _ uuid.UUID) (*organization.Organization, error) {
			// Stripe account exists → KYC completed, not blocked even after 15 days.
			return &organization.Organization{
				ID:                uuid.New(),
				Type:              organization.OrgTypeProviderPersonal,
				KYCFirstEarningAt: &past15,
				StripeAccountID:   &stripeID,
			}, nil
		},
	}
	proposalRepo := &mockProposalRepo{}
	svc := newTestServiceWithCreditsAndOrgs(proposalRepo, userRepo, orgRepo, nil, nil, nil)

	p, err := svc.CreateProposal(context.Background(), CreateProposalInput{
		ConversationID: uuid.New(),
		SenderID:       providerID,
		RecipientID:    enterpriseID,
		Title:          "Test proposal",
		Description:    "Test description",
		Amount:         5000,
	})

	assert.NoError(t, err)
	assert.NotNil(t, p)
}
