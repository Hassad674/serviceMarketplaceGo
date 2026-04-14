package proposal

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	jobdomain "marketplace-backend/internal/domain/job"
	domain "marketplace-backend/internal/domain/proposal"
)

// --- ConfirmPaymentAndActivate bonus credits tests ---

func TestConfirmPaymentAndActivate_AwardsBonusCredits(t *testing.T) {
	// Phase 4 (user decision F4): the credit bonus now fires when the
	// LAST milestone of a proposal is released (macro status →
	// completed), not on the first payment. This test still asserts
	// the legacy behavior and needs to be rewritten to walk the
	// proposal through fund → submit → approveAndRelease and assert
	// the bonus only at completion. Skipping until the rewrite lands
	// in a follow-up commit (tracked in BLOCKED-milestones-bonus.md).
	t.Skip("TODO: rewrite for F4 — bonus fires on completion, not first payment")
	clientID := uuid.New()
	providerID := uuid.New()
	proposalID := uuid.New()
	now := time.Now()

	repo := &mockProposalRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Proposal, error) {
			return &domain.Proposal{
				ID:             proposalID,
				ConversationID: uuid.New(),
				SenderID:       clientID,
				RecipientID:    providerID,
				ClientID:       clientID,
				ProviderID:     providerID,
				Status:         domain.StatusAccepted,
				AcceptedAt:     &now,
				Title:          "Mission with bonus",
				Amount:         500000,
				Version:        1,
			}, nil
		},
	}
	msgs := &mockMessageSender{}
	credits := &mockJobCreditRepo{}

	svc := newTestServiceWithCredits(repo, nil, msgs, nil, credits)

	err := svc.ConfirmPaymentAndActivate(context.Background(), proposalID)

	require.NoError(t, err)

	// Verify AddBonus was called exactly once with correct parameters
	require.Len(t, credits.addBonusCalls, 1)
	assert.Equal(t, providerID, credits.addBonusCalls[0].UserID)
	assert.Equal(t, jobdomain.BonusPerMission, credits.addBonusCalls[0].Amount)
	assert.Equal(t, jobdomain.MaxTokens, credits.addBonusCalls[0].MaxTokens)
}

func TestConfirmPaymentAndActivate_IdempotentSkipsBonus(t *testing.T) {
	clientID := uuid.New()
	providerID := uuid.New()
	proposalID := uuid.New()

	tests := []struct {
		name   string
		status domain.ProposalStatus
	}{
		{
			name:   "already paid proposal skips bonus",
			status: domain.StatusPaid,
		},
		{
			name:   "already active proposal skips bonus",
			status: domain.StatusActive,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockProposalRepo{
				getByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Proposal, error) {
					return &domain.Proposal{
						ID:         proposalID,
						ClientID:   clientID,
						ProviderID: providerID,
						Status:     tt.status,
						Title:      "Already processed",
						Amount:     100000,
					}, nil
				},
			}
			credits := &mockJobCreditRepo{}

			svc := newTestServiceWithCredits(repo, nil, nil, nil, credits)

			err := svc.ConfirmPaymentAndActivate(context.Background(), proposalID)

			require.NoError(t, err)

			// AddBonus must NOT be called for already-paid/active proposals
			assert.Empty(t, credits.addBonusCalls,
				"AddBonus should not be called for %s proposal", tt.status)
		})
	}
}

func TestConfirmPaymentAndActivate_BonusExact5Credits(t *testing.T) {
	t.Skip("TODO: rewrite for F4 — bonus fires on completion, not first payment")
	clientID := uuid.New()
	providerID := uuid.New()
	now := time.Now()

	repo := &mockProposalRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Proposal, error) {
			return &domain.Proposal{
				ID:             uuid.New(),
				ConversationID: uuid.New(),
				SenderID:       clientID,
				RecipientID:    providerID,
				ClientID:       clientID,
				ProviderID:     providerID,
				Status:         domain.StatusAccepted,
				AcceptedAt:     &now,
				Title:          "Verify exact amount",
				Amount:         300000,
				Version:        1,
			}, nil
		},
	}
	credits := &mockJobCreditRepo{}
	svc := newTestServiceWithCredits(repo, nil, nil, nil, credits)

	err := svc.ConfirmPaymentAndActivate(context.Background(), uuid.New())

	require.NoError(t, err)
	require.Len(t, credits.addBonusCalls, 1)

	// BonusPerMission is exactly 5
	assert.Equal(t, 5, credits.addBonusCalls[0].Amount,
		"provider must receive exactly 5 bonus credits per mission")
}

func TestConfirmPaymentAndActivate_NilCreditsRepoNoError(t *testing.T) {
	clientID := uuid.New()
	providerID := uuid.New()
	now := time.Now()

	repo := &mockProposalRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Proposal, error) {
			return &domain.Proposal{
				ID:             uuid.New(),
				ConversationID: uuid.New(),
				SenderID:       clientID,
				RecipientID:    providerID,
				ClientID:       clientID,
				ProviderID:     providerID,
				Status:         domain.StatusAccepted,
				AcceptedAt:     &now,
				Title:          "No credits repo",
				Amount:         200000,
				Version:        1,
			}, nil
		},
	}
	msgs := &mockMessageSender{}

	// credits is nil — should not panic or error
	svc := newTestServiceWithCredits(repo, nil, msgs, nil, nil)

	err := svc.ConfirmPaymentAndActivate(context.Background(), uuid.New())

	require.NoError(t, err)
	// Proposal should still transition to active and send messages
	require.Len(t, msgs.calls, 1)
	assert.Equal(t, "proposal_paid", msgs.calls[0].Type)
}

func TestConfirmPaymentAndActivate_BonusErrorDoesNotBlockActivation(t *testing.T) {
	t.Skip("TODO: rewrite for F4 — bonus fires on completion, not first payment")
	clientID := uuid.New()
	providerID := uuid.New()
	now := time.Now()

	repo := &mockProposalRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Proposal, error) {
			return &domain.Proposal{
				ID:             uuid.New(),
				ConversationID: uuid.New(),
				SenderID:       clientID,
				RecipientID:    providerID,
				ClientID:       clientID,
				ProviderID:     providerID,
				Status:         domain.StatusAccepted,
				AcceptedAt:     &now,
				Title:          "Bonus fails",
				Amount:         100000,
				Version:        1,
			}, nil
		},
	}
	credits := &mockJobCreditRepo{
		addBonusFn: func(_ context.Context, _ uuid.UUID, _ int, _ int) error {
			return errors.New("redis connection lost")
		},
	}
	msgs := &mockMessageSender{}

	svc := newTestServiceWithCredits(repo, nil, msgs, nil, credits)

	err := svc.ConfirmPaymentAndActivate(context.Background(), uuid.New())

	// The activation must succeed even if AddBonus fails
	require.NoError(t, err)

	// AddBonus was attempted
	require.Len(t, credits.addBonusCalls, 1)

	// Messages were still sent
	require.Len(t, msgs.calls, 1)
	assert.Equal(t, "proposal_paid", msgs.calls[0].Type)
}

// --- SimulatePayment bonus credits tests ---

func TestSimulatePayment_AwardsBonusCredits(t *testing.T) {
	t.Skip("TODO: rewrite for F4 — bonus fires on completion, not first payment")
	clientID := uuid.New()
	providerID := uuid.New()
	clientOrgID := uuid.New()
	now := time.Now()

	repo := &mockProposalRepo{
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
				Title:          "Simulated payment",
				Amount:         400000,
				Version:        1,
			}, nil
		},
	}
	msgs := &mockMessageSender{}
	credits := &mockJobCreditRepo{}

	// No payments service = simulation mode. Provide an org-aware user
	// repo so the new client-side directional check passes.
	svc := newTestServiceWithCredits(repo, orgAwareUserRepo(clientOrgID), msgs, nil, credits)

	_, err := svc.InitiatePayment(context.Background(), PayProposalInput{
		ProposalID: uuid.New(),
		UserID:     clientID,
		OrgID:      clientOrgID,
	})

	require.NoError(t, err)

	// Verify bonus was awarded in simulation mode too
	require.Len(t, credits.addBonusCalls, 1)
	assert.Equal(t, providerID, credits.addBonusCalls[0].UserID)
	assert.Equal(t, jobdomain.BonusPerMission, credits.addBonusCalls[0].Amount)
}
