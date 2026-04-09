package dispute

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// NewDispute validation
// ---------------------------------------------------------------------------

func TestNewDispute_ClientReasons(t *testing.T) {
	base := validInput()
	base.InitiatorID = base.ClientID // initiator is client

	tests := []struct {
		name    string
		reason  Reason
		wantErr error
	}{
		{"work not conforming", ReasonWorkNotConforming, nil},
		{"non delivery", ReasonNonDelivery, nil},
		{"insufficient quality", ReasonInsufficientQuality, nil},
		{"other", ReasonOther, nil},
		{"client ghosting — invalid for client", ReasonClientGhosting, ErrInvalidReason},
		{"scope creep — invalid for client", ReasonScopeCreep, ErrInvalidReason},
		{"harassment — invalid for client", ReasonHarassment, ErrInvalidReason},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := base
			in.Reason = tt.reason
			d, err := NewDispute(in)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, d)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, d)
			}
		})
	}
}

func TestNewDispute_ProviderReasons(t *testing.T) {
	base := validInput()
	base.InitiatorID = base.ProviderID // initiator is provider

	tests := []struct {
		name    string
		reason  Reason
		wantErr error
	}{
		{"client ghosting", ReasonClientGhosting, nil},
		{"scope creep", ReasonScopeCreep, nil},
		{"refusal to validate", ReasonRefusalToValidate, nil},
		{"harassment", ReasonHarassment, nil},
		{"other", ReasonOther, nil},
		{"work not conforming — invalid for provider", ReasonWorkNotConforming, ErrInvalidReason},
		{"non delivery — invalid for provider", ReasonNonDelivery, ErrInvalidReason},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := base
			in.Reason = tt.reason
			d, err := NewDispute(in)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, d)
			}
		})
	}
}

func TestNewDispute_Validation(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*NewDisputeInput)
		wantErr error
	}{
		{"description too long", func(in *NewDisputeInput) {
			in.Description = string(make([]byte, 5001))
		}, ErrDescriptionTooLong},
		{"zero amount", func(in *NewDisputeInput) { in.RequestedAmount = 0 }, ErrInvalidAmount},
		{"negative amount", func(in *NewDisputeInput) { in.RequestedAmount = -100 }, ErrInvalidAmount},
		{"amount exceeds proposal", func(in *NewDisputeInput) { in.RequestedAmount = 100001 }, ErrInvalidAmount},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := validInput()
			tt.mutate(&in)
			d, err := NewDispute(in)
			assert.ErrorIs(t, err, tt.wantErr)
			assert.Nil(t, d)
		})
	}
}

func TestNewDispute_Success(t *testing.T) {
	in := validInput()
	d, err := NewDispute(in)
	require.NoError(t, err)
	assert.Equal(t, StatusOpen, d.Status)
	assert.Equal(t, 1, d.Version)
	assert.Nil(t, d.RespondentFirstReplyAt)
}

// ---------------------------------------------------------------------------
// State machine
// ---------------------------------------------------------------------------

func TestDispute_MarkNegotiation(t *testing.T) {
	d := makeDispute(StatusOpen)
	assert.NoError(t, d.MarkNegotiation())
	assert.Equal(t, StatusNegotiation, d.Status)

	// Cannot transition from negotiation again
	assert.ErrorIs(t, d.MarkNegotiation(), ErrInvalidStatus)
}

func TestDispute_Escalate(t *testing.T) {
	tests := []struct {
		name   string
		status Status
		ok     bool
	}{
		{"from open", StatusOpen, true},
		{"from negotiation", StatusNegotiation, true},
		{"from escalated", StatusEscalated, false},
		{"from resolved", StatusResolved, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := makeDispute(tt.status)
			err := d.Escalate()
			if tt.ok {
				assert.NoError(t, err)
				assert.Equal(t, StatusEscalated, d.Status)
				assert.NotNil(t, d.EscalatedAt)
			} else {
				assert.ErrorIs(t, err, ErrInvalidStatus)
			}
		})
	}
}

func TestDispute_Resolve(t *testing.T) {
	d := makeDispute(StatusEscalated)
	adminID := uuid.New()

	err := d.Resolve(ResolveInput{
		ResolvedBy:     adminID,
		AmountClient:   30000,
		AmountProvider: 70000,
		Note:           "Partial refund",
	})
	assert.NoError(t, err)
	assert.Equal(t, StatusResolved, d.Status)
	assert.Equal(t, int64(30000), *d.ResolutionAmountClient)
	assert.Equal(t, int64(70000), *d.ResolutionAmountProvider)
	assert.NotNil(t, d.ResolvedAt)
}

func TestDispute_Resolve_AmountMismatch(t *testing.T) {
	d := makeDispute(StatusEscalated)
	err := d.Resolve(ResolveInput{
		AmountClient:   30000,
		AmountProvider: 30000, // sum != 100000
	})
	assert.ErrorIs(t, err, ErrAmountMismatch)
}

func TestDispute_AutoResolveForInitiator_Client(t *testing.T) {
	d := makeDispute(StatusOpen)
	d.InitiatorID = d.ClientID
	d.RequestedAmount = 80000

	err := d.AutoResolveForInitiator()
	assert.NoError(t, err)
	assert.Equal(t, StatusResolved, d.Status)
	assert.Equal(t, int64(80000), *d.ResolutionAmountClient)
	assert.Equal(t, int64(20000), *d.ResolutionAmountProvider)
}

func TestDispute_AutoResolveForInitiator_Provider(t *testing.T) {
	d := makeDispute(StatusOpen)
	d.InitiatorID = d.ProviderID
	d.RequestedAmount = 100000

	err := d.AutoResolveForInitiator()
	assert.NoError(t, err)
	assert.Equal(t, int64(0), *d.ResolutionAmountClient)
	assert.Equal(t, int64(100000), *d.ResolutionAmountProvider)
}

// ---------------------------------------------------------------------------
// Cancel
// ---------------------------------------------------------------------------

func TestDispute_Cancel_BeforeReply(t *testing.T) {
	d := makeDispute(StatusOpen)
	cancelled, err := d.Cancel(d.InitiatorID)
	assert.NoError(t, err)
	assert.True(t, cancelled)
	assert.Equal(t, StatusCancelled, d.Status)
}

func TestDispute_Cancel_AfterReply_CreatesRequest(t *testing.T) {
	d := makeDispute(StatusOpen)
	d.RecordRespondentReply()
	cancelled, err := d.Cancel(d.InitiatorID)
	assert.NoError(t, err)
	assert.False(t, cancelled)
	assert.NotNil(t, d.CancellationRequestedBy)
	assert.Equal(t, d.InitiatorID, *d.CancellationRequestedBy)
	assert.NotNil(t, d.CancellationRequestedAt)
	// The dispute is still open — only terminal after the respondent accepts.
	assert.Equal(t, StatusOpen, d.Status)
}

func TestDispute_Cancel_AfterReply_AlreadyRequested(t *testing.T) {
	d := makeDispute(StatusOpen)
	d.RecordRespondentReply()
	_, _ = d.Cancel(d.InitiatorID)
	_, err := d.Cancel(d.InitiatorID)
	assert.ErrorIs(t, err, ErrCancellationAlreadyRequested)
}

func TestDispute_Cancel_RespondentAlwaysGoesThroughRequest(t *testing.T) {
	// Even before the respondent has "replied" via a counter-proposal,
	// the respondent themselves can never directly cancel — they only
	// have the right to ASK the initiator for cancellation.
	d := makeDispute(StatusOpen)
	cancelled, err := d.Cancel(d.RespondentID)
	assert.NoError(t, err)
	assert.False(t, cancelled, "respondent must never directly cancel a dispute")
	assert.NotNil(t, d.CancellationRequestedBy)
	assert.Equal(t, d.RespondentID, *d.CancellationRequestedBy)
	assert.Equal(t, StatusOpen, d.Status, "dispute stays open until the initiator consents")
}

func TestDispute_Cancel_NonParticipant(t *testing.T) {
	d := makeDispute(StatusOpen)
	stranger := uuid.New()
	_, err := d.Cancel(stranger)
	assert.ErrorIs(t, err, ErrNotParticipant)
}

func TestDispute_Cancel_TerminalStatus(t *testing.T) {
	d := makeDispute(StatusResolved)
	_, err := d.Cancel(d.InitiatorID)
	assert.ErrorIs(t, err, ErrInvalidStatus)
}

func TestDispute_RespondToCancellationRequest_Accept(t *testing.T) {
	d := makeDispute(StatusOpen)
	d.RecordRespondentReply()
	_, _ = d.Cancel(d.InitiatorID)

	err := d.RespondToCancellationRequest(d.RespondentID, true)
	assert.NoError(t, err)
	assert.Equal(t, StatusCancelled, d.Status)
	assert.NotNil(t, d.CancelledAt)
}

func TestDispute_RespondToCancellationRequest_Refuse(t *testing.T) {
	d := makeDispute(StatusOpen)
	d.RecordRespondentReply()
	_, _ = d.Cancel(d.InitiatorID)

	err := d.RespondToCancellationRequest(d.RespondentID, false)
	assert.NoError(t, err)
	assert.Equal(t, StatusOpen, d.Status)
	assert.Nil(t, d.CancellationRequestedBy)
	assert.Nil(t, d.CancellationRequestedAt)
}

func TestDispute_RespondToCancellationRequest_InitiatorCannotRespond(t *testing.T) {
	d := makeDispute(StatusOpen)
	d.RecordRespondentReply()
	_, _ = d.Cancel(d.InitiatorID)

	err := d.RespondToCancellationRequest(d.InitiatorID, true)
	assert.ErrorIs(t, err, ErrNotAuthorized)
}

func TestDispute_RespondToCancellationRequest_NoPending(t *testing.T) {
	d := makeDispute(StatusOpen)
	err := d.RespondToCancellationRequest(d.RespondentID, true)
	assert.ErrorIs(t, err, ErrNoCancellationPending)
}

// Symmetric flow: when the respondent initiates the cancellation request,
// only the initiator can accept or refuse it.
func TestDispute_RespondToCancellationRequest_RespondentInitiated_InitiatorAccepts(t *testing.T) {
	d := makeDispute(StatusOpen)
	// Respondent asks to cancel — always goes through the request flow.
	cancelled, err := d.Cancel(d.RespondentID)
	assert.NoError(t, err)
	assert.False(t, cancelled)

	// The respondent (the requester) cannot self-respond.
	err = d.RespondToCancellationRequest(d.RespondentID, true)
	assert.ErrorIs(t, err, ErrNotAuthorized)

	// The initiator accepts → dispute cancelled.
	err = d.RespondToCancellationRequest(d.InitiatorID, true)
	assert.NoError(t, err)
	assert.Equal(t, StatusCancelled, d.Status)
}

func TestDispute_ClearCancellationRequest(t *testing.T) {
	d := makeDispute(StatusOpen)
	d.RecordRespondentReply()
	_, _ = d.Cancel(d.InitiatorID)

	d.ClearCancellationRequest()
	assert.Nil(t, d.CancellationRequestedBy)
	assert.Nil(t, d.CancellationRequestedAt)
}

// ---------------------------------------------------------------------------
// Counter-proposal
// ---------------------------------------------------------------------------

func TestNewCounterProposal_Success(t *testing.T) {
	cp, err := NewCounterProposal(NewCounterProposalInput{
		DisputeID:      uuid.New(),
		ProposerID:     uuid.New(),
		AmountClient:   40000,
		AmountProvider: 60000,
		ProposalAmount: 100000,
		Message:        "I propose 40/60",
	})
	assert.NoError(t, err)
	assert.Equal(t, CPStatusPending, cp.Status)
}

func TestNewCounterProposal_AmountMismatch(t *testing.T) {
	_, err := NewCounterProposal(NewCounterProposalInput{
		DisputeID:      uuid.New(),
		ProposerID:     uuid.New(),
		AmountClient:   40000,
		AmountProvider: 40000,
		ProposalAmount: 100000,
	})
	assert.ErrorIs(t, err, ErrAmountMismatch)
}

func TestCounterProposal_Accept(t *testing.T) {
	cp := makePendingCP()
	otherID := uuid.New()
	assert.NoError(t, cp.Accept(otherID))
	assert.Equal(t, CPStatusAccepted, cp.Status)
}

func TestCounterProposal_Accept_SelfRespond(t *testing.T) {
	cp := makePendingCP()
	assert.ErrorIs(t, cp.Accept(cp.ProposerID), ErrCannotRespondToOwnProposal)
}

func TestCounterProposal_Reject(t *testing.T) {
	cp := makePendingCP()
	otherID := uuid.New()
	assert.NoError(t, cp.Reject(otherID))
	assert.Equal(t, CPStatusRejected, cp.Status)
}

func TestCounterProposal_AcceptNotPending(t *testing.T) {
	cp := makePendingCP()
	cp.Status = CPStatusRejected
	assert.ErrorIs(t, cp.Accept(uuid.New()), ErrCounterProposalNotPending)
}

func TestCounterProposal_Supersede(t *testing.T) {
	cp := makePendingCP()
	cp.Supersede()
	assert.Equal(t, CPStatusSuperseded, cp.Status)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func TestDispute_IsParticipant(t *testing.T) {
	d := makeDispute(StatusOpen)
	assert.True(t, d.IsParticipant(d.InitiatorID))
	assert.True(t, d.IsParticipant(d.RespondentID))
	assert.False(t, d.IsParticipant(uuid.New()))
}

func TestDispute_CanBeCancelledBy(t *testing.T) {
	d := makeDispute(StatusOpen)
	assert.True(t, d.CanBeCancelledBy(d.InitiatorID))
	assert.False(t, d.CanBeCancelledBy(d.RespondentID))

	d.RecordRespondentReply()
	assert.False(t, d.CanBeCancelledBy(d.InitiatorID))
}

func TestDispute_InitiatorRole(t *testing.T) {
	d := makeDispute(StatusOpen)
	d.InitiatorID = d.ClientID
	assert.Equal(t, "client", d.InitiatorRole())

	d.InitiatorID = d.ProviderID
	assert.Equal(t, "provider", d.InitiatorRole())
}

func TestDispute_RecordRespondentReply_Idempotent(t *testing.T) {
	d := makeDispute(StatusOpen)
	d.RecordRespondentReply()
	first := d.RespondentFirstReplyAt

	time.Sleep(time.Millisecond)
	d.RecordRespondentReply()
	assert.Equal(t, first, d.RespondentFirstReplyAt) // not changed
}

// ---------------------------------------------------------------------------
// Test factories
// ---------------------------------------------------------------------------

func validInput() NewDisputeInput {
	clientID := uuid.New()
	providerID := uuid.New()
	return NewDisputeInput{
		ProposalID:      uuid.New(),
		ConversationID:  uuid.New(),
		InitiatorID:     clientID,
		RespondentID:    providerID,
		ClientID:        clientID,
		ProviderID:      providerID,
		Reason:          ReasonWorkNotConforming,
		Description:     "The deliverable does not match the scope.",
		RequestedAmount: 50000,
		ProposalAmount:  100000,
	}
}

func makeDispute(status Status) *Dispute {
	clientID := uuid.New()
	providerID := uuid.New()
	return &Dispute{
		ID:              uuid.New(),
		ProposalID:      uuid.New(),
		ConversationID:  uuid.New(),
		InitiatorID:     clientID,
		RespondentID:    providerID,
		ClientID:        clientID,
		ProviderID:      providerID,
		Reason:          ReasonWorkNotConforming,
		Description:     "Test dispute",
		RequestedAmount: 50000,
		ProposalAmount:  100000,
		Status:          status,
		LastActivityAt:  time.Now(),
		Version:         1,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
}

func makePendingCP() *CounterProposal {
	return &CounterProposal{
		ID:             uuid.New(),
		DisputeID:      uuid.New(),
		ProposerID:     uuid.New(),
		AmountClient:   40000,
		AmountProvider: 60000,
		Status:         CPStatusPending,
		CreatedAt:      time.Now(),
	}
}
