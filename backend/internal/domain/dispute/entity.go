package dispute

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// Dispute statuses
// ---------------------------------------------------------------------------

type Status string

const (
	StatusOpen        Status = "open"
	StatusNegotiation Status = "negotiation"
	StatusEscalated   Status = "escalated"
	StatusResolved    Status = "resolved"
	StatusCancelled   Status = "cancelled"
)

func (s Status) IsTerminal() bool {
	return s == StatusResolved || s == StatusCancelled
}

// ---------------------------------------------------------------------------
// Dispute reasons (role-specific)
// ---------------------------------------------------------------------------

type Reason string

const (
	ReasonWorkNotConforming  Reason = "work_not_conforming"
	ReasonNonDelivery        Reason = "non_delivery"
	ReasonInsufficientQuality Reason = "insufficient_quality"
	ReasonClientGhosting     Reason = "client_ghosting"
	ReasonScopeCreep         Reason = "scope_creep"
	ReasonRefusalToValidate  Reason = "refusal_to_validate"
	ReasonHarassment         Reason = "harassment"
	ReasonOther              Reason = "other"
)

var clientReasons = map[Reason]bool{
	ReasonWorkNotConforming:   true,
	ReasonNonDelivery:         true,
	ReasonInsufficientQuality: true,
	ReasonOther:               true,
}

var providerReasons = map[Reason]bool{
	ReasonClientGhosting:    true,
	ReasonScopeCreep:        true,
	ReasonRefusalToValidate: true,
	ReasonHarassment:        true,
	ReasonOther:             true,
}

// IsValidForRole checks whether the reason is allowed for the given role.
// role is "client" or "provider".
func (r Reason) IsValidForRole(role string) bool {
	if role == "client" {
		return clientReasons[r]
	}
	return providerReasons[r]
}

// ---------------------------------------------------------------------------
// Resolution type
// ---------------------------------------------------------------------------

type ResolutionType string

const (
	ResolutionFullRefund    ResolutionType = "full_refund"
	ResolutionPartialRefund ResolutionType = "partial_refund"
	ResolutionFullRelease   ResolutionType = "full_release"
	ResolutionCustom        ResolutionType = "custom"
)

// ---------------------------------------------------------------------------
// Dispute entity
// ---------------------------------------------------------------------------

type Dispute struct {
	ID             uuid.UUID
	ProposalID     uuid.UUID
	ConversationID uuid.UUID
	InitiatorID    uuid.UUID
	RespondentID   uuid.UUID
	ClientID       uuid.UUID
	ProviderID     uuid.UUID

	Reason          Reason
	Description     string
	RequestedAmount int64
	ProposalAmount  int64

	Status Status

	ResolutionType          *ResolutionType
	ResolutionAmountClient  *int64
	ResolutionAmountProvider *int64
	ResolvedBy              *uuid.UUID
	ResolutionNote          *string
	AISummary               *string

	EscalatedAt            *time.Time
	ResolvedAt             *time.Time
	CancelledAt            *time.Time
	LastActivityAt         time.Time
	RespondentFirstReplyAt *time.Time

	// Cancellation request: set when the initiator asks to cancel after the
	// respondent has already replied. The respondent must accept or refuse.
	CancellationRequestedBy *uuid.UUID
	CancellationRequestedAt *time.Time

	Version   int
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ---------------------------------------------------------------------------
// Constructor
// ---------------------------------------------------------------------------

type NewDisputeInput struct {
	ProposalID      uuid.UUID
	ConversationID  uuid.UUID
	InitiatorID     uuid.UUID
	RespondentID    uuid.UUID
	ClientID        uuid.UUID
	ProviderID      uuid.UUID
	Reason          Reason
	Description     string
	RequestedAmount int64
	ProposalAmount  int64
}

func NewDispute(in NewDisputeInput) (*Dispute, error) {
	if len(in.Description) > 5000 {
		return nil, ErrDescriptionTooLong
	}
	if in.RequestedAmount <= 0 || in.RequestedAmount > in.ProposalAmount {
		return nil, ErrInvalidAmount
	}

	role := "provider"
	if in.InitiatorID == in.ClientID {
		role = "client"
	}
	if !in.Reason.IsValidForRole(role) {
		return nil, ErrInvalidReason
	}

	now := time.Now()
	return &Dispute{
		ID:              uuid.New(),
		ProposalID:      in.ProposalID,
		ConversationID:  in.ConversationID,
		InitiatorID:     in.InitiatorID,
		RespondentID:    in.RespondentID,
		ClientID:        in.ClientID,
		ProviderID:      in.ProviderID,
		Reason:          in.Reason,
		Description:     in.Description,
		RequestedAmount: in.RequestedAmount,
		ProposalAmount:  in.ProposalAmount,
		Status:          StatusOpen,
		LastActivityAt:  now,
		Version:         1,
		CreatedAt:       now,
		UpdatedAt:       now,
	}, nil
}

// ---------------------------------------------------------------------------
// State machine
// ---------------------------------------------------------------------------

func (d *Dispute) MarkNegotiation() error {
	if d.Status != StatusOpen {
		return ErrInvalidStatus
	}
	d.Status = StatusNegotiation
	d.UpdatedAt = time.Now()
	return nil
}

func (d *Dispute) Escalate() error {
	if d.Status != StatusOpen && d.Status != StatusNegotiation {
		return ErrInvalidStatus
	}
	now := time.Now()
	d.Status = StatusEscalated
	d.EscalatedAt = &now
	d.UpdatedAt = now
	return nil
}

func (d *Dispute) Resolve(in ResolveInput) error {
	if d.Status != StatusEscalated && d.Status != StatusOpen && d.Status != StatusNegotiation {
		return ErrInvalidStatus
	}
	if in.AmountClient+in.AmountProvider != d.ProposalAmount {
		return ErrAmountMismatch
	}
	now := time.Now()
	d.Status = StatusResolved
	rt := classifyResolution(in.AmountClient, d.ProposalAmount)
	d.ResolutionType = &rt
	d.ResolutionAmountClient = &in.AmountClient
	d.ResolutionAmountProvider = &in.AmountProvider
	if in.ResolvedBy != uuid.Nil {
		d.ResolvedBy = &in.ResolvedBy
	}
	if in.Note != "" {
		d.ResolutionNote = &in.Note
	}
	d.ResolvedAt = &now
	d.UpdatedAt = now
	return nil
}

func (d *Dispute) AutoResolveForInitiator() error {
	if d.Status != StatusOpen {
		return ErrInvalidStatus
	}
	// Initiator gets what they asked for
	var clientAmt, providerAmt int64
	if d.InitiatorID == d.ClientID {
		clientAmt = d.RequestedAmount
		providerAmt = d.ProposalAmount - d.RequestedAmount
	} else {
		providerAmt = d.RequestedAmount
		clientAmt = d.ProposalAmount - d.RequestedAmount
	}
	return d.Resolve(ResolveInput{
		AmountClient:  clientAmt,
		AmountProvider: providerAmt,
		Note:          "Auto-resolved: respondent did not reply within 7 days.",
	})
}

// Cancel attempts to cancel a dispute on behalf of one of its participants.
//
// The path taken depends on who is asking and whether the respondent has
// already engaged with the dispute:
//
//   - Initiator + respondent has NOT yet replied → direct cancellation.
//     The initiator may freely retract the dispute as long as the other side
//     has not invested any effort.
//
//   - Initiator + respondent HAS replied → creates a cancellation request.
//     The respondent now has a stake and must explicitly consent.
//
//   - Respondent (non-initiator), at any point → ALWAYS creates a
//     cancellation request, never a direct cancellation. The respondent
//     never had the unilateral right to terminate a dispute they did not
//     open; they can only ask the initiator for permission to cancel.
//
// Returns (true, nil) when the dispute was cancelled directly,
// or (false, nil) when a cancellation request was created.
func (d *Dispute) Cancel(userID uuid.UUID) (cancelled bool, err error) {
	if d.Status != StatusOpen && d.Status != StatusNegotiation {
		return false, ErrInvalidStatus
	}
	if !d.IsParticipant(userID) {
		return false, ErrNotParticipant
	}

	// Direct cancellation path — reserved to the initiator and only valid
	// while the respondent has not yet engaged.
	if userID == d.InitiatorID && d.RespondentFirstReplyAt == nil {
		now := time.Now()
		d.Status = StatusCancelled
		d.CancelledAt = &now
		d.UpdatedAt = now
		return true, nil
	}

	// All other cases (initiator after a reply, or respondent at any time)
	// must go through a cancellation request that the OTHER party accepts.
	if d.CancellationRequestedBy != nil {
		return false, ErrCancellationAlreadyRequested
	}
	now := time.Now()
	d.CancellationRequestedBy = &userID
	d.CancellationRequestedAt = &now
	d.UpdatedAt = now
	return false, nil
}

// RespondToCancellationRequest processes the other party's decision on a
// pending cancellation request. Either the initiator or the respondent may
// be the requester (since both can ask), so the only invariant is that the
// requester themselves cannot self-accept their own request — only the
// OTHER participant can.
// If accepted, the dispute is cancelled; if refused, the request is cleared.
func (d *Dispute) RespondToCancellationRequest(userID uuid.UUID, accept bool) error {
	if d.CancellationRequestedBy == nil {
		return ErrNoCancellationPending
	}
	if !d.IsParticipant(userID) {
		return ErrNotParticipant
	}
	if userID == *d.CancellationRequestedBy {
		return ErrNotAuthorized
	}

	now := time.Now()
	if accept {
		d.Status = StatusCancelled
		d.CancelledAt = &now
	} else {
		d.CancellationRequestedBy = nil
		d.CancellationRequestedAt = nil
	}
	d.UpdatedAt = now
	return nil
}

// ClearCancellationRequest removes any pending cancellation request.
// Called implicitly when the dispute state changes (e.g. a counter-proposal
// signals that negotiation is still active).
func (d *Dispute) ClearCancellationRequest() {
	if d.CancellationRequestedBy != nil {
		d.CancellationRequestedBy = nil
		d.CancellationRequestedAt = nil
		d.UpdatedAt = time.Now()
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func (d *Dispute) RecordActivity() {
	d.LastActivityAt = time.Now()
	d.UpdatedAt = d.LastActivityAt
}

func (d *Dispute) RecordRespondentReply() {
	if d.RespondentFirstReplyAt == nil {
		now := time.Now()
		d.RespondentFirstReplyAt = &now
	}
}

func (d *Dispute) SetAISummary(summary string) {
	d.AISummary = &summary
	d.UpdatedAt = time.Now()
}

func (d *Dispute) IsParticipant(userID uuid.UUID) bool {
	return userID == d.InitiatorID || userID == d.RespondentID
}

func (d *Dispute) CanBeCancelledBy(userID uuid.UUID) bool {
	if d.Status.IsTerminal() {
		return false
	}
	return userID == d.InitiatorID && d.RespondentFirstReplyAt == nil
}

func (d *Dispute) InitiatorRole() string {
	if d.InitiatorID == d.ClientID {
		return "client"
	}
	return "provider"
}

func classifyResolution(clientAmount, proposalAmount int64) ResolutionType {
	if clientAmount == proposalAmount {
		return ResolutionFullRefund
	}
	if clientAmount == 0 {
		return ResolutionFullRelease
	}
	return ResolutionCustom
}

// ---------------------------------------------------------------------------
// ResolveInput
// ---------------------------------------------------------------------------

type ResolveInput struct {
	ResolvedBy     uuid.UUID
	AmountClient   int64
	AmountProvider int64
	Note           string
}

// ---------------------------------------------------------------------------
// Evidence
// ---------------------------------------------------------------------------

type Evidence struct {
	ID                uuid.UUID
	DisputeID         uuid.UUID
	CounterProposalID *uuid.UUID // nil = attached to dispute opening, set = attached to a counter-proposal
	UploaderID        uuid.UUID
	Filename          string
	URL               string
	Size              int64
	MimeType          string
	CreatedAt         time.Time
}

// ---------------------------------------------------------------------------
// Counter-proposal
// ---------------------------------------------------------------------------

type CounterProposalStatus string

const (
	CPStatusPending    CounterProposalStatus = "pending"
	CPStatusAccepted   CounterProposalStatus = "accepted"
	CPStatusRejected   CounterProposalStatus = "rejected"
	CPStatusSuperseded CounterProposalStatus = "superseded"
)

type CounterProposal struct {
	ID             uuid.UUID
	DisputeID      uuid.UUID
	ProposerID     uuid.UUID
	AmountClient   int64
	AmountProvider int64
	Message        string
	Status         CounterProposalStatus
	RespondedAt    *time.Time
	CreatedAt      time.Time
}

type NewCounterProposalInput struct {
	DisputeID      uuid.UUID
	ProposerID     uuid.UUID
	AmountClient   int64
	AmountProvider int64
	ProposalAmount int64
	Message        string
}

func NewCounterProposal(in NewCounterProposalInput) (*CounterProposal, error) {
	if in.AmountClient < 0 || in.AmountProvider < 0 {
		return nil, ErrInvalidAmount
	}
	if in.AmountClient+in.AmountProvider != in.ProposalAmount {
		return nil, ErrAmountMismatch
	}
	return &CounterProposal{
		ID:             uuid.New(),
		DisputeID:      in.DisputeID,
		ProposerID:     in.ProposerID,
		AmountClient:   in.AmountClient,
		AmountProvider: in.AmountProvider,
		Message:        in.Message,
		Status:         CPStatusPending,
		CreatedAt:      time.Now(),
	}, nil
}

func (cp *CounterProposal) Accept(userID uuid.UUID) error {
	if cp.Status != CPStatusPending {
		return ErrCounterProposalNotPending
	}
	if userID == cp.ProposerID {
		return ErrCannotRespondToOwnProposal
	}
	now := time.Now()
	cp.Status = CPStatusAccepted
	cp.RespondedAt = &now
	return nil
}

func (cp *CounterProposal) Reject(userID uuid.UUID) error {
	if cp.Status != CPStatusPending {
		return ErrCounterProposalNotPending
	}
	if userID == cp.ProposerID {
		return ErrCannotRespondToOwnProposal
	}
	now := time.Now()
	cp.Status = CPStatusRejected
	cp.RespondedAt = &now
	return nil
}

func (cp *CounterProposal) Supersede() {
	if cp.Status == CPStatusPending {
		cp.Status = CPStatusSuperseded
	}
}

// MustJSON marshals v to json.RawMessage, ignoring errors (for metadata).
func MustJSON(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}
