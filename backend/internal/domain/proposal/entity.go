package proposal

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ProposalStatus represents the lifecycle state of a proposal.
type ProposalStatus string

const (
	StatusPending   ProposalStatus = "pending"
	StatusAccepted  ProposalStatus = "accepted"
	StatusDeclined  ProposalStatus = "declined"
	StatusWithdrawn ProposalStatus = "withdrawn"
	StatusPaid      ProposalStatus = "paid"
	StatusActive    ProposalStatus = "active"
	StatusCompleted ProposalStatus = "completed"
)

func (s ProposalStatus) IsValid() bool {
	switch s {
	case StatusPending, StatusAccepted, StatusDeclined,
		StatusWithdrawn, StatusPaid, StatusActive, StatusCompleted:
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
	Metadata       json.RawMessage
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

// MarkCompleted transitions an active proposal to completed.
func (p *Proposal) MarkCompleted() error {
	if p.Status != StatusActive {
		return ErrInvalidStatus
	}
	now := time.Now()
	p.Status = StatusCompleted
	p.CompletedAt = &now
	p.UpdatedAt = now
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
