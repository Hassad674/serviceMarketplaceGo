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
	"marketplace-backend/internal/system"
)

// Phase 4 (user decision F4) moved the bonus award out of
// ConfirmPaymentAndActivate — it now fires when the LAST milestone of a
// proposal is released (macro status → completed). The tests below
// invoke awardBonusWithFraudCheck directly with a hand-built proposal
// so they can assert the award mechanism in isolation, the same way
// TestFraudCheck_* exercises the fraud-evaluation path. Full-lifecycle
// coverage (milestones walking through fund → submit → release →
// bonus) lives in the macro-status suite.

// --- Direct awardBonusWithFraudCheck tests (fallback-direct path) ---

func TestAwardBonus_CleanProposal_AwardsBonusCredits(t *testing.T) {
	clientID := uuid.New()
	providerID := uuid.New()
	now := time.Now()

	p := &domain.Proposal{
		ID:         uuid.New(),
		ClientID:   clientID,
		ProviderID: providerID,
		Status:     domain.StatusCompleted,
		Title:      "Mission with bonus",
		Amount:     500000,
		CreatedAt:  now.Add(-10 * time.Minute),
	}

	credits := &mockJobCreditRepo{}
	// No bonusLog wired → fraud check falls through to awardBonusDirect.
	svc := newTestServiceWithCredits(nil, nil, nil, nil, credits)

	svc.awardBonusWithFraudCheck(context.Background(), p)

	require.Len(t, credits.addBonusCalls, 1)
	assert.Equal(t, providerID, credits.addBonusCalls[0].UserID)
	assert.Equal(t, jobdomain.BonusPerMission, credits.addBonusCalls[0].Amount)
	assert.Equal(t, jobdomain.MaxTokens, credits.addBonusCalls[0].MaxTokens)
}

// --- ConfirmPaymentAndActivate idempotency (orthogonal to bonus timing) ---

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

			// ConfirmPaymentAndActivate is invoked by the Stripe
			// webhook which runs as a system actor — mark the test
			// context the same way so the proposal service takes
			// the system-actor branch of loadProposalForActor.
			err := svc.ConfirmPaymentAndActivate(system.WithSystemActor(context.Background()), proposalID)

			require.NoError(t, err)

			// AddBonus must NOT be called on already-paid/active proposals:
			// activation short-circuits on status, and (post-F4) never
			// touches the bonus path regardless.
			assert.Empty(t, credits.addBonusCalls,
				"AddBonus should not be called for %s proposal", tt.status)
		})
	}
}

func TestAwardBonus_ExactFiveCredits(t *testing.T) {
	providerID := uuid.New()
	now := time.Now()

	p := &domain.Proposal{
		ID:         uuid.New(),
		ClientID:   uuid.New(),
		ProviderID: providerID,
		Status:     domain.StatusCompleted,
		Title:      "Verify exact amount",
		Amount:     300000,
		CreatedAt:  now.Add(-10 * time.Minute),
	}

	credits := &mockJobCreditRepo{}
	svc := newTestServiceWithCredits(nil, nil, nil, nil, credits)

	svc.awardBonusWithFraudCheck(context.Background(), p)

	require.Len(t, credits.addBonusCalls, 1)
	// Guard against accidental changes to the BonusPerMission constant —
	// the product requirement is exactly 5 credits per completed mission.
	assert.Equal(t, 5, credits.addBonusCalls[0].Amount,
		"provider must receive exactly 5 bonus credits per completed mission")
}

// --- ConfirmPaymentAndActivate nil-credits safety (orthogonal to bonus) ---

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

	// credits is nil — should not panic or error.
	svc := newTestServiceWithCredits(repo, nil, msgs, nil, nil)

	err := svc.ConfirmPaymentAndActivate(system.WithSystemActor(context.Background()), uuid.New())

	require.NoError(t, err)
	// Proposal should still transition to active and send messages.
	require.Len(t, msgs.calls, 1)
	assert.Equal(t, "proposal_paid", msgs.calls[0].Type)
}

func TestAwardBonus_CreditsErrorIsSwallowed(t *testing.T) {
	providerID := uuid.New()
	now := time.Now()

	p := &domain.Proposal{
		ID:         uuid.New(),
		ClientID:   uuid.New(),
		ProviderID: providerID,
		Status:     domain.StatusCompleted,
		Title:      "Bonus fails",
		Amount:     100000,
		CreatedAt:  now.Add(-10 * time.Minute),
	}

	credits := &mockJobCreditRepo{
		addBonusFn: func(_ context.Context, _ uuid.UUID, _ int, _ int) error {
			return errors.New("redis connection lost")
		},
	}
	svc := newTestServiceWithCredits(nil, nil, nil, nil, credits)

	// Errors from credits.AddBonus must be logged and swallowed so a
	// transient credits-store outage never blocks proposal completion.
	require.NotPanics(t, func() {
		svc.awardBonusWithFraudCheck(context.Background(), p)
	})

	require.Len(t, credits.addBonusCalls, 1, "AddBonus must still be attempted")
}
