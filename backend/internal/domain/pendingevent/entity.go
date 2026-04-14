// Package pendingevent is the pure domain layer for the unified
// scheduler + outbox queue.
//
// A PendingEvent represents a piece of work the system needs to perform
// at or after a specific moment in the future. Two patterns share the
// same backing table:
//
//  1. **Scheduled events** — fire-once jobs queued in advance:
//     auto-approve a submitted milestone after 7 days, send a fund
//     reminder after 7 days of no payment, auto-close a proposal after
//     14 days of client ghosting.
//
//  2. **Outbox events** — durable side effects that must happen
//     exactly once with retry semantics: Stripe transfers, webhook
//     notifications, anything that touches an external system inside
//     a transaction boundary.
//
// Both patterns share the same lifecycle: pending → processing → done
// (or failed with retry). A background worker pops due events with
// FOR UPDATE SKIP LOCKED, dispatches them to type-specific handlers,
// and updates the row according to the handler outcome.
//
// This package has zero external dependencies beyond the Go stdlib
// and github.com/google/uuid. The persistence and worker live in the
// adapter and app layers respectively.
package pendingevent

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// EventType is the discriminator that the worker uses to route an
// event to its registered handler. Every event type has a typed
// payload struct elsewhere in the codebase that round-trips through
// the JSONB payload column.
type EventType string

const (
	// TypeMilestoneAutoApprove fires after the auto-approval delay
	// (default 7 days) on a submitted milestone, transitioning it
	// to approved → released without explicit client action.
	TypeMilestoneAutoApprove EventType = "milestone_auto_approve"

	// TypeMilestoneFundReminder fires after the fund-reminder delay
	// (default 7 days) when the next milestone is awaiting funding,
	// nudging the client by email + push.
	TypeMilestoneFundReminder EventType = "milestone_fund_reminder"

	// TypeProposalAutoClose fires after the auto-close delay
	// (default 14 days) when the client has ghosted on the next
	// milestone, gracefully closing the proposal in closed_partial.
	TypeProposalAutoClose EventType = "proposal_auto_close"

	// TypeStripeTransfer fires through the outbox path for every
	// milestone release, executing the Stripe Transfer with retry
	// + idempotency on the milestone_id.
	TypeStripeTransfer EventType = "stripe_transfer"
)

// IsValid reports whether the type is one of the recognised values.
func (t EventType) IsValid() bool {
	switch t {
	case TypeMilestoneAutoApprove, TypeMilestoneFundReminder,
		TypeProposalAutoClose, TypeStripeTransfer:
		return true
	}
	return false
}

// Status is the worker-facing lifecycle of a pending event. A row is
// born "pending", briefly held in "processing" while a worker pops
// and runs it, then settled to "done" on success or "failed" on a
// handler error (which schedules a retry by bumping fires_at).
type Status string

const (
	StatusPending    Status = "pending"
	StatusProcessing Status = "processing"
	StatusDone       Status = "done"
	StatusFailed     Status = "failed"
)

// IsValid reports whether the status is one of the recognised values.
func (s Status) IsValid() bool {
	switch s {
	case StatusPending, StatusProcessing, StatusDone, StatusFailed:
		return true
	}
	return false
}

// PendingEvent is the unified scheduler + outbox row.
//
// Payload is JSONB at rest; the worker decodes it into a typed struct
// based on EventType before dispatching to the handler. Handlers MUST
// be idempotent — a failed run will be retried, and a successful run
// that crashes before marking the row done will be re-attempted at
// the next tick.
type PendingEvent struct {
	ID         uuid.UUID
	EventType  EventType
	Payload    json.RawMessage
	FiresAt    time.Time
	Status     Status
	Attempts   int
	LastError  *string

	ProcessedAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// MaxAttempts caps how many times the worker will retry a failing
// event before giving up. After this many attempts the row is left
// in failed status and surfaced via the admin pending-events table
// for manual inspection.
const MaxAttempts = 5

// NewPendingEventInput is the validated factory input.
type NewPendingEventInput struct {
	EventType EventType
	Payload   json.RawMessage
	FiresAt   time.Time
}

// NewPendingEvent builds a validated PendingEvent ready for INSERT.
// The caller marshals the typed payload struct into the JSONB
// payload before calling.
func NewPendingEvent(input NewPendingEventInput) (*PendingEvent, error) {
	if !input.EventType.IsValid() {
		return nil, ErrInvalidEventType
	}
	if len(input.Payload) == 0 {
		return nil, ErrEmptyPayload
	}
	if input.FiresAt.IsZero() {
		return nil, ErrZeroFiresAt
	}
	now := time.Now()
	return &PendingEvent{
		ID:        uuid.New(),
		EventType: input.EventType,
		Payload:   input.Payload,
		FiresAt:   input.FiresAt,
		Status:    StatusPending,
		Attempts:  0,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// MarkProcessing transitions a pending or failed event into the
// processing window. Called by the worker right after FOR UPDATE
// SKIP LOCKED claims the row.
func (e *PendingEvent) MarkProcessing() error {
	if e.Status != StatusPending && e.Status != StatusFailed {
		return ErrInvalidStatus
	}
	e.Status = StatusProcessing
	e.Attempts++
	e.UpdatedAt = time.Now()
	return nil
}

// MarkDone settles a processing event as completed. The worker calls
// this after the handler returns nil. processed_at is recorded for
// audit / debugging.
func (e *PendingEvent) MarkDone() error {
	if e.Status != StatusProcessing {
		return ErrInvalidStatus
	}
	now := time.Now()
	e.Status = StatusDone
	e.ProcessedAt = &now
	e.LastError = nil
	e.UpdatedAt = now
	return nil
}

// MarkFailed records a handler failure and schedules a retry by
// bumping FiresAt forward according to an exponential backoff
// (1m, 5m, 15m, 1h, 6h). After MaxAttempts the row stays failed
// without a future fires_at — the worker won't pick it up again.
func (e *PendingEvent) MarkFailed(handlerErr error) error {
	if e.Status != StatusProcessing {
		return ErrInvalidStatus
	}
	msg := handlerErr.Error()
	e.LastError = &msg
	e.Status = StatusFailed
	e.UpdatedAt = time.Now()
	if e.Attempts < MaxAttempts {
		e.FiresAt = time.Now().Add(backoffFor(e.Attempts))
	}
	return nil
}

// HasExceededMaxAttempts reports whether the event has been retried
// MaxAttempts times. The admin dashboard can use this to surface
// stuck events for manual intervention.
func (e *PendingEvent) HasExceededMaxAttempts() bool {
	return e.Attempts >= MaxAttempts
}

// backoffFor returns the exponential backoff delay before the
// (attempt+1)-th retry. The schedule is intentionally coarse:
// 1 minute → 5 → 15 → 1 hour → 6 hours.
func backoffFor(attempts int) time.Duration {
	switch attempts {
	case 1:
		return 1 * time.Minute
	case 2:
		return 5 * time.Minute
	case 3:
		return 15 * time.Minute
	case 4:
		return 1 * time.Hour
	}
	return 6 * time.Hour
}
