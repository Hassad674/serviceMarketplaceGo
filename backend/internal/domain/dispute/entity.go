package dispute

import (
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
	ReasonWorkNotConforming   Reason = "work_not_conforming"
	ReasonNonDelivery         Reason = "non_delivery"
	ReasonInsufficientQuality Reason = "insufficient_quality"
	ReasonClientGhosting      Reason = "client_ghosting"
	ReasonScopeCreep          Reason = "scope_creep"
	ReasonRefusalToValidate   Reason = "refusal_to_validate"
	ReasonHarassment          Reason = "harassment"
	ReasonOther               Reason = "other"
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
	ID         uuid.UUID
	ProposalID uuid.UUID
	// MilestoneID scopes the dispute to a single milestone of the
	// proposal (phase 8). Resolution splits the milestone amount,
	// not the proposal amount. Existing pre-phase-8 disputes were
	// backfilled to their proposal's only synthetic milestone.
	MilestoneID    uuid.UUID
	ConversationID uuid.UUID
	InitiatorID    uuid.UUID
	RespondentID   uuid.UUID
	ClientID       uuid.UUID
	ProviderID     uuid.UUID

	// Denormalized org anchors (R3 extended): the client's and
	// provider's current organization at the moment the dispute was
	// opened. Used to scope ListByOrganization so every operator of
	// either org sees the dispute in their list.
	ClientOrganizationID   uuid.UUID
	ProviderOrganizationID uuid.UUID

	Reason          Reason
	Description     string
	RequestedAmount int64
	ProposalAmount  int64

	Status Status

	ResolutionType           *ResolutionType
	ResolutionAmountClient   *int64
	ResolutionAmountProvider *int64
	ResolvedBy               *uuid.UUID
	ResolutionNote           *string
	AISummary                *string

	EscalatedAt            *time.Time
	ResolvedAt             *time.Time
	CancelledAt            *time.Time
	LastActivityAt         time.Time
	RespondentFirstReplyAt *time.Time

	// Cancellation request: set when the initiator asks to cancel after the
	// respondent has already replied. The respondent must accept or refuse.
	CancellationRequestedBy *uuid.UUID
	CancellationRequestedAt *time.Time

	// AI budget tracking — cumulative across the dispute lifetime. Summary
	// and chat tokens are tracked separately so the admin UI can show
	// distinct progress bars per category. AIBudgetBonusTokens grows each
	// time the admin clicks "Augmenter le budget" and is added to BOTH
	// the summary and chat caps (whichever the admin needs more of).
	AISummaryInputTokens  int
	AISummaryOutputTokens int
	AIChatInputTokens     int
	AIChatOutputTokens    int
	AIBudgetBonusTokens   int

	Version   int
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ---------------------------------------------------------------------------
// Constructor
// ---------------------------------------------------------------------------

type NewDisputeInput struct {
	ProposalID             uuid.UUID
	MilestoneID            uuid.UUID
	ConversationID         uuid.UUID
	InitiatorID            uuid.UUID
	RespondentID           uuid.UUID
	ClientID               uuid.UUID
	ProviderID             uuid.UUID
	ClientOrganizationID   uuid.UUID
	ProviderOrganizationID uuid.UUID
	Reason                 Reason
	Description            string
	RequestedAmount        int64
	// ProposalAmount carries the milestone amount (post-phase-8)
	// rather than the proposal total. Field name kept for backward
	// compatibility with existing SQL queries; the semantics moved
	// to "the milestone in dispute".
	ProposalAmount int64
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
		ID:                     uuid.New(),
		ProposalID:             in.ProposalID,
		MilestoneID:            in.MilestoneID,
		ConversationID:         in.ConversationID,
		InitiatorID:            in.InitiatorID,
		RespondentID:           in.RespondentID,
		ClientID:               in.ClientID,
		ProviderID:             in.ProviderID,
		ClientOrganizationID:   in.ClientOrganizationID,
		ProviderOrganizationID: in.ProviderOrganizationID,
		Reason:                 in.Reason,
		Description:            in.Description,
		RequestedAmount:        in.RequestedAmount,
		ProposalAmount:         in.ProposalAmount,
		Status:                 StatusOpen,
		LastActivityAt:         now,
		Version:                1,
		CreatedAt:              now,
		UpdatedAt:              now,
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
		AmountClient:   clientAmt,
		AmountProvider: providerAmt,
		Note:           "Auto-resolved: respondent did not reply within 7 days.",
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
	// Cancellation is allowed all the way through admin mediation: as long
	// as the admin has not rendered a final decision, the parties can still
	// reach an amicable agreement (whichever comes first wins).
	if d.Status != StatusOpen && d.Status != StatusNegotiation && d.Status != StatusEscalated {
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
