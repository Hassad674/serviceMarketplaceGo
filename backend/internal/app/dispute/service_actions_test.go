package dispute

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	disputedomain "marketplace-backend/internal/domain/dispute"
	"marketplace-backend/internal/domain/proposal"
)

// ---------------------------------------------------------------------------
// OpenDispute
// ---------------------------------------------------------------------------

func TestOpenDispute_Success(t *testing.T) {
	svc, dr, pr, ms, ns, _ := newTestService()
	clientID := uuid.New()
	providerID := uuid.New()
	p := makeActiveProposal(clientID, providerID)

	pr.getByIDFn = func(_ context.Context, _ uuid.UUID) (*proposal.Proposal, error) { return p, nil }
	dr.getByProposalIDFn = func(_ context.Context, _ uuid.UUID) (*disputedomain.Dispute, error) { return nil, nil }

	d, err := svc.OpenDispute(context.Background(), OpenDisputeInput{
		ProposalID:      p.ID,
		InitiatorID:     clientID,
		Reason:          "work_not_conforming",
		Description:     "The deliverable is incomplete",
		RequestedAmount: 80000,
	})

	require.NoError(t, err)
	assert.Equal(t, disputedomain.StatusOpen, d.Status)
	assert.Equal(t, clientID, d.InitiatorID)
	assert.Equal(t, providerID, d.RespondentID)
	assert.NotNil(t, ms.lastInput) // system message sent
	assert.Len(t, ns.sent, 1)     // notification sent to respondent
}

func TestOpenDispute_AlreadyDisputed(t *testing.T) {
	svc, dr, pr, _, _, _ := newTestService()
	clientID := uuid.New()
	providerID := uuid.New()
	p := makeActiveProposal(clientID, providerID)

	pr.getByIDFn = func(_ context.Context, _ uuid.UUID) (*proposal.Proposal, error) { return p, nil }
	dr.getByProposalIDFn = func(_ context.Context, _ uuid.UUID) (*disputedomain.Dispute, error) {
		return &disputedomain.Dispute{ID: uuid.New()}, nil // existing dispute
	}

	_, err := svc.OpenDispute(context.Background(), OpenDisputeInput{
		ProposalID:      p.ID,
		InitiatorID:     clientID,
		Reason:          "work_not_conforming",
		Description:     "Test",
		RequestedAmount: 50000,
	})

	assert.ErrorIs(t, err, disputedomain.ErrAlreadyDisputed)
}

func TestOpenDispute_ProposalNotActive(t *testing.T) {
	svc, _, pr, _, _, _ := newTestService()
	clientID := uuid.New()
	p := makeActiveProposal(clientID, uuid.New())
	p.Status = proposal.StatusCompleted

	pr.getByIDFn = func(_ context.Context, _ uuid.UUID) (*proposal.Proposal, error) { return p, nil }

	_, err := svc.OpenDispute(context.Background(), OpenDisputeInput{
		ProposalID:      p.ID,
		InitiatorID:     clientID,
		Reason:          "work_not_conforming",
		Description:     "Test",
		RequestedAmount: 50000,
	})

	assert.ErrorIs(t, err, disputedomain.ErrProposalNotDisputable)
}

func TestOpenDispute_InvalidReasonForRole(t *testing.T) {
	svc, dr, pr, _, _, _ := newTestService()
	clientID := uuid.New()
	p := makeActiveProposal(clientID, uuid.New())

	pr.getByIDFn = func(_ context.Context, _ uuid.UUID) (*proposal.Proposal, error) { return p, nil }
	dr.getByProposalIDFn = func(_ context.Context, _ uuid.UUID) (*disputedomain.Dispute, error) { return nil, nil }

	_, err := svc.OpenDispute(context.Background(), OpenDisputeInput{
		ProposalID:      p.ID,
		InitiatorID:     clientID,
		Reason:          "client_ghosting", // invalid for client
		Description:     "Test",
		RequestedAmount: 50000,
	})

	assert.ErrorIs(t, err, disputedomain.ErrInvalidReason)
}

// ---------------------------------------------------------------------------
// CounterPropose
// ---------------------------------------------------------------------------

func TestCounterPropose_Success(t *testing.T) {
	svc, dr, _, ms, ns, _ := newTestService()
	clientID := uuid.New()
	providerID := uuid.New()
	disputeID := uuid.New()

	dr.getByIDFn = func(_ context.Context, _ uuid.UUID) (*disputedomain.Dispute, error) {
		return &disputedomain.Dispute{
			ID: disputeID, InitiatorID: clientID, RespondentID: providerID,
			ClientID: clientID, ProviderID: providerID,
			Status: disputedomain.StatusOpen, ProposalAmount: 100000,
			Version: 1,
		}, nil
	}

	cp, err := svc.CounterPropose(context.Background(), CounterProposeInput{
		DisputeID:      disputeID,
		ProposerID:     providerID,
		AmountClient:   30000,
		AmountProvider: 70000,
		Message:        "I did most of the work",
	})

	require.NoError(t, err)
	assert.Equal(t, int64(30000), cp.AmountClient)
	assert.NotNil(t, ms.lastInput)
	assert.Len(t, ns.sent, 1)
}

func TestCounterPropose_AmountMismatch(t *testing.T) {
	svc, dr, _, _, _, _ := newTestService()
	disputeID := uuid.New()
	providerID := uuid.New()

	dr.getByIDFn = func(_ context.Context, _ uuid.UUID) (*disputedomain.Dispute, error) {
		return &disputedomain.Dispute{
			ID: disputeID, InitiatorID: uuid.New(), RespondentID: providerID,
			ClientID: uuid.New(), ProviderID: providerID,
			Status: disputedomain.StatusOpen, ProposalAmount: 100000,
			Version: 1,
		}, nil
	}

	_, err := svc.CounterPropose(context.Background(), CounterProposeInput{
		DisputeID:      disputeID,
		ProposerID:     providerID, // must be a participant
		AmountClient:   30000,
		AmountProvider: 30000, // sum != 100000
	})

	assert.ErrorIs(t, err, disputedomain.ErrAmountMismatch)
}

func TestCounterPropose_InvalidStatus(t *testing.T) {
	svc, dr, _, _, _, _ := newTestService()
	disputeID := uuid.New()

	dr.getByIDFn = func(_ context.Context, _ uuid.UUID) (*disputedomain.Dispute, error) {
		return &disputedomain.Dispute{
			ID: disputeID, Status: disputedomain.StatusResolved,
			InitiatorID: uuid.New(), RespondentID: uuid.New(),
			Version: 1,
		}, nil
	}

	_, err := svc.CounterPropose(context.Background(), CounterProposeInput{
		DisputeID:      disputeID,
		ProposerID:     uuid.New(),
		AmountClient:   50000,
		AmountProvider: 50000,
	})

	assert.ErrorIs(t, err, disputedomain.ErrInvalidStatus)
}

// ---------------------------------------------------------------------------
// CancelDispute
// ---------------------------------------------------------------------------

func TestCancelDispute_BeforeReply(t *testing.T) {
	svc, dr, pr, ms, _, _ := newTestService()
	clientID := uuid.New()
	providerID := uuid.New()
	proposalID := uuid.New()
	disputeID := uuid.New()

	dr.getByIDFn = func(_ context.Context, _ uuid.UUID) (*disputedomain.Dispute, error) {
		return &disputedomain.Dispute{
			ID: disputeID, ProposalID: proposalID,
			ConversationID: uuid.New(),
			InitiatorID: clientID, RespondentID: providerID,
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

	result, err := svc.CancelDispute(context.Background(), CancelDisputeInput{
		DisputeID: disputeID, UserID: clientID,
	})

	assert.NoError(t, err)
	assert.True(t, result.Cancelled)
	assert.False(t, result.Requested)
	assert.NotNil(t, ms.lastInput)
}

func TestCancelDispute_AfterReply_CreatesRequest(t *testing.T) {
	svc, dr, _, ms, _, _ := newTestService()
	clientID := uuid.New()
	providerID := uuid.New()
	disputeID := uuid.New()

	dr.getByIDFn = func(_ context.Context, _ uuid.UUID) (*disputedomain.Dispute, error) {
		d := &disputedomain.Dispute{
			ID: disputeID, InitiatorID: clientID, RespondentID: providerID,
			ClientID: clientID, ProviderID: providerID,
			Status: disputedomain.StatusNegotiation, ProposalAmount: 100000,
			Version: 1,
		}
		d.RecordRespondentReply() // respondent has replied
		return d, nil
	}

	supersedeCalled := false
	dr.supersedeAllFn = func(_ context.Context, id uuid.UUID) error {
		supersedeCalled = true
		assert.Equal(t, disputeID, id)
		return nil
	}

	result, err := svc.CancelDispute(context.Background(), CancelDisputeInput{
		DisputeID: disputeID, UserID: clientID,
	})

	assert.NoError(t, err)
	assert.False(t, result.Cancelled)
	assert.True(t, result.Requested)
	assert.True(t, supersedeCalled, "creating a cancellation request must supersede pending counter-proposals")
	// System message emitted should be dispute_cancellation_requested
	assert.NotNil(t, ms.lastInput)
}

// When the respondent (non-initiator) requests cancellation, the same
// supersede-pending-CPs rule must apply.
func TestCancelDispute_RespondentRequest_SupersedesCPs(t *testing.T) {
	svc, dr, _, _, _, _ := newTestService()
	clientID := uuid.New()
	providerID := uuid.New()
	disputeID := uuid.New()

	dr.getByIDFn = func(_ context.Context, _ uuid.UUID) (*disputedomain.Dispute, error) {
		// Initiator is the client, respondent is the provider — and the
		// respondent has NOT yet engaged. The respondent should still go
		// through the request flow (never direct cancel).
		return &disputedomain.Dispute{
			ID: disputeID, InitiatorID: clientID, RespondentID: providerID,
			ClientID: clientID, ProviderID: providerID,
			Status: disputedomain.StatusOpen, ProposalAmount: 100000,
			Version: 1,
		}, nil
	}

	supersedeCalled := false
	dr.supersedeAllFn = func(_ context.Context, _ uuid.UUID) error {
		supersedeCalled = true
		return nil
	}

	// Respondent (provider) asks to cancel.
	result, err := svc.CancelDispute(context.Background(), CancelDisputeInput{
		DisputeID: disputeID, UserID: providerID,
	})

	assert.NoError(t, err)
	assert.False(t, result.Cancelled, "respondent never directly cancels")
	assert.True(t, result.Requested)
	assert.True(t, supersedeCalled)
}

// ---------------------------------------------------------------------------
// AdminResolve
// ---------------------------------------------------------------------------

func TestAdminResolve_Success(t *testing.T) {
	svc, dr, pr, ms, ns, pp := newTestService()
	adminID := uuid.New()
	clientID := uuid.New()
	providerID := uuid.New()
	proposalID := uuid.New()
	disputeID := uuid.New()

	dr.getByIDFn = func(_ context.Context, _ uuid.UUID) (*disputedomain.Dispute, error) {
		return &disputedomain.Dispute{
			ID: disputeID, ProposalID: proposalID,
			ConversationID: uuid.New(),
			InitiatorID: clientID, RespondentID: providerID,
			ClientID: clientID, ProviderID: providerID,
			Status: disputedomain.StatusEscalated, ProposalAmount: 100000,
			Version: 1,
		}, nil
	}
	pr.getByIDFn = func(_ context.Context, _ uuid.UUID) (*proposal.Proposal, error) {
		return &proposal.Proposal{
			ID: proposalID, Status: proposal.StatusDisputed,
			ActiveDisputeID: &disputeID,
		}, nil
	}

	err := svc.AdminResolve(context.Background(), AdminResolveInput{
		DisputeID:      disputeID,
		AdminID:        adminID,
		AmountClient:   40000,
		AmountProvider: 60000,
		Note:           "Partial refund — provider did most of the work.",
	})

	assert.NoError(t, err)
	assert.NotNil(t, ms.lastInput)
	assert.Len(t, ns.sent, 2) // both parties notified
	assert.True(t, pp.transferCalled)
}

func TestAdminResolve_NotEscalated(t *testing.T) {
	svc, dr, _, _, _, _ := newTestService()

	dr.getByIDFn = func(_ context.Context, _ uuid.UUID) (*disputedomain.Dispute, error) {
		return &disputedomain.Dispute{
			ID: uuid.New(), Status: disputedomain.StatusOpen,
			InitiatorID: uuid.New(), RespondentID: uuid.New(),
			Version: 1,
		}, nil
	}

	err := svc.AdminResolve(context.Background(), AdminResolveInput{
		DisputeID:      uuid.New(),
		AdminID:        uuid.New(),
		AmountClient:   50000,
		AmountProvider: 50000,
	})

	assert.ErrorIs(t, err, disputedomain.ErrInvalidStatus)
}

func TestAdminResolve_AmountMismatch(t *testing.T) {
	svc, dr, _, _, _, _ := newTestService()

	dr.getByIDFn = func(_ context.Context, _ uuid.UUID) (*disputedomain.Dispute, error) {
		return &disputedomain.Dispute{
			ID: uuid.New(), Status: disputedomain.StatusEscalated,
			InitiatorID: uuid.New(), RespondentID: uuid.New(),
			ProposalAmount: 100000, Version: 1,
		}, nil
	}

	err := svc.AdminResolve(context.Background(), AdminResolveInput{
		DisputeID:      uuid.New(),
		AdminID:        uuid.New(),
		AmountClient:   50000,
		AmountProvider: 40000, // sum != 100000
	})

	assert.ErrorIs(t, err, disputedomain.ErrAmountMismatch)
}

// ---------------------------------------------------------------------------
// GetDispute
// ---------------------------------------------------------------------------

func TestGetDispute_Participant(t *testing.T) {
	svc, dr, _, _, _, _ := newTestService()
	clientID := uuid.New()
	disputeID := uuid.New()

	dr.getByIDFn = func(_ context.Context, _ uuid.UUID) (*disputedomain.Dispute, error) {
		return &disputedomain.Dispute{
			ID: disputeID, InitiatorID: clientID, RespondentID: uuid.New(),
			Version: 1,
		}, nil
	}

	detail, err := svc.GetDispute(context.Background(), clientID, disputeID)
	assert.NoError(t, err)
	assert.NotNil(t, detail)
}

func TestGetDispute_NotParticipant(t *testing.T) {
	svc, dr, _, _, _, _ := newTestService()
	disputeID := uuid.New()

	dr.getByIDFn = func(_ context.Context, _ uuid.UUID) (*disputedomain.Dispute, error) {
		return &disputedomain.Dispute{
			ID: disputeID, InitiatorID: uuid.New(), RespondentID: uuid.New(),
			Version: 1,
		}, nil
	}

	_, err := svc.GetDispute(context.Background(), uuid.New(), disputeID) // random user
	assert.ErrorIs(t, err, disputedomain.ErrNotParticipant)
}
