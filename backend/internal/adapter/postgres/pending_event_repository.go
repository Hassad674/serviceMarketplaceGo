package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/pendingevent"
)

// staleProcessingThreshold is how long a row is allowed to sit in
// 'processing' before another worker treats it as orphaned and
// reclaims it. Tuned wider than any reasonable handler runtime — if
// any single dispatch ever takes longer than this, two workers will
// race the same row. 5 minutes covers slow Stripe / search-index
// roundtrips by an order of magnitude.
//
// Exported via PopDueWithStaleThreshold so integration tests can drive
// faster recoveries without waiting 5 real minutes.
const staleProcessingThreshold = 5 * time.Minute

// PendingEventRepository is the postgres-backed implementation of the
// scheduler + outbox queue. It is intentionally small: 5 methods, all
// driven by single SQL statements in pending_event_queries.go.
//
// The hot path (PopDue) uses FOR UPDATE SKIP LOCKED inside a CTE so
// concurrent workers can pop disjoint batches of events without ever
// blocking each other or claiming the same row twice.
type PendingEventRepository struct {
	db *sql.DB
}

// NewPendingEventRepository wires the adapter against a sql.DB pool.
func NewPendingEventRepository(db *sql.DB) *PendingEventRepository {
	return &PendingEventRepository{db: db}
}

// Schedule inserts a new pending event using the repository's own
// connection pool. Callers that need outbox semantics (event row
// committed in the same transaction as the domain mutation) must
// use ScheduleTx instead — losing the atomic boundary on this path
// is precisely the data-drift class of bug ScheduleTx was added to
// eliminate (see BUG-05).
func (r *PendingEventRepository) Schedule(ctx context.Context, e *pendingevent.PendingEvent) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	_, err := r.db.ExecContext(ctx, queryInsertPendingEvent,
		e.ID, string(e.EventType), e.Payload, e.FiresAt,
		string(e.Status), e.Attempts, e.LastError,
		e.ProcessedAt, e.CreatedAt, e.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert pending event: %w", err)
	}
	return nil
}

// ScheduleTx inserts a pending event inside an existing transaction.
// The caller owns the transaction lifecycle — Begin / Commit /
// Rollback all happen outside this method. We share the same INSERT
// statement as Schedule so the column list (and any future schema
// migration) stays in lock-step between the two paths.
func (r *PendingEventRepository) ScheduleTx(ctx context.Context, tx *sql.Tx, e *pendingevent.PendingEvent) error {
	if tx == nil {
		return fmt.Errorf("schedule pending event: tx is required")
	}
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	_, err := tx.ExecContext(ctx, queryInsertPendingEvent,
		e.ID, string(e.EventType), e.Payload, e.FiresAt,
		string(e.Status), e.Attempts, e.LastError,
		e.ProcessedAt, e.CreatedAt, e.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert pending event in tx: %w", err)
	}
	return nil
}

// PopDue claims up to `limit` due events in a single round trip,
// marking them processing inside the same statement. Returns the
// freshly-claimed events so the worker can dispatch them to handlers
// without a second SELECT.
//
// Concurrent workers are safe: FOR UPDATE SKIP LOCKED hands disjoint
// batches to each caller. There is no possibility of double-pop, no
// row-level deadlock, and no need for an external lock.
//
// BUG-NEW-03 — also reclaims rows stuck in 'processing' whose
// updated_at is older than staleProcessingThreshold. This recovers
// from worker crashes between claim and MarkDone/Failed.
func (r *PendingEventRepository) PopDue(ctx context.Context, limit int) ([]*pendingevent.PendingEvent, error) {
	return r.PopDueWithStaleThreshold(ctx, limit, staleProcessingThreshold)
}

// PopDueWithStaleThreshold lets callers override the stale-processing
// recovery window. Used by integration tests to verify the recovery
// path without waiting the full production threshold (5 minutes).
// Production code must use PopDue.
func (r *PendingEventRepository) PopDueWithStaleThreshold(ctx context.Context, limit int, stale time.Duration) ([]*pendingevent.PendingEvent, error) {
	if limit <= 0 {
		limit = 10
	}
	if stale <= 0 {
		stale = staleProcessingThreshold
	}
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	// Pass stale as seconds — make_interval(secs => $2) on the SQL
	// side. Float so sub-second test thresholds round correctly.
	staleSeconds := stale.Seconds()
	rows, err := r.db.QueryContext(ctx, queryPopDuePendingEvents, limit, staleSeconds)
	if err != nil {
		return nil, fmt.Errorf("pop due pending events: %w", err)
	}
	defer rows.Close()

	var events []*pendingevent.PendingEvent
	for rows.Next() {
		e, err := scanPendingEvent(rows)
		if err != nil {
			return nil, fmt.Errorf("scan pending event: %w", err)
		}
		events = append(events, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows err: %w", err)
	}
	return events, nil
}

// MarkDone settles a processing event as completed. Called by the
// worker after a successful handler run.
func (r *PendingEventRepository) MarkDone(ctx context.Context, e *pendingevent.PendingEvent) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	result, err := r.db.ExecContext(ctx, queryMarkPendingEventDone, e.ID)
	if err != nil {
		return fmt.Errorf("mark pending event done: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if affected == 0 {
		// Row was not in processing status — likely another worker
		// claimed it concurrently. Surface as not found so the worker
		// can move on without crashing.
		return pendingevent.ErrEventNotFound
	}
	return nil
}

// MarkFailed records a handler error and reschedules the event for
// a later retry according to the backoff already computed by the
// domain entity (in MarkFailed on the in-memory copy).
func (r *PendingEventRepository) MarkFailed(ctx context.Context, e *pendingevent.PendingEvent) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	var lastErr sql.NullString
	if e.LastError != nil {
		lastErr.String = *e.LastError
		lastErr.Valid = true
	}
	result, err := r.db.ExecContext(ctx, queryMarkPendingEventFailed, e.ID, lastErr, e.FiresAt)
	if err != nil {
		return fmt.Errorf("mark pending event failed: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if affected == 0 {
		return pendingevent.ErrEventNotFound
	}
	return nil
}

// GetByID fetches a single event without locking. Used by admin
// inspection paths.
func (r *PendingEventRepository) GetByID(ctx context.Context, id uuid.UUID) (*pendingevent.PendingEvent, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	row := r.db.QueryRowContext(ctx, queryGetPendingEventByID, id)
	e, err := scanPendingEvent(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, pendingevent.ErrEventNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get pending event by id: %w", err)
	}
	return e, nil
}

// scanPendingEvent materialises a row into a domain entity. Status
// and EventType are converted from TEXT to typed enums.
func scanPendingEvent(s scanner) (*pendingevent.PendingEvent, error) {
	var e pendingevent.PendingEvent
	var (
		eventType string
		status    string
		lastError sql.NullString
	)
	if err := s.Scan(
		&e.ID, &eventType, &e.Payload, &e.FiresAt,
		&status, &e.Attempts, &lastError,
		&e.ProcessedAt, &e.CreatedAt, &e.UpdatedAt,
	); err != nil {
		return nil, err
	}
	e.EventType = pendingevent.EventType(eventType)
	e.Status = pendingevent.Status(status)
	if lastError.Valid {
		err := lastError.String
		e.LastError = &err
	}
	return &e, nil
}
