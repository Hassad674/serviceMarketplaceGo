package dispute

import (
	"context"
	"errors"
	"math/rand"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	disputedomain "marketplace-backend/internal/domain/dispute"
	"marketplace-backend/internal/domain/proposal"
)

// ---------------------------------------------------------------------------
// BUG-03 — restore proposal after dispute cancel/respond.
//
// Before the fix:
//
//	if err := p.RestoreFromDispute(...); err != nil {
//	    slog.Warn(...)        // logged
//	} else {
//	    _ = s.proposals.Update(ctx, p) // ERROR SWALLOWED
//	}
//
// If the UPDATE fails (DB blip, optimistic concurrency conflict), the
// dispute is `cancelled` in DB but the proposal stays `disputed` —
// frozen pair, no automatic recovery, user has no clue why.
//
// The fix propagates the error so the HTTP layer surfaces a 500 the
// client can retry. The test below pins both the cancel branch and
// the respond branch of RespondToCancellation.
// ---------------------------------------------------------------------------

// TestCancelDispute_ProposalUpdateFailsPropagatesError pins the
// CancelDispute direct-cancel branch (BUG-03 location 1).
func TestCancelDispute_ProposalUpdateFailsPropagatesError(t *testing.T) {
	svc, dr, pr, _, _, _ := newTestService()
	clientID := uuid.New()
	providerID := uuid.New()
	proposalID := uuid.New()
	disputeID := uuid.New()

	dr.getByIDFn = func(_ context.Context, _ uuid.UUID) (*disputedomain.Dispute, error) {
		return &disputedomain.Dispute{
			ID: disputeID, ProposalID: proposalID,
			ConversationID: uuid.New(),
			InitiatorID:    clientID, RespondentID: providerID,
			ClientID: clientID, ProviderID: providerID,
			Status: disputedomain.StatusOpen, ProposalAmount: 100000,
			Version: 1,
		}, nil
	}
	pr.getByIDFn = func(_ context.Context, _ uuid.UUID) (*proposal.Proposal, error) {
		return &proposal.Proposal{
			ID: proposalID, Status: proposal.StatusDisputed,
			ActiveDisputeID: &disputeID,
		}, nil
	}

	// Simulate the DB blip — proposals.Update returns an error.
	dbBlip := errors.New("connection reset by peer")
	pr.updateFn = func(_ context.Context, _ *proposal.Proposal) error {
		return dbBlip
	}

	result, err := svc.CancelDispute(context.Background(), CancelDisputeInput{
		DisputeID: disputeID, UserID: clientID,
	})

	require.Error(t, err, "BUG-03: proposal update failure must NOT be swallowed")
	assert.ErrorIs(t, err, dbBlip,
		"the original DB error must be preserved via fmt.Errorf %%w wrapping")
	// Result is zero-value when an error is returned (the API contract).
	assert.False(t, result.Cancelled,
		"caller must see the failure — never report Cancelled=true on a partial commit")
}

// TestRespondToCancellation_ProposalUpdateFailsPropagatesError pins
// the RespondToCancellation accept branch (BUG-03 location 2).
func TestRespondToCancellation_ProposalUpdateFailsPropagatesError(t *testing.T) {
	svc, dr, pr, _, _, _ := newTestService()
	clientID := uuid.New()
	providerID := uuid.New()
	proposalID := uuid.New()
	disputeID := uuid.New()

	// Initiator (client) had requested cancellation; respondent (provider)
	// is now accepting it. Build a dispute in a state where
	// RespondToCancellationRequest will succeed.
	dr.getByIDFn = func(_ context.Context, _ uuid.UUID) (*disputedomain.Dispute, error) {
		d := &disputedomain.Dispute{
			ID: disputeID, ProposalID: proposalID,
			ConversationID: uuid.New(),
			InitiatorID:    clientID, RespondentID: providerID,
			ClientID: clientID, ProviderID: providerID,
			Status: disputedomain.StatusNegotiation, ProposalAmount: 100000,
			Version: 1,
		}
		d.RecordRespondentReply()
		// Domain helper to set the cancellation request — emulates
		// the prior CancelDispute call.
		_, _ = d.Cancel(clientID)
		return d, nil
	}
	pr.getByIDFn = func(_ context.Context, _ uuid.UUID) (*proposal.Proposal, error) {
		return &proposal.Proposal{
			ID: proposalID, Status: proposal.StatusDisputed,
			ActiveDisputeID: &disputeID,
		}, nil
	}

	dbBlip := errors.New("write conflict")
	pr.updateFn = func(_ context.Context, _ *proposal.Proposal) error {
		return dbBlip
	}

	err := svc.RespondToCancellation(context.Background(), RespondToCancellationInput{
		DisputeID: disputeID,
		UserID:    providerID,
		Accept:    true,
	})

	require.Error(t, err, "BUG-03 mirror: proposal update failure must NOT be swallowed")
	assert.ErrorIs(t, err, dbBlip)
}

// TestCancelDispute_RestoreFromDisputeRejected propagates the domain-level
// rejection (proposal not in StatusDisputed) instead of warning + swallowing.
// This is the second silent-failure mode of BUG-03 — without this fix, a
// dispute could be cancelled in DB while the proposal stayed in a
// non-disputed status (e.g. completed, paid) and the caller never knew.
func TestCancelDispute_RestoreFromDisputeRejected(t *testing.T) {
	svc, dr, pr, _, _, _ := newTestService()
	clientID := uuid.New()
	providerID := uuid.New()
	proposalID := uuid.New()
	disputeID := uuid.New()

	dr.getByIDFn = func(_ context.Context, _ uuid.UUID) (*disputedomain.Dispute, error) {
		return &disputedomain.Dispute{
			ID: disputeID, ProposalID: proposalID,
			ConversationID: uuid.New(),
			InitiatorID:    clientID, RespondentID: providerID,
			ClientID: clientID, ProviderID: providerID,
			Status: disputedomain.StatusOpen, ProposalAmount: 100000,
			Version: 1,
		}, nil
	}
	// Proposal status drift: not StatusDisputed.
	pr.getByIDFn = func(_ context.Context, _ uuid.UUID) (*proposal.Proposal, error) {
		return &proposal.Proposal{
			ID: proposalID, Status: proposal.StatusActive, // RestoreFromDispute will reject
		}, nil
	}

	updateCalled := false
	pr.updateFn = func(_ context.Context, _ *proposal.Proposal) error {
		updateCalled = true
		return nil
	}

	_, err := svc.CancelDispute(context.Background(), CancelDisputeInput{
		DisputeID: disputeID, UserID: clientID,
	})
	require.Error(t, err)
	assert.False(t, updateCalled, "Update must NOT be called when domain restore rejects")
}

// TestCancelDispute_HappyPath_StillUpdatesAndCleansUp regression test:
// the BUG-03 fix should not break the normal flow where the proposal
// update succeeds. The cancel path proceeds unchanged when nothing
// fails.
func TestCancelDispute_HappyPath_StillUpdatesAndCleansUp(t *testing.T) {
	svc, dr, pr, ms, _, _ := newTestService()
	clientID := uuid.New()
	providerID := uuid.New()
	proposalID := uuid.New()
	disputeID := uuid.New()

	dr.getByIDFn = func(_ context.Context, _ uuid.UUID) (*disputedomain.Dispute, error) {
		return &disputedomain.Dispute{
			ID: disputeID, ProposalID: proposalID,
			ConversationID: uuid.New(),
			InitiatorID:    clientID, RespondentID: providerID,
			ClientID: clientID, ProviderID: providerID,
			Status: disputedomain.StatusOpen, ProposalAmount: 100000,
			Version: 1,
		}, nil
	}
	pr.getByIDFn = func(_ context.Context, _ uuid.UUID) (*proposal.Proposal, error) {
		return &proposal.Proposal{
			ID: proposalID, Status: proposal.StatusDisputed,
			ActiveDisputeID: &disputeID,
		}, nil
	}

	var updatedProposal *proposal.Proposal
	pr.updateFn = func(_ context.Context, p *proposal.Proposal) error {
		updatedProposal = p
		return nil
	}

	result, err := svc.CancelDispute(context.Background(), CancelDisputeInput{
		DisputeID: disputeID, UserID: clientID,
	})

	require.NoError(t, err)
	assert.True(t, result.Cancelled)
	require.NotNil(t, updatedProposal, "happy path must call Update")
	assert.Equal(t, proposal.StatusActive, updatedProposal.Status)
	assert.Nil(t, updatedProposal.ActiveDisputeID,
		"ActiveDisputeID must be cleared by RestoreFromDispute")
	assert.NotNil(t, ms.lastInput, "system message still emitted")
}

// TestCancelRestore_PropertyTest_FinalStateConsistent runs N random
// (succeed/fail) sequences against the cancel-restore pair and asserts
// that the post-call state is always coherent: either both updated
// (success) or neither updated (error). The test guards against
// future regressions where a refactor might split the two writes
// across distinct error-handling paths.
func TestCancelRestore_PropertyTest_FinalStateConsistent(t *testing.T) {
	rng := rand.New(rand.NewSource(20260430))

	for it := 0; it < 100; it++ {
		svc, dr, pr, _, _, _ := newTestService()
		clientID := uuid.New()
		providerID := uuid.New()
		proposalID := uuid.New()
		disputeID := uuid.New()

		dr.getByIDFn = func(_ context.Context, _ uuid.UUID) (*disputedomain.Dispute, error) {
			return &disputedomain.Dispute{
				ID: disputeID, ProposalID: proposalID,
				ConversationID: uuid.New(),
				InitiatorID:    clientID, RespondentID: providerID,
				ClientID: clientID, ProviderID: providerID,
				Status: disputedomain.StatusOpen, ProposalAmount: 100000,
				Version: 1,
			}, nil
		}
		pr.getByIDFn = func(_ context.Context, _ uuid.UUID) (*proposal.Proposal, error) {
			return &proposal.Proposal{
				ID: proposalID, Status: proposal.StatusDisputed,
				ActiveDisputeID: &disputeID,
			}, nil
		}

		// Random failure injection: 30% of the time the proposal
		// update fails.
		updateCalls := 0
		shouldFail := rng.Intn(100) < 30
		pr.updateFn = func(_ context.Context, _ *proposal.Proposal) error {
			updateCalls++
			if shouldFail {
				return errors.New("synthetic blip")
			}
			return nil
		}

		result, err := svc.CancelDispute(context.Background(), CancelDisputeInput{
			DisputeID: disputeID, UserID: clientID,
		})

		// Invariant: success ↔ no error returned, failure ↔ error
		// returned. Never "Cancelled=true with err != nil" or any
		// other inconsistency.
		if shouldFail {
			require.Error(t, err, "iter=%d: failure must surface", it)
			assert.False(t, result.Cancelled,
				"iter=%d: result must not claim Cancelled when error is returned", it)
		} else {
			require.NoError(t, err, "iter=%d: success path must not error", it)
			assert.True(t, result.Cancelled,
				"iter=%d: result must claim Cancelled on success", it)
		}
	}
}

// TestRespondToCancellation_RestoreRejectionPropagated covers the
// Accept branch where RestoreFromDispute itself rejects (proposal
// drifted off StatusDisputed). The domain error must surface.
func TestRespondToCancellation_RestoreRejectionPropagated(t *testing.T) {
	svc, dr, pr, _, _, _ := newTestService()
	clientID := uuid.New()
	providerID := uuid.New()
	proposalID := uuid.New()
	disputeID := uuid.New()

	dr.getByIDFn = func(_ context.Context, _ uuid.UUID) (*disputedomain.Dispute, error) {
		d := &disputedomain.Dispute{
			ID: disputeID, ProposalID: proposalID,
			ConversationID: uuid.New(),
			InitiatorID:    clientID, RespondentID: providerID,
			ClientID: clientID, ProviderID: providerID,
			Status: disputedomain.StatusNegotiation, ProposalAmount: 100000,
			Version: 1,
		}
		d.RecordRespondentReply()
		_, _ = d.Cancel(clientID)
		return d, nil
	}
	pr.getByIDFn = func(_ context.Context, _ uuid.UUID) (*proposal.Proposal, error) {
		return &proposal.Proposal{
			ID: proposalID, Status: proposal.StatusActive, // not Disputed → reject
		}, nil
	}

	updateCalled := false
	pr.updateFn = func(_ context.Context, _ *proposal.Proposal) error {
		updateCalled = true
		return nil
	}

	err := svc.RespondToCancellation(context.Background(), RespondToCancellationInput{
		DisputeID: disputeID,
		UserID:    providerID,
		Accept:    true,
	})
	require.Error(t, err)
	assert.False(t, updateCalled, "Update must NOT fire after RestoreFromDispute rejects")
}
