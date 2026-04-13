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
	return newTestServiceWithCredits(proposalRepo, userRepo, msgSender, storage, nil)
}

func newTestServiceWithCredits(
	proposalRepo *mockProposalRepo,
	userRepo *mockUserRepo,
	msgSender *mockMessageSender,
	storage *mockStorageService,
	credits *mockJobCreditRepo,
) *Service {
	return newTestServiceWithCreditsAndOrgs(proposalRepo, userRepo, nil, msgSender, storage, credits)
}

func newTestServiceWithCreditsAndOrgs(
	proposalRepo *mockProposalRepo,
	userRepo *mockUserRepo,
	orgRepo *mockOrgRepo,
	msgSender *mockMessageSender,
	storage *mockStorageService,
	credits *mockJobCreditRepo,
) *Service {
	if proposalRepo == nil {
		proposalRepo = &mockProposalRepo{}
	}
	if userRepo == nil {
		userRepo = &mockUserRepo{}
	}
	if orgRepo == nil {
		orgRepo = &mockOrgRepo{}
	}
	if msgSender == nil {
		msgSender = &mockMessageSender{}
	}
	if storage == nil {
		storage = &mockStorageService{}
	}
	deps := ServiceDeps{
		Proposals:     proposalRepo,
		Users:         userRepo,
		Organizations: orgRepo,
		Messages:      msgSender,
		Storage:       storage,
		Notifications: &mockNotificationSender{},
	}
	if credits != nil {
		deps.Credits = credits
	}
	return NewService(deps)
}

func makeUser(id uuid.UUID, role user.Role) *user.User {
	return &user.User{ID: id, Role: role, DisplayName: "Test " + string(role)}
}

// orgAwareUserRepo builds a mockUserRepo that returns, for every
// requested id, a user whose OrganizationID is set to the given
// orgID. Used by the authorization tests to simulate a proposal's
// side users (client/provider/recipient) all belonging to the same
// organization — the Stripe Dashboard shared-workspace shape.
func orgAwareUserRepo(orgID uuid.UUID) *mockUserRepo {
	return &mockUserRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*user.User, error) {
			o := orgID
			return &user.User{
				ID:             id,
				Role:           user.RoleEnterprise,
				DisplayName:    "Member of " + orgID.String()[:8],
				OrganizationID: &o,
			}, nil
		},
	}
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
		Amount:         5000,
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

func TestCreateProposal_WithDocuments(t *testing.T) {
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

	var capturedDocs []*domain.ProposalDocument
	proposalRepo := &mockProposalRepo{
		createWithDocsFn: func(_ context.Context, _ *domain.Proposal, docs []*domain.ProposalDocument) error {
			capturedDocs = docs
			return nil
		},
	}
	msgSender := &mockMessageSender{}

	svc := newTestService(proposalRepo, userRepo, msgSender, nil)

	p, err := svc.CreateProposal(context.Background(), CreateProposalInput{
		ConversationID: convID,
		SenderID:       enterpriseID,
		RecipientID:    providerID,
		Title:          "Build website",
		Description:    "Full redesign of the corporate website",
		Amount:         500000,
		Documents: []DocumentInput{
			{Filename: "spec.pdf", URL: "https://storage.example.com/spec.pdf", Size: 2048, MimeType: "application/pdf"},
			{Filename: "mockup.png", URL: "https://storage.example.com/mockup.png", Size: 4096, MimeType: "image/png"},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, "Build website", p.Title)
	require.Len(t, capturedDocs, 2)
	assert.Equal(t, "spec.pdf", capturedDocs[0].Filename)
	assert.Equal(t, "mockup.png", capturedDocs[1].Filename)
	assert.Equal(t, p.ID, capturedDocs[0].ProposalID)
	assert.Equal(t, p.ID, capturedDocs[1].ProposalID)
}

func TestCreateProposal_WithDeadline(t *testing.T) {
	enterpriseID := uuid.New()
	providerID := uuid.New()
	convID := uuid.New()
	deadline := time.Now().Add(30 * 24 * time.Hour)

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
		Description:    "Full redesign",
		Amount:         500000,
		Deadline:       &deadline,
	})

	require.NoError(t, err)
	require.NotNil(t, p.Deadline)
	assert.WithinDuration(t, deadline, *p.Deadline, time.Second)
}

func TestModifyProposal_VersionChain(t *testing.T) {
	senderID := uuid.New()
	recipientID := uuid.New()
	recipientOrgID := uuid.New()
	rootProposalID := uuid.New()
	// Simulate modifying a version 2 proposal that already has a parent
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
				Title:          "V2",
				Amount:         2000,
				Version:        2,
				ParentID:       &rootProposalID,
			}, nil
		},
	}
	msgSender := &mockMessageSender{}

	svc := newTestService(proposalRepo, orgAwareUserRepo(recipientOrgID), msgSender, nil)

	modified, err := svc.ModifyProposal(context.Background(), ModifyProposalInput{
		ProposalID:  uuid.New(),
		UserID:      recipientID,
		OrgID:       recipientOrgID,
		Title:       "V3 counter",
		Description: "Third version",
		Amount:      3000,
	})

	require.NoError(t, err)
	assert.Equal(t, 3, modified.Version)
	// ParentID should point to root, not to the immediate parent
	require.NotNil(t, modified.ParentID)
	assert.Equal(t, rootProposalID, *modified.ParentID)
}

// --- AcceptProposal tests ---

func TestAcceptProposal_ByRecipient(t *testing.T) {
	senderID := uuid.New()
	recipientID := uuid.New()
	orgID := uuid.New()
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

	svc := newTestService(proposalRepo, orgAwareUserRepo(orgID), msgSender, nil)

	err := svc.AcceptProposal(context.Background(), AcceptProposalInput{
		ProposalID: proposalID,
		UserID:     recipientID,
		OrgID:      orgID,
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
	senderOrgID := uuid.New()
	recipientOrgID := uuid.New()

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

	// Sender's org is NOT the recipient's org — the org-level directional
	// check must reject the accept attempt.
	userRepo := &mockUserRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*user.User, error) {
			o := recipientOrgID
			if id == senderID {
				o = senderOrgID
			}
			return &user.User{ID: id, Role: user.RoleEnterprise, OrganizationID: &o}, nil
		},
	}

	svc := newTestService(proposalRepo, userRepo, nil, nil)

	err := svc.AcceptProposal(context.Background(), AcceptProposalInput{
		ProposalID: uuid.New(),
		UserID:     senderID,
		OrgID:      senderOrgID, // caller is on sender side — wrong for accept
	})

	assert.ErrorIs(t, err, domain.ErrNotAuthorized)
}

func TestAcceptProposal_ProviderAccepts_SendsPaymentRequest(t *testing.T) {
	clientID := uuid.New()
	providerID := uuid.New()
	providerOrgID := uuid.New()

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

	svc := newTestService(proposalRepo, orgAwareUserRepo(providerOrgID), msgSender, nil)

	err := svc.AcceptProposal(context.Background(), AcceptProposalInput{
		ProposalID: uuid.New(),
		UserID:     providerID,
		OrgID:      providerOrgID,
	})

	require.NoError(t, err)
	assert.Len(t, msgSender.calls, 2)
	assert.Equal(t, "proposal_payment_requested", msgSender.calls[1].Type)
}

// --- DeclineProposal tests ---

func TestDeclineProposal_Success(t *testing.T) {
	senderID := uuid.New()
	recipientID := uuid.New()
	recipientOrgID := uuid.New()

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

	svc := newTestService(proposalRepo, orgAwareUserRepo(recipientOrgID), msgSender, nil)

	err := svc.DeclineProposal(context.Background(), DeclineProposalInput{
		ProposalID: uuid.New(),
		UserID:     recipientID,
		OrgID:      recipientOrgID,
	})

	require.NoError(t, err)
	assert.Len(t, msgSender.calls, 1)
	assert.Equal(t, "proposal_declined", msgSender.calls[0].Type)
}

// --- ModifyProposal tests ---

func TestModifyProposal_Success(t *testing.T) {
	senderID := uuid.New()
	recipientID := uuid.New()
	recipientOrgID := uuid.New()
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
				Amount:         5000,
				Version:        1,
			}, nil
		},
	}
	msgSender := &mockMessageSender{}

	svc := newTestService(proposalRepo, orgAwareUserRepo(recipientOrgID), msgSender, nil)

	modified, err := svc.ModifyProposal(context.Background(), ModifyProposalInput{
		ProposalID:  proposalID,
		UserID:      recipientID,
		OrgID:       recipientOrgID,
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
	senderOrgID := uuid.New()
	recipientOrgID := uuid.New()

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
	// Sender and recipient live in different orgs; caller presents the
	// sender's org, which is NOT the recipient side that ModifyProposal
	// requires.
	userRepo := &mockUserRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*user.User, error) {
			o := recipientOrgID
			if id == senderID {
				o = senderOrgID
			}
			return &user.User{ID: id, Role: user.RoleEnterprise, OrganizationID: &o}, nil
		},
	}

	svc := newTestService(proposalRepo, userRepo, nil, nil)

	_, err := svc.ModifyProposal(context.Background(), ModifyProposalInput{
		ProposalID:  uuid.New(),
		UserID:      senderID,
		OrgID:       senderOrgID,
		Title:       "Test",
		Description: "Test",
		Amount:      5000,
	})

	assert.ErrorIs(t, err, domain.ErrCannotModify)
}

func TestModifyProposal_NotPending_Fails(t *testing.T) {
	senderID := uuid.New()
	recipientID := uuid.New()
	recipientOrgID := uuid.New()

	proposalRepo := &mockProposalRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Proposal, error) {
			return &domain.Proposal{
				ID:          uuid.New(),
				SenderID:    senderID,
				RecipientID: recipientID,
				ClientID:    senderID,
				ProviderID:  recipientID,
				Status:      domain.StatusAccepted,
			}, nil
		},
	}

	// Correct org on recipient side, but status is already accepted
	// so the modify must still fail with ErrCannotModify.
	svc := newTestService(proposalRepo, orgAwareUserRepo(recipientOrgID), nil, nil)

	_, err := svc.ModifyProposal(context.Background(), ModifyProposalInput{
		ProposalID:  uuid.New(),
		UserID:      recipientID,
		OrgID:       recipientOrgID,
		Title:       "Test",
		Description: "Test",
		Amount:      5000,
	})

	assert.ErrorIs(t, err, domain.ErrCannotModify)
}

// --- SimulatePayment tests ---

func TestSimulatePayment_Success(t *testing.T) {
	clientID := uuid.New()
	providerID := uuid.New()
	clientOrgID := uuid.New()
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
				Amount:         5000,
			}, nil
		},
	}
	msgSender := &mockMessageSender{}

	svc := newTestService(proposalRepo, orgAwareUserRepo(clientOrgID), msgSender, nil)

	_, err := svc.InitiatePayment(context.Background(), PayProposalInput{
		ProposalID: uuid.New(),
		UserID:     clientID,
		OrgID:      clientOrgID,
	})

	require.NoError(t, err)
	assert.Len(t, msgSender.calls, 1)
	assert.Equal(t, "proposal_paid", msgSender.calls[0].Type)
}

func TestInitiatePayment_ByProvider_Fails(t *testing.T) {
	clientID := uuid.New()
	providerID := uuid.New()
	clientOrgID := uuid.New()
	providerOrgID := uuid.New()
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
	userRepo := &mockUserRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*user.User, error) {
			o := providerOrgID
			if id == clientID {
				o = clientOrgID
			}
			return &user.User{ID: id, Role: user.RoleEnterprise, OrganizationID: &o}, nil
		},
	}

	svc := newTestService(proposalRepo, userRepo, nil, nil)

	_, err := svc.InitiatePayment(context.Background(), PayProposalInput{
		ProposalID: uuid.New(),
		UserID:     providerID,
		OrgID:      providerOrgID, // caller is on provider side — wrong for pay
	})

	assert.ErrorIs(t, err, domain.ErrNotAuthorized)
}

func TestInitiatePayment_NotAccepted_Fails(t *testing.T) {
	clientID := uuid.New()
	providerID := uuid.New()
	clientOrgID := uuid.New()

	proposalRepo := &mockProposalRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Proposal, error) {
			return &domain.Proposal{
				ID:         uuid.New(),
				ClientID:   clientID,
				ProviderID: providerID,
				Status:     domain.StatusPending,
			}, nil
		},
	}

	// Caller IS on the client side — auth passes — but the status check
	// must still reject because the proposal isn't accepted yet.
	svc := newTestService(proposalRepo, orgAwareUserRepo(clientOrgID), nil, nil)

	_, err := svc.InitiatePayment(context.Background(), PayProposalInput{
		ProposalID: uuid.New(),
		UserID:     clientID,
		OrgID:      clientOrgID,
	})

	assert.ErrorIs(t, err, domain.ErrInvalidStatus)
}

// --- GetProposal tests ---

func TestGetProposal_Authorized(t *testing.T) {
	senderID := uuid.New()
	recipientID := uuid.New()
	orgID := uuid.New()
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
		isOrgAuthorizedFn: func(_ context.Context, _ uuid.UUID, got uuid.UUID) (bool, error) {
			return got == orgID, nil
		},
	}

	svc := newTestService(proposalRepo, nil, nil, nil)

	p, docs, err := svc.GetProposal(context.Background(), senderID, orgID, proposalID)

	require.NoError(t, err)
	assert.Equal(t, proposalID, p.ID)
	assert.NotNil(t, docs)
}

// TestGetProposal_OrgOperatorCanRead is the direct regression test for
// R14: an operator who is neither the original sender nor the original
// recipient, but whose organization DOES own one of the sides, must
// be able to fetch the proposal. Before R14 this returned 404 / not
// authorized; after R14 it returns the proposal.
func TestGetProposal_OrgOperatorCanRead(t *testing.T) {
	senderID := uuid.New()      // Alice, the original sender
	recipientID := uuid.New()   // Charlie, the original recipient
	bobUserID := uuid.New()     // Bob, an operator of Alice's org
	agencyOrgID := uuid.New()   // Alice's agency X — now also Bob's org
	proposalID := uuid.New()

	proposalRepo := &mockProposalRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Proposal, error) {
			return &domain.Proposal{
				ID:          proposalID,
				SenderID:    senderID,
				RecipientID: recipientID,
				ClientID:    senderID,
				ProviderID:  recipientID,
				Title:       "Active project",
				Amount:      500000,
			}, nil
		},
		getDocumentsFn: func(_ context.Context, _ uuid.UUID) ([]*domain.ProposalDocument, error) {
			return []*domain.ProposalDocument{}, nil
		},
		isOrgAuthorizedFn: func(_ context.Context, _ uuid.UUID, got uuid.UUID) (bool, error) {
			// Agency X is a party on the client side — anyone from
			// that org (including Bob, who joined after the proposal
			// was sent) can read the row.
			return got == agencyOrgID, nil
		},
	}

	svc := newTestService(proposalRepo, nil, nil, nil)

	p, docs, err := svc.GetProposal(context.Background(), bobUserID, agencyOrgID, proposalID)

	require.NoError(t, err)
	assert.Equal(t, proposalID, p.ID)
	assert.NotNil(t, docs)
}

// TestAcceptProposal_OrgOperatorOfRecipientOrgCanAccept verifies that
// Dave — a team member of Charlie's personal org who was NOT the
// original recipient — can still accept the proposal on behalf of
// Charlie's side. The org-level directional check must pass because
// Dave's org matches the recipient user's org.
func TestAcceptProposal_OrgOperatorOfRecipientOrgCanAccept(t *testing.T) {
	senderID := uuid.New()      // Alice (sender)
	recipientID := uuid.New()   // Charlie (original recipient)
	daveUserID := uuid.New()    // Dave, member of Charlie's org
	charlieOrgID := uuid.New()  // Shared org: Charlie + Dave
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
				Title:          "Shared proposal",
				Amount:         500000,
			}, nil
		},
	}
	userRepo := &mockUserRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*user.User, error) {
			// Every user we look up during this test resolves to
			// Charlie's org — sender side users are not looked up
			// on this code path so the lookup is keyed on recipient
			// id. The helper returns a matching org by default.
			_ = id
			o := charlieOrgID
			return &user.User{ID: id, Role: user.RoleProvider, OrganizationID: &o}, nil
		},
	}
	msgSender := &mockMessageSender{}

	svc := newTestService(proposalRepo, userRepo, msgSender, nil)

	err := svc.AcceptProposal(context.Background(), AcceptProposalInput{
		ProposalID: proposalID,
		UserID:     daveUserID, // Dave is not the original recipient
		OrgID:      charlieOrgID,
	})

	require.NoError(t, err)
	// Recipient is the provider, so 2 messages are sent.
	require.Len(t, msgSender.calls, 2)
	assert.Equal(t, "proposal_accepted", msgSender.calls[0].Type)
	assert.Equal(t, "proposal_payment_requested", msgSender.calls[1].Type)
}

// TestInitiatePayment_OrgOperatorOfProviderOrgCannotPay is the
// directional safety test: payment is a client-only action, so an
// operator on the PROVIDER side (even though they are a party to the
// proposal and can read it) must NOT be able to initiate the payment.
// Protects against a bug where the any-side org check replaces the
// directional check — that would let the provider's team pay on
// behalf of the client, which is catastrophic for escrow.
func TestInitiatePayment_OrgOperatorOfProviderOrgCannotPay(t *testing.T) {
	clientID := uuid.New()
	providerID := uuid.New()
	clientOrgID := uuid.New()
	providerOrgID := uuid.New()
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
	userRepo := &mockUserRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*user.User, error) {
			o := providerOrgID
			if id == clientID {
				o = clientOrgID
			}
			return &user.User{ID: id, Role: user.RoleEnterprise, OrganizationID: &o}, nil
		},
	}

	svc := newTestService(proposalRepo, userRepo, nil, nil)

	_, err := svc.InitiatePayment(context.Background(), PayProposalInput{
		ProposalID: uuid.New(),
		UserID:     uuid.New(),     // some operator of provider org
		OrgID:      providerOrgID,  // provider-side org — wrong for pay
	})

	assert.ErrorIs(t, err, domain.ErrNotAuthorized)
}

func TestGetProposal_NotAuthorized(t *testing.T) {
	senderID := uuid.New()
	recipientID := uuid.New()
	outsiderOrgID := uuid.New()

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
		// isOrgAuthorizedFn left nil → default denies.
	}

	svc := newTestService(proposalRepo, nil, nil, nil)

	_, _, err := svc.GetProposal(context.Background(), uuid.New(), outsiderOrgID, uuid.New())

	assert.ErrorIs(t, err, domain.ErrNotAuthorized)
}

// --- RequestCompletion tests ---

func TestRequestCompletion_Success(t *testing.T) {
	clientID := uuid.New()
	providerID := uuid.New()
	providerOrgID := uuid.New()

	proposalRepo := &mockProposalRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Proposal, error) {
			return &domain.Proposal{
				ID:             uuid.New(),
				ConversationID: uuid.New(),
				SenderID:       clientID,
				RecipientID:    providerID,
				ClientID:       clientID,
				ProviderID:     providerID,
				Status:         domain.StatusActive,
				Title:          "Active project",
				Amount:         500000,
			}, nil
		},
	}
	msgSender := &mockMessageSender{}

	svc := newTestService(proposalRepo, orgAwareUserRepo(providerOrgID), msgSender, nil)

	err := svc.RequestCompletion(context.Background(), RequestCompletionInput{
		ProposalID: uuid.New(),
		UserID:     providerID,
		OrgID:      providerOrgID,
	})

	require.NoError(t, err)
	assert.Len(t, msgSender.calls, 1)
	assert.Equal(t, "proposal_completion_requested", msgSender.calls[0].Type)
}

func TestRequestCompletion_NotProvider(t *testing.T) {
	clientID := uuid.New()
	providerID := uuid.New()
	clientOrgID := uuid.New()
	providerOrgID := uuid.New()

	proposalRepo := &mockProposalRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Proposal, error) {
			return &domain.Proposal{
				ID:         uuid.New(),
				ClientID:   clientID,
				ProviderID: providerID,
				Status:     domain.StatusActive,
			}, nil
		},
	}
	userRepo := &mockUserRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*user.User, error) {
			o := providerOrgID
			if id == clientID {
				o = clientOrgID
			}
			return &user.User{ID: id, Role: user.RoleEnterprise, OrganizationID: &o}, nil
		},
	}

	svc := newTestService(proposalRepo, userRepo, nil, nil)

	err := svc.RequestCompletion(context.Background(), RequestCompletionInput{
		ProposalID: uuid.New(),
		UserID:     clientID,
		OrgID:      clientOrgID, // client org cannot request completion
	})

	assert.ErrorIs(t, err, domain.ErrNotProvider)
}

func TestRequestCompletion_NotActive(t *testing.T) {
	providerID := uuid.New()
	providerOrgID := uuid.New()

	proposalRepo := &mockProposalRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Proposal, error) {
			return &domain.Proposal{
				ID:         uuid.New(),
				ClientID:   uuid.New(),
				ProviderID: providerID,
				Status:     domain.StatusPending,
			}, nil
		},
	}

	// Provider-side auth passes — status check must still reject.
	svc := newTestService(proposalRepo, orgAwareUserRepo(providerOrgID), nil, nil)

	err := svc.RequestCompletion(context.Background(), RequestCompletionInput{
		ProposalID: uuid.New(),
		UserID:     providerID,
		OrgID:      providerOrgID,
	})

	assert.ErrorIs(t, err, domain.ErrInvalidStatus)
}

// --- CompleteProposal tests ---

func TestCompleteProposal_Success(t *testing.T) {
	clientID := uuid.New()
	providerID := uuid.New()
	clientOrgID := uuid.New()

	proposalRepo := &mockProposalRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Proposal, error) {
			return &domain.Proposal{
				ID:             uuid.New(),
				ConversationID: uuid.New(),
				SenderID:       providerID,
				RecipientID:    clientID,
				ClientID:       clientID,
				ProviderID:     providerID,
				Status:         domain.StatusCompletionRequested,
				Title:          "Completion pending",
				Amount:         500000,
			}, nil
		},
	}
	msgSender := &mockMessageSender{}

	svc := newTestService(proposalRepo, orgAwareUserRepo(clientOrgID), msgSender, nil)

	err := svc.CompleteProposal(context.Background(), CompleteProposalInput{
		ProposalID: uuid.New(),
		UserID:     clientID,
		OrgID:      clientOrgID,
	})

	require.NoError(t, err)
	// proposal_completed + evaluation_request (client) +
	// evaluation_request (provider). Since R18 both sides get kicked
	// off so either party can leave a double-blind review.
	assert.Len(t, msgSender.calls, 3)
	assert.Equal(t, "proposal_completed", msgSender.calls[0].Type)
	assert.Equal(t, "evaluation_request", msgSender.calls[1].Type)
	assert.Equal(t, "evaluation_request", msgSender.calls[2].Type)
}

func TestCompleteProposal_NotClient(t *testing.T) {
	clientID := uuid.New()
	providerID := uuid.New()
	clientOrgID := uuid.New()
	providerOrgID := uuid.New()

	proposalRepo := &mockProposalRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Proposal, error) {
			return &domain.Proposal{
				ID:         uuid.New(),
				ClientID:   clientID,
				ProviderID: providerID,
				Status:     domain.StatusCompletionRequested,
			}, nil
		},
	}
	userRepo := &mockUserRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*user.User, error) {
			o := providerOrgID
			if id == clientID {
				o = clientOrgID
			}
			return &user.User{ID: id, Role: user.RoleEnterprise, OrganizationID: &o}, nil
		},
	}

	svc := newTestService(proposalRepo, userRepo, nil, nil)

	err := svc.CompleteProposal(context.Background(), CompleteProposalInput{
		ProposalID: uuid.New(),
		UserID:     providerID,
		OrgID:      providerOrgID, // provider org cannot confirm completion
	})

	assert.ErrorIs(t, err, domain.ErrNotClient)
}

// --- RejectCompletion tests ---

func TestRejectCompletion_Success(t *testing.T) {
	clientID := uuid.New()
	providerID := uuid.New()
	clientOrgID := uuid.New()

	proposalRepo := &mockProposalRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Proposal, error) {
			return &domain.Proposal{
				ID:             uuid.New(),
				ConversationID: uuid.New(),
				SenderID:       providerID,
				RecipientID:    clientID,
				ClientID:       clientID,
				ProviderID:     providerID,
				Status:         domain.StatusCompletionRequested,
				Title:          "Completion pending",
				Amount:         500000,
			}, nil
		},
	}
	msgSender := &mockMessageSender{}

	svc := newTestService(proposalRepo, orgAwareUserRepo(clientOrgID), msgSender, nil)

	err := svc.RejectCompletion(context.Background(), RejectCompletionInput{
		ProposalID: uuid.New(),
		UserID:     clientID,
		OrgID:      clientOrgID,
	})

	require.NoError(t, err)
	assert.Len(t, msgSender.calls, 1)
	assert.Equal(t, "proposal_completion_rejected", msgSender.calls[0].Type)
}
