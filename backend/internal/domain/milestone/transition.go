package milestone

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Transition is an append-only audit record of a single milestone
// state change. Every successful withMilestoneLock in the proposal
// service writes one of these so the admin dashboard, dispute
// arbitration, and post-incident review can replay the exact who/
// when/why of every transition.
//
// The table is INSERT-only at the application level — there are no
// Update or Delete methods on the repository. The production DB
// user holds INSERT/SELECT only on milestone_transitions.
type Transition struct {
	ID          uuid.UUID
	MilestoneID uuid.UUID
	ProposalID  uuid.UUID

	FromStatus MilestoneStatus
	ToStatus   MilestoneStatus

	// ActorID is NULL when the transition is performed by the system
	// (auto-approve scheduler, outbox handler). Populated otherwise
	// for the user/operator who triggered the change.
	ActorID    *uuid.UUID
	ActorOrgID *uuid.UUID

	// Reason is a free-form short string the caller passes to
	// describe the transition — e.g. "auto-approved after 7d",
	// "rejected: needs revision", "boundary cancel". Optional.
	Reason string

	// Metadata is structured arbitrary context — payment intent id
	// for fund events, dispute id for dispute events, bytes
	// transferred for stripe transfer events, etc. Marshalled as
	// JSONB at the adapter layer.
	Metadata json.RawMessage

	CreatedAt time.Time
}

// NewTransition builds a fresh transition row. Validation is light
// because the entity is purely data — the adapter enforces FK
// integrity and the caller owns the from/to statuses.
func NewTransition(
	milestoneID, proposalID uuid.UUID,
	from, to MilestoneStatus,
	actorID, actorOrgID *uuid.UUID,
	reason string,
	metadata json.RawMessage,
) *Transition {
	return &Transition{
		ID:          uuid.New(),
		MilestoneID: milestoneID,
		ProposalID:  proposalID,
		FromStatus:  from,
		ToStatus:    to,
		ActorID:     actorID,
		ActorOrgID:  actorOrgID,
		Reason:      reason,
		Metadata:    metadata,
		CreatedAt:   time.Now(),
	}
}
