package proposal

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ProposalStatus represents the lifecycle state of a proposal.
type ProposalStatus string

const (
	StatusPending             ProposalStatus = "pending"
	StatusAccepted            ProposalStatus = "accepted"
	StatusDeclined            ProposalStatus = "declined"
	StatusWithdrawn           ProposalStatus = "withdrawn"
	StatusPaid                ProposalStatus = "paid"
	StatusActive              ProposalStatus = "active"
	StatusCompletionRequested ProposalStatus = "completion_requested"
	StatusCompleted           ProposalStatus = "completed"
	StatusDisputed            ProposalStatus = "disputed"
)

func (s ProposalStatus) IsValid() bool {
	switch s {
	case StatusPending, StatusAccepted, StatusDeclined,
		StatusWithdrawn, StatusPaid, StatusActive,
		StatusCompletionRequested, StatusCompleted, StatusDisputed:
		return true
	}
	return false
}

// Proposal represents a commercial proposal exchanged within a conversation.
// Amount is stored in centimes (1 EUR = 100 centimes).
type Proposal struct {
	ID             uuid.UUID
	ConversationID uuid.UUID
	SenderID       uuid.UUID
	RecipientID    uuid.UUID
	Title          string
	Description    string
	Amount         int64
	Deadline       *time.Time
	Status         ProposalStatus
	ParentID       *uuid.UUID
	Version        int
	ClientID       uuid.UUID
	ProviderID     uuid.UUID
	Metadata        json.RawMessage
	// ActiveDisputeID points to the dispute currently in progress on this
	// proposal (status open/negotiation/escalated). Cleared on resolution
	// or cancellation by RestoreFromDispute.
	ActiveDisputeID *uuid.UUID
	// LastDisputeID is the most recent dispute that has ever existed on
	// this proposal, regardless of its current status. Set when a dispute
	// is opened and NEVER cleared, so the project page can always show
	// the historical decision (split + admin note) after resolution.
	LastDisputeID  *uuid.UUID
	AcceptedAt     *time.Time
	DeclinedAt     *time.Time
	PaidAt         *time.Time
	CompletedAt    *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// ProposalDocument represents a file attached to a proposal.
type ProposalDocument struct {
	ID         uuid.UUID
	ProposalID uuid.UUID
	Filename   string
	URL        string
	Size       int64
	MimeType   string
	CreatedAt  time.Time
}

// NewProposalInput contains the required fields to create a new Proposal.
type NewProposalInput struct {
	ConversationID uuid.UUID
	SenderID       uuid.UUID
	RecipientID    uuid.UUID
	Title          string
	Description    string
	Amount         int64
	Deadline       *time.Time
	ClientID       uuid.UUID
	ProviderID     uuid.UUID
	ParentID       *uuid.UUID
	Version        int
}

// NewProposal creates a validated Proposal from the given input.
func NewProposal(input NewProposalInput) (*Proposal, error) {
	if input.Title == "" {
		return nil, ErrEmptyTitle
	}
	if input.Description == "" {
		return nil, ErrEmptyDescription
	}
	if input.Amount <= 0 {
		return nil, ErrInvalidAmount
	}
	if input.Amount < 3000 {
		return nil, ErrBelowMinimumAmount
	}
	if input.SenderID == input.RecipientID {
		return nil, ErrSameUser
	}
	if input.ClientID == input.ProviderID {
		return nil, ErrSameUser
	}

	version := input.Version
	if version < 1 {
		version = 1
	}

	now := time.Now()
	return &Proposal{
		ID:             uuid.New(),
		ConversationID: input.ConversationID,
		SenderID:       input.SenderID,
		RecipientID:    input.RecipientID,
		Title:          input.Title,
		Description:    input.Description,
		Amount:         input.Amount,
		Deadline:       input.Deadline,
		Status:         StatusPending,
		ParentID:       input.ParentID,
		Version:        version,
		ClientID:       input.ClientID,
		ProviderID:     input.ProviderID,
		CreatedAt:      now,
		UpdatedAt:      now,
	}, nil
}

// Accept transitions a pending proposal to accepted.
// Only a participant who is NOT the sender of this version can accept.
func (p *Proposal) Accept(userID uuid.UUID) error {
	if p.Status != StatusPending {
		return ErrInvalidStatus
	}
	if userID == p.SenderID {
		return ErrNotAuthorized
	}
	if userID != p.RecipientID {
		return ErrNotAuthorized
	}
	now := time.Now()
	p.Status = StatusAccepted
	p.AcceptedAt = &now
	p.UpdatedAt = now
	return nil
}

// Decline transitions a pending proposal to declined.
func (p *Proposal) Decline(userID uuid.UUID) error {
	if p.Status != StatusPending {
		return ErrInvalidStatus
	}
	if userID != p.RecipientID {
		return ErrNotAuthorized
	}
	now := time.Now()
	p.Status = StatusDeclined
	p.DeclinedAt = &now
	p.UpdatedAt = now
	return nil
}

// Withdraw transitions a pending proposal to withdrawn.
// Only the sender can withdraw their own proposal.
func (p *Proposal) Withdraw(userID uuid.UUID) error {
	if p.Status != StatusPending {
		return ErrInvalidStatus
	}
	if userID != p.SenderID {
		return ErrNotAuthorized
	}
	now := time.Now()
	p.Status = StatusWithdrawn
	p.UpdatedAt = now
	return nil
}

// MarkPaid transitions an accepted proposal to paid.
func (p *Proposal) MarkPaid() error {
	if p.Status != StatusAccepted {
		return ErrInvalidStatus
	}
	now := time.Now()
	p.Status = StatusPaid
	p.PaidAt = &now
	p.UpdatedAt = now
	return nil
}

// MarkActive transitions a paid proposal to active.
func (p *Proposal) MarkActive() error {
	if p.Status != StatusPaid {
		return ErrInvalidStatus
	}
	p.Status = StatusActive
	p.UpdatedAt = time.Now()
	return nil
}

// RequestCompletion transitions an active proposal to completion_requested.
// Only the provider can request completion.
func (p *Proposal) RequestCompletion(userID uuid.UUID) error {
	if p.Status != StatusActive {
		return ErrInvalidStatus
	}
	if userID != p.ProviderID {
		return ErrNotProvider
	}
	p.Status = StatusCompletionRequested
	p.UpdatedAt = time.Now()
	return nil
}

// ConfirmCompletion transitions a completion_requested proposal to completed.
// Only the client can confirm completion.
func (p *Proposal) ConfirmCompletion(userID uuid.UUID) error {
	if p.Status != StatusCompletionRequested {
		return ErrInvalidStatus
	}
	if userID != p.ClientID {
		return ErrNotClient
	}
	now := time.Now()
	p.Status = StatusCompleted
	p.CompletedAt = &now
	p.UpdatedAt = now
	return nil
}

// RejectCompletion transitions a completion_requested proposal back to active.
// Only the client can reject a completion request.
func (p *Proposal) RejectCompletion(userID uuid.UUID) error {
	if p.Status != StatusCompletionRequested {
		return ErrInvalidStatus
	}
	if userID != p.ClientID {
		return ErrNotClient
	}
	p.Status = StatusActive
	p.UpdatedAt = time.Now()
	return nil
}

// MarkDisputed transitions the proposal to disputed status and records the
// dispute ID. Only active or completion_requested proposals can be disputed.
//
// Both ActiveDisputeID (cleared later by RestoreFromDispute) and
// LastDisputeID (kept forever) are set to the same value, so the project
// page can keep displaying the historical decision after restoration.
func (p *Proposal) MarkDisputed(disputeID uuid.UUID) error {
	if p.Status != StatusActive && p.Status != StatusCompletionRequested {
		return ErrInvalidStatus
	}
	p.ActiveDisputeID = &disputeID
	p.LastDisputeID = &disputeID
	p.Status = StatusDisputed
	p.UpdatedAt = time.Now()
	return nil
}

// RestoreFromDispute returns the proposal to the given target status after a
// dispute is resolved or cancelled. Clears the active dispute reference but
// keeps LastDisputeID intact for the historical display.
func (p *Proposal) RestoreFromDispute(target ProposalStatus) error {
	if p.Status != StatusDisputed {
		return ErrInvalidStatus
	}
	p.ActiveDisputeID = nil
	p.Status = target
	p.UpdatedAt = time.Now()
	return nil
}

// CanBeModifiedBy returns true if the given user can create a counter-proposal.
// Only the recipient of a pending proposal can modify it.
func (p *Proposal) CanBeModifiedBy(userID uuid.UUID) bool {
	return p.Status == StatusPending && userID == p.RecipientID
}

// DetermineRoles assigns client and provider roles based on user roles.
// Enterprise is always the client. Provider/freelance is always the provider.
// Agency acts as client when paired with a provider, and as provider when paired with an enterprise.
func DetermineRoles(
	senderID uuid.UUID, senderRole string,
	recipientID uuid.UUID, recipientRole string,
) (clientID, providerID uuid.UUID, err error) {
	roleA := senderRole
	roleB := recipientRole

	switch {
	case roleA == "enterprise" && (roleB == "provider" || roleB == "agency"):
		return senderID, recipientID, nil
	case (roleA == "provider" || roleA == "agency") && roleB == "enterprise":
		return recipientID, senderID, nil
	case roleA == "agency" && roleB == "provider":
		return senderID, recipientID, nil
	case roleA == "provider" && roleB == "agency":
		return recipientID, senderID, nil
	default:
		return uuid.Nil, uuid.Nil, ErrInvalidRoleCombination
	}
}
