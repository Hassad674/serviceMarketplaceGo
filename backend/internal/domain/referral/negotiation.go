package referral

import (
	"time"

	"github.com/google/uuid"
)

// NegotiationAction is the kind of action an actor took during a negotiation
// round. Stored on every Negotiation row to power the timeline UI.
type NegotiationAction string

const (
	NegoActionProposed  NegotiationAction = "proposed"
	NegoActionCountered NegotiationAction = "countered"
	NegoActionAccepted  NegotiationAction = "accepted"
	NegoActionRejected  NegotiationAction = "rejected"
)

// IsValid reports whether a is one of the known actions.
func (a NegotiationAction) IsValid() bool {
	switch a {
	case NegoActionProposed, NegoActionCountered, NegoActionAccepted, NegoActionRejected:
		return true
	}
	return false
}

// Negotiation is one immutable row in the bilateral negotiation audit trail
// between the apporteur and the provider. The client never produces a
// Negotiation row — they only Accept or Reject the activated terms (Modèle A).
//
// Each row captures: who acted, what they did, the rate that was on the table
// at that moment, and an optional free-text justification.
type Negotiation struct {
	ID         uuid.UUID
	ReferralID uuid.UUID
	Version    int
	ActorID    uuid.UUID
	ActorRole  ActorRole
	Action     NegotiationAction
	RatePct    float64
	Message    string
	CreatedAt  time.Time
}

// NewNegotiationInput is the validated input for NewNegotiation.
type NewNegotiationInput struct {
	ReferralID uuid.UUID
	Version    int
	ActorID    uuid.UUID
	ActorRole  ActorRole
	Action     NegotiationAction
	RatePct    float64
	Message    string
}

// NewNegotiation builds a validated Negotiation row.
func NewNegotiation(input NewNegotiationInput) (*Negotiation, error) {
	if input.ReferralID == uuid.Nil || input.ActorID == uuid.Nil {
		return nil, ErrNotAuthorized
	}
	if !input.ActorRole.IsValid() {
		return nil, ErrNotAuthorized
	}
	if !input.Action.IsValid() {
		return nil, ErrInvalidTransition
	}
	if input.Version < 1 {
		return nil, ErrInvalidTransition
	}
	if input.RatePct < MinRatePct || input.RatePct > MaxRatePct {
		return nil, ErrRateOutOfRange
	}
	if len([]rune(input.Message)) > MaxIntroMessageLen {
		return nil, ErrMessageTooLong
	}
	return &Negotiation{
		ID:         uuid.New(),
		ReferralID: input.ReferralID,
		Version:    input.Version,
		ActorID:    input.ActorID,
		ActorRole:  input.ActorRole,
		Action:     input.Action,
		RatePct:    input.RatePct,
		Message:    input.Message,
		CreatedAt:  time.Now().UTC(),
	}, nil
}
