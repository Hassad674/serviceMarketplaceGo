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

// --- AcceptProposal table-driven tests ---

func TestAcceptProposal_TableDriven(t *testing.T) {
	senderID := uuid.New()
	recipientID := uuid.New()
	outsiderID := uuid.New()
	senderOrgID := uuid.New()
	recipientOrgID := uuid.New()
	outsiderOrgID := uuid.New()

	// orgLookup returns, for every requested user id, a user whose
	// OrganizationID matches the side they represent. Keeps the table
	// compact while still exercising the new org-directional checks.
	orgLookup := func(id uuid.UUID) *user.User {
		var o uuid.UUID
		switch id {
		case senderID:
			o = senderOrgID
		case recipientID:
			o = recipientOrgID
		default:
			o = outsiderOrgID
		}
		return &user.User{ID: id, Role: user.RoleEnterprise, OrganizationID: &o}
	}

	tests := []struct {
		name      string
		status    domain.ProposalStatus
		actorID   uuid.UUID
		actorOrg  uuid.UUID
		wantErr   error
		wantMsgs  int
		wantTypes []string
	}{
		{
			name:      "recipient accepts pending proposal",
			status:    domain.StatusPending,
			actorID:   recipientID,
			actorOrg:  recipientOrgID,
			wantMsgs:  2,
			wantTypes: []string{"proposal_accepted", "proposal_payment_requested"},
		},
		{
			name:     "sender cannot accept own proposal",
			status:   domain.StatusPending,
			actorID:  senderID,
			actorOrg: senderOrgID,
			wantErr:  domain.ErrNotAuthorized,
		},
		{
			name:     "outsider cannot accept proposal",
			status:   domain.StatusPending,
			actorID:  outsiderID,
			actorOrg: outsiderOrgID,
			wantErr:  domain.ErrNotAuthorized,
		},
		{
			name:     "cannot accept already accepted proposal",
			status:   domain.StatusAccepted,
			actorID:  recipientID,
			actorOrg: recipientOrgID,
			wantErr:  domain.ErrInvalidStatus,
		},
		{
			name:     "cannot accept declined proposal",
			status:   domain.StatusDeclined,
			actorID:  recipientID,
			actorOrg: recipientOrgID,
			wantErr:  domain.ErrInvalidStatus,
		},
		{
			name:     "cannot accept withdrawn proposal",
			status:   domain.StatusWithdrawn,
			actorID:  recipientID,
			actorOrg: recipientOrgID,
			wantErr:  domain.ErrInvalidStatus,
		},
		{
			name:     "cannot accept active proposal",
			status:   domain.StatusActive,
			actorID:  recipientID,
			actorOrg: recipientOrgID,
			wantErr:  domain.ErrInvalidStatus,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockProposalRepo{
				getByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Proposal, error) {
					return newStubProposal(senderID, recipientID, tt.status), nil
				},
			}
			msgs := &mockMessageSender{}
			userRepo := &mockUserRepo{
				getByIDFn: func(_ context.Context, id uuid.UUID) (*user.User, error) {
					return orgLookup(id), nil
				},
			}
			svc := newTestService(repo, userRepo, msgs, nil)

			err := svc.AcceptProposal(context.Background(), AcceptProposalInput{
				ProposalID: uuid.New(),
				UserID:     tt.actorID,
				OrgID:      tt.actorOrg,
			})

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Len(t, msgs.calls, tt.wantMsgs)
			for i, typ := range tt.wantTypes {
				assert.Equal(t, typ, msgs.calls[i].Type)
			}
		})
	}
}

func TestAcceptProposal_RepoGetError(t *testing.T) {
	repo := &mockProposalRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Proposal, error) {
			return nil, errors.New("db connection lost")
		},
	}
	svc := newTestService(repo, nil, nil, nil)

	err := svc.AcceptProposal(context.Background(), AcceptProposalInput{
		ProposalID: uuid.New(),
		UserID:     uuid.New(),
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "get proposal")
}

func TestAcceptProposal_RepoUpdateError(t *testing.T) {
	senderID := uuid.New()
	recipientID := uuid.New()
	recipientOrgID := uuid.New()

	repo := &mockProposalRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Proposal, error) {
			return newStubProposal(senderID, recipientID, domain.StatusPending), nil
		},
		updateFn: func(_ context.Context, _ *domain.Proposal) error {
			return errors.New("write failed")
		},
	}
	svc := newTestService(repo, orgAwareUserRepo(recipientOrgID), nil, nil)

	err := svc.AcceptProposal(context.Background(), AcceptProposalInput{
		ProposalID: uuid.New(),
		UserID:     recipientID,
		OrgID:      recipientOrgID,
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "update proposal")
}

// --- DeclineProposal table-driven tests ---

func TestDeclineProposal_TableDriven(t *testing.T) {
	senderID := uuid.New()
	recipientID := uuid.New()
	outsiderID := uuid.New()
	senderOrgID := uuid.New()
	recipientOrgID := uuid.New()
	outsiderOrgID := uuid.New()

	orgLookup := func(id uuid.UUID) *user.User {
		var o uuid.UUID
		switch id {
		case senderID:
			o = senderOrgID
		case recipientID:
			o = recipientOrgID
		default:
			o = outsiderOrgID
		}
		return &user.User{ID: id, Role: user.RoleEnterprise, OrganizationID: &o}
	}

	tests := []struct {
		name     string
		status   domain.ProposalStatus
		actorID  uuid.UUID
		actorOrg uuid.UUID
		wantErr  error
	}{
		{
			name:     "recipient declines pending proposal",
			status:   domain.StatusPending,
			actorID:  recipientID,
			actorOrg: recipientOrgID,
		},
		{
			name:     "sender cannot decline own proposal",
			status:   domain.StatusPending,
			actorID:  senderID,
			actorOrg: senderOrgID,
			wantErr:  domain.ErrNotAuthorized,
		},
		{
			name:     "outsider cannot decline proposal",
			status:   domain.StatusPending,
			actorID:  outsiderID,
			actorOrg: outsiderOrgID,
			wantErr:  domain.ErrNotAuthorized,
		},
		{
			name:     "cannot decline accepted proposal",
			status:   domain.StatusAccepted,
			actorID:  recipientID,
			actorOrg: recipientOrgID,
			wantErr:  domain.ErrInvalidStatus,
		},
		{
			name:     "cannot decline already declined proposal",
			status:   domain.StatusDeclined,
			actorID:  recipientID,
			actorOrg: recipientOrgID,
			wantErr:  domain.ErrInvalidStatus,
		},
		{
			name:     "cannot decline active proposal",
			status:   domain.StatusActive,
			actorID:  recipientID,
			actorOrg: recipientOrgID,
			wantErr:  domain.ErrInvalidStatus,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockProposalRepo{
				getByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Proposal, error) {
					return newStubProposal(senderID, recipientID, tt.status), nil
				},
			}
			msgs := &mockMessageSender{}
			userRepo := &mockUserRepo{
				getByIDFn: func(_ context.Context, id uuid.UUID) (*user.User, error) {
					return orgLookup(id), nil
				},
			}
			svc := newTestService(repo, userRepo, msgs, nil)

			err := svc.DeclineProposal(context.Background(), DeclineProposalInput{
				ProposalID: uuid.New(),
				UserID:     tt.actorID,
				OrgID:      tt.actorOrg,
			})

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.Len(t, msgs.calls, 1)
			assert.Equal(t, "proposal_declined", msgs.calls[0].Type)
		})
	}
}

func TestDeclineProposal_RepoGetError(t *testing.T) {
	repo := &mockProposalRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Proposal, error) {
			return nil, errors.New("db timeout")
		},
	}
	svc := newTestService(repo, nil, nil, nil)

	err := svc.DeclineProposal(context.Background(), DeclineProposalInput{
		ProposalID: uuid.New(),
		UserID:     uuid.New(),
		OrgID:      uuid.New(),
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "get proposal")
}

// --- Withdraw tests (not covered in service_test.go) ---

func TestWithdrawProposal_TableDriven(t *testing.T) {
	senderID := uuid.New()
	recipientID := uuid.New()

	tests := []struct {
		name    string
		status  domain.ProposalStatus
		actorID uuid.UUID
		wantErr error
	}{
		{
			name:    "sender withdraws pending proposal",
			status:  domain.StatusPending,
			actorID: senderID,
		},
		{
			name:    "recipient cannot withdraw proposal",
			status:  domain.StatusPending,
			actorID: recipientID,
			wantErr: domain.ErrNotAuthorized,
		},
		{
			name:    "cannot withdraw accepted proposal",
			status:  domain.StatusAccepted,
			actorID: senderID,
			wantErr: domain.ErrInvalidStatus,
		},
		{
			name:    "cannot withdraw declined proposal",
			status:  domain.StatusDeclined,
			actorID: senderID,
			wantErr: domain.ErrInvalidStatus,
		},
		{
			name:    "cannot withdraw active proposal",
			status:  domain.StatusActive,
			actorID: senderID,
			wantErr: domain.ErrInvalidStatus,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockProposalRepo{
				getByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Proposal, error) {
					return newStubProposal(senderID, recipientID, tt.status), nil
				},
			}
			svc := newTestService(repo, nil, nil, nil)

			p, err := svc.proposals.GetByID(context.Background(), uuid.New())
			require.NoError(t, err)

			err = p.Withdraw(tt.actorID)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, domain.StatusWithdrawn, p.Status)
		})
	}
}

// --- RequestCompletion table-driven tests ---

func TestRequestCompletion_TableDriven(t *testing.T) {
	clientID := uuid.New()
	providerID := uuid.New()
	clientOrgID := uuid.New()
	providerOrgID := uuid.New()

	orgLookup := func(id uuid.UUID) *user.User {
		o := providerOrgID
		if id == clientID {
			o = clientOrgID
		}
		return &user.User{ID: id, Role: user.RoleEnterprise, OrganizationID: &o}
	}

	tests := []struct {
		name     string
		status   domain.ProposalStatus
		actorID  uuid.UUID
		actorOrg uuid.UUID
		wantErr  error
	}{
		{
			name:     "provider requests completion on active",
			status:   domain.StatusActive,
			actorID:  providerID,
			actorOrg: providerOrgID,
		},
		{
			name:     "client cannot request completion",
			status:   domain.StatusActive,
			actorID:  clientID,
			actorOrg: clientOrgID,
			wantErr:  domain.ErrNotProvider,
		},
		{
			name:     "cannot request completion on pending",
			status:   domain.StatusPending,
			actorID:  providerID,
			actorOrg: providerOrgID,
			wantErr:  domain.ErrInvalidStatus,
		},
		{
			name:     "cannot request completion on accepted",
			status:   domain.StatusAccepted,
			actorID:  providerID,
			actorOrg: providerOrgID,
			wantErr:  domain.ErrInvalidStatus,
		},
		{
			name:     "cannot request completion on completed",
			status:   domain.StatusCompleted,
			actorID:  providerID,
			actorOrg: providerOrgID,
			wantErr:  domain.ErrInvalidStatus,
		},
		{
			name:     "cannot request completion on already requested",
			status:   domain.StatusCompletionRequested,
			actorID:  providerID,
			actorOrg: providerOrgID,
			wantErr:  domain.ErrInvalidStatus,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockProposalRepo{
				getByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Proposal, error) {
					p := newStubProposal(clientID, providerID, tt.status)
					p.ClientID = clientID
					p.ProviderID = providerID
					return p, nil
				},
			}
			msgs := &mockMessageSender{}
			userRepo := &mockUserRepo{
				getByIDFn: func(_ context.Context, id uuid.UUID) (*user.User, error) {
					return orgLookup(id), nil
				},
			}
			svc := newTestService(repo, userRepo, msgs, nil)

			err := svc.RequestCompletion(context.Background(), RequestCompletionInput{
				ProposalID: uuid.New(),
				UserID:     tt.actorID,
				OrgID:      tt.actorOrg,
			})

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.Len(t, msgs.calls, 1)
			assert.Equal(t, "proposal_completion_requested", msgs.calls[0].Type)
		})
	}
}

// --- CompleteProposal table-driven tests ---

func TestCompleteProposal_TableDriven(t *testing.T) {
	clientID := uuid.New()
	providerID := uuid.New()
	clientOrgID := uuid.New()
	providerOrgID := uuid.New()

	orgLookup := func(id uuid.UUID) *user.User {
		o := providerOrgID
		if id == clientID {
			o = clientOrgID
		}
		return &user.User{ID: id, Role: user.RoleEnterprise, OrganizationID: &o}
	}

	tests := []struct {
		name     string
		status   domain.ProposalStatus
		actorID  uuid.UUID
		actorOrg uuid.UUID
		wantErr  error
	}{
		{
			name:     "client confirms completion",
			status:   domain.StatusCompletionRequested,
			actorID:  clientID,
			actorOrg: clientOrgID,
		},
		{
			name:     "provider cannot confirm completion",
			status:   domain.StatusCompletionRequested,
			actorID:  providerID,
			actorOrg: providerOrgID,
			wantErr:  domain.ErrNotClient,
		},
		{
			name:     "cannot complete from active status",
			status:   domain.StatusActive,
			actorID:  clientID,
			actorOrg: clientOrgID,
			wantErr:  domain.ErrInvalidStatus,
		},
		{
			name:     "cannot complete from pending status",
			status:   domain.StatusPending,
			actorID:  clientID,
			actorOrg: clientOrgID,
			wantErr:  domain.ErrInvalidStatus,
		},
		{
			name:     "cannot complete already completed",
			status:   domain.StatusCompleted,
			actorID:  clientID,
			actorOrg: clientOrgID,
			wantErr:  domain.ErrInvalidStatus,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockProposalRepo{
				getByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Proposal, error) {
					p := newStubProposal(providerID, clientID, tt.status)
					p.ClientID = clientID
					p.ProviderID = providerID
					return p, nil
				},
			}
			msgs := &mockMessageSender{}
			userRepo := &mockUserRepo{
				getByIDFn: func(_ context.Context, id uuid.UUID) (*user.User, error) {
					return orgLookup(id), nil
				},
			}
			svc := newTestService(repo, userRepo, msgs, nil)

			err := svc.CompleteProposal(context.Background(), CompleteProposalInput{
				ProposalID: uuid.New(),
				UserID:     tt.actorID,
				OrgID:      tt.actorOrg,
			})

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			// proposal_completed + a single shared evaluation_request
			// (since R18-fix). Both parties see the one evaluation_request
			// and the frontend picks the right review variant from the
			// organization ids enriched into metadata.
			require.Len(t, msgs.calls, 2)
			assert.Equal(t, "proposal_completed", msgs.calls[0].Type)
			assert.Equal(t, "evaluation_request", msgs.calls[1].Type)
		})
	}
}

func TestCompleteProposal_RepoUpdateError(t *testing.T) {
	clientID := uuid.New()
	providerID := uuid.New()
	clientOrgID := uuid.New()

	repo := &mockProposalRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Proposal, error) {
			p := newStubProposal(providerID, clientID, domain.StatusCompletionRequested)
			p.ClientID = clientID
			p.ProviderID = providerID
			return p, nil
		},
		updateFn: func(_ context.Context, _ *domain.Proposal) error {
			return errors.New("disk full")
		},
	}
	svc := newTestService(repo, orgAwareUserRepo(clientOrgID), nil, nil)

	err := svc.CompleteProposal(context.Background(), CompleteProposalInput{
		ProposalID: uuid.New(),
		UserID:     clientID,
		OrgID:      clientOrgID,
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "update proposal")
}

// --- RejectCompletion table-driven tests ---

func TestRejectCompletion_TableDriven(t *testing.T) {
	clientID := uuid.New()
	providerID := uuid.New()
	clientOrgID := uuid.New()
	providerOrgID := uuid.New()

	orgLookup := func(id uuid.UUID) *user.User {
		o := providerOrgID
		if id == clientID {
			o = clientOrgID
		}
		return &user.User{ID: id, Role: user.RoleEnterprise, OrganizationID: &o}
	}

	tests := []struct {
		name     string
		status   domain.ProposalStatus
		actorID  uuid.UUID
		actorOrg uuid.UUID
		wantErr  error
	}{
		{
			name:     "client rejects completion",
			status:   domain.StatusCompletionRequested,
			actorID:  clientID,
			actorOrg: clientOrgID,
		},
		{
			name:     "provider cannot reject completion",
			status:   domain.StatusCompletionRequested,
			actorID:  providerID,
			actorOrg: providerOrgID,
			wantErr:  domain.ErrNotClient,
		},
		{
			name:     "cannot reject from active status",
			status:   domain.StatusActive,
			actorID:  clientID,
			actorOrg: clientOrgID,
			wantErr:  domain.ErrInvalidStatus,
		},
		{
			name:     "cannot reject from pending status",
			status:   domain.StatusPending,
			actorID:  clientID,
			actorOrg: clientOrgID,
			wantErr:  domain.ErrInvalidStatus,
		},
		{
			name:     "cannot reject from completed status",
			status:   domain.StatusCompleted,
			actorID:  clientID,
			actorOrg: clientOrgID,
			wantErr:  domain.ErrInvalidStatus,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockProposalRepo{
				getByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Proposal, error) {
					p := newStubProposal(providerID, clientID, tt.status)
					p.ClientID = clientID
					p.ProviderID = providerID
					return p, nil
				},
			}
			msgs := &mockMessageSender{}
			userRepo := &mockUserRepo{
				getByIDFn: func(_ context.Context, id uuid.UUID) (*user.User, error) {
					return orgLookup(id), nil
				},
			}
			svc := newTestService(repo, userRepo, msgs, nil)

			err := svc.RejectCompletion(context.Background(), RejectCompletionInput{
				ProposalID: uuid.New(),
				UserID:     tt.actorID,
				OrgID:      tt.actorOrg,
			})

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.Len(t, msgs.calls, 1)
			assert.Equal(t, "proposal_completion_rejected", msgs.calls[0].Type)
		})
	}
}

func TestRejectCompletion_RepoGetError(t *testing.T) {
	repo := &mockProposalRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Proposal, error) {
			return nil, errors.New("not found")
		},
	}
	svc := newTestService(repo, nil, nil, nil)

	err := svc.RejectCompletion(context.Background(), RejectCompletionInput{
		ProposalID: uuid.New(),
		UserID:     uuid.New(),
		OrgID:      uuid.New(),
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "get proposal")
}

// --- helper ---

// --- KYC enforcement tests ---

func TestAcceptProposal_KYCBlocked(t *testing.T) {
	senderID := uuid.New()
	providerID := uuid.New()
	providerOrgID := uuid.New()
	past15 := time.Now().Add(-15 * 24 * time.Hour)

	p := newStubProposal(senderID, providerID, domain.StatusPending)

	proposalRepo := &mockProposalRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Proposal, error) { return p, nil },
	}
	userRepo := &mockUserRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*user.User, error) {
			u := makeUser(id, user.RoleProvider)
			o := providerOrgID
			u.OrganizationID = &o
			return u, nil
		},
	}
	orgRepo := &mockOrgRepo{
		findByUserIDFn: func(_ context.Context, _ uuid.UUID) (*organization.Organization, error) {
			return &organization.Organization{
				ID:                uuid.New(),
				Type:              organization.OrgTypeProviderPersonal,
				KYCFirstEarningAt: &past15,
			}, nil
		},
	}
	svc := newTestServiceWithCreditsAndOrgs(proposalRepo, userRepo, orgRepo, nil, nil, nil)

	err := svc.AcceptProposal(context.Background(), AcceptProposalInput{
		ProposalID: p.ID,
		UserID:     providerID,
		OrgID:      providerOrgID,
	})

	assert.ErrorIs(t, err, user.ErrKYCRestricted)
}

func TestAcceptProposal_KYCNotBlocked_Passes(t *testing.T) {
	senderID := uuid.New()
	providerID := uuid.New()
	providerOrgID := uuid.New()
	past5 := time.Now().Add(-5 * 24 * time.Hour)

	p := newStubProposal(senderID, providerID, domain.StatusPending)

	proposalRepo := &mockProposalRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Proposal, error) { return p, nil },
	}
	userRepo := &mockUserRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*user.User, error) {
			u := makeUser(id, user.RoleProvider)
			o := providerOrgID
			u.OrganizationID = &o
			return u, nil
		},
	}
	orgRepo := &mockOrgRepo{
		findByUserIDFn: func(_ context.Context, _ uuid.UUID) (*organization.Organization, error) {
			// 5 days elapsed — still within the 14-day deadline.
			return &organization.Organization{
				ID:                uuid.New(),
				Type:              organization.OrgTypeProviderPersonal,
				KYCFirstEarningAt: &past5,
			}, nil
		},
	}
	svc := newTestServiceWithCreditsAndOrgs(proposalRepo, userRepo, orgRepo, nil, nil, nil)

	err := svc.AcceptProposal(context.Background(), AcceptProposalInput{
		ProposalID: p.ID,
		UserID:     providerID,
		OrgID:      providerOrgID,
	})

	assert.NoError(t, err)
}

func newStubProposal(senderID, recipientID uuid.UUID, status domain.ProposalStatus) *domain.Proposal {
	return &domain.Proposal{
		ID:             uuid.New(),
		ConversationID: uuid.New(),
		SenderID:       senderID,
		RecipientID:    recipientID,
		ClientID:       senderID,
		ProviderID:     recipientID,
		Status:         status,
		Title:          "Stub proposal",
		Amount:         500000,
		Version:        1,
	}
}
