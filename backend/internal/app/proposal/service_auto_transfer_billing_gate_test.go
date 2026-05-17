package proposal

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/domain/milestone"
	domain "marketplace-backend/internal/domain/proposal"
)

// ---------------------------------------------------------------------------
// Volet 3 — milestone-approval auto-transfer gate
// (fix/wallet-kyc-billing-regression)
//
// Asserts the proposal milestone-approval path consults the combined
// KYC+billing gate (ProviderReadyForAutoTransfer) and ONLY auto-
// transfers when it returns true. When the provider's billing profile
// is incomplete (gate → false) the milestone is approved/released but
// the funds are NOT transferred — they stay in the wallet's Available
// bucket (the payment record keeps Succeeded+TransferPending) until the
// provider completes their profile and drains them via "Retirer".
//
// The mid-project branch additionally requires HasAutoPayoutConsent;
// the test stamps it so the ONLY variable under test is the
// billing/KYC gate.
// ---------------------------------------------------------------------------

func TestCompleteProposal_MidProject_AutoTransferGate(t *testing.T) {
	tests := []struct {
		name                 string
		readyForAutoTransfer bool
		wantTransferCalls    int
		reason               string
	}{
		{
			name:                 "KYC ok + billing ok → auto-transfer fires (Transféré)",
			readyForAutoTransfer: true,
			wantTransferCalls:    1,
			reason:               "both gates green → money auto-transferred",
		},
		{
			name:                 "billing incomplete (gate false) → NO auto-transfer (funds stay Disponible)",
			readyForAutoTransfer: false,
			wantTransferCalls:    0,
			reason:               "gate false → record stays Succeeded+TransferPending → Available",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clientID := uuid.New()
			providerID := uuid.New()
			clientOrgID := uuid.New()
			providerOrgID := uuid.New()
			proposalID := uuid.New()

			repo := &mockProposalRepo{
				getByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Proposal, error) {
					return &domain.Proposal{
						ID:             proposalID,
						ConversationID: uuid.New(),
						ClientID:       clientID,
						ProviderID:     providerID,
						// Active (not the last milestone) → mid-project
						// branch, which requires consent + the gate.
						Status: domain.StatusActive,
						Title:  "Test",
						Amount: 500000,
					}, nil
				},
			}
			msgs := &mockMessageSender{}
			notifs := &mockNotificationSender{}
			payments := &mockPaymentProcessor{
				readyForAutoTransferFn: func(_ context.Context, _ uuid.UUID) (bool, error) {
					return tt.readyForAutoTransfer, nil
				},
				// Consent already stamped — isolates the gate as the
				// only variable affecting the mid-project decision.
				hasAutoPayoutConsentFn: func() (bool, error) { return true, nil },
			}

			svc := newTestServiceWithPayments(repo,
				orgAwareUserRepoSplit(clientID, providerID, clientOrgID, providerOrgID),
				msgs, notifs, payments)

			// Two milestones so the macro status stays Active after the
			// first approval → exercises the mid-project branch.
			mrepo := svc.milestones.(*mockMilestoneRepo)
			m := mrepo.seedMilestone(proposalID, milestone.StatusSubmitted, 250000)
			mrepo.seedMilestone(proposalID, milestone.StatusPendingFunding, 250000)

			err := svc.CompleteProposal(context.Background(), CompleteProposalInput{
				ProposalID: proposalID,
				UserID:     clientID,
				OrgID:      clientOrgID,
			})

			require.NoError(t, err)
			assert.Equal(t, milestone.StatusReleased, m.Status,
				"milestone must still be approved+released regardless of the gate")
			assert.Equal(t, tt.wantTransferCalls, payments.transferMilestoneCalls, tt.reason)
		})
	}
}

// TestCompleteProposal_GateError_NoAutoTransfer pins the conservative
// posture: when the combined gate returns an error the milestone is
// NOT auto-transferred (funds stay Available, never moved on partial
// information).
func TestCompleteProposal_GateError_NoAutoTransfer(t *testing.T) {
	clientID := uuid.New()
	providerID := uuid.New()
	clientOrgID := uuid.New()
	providerOrgID := uuid.New()
	proposalID := uuid.New()

	repo := &mockProposalRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Proposal, error) {
			return &domain.Proposal{
				ID:             proposalID,
				ConversationID: uuid.New(),
				ClientID:       clientID,
				ProviderID:     providerID,
				Status:         domain.StatusActive,
				Title:          "Test",
				Amount:         500000,
			}, nil
		},
	}
	payments := &mockPaymentProcessor{
		readyForAutoTransferFn: func(_ context.Context, _ uuid.UUID) (bool, error) {
			return false, assert.AnError
		},
		hasAutoPayoutConsentFn: func() (bool, error) { return true, nil },
	}

	svc := newTestServiceWithPayments(repo,
		orgAwareUserRepoSplit(clientID, providerID, clientOrgID, providerOrgID),
		&mockMessageSender{}, &mockNotificationSender{}, payments)

	mrepo := svc.milestones.(*mockMilestoneRepo)
	mrepo.seedMilestone(proposalID, milestone.StatusSubmitted, 250000)
	mrepo.seedMilestone(proposalID, milestone.StatusPendingFunding, 250000)

	err := svc.CompleteProposal(context.Background(), CompleteProposalInput{
		ProposalID: proposalID,
		UserID:     clientID,
		OrgID:      clientOrgID,
	})

	require.NoError(t, err, "approval must not fail just because the payout gate errored")
	assert.Equal(t, 0, payments.transferMilestoneCalls,
		"gate error → no auto-transfer (conservative: funds stay Available)")
}
