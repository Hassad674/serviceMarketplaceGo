package repository

import (
	"context"
	"database/sql"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/pendingevent"
)

// PendingEventRepository defines persistence operations for the
// unified scheduler + outbox queue.
//
// The worker is the only writer that owns the lifecycle: it pops
// due events with PopDue (which uses FOR UPDATE SKIP LOCKED so two
// workers never claim the same row), runs the handler, then calls
// MarkDone or MarkFailed on the result.
//
// Schedule is the public writer for everywhere else — services call
// it inside their own transactions when they want a future side
// effect (e.g. proposal service schedules milestone_auto_approve
// when a milestone is submitted).
type PendingEventRepository interface {
	// Schedule inserts a new pending event using the repository's
	// own connection pool. Idempotency at the caller level is the
	// writer's responsibility — schedule the same payload twice and
	// you get two rows.
	Schedule(ctx context.Context, e *pendingevent.PendingEvent) error

	// ScheduleStripe inserts a Stripe-webhook pending event with
	// at-most-once-per-Stripe-event-id semantics. The implementation
	// uses ON CONFLICT (stripe_event_id) DO NOTHING so a Stripe
	// re-delivery (Stripe retries on any non-2xx response) is a
	// silent no-op rather than a duplicate row.
	//
	// Returns (true, nil) on first delivery, (false, nil) when the
	// event was already enqueued, or (_, err) on a database error.
	// The caller must reply 200 OK to Stripe in both the inserted
	// and deduplicated cases.
	ScheduleStripe(ctx context.Context, e *pendingevent.PendingEvent) (bool, error)

	// ScheduleTx inserts a new pending event inside an existing
	// transaction. Used by the outbox pattern (BUG-05): callers
	// committing a domain mutation alongside an `event` (e.g. a
	// profile UPDATE plus a `search.reindex` row) MUST use this
	// variant so the two writes share a single atomic boundary.
	// A rollback on the surrounding tx — for any reason — also
	// drops the event, guaranteeing Postgres and the downstream
	// search index never drift.
	ScheduleTx(ctx context.Context, tx *sql.Tx, e *pendingevent.PendingEvent) error

	// PopDue claims up to `limit` events whose fires_at is in the
	// past, marks them processing in the same transaction, and
	// returns them. Uses FOR UPDATE SKIP LOCKED so concurrent
	// workers never see the same row twice. The attempts counter is
	// bumped per-row inside the same transaction.
	//
	// The events returned are already in StatusProcessing — the
	// worker calls MarkDone or MarkFailed once the handler returns.
	PopDue(ctx context.Context, limit int) ([]*pendingevent.PendingEvent, error)

	// MarkDone settles a processing event as completed. Called by
	// the worker after a handler returns nil.
	MarkDone(ctx context.Context, e *pendingevent.PendingEvent) error

	// MarkFailed settles a processing event as failed and reschedules
	// it according to the backoff embedded in the entity. Called by
	// the worker after a handler returns an error.
	MarkFailed(ctx context.Context, e *pendingevent.PendingEvent) error

	// GetByID fetches a single event without taking any lock. Used
	// by admin endpoints to inspect a stuck event.
	GetByID(ctx context.Context, id uuid.UUID) (*pendingevent.PendingEvent, error)
}
