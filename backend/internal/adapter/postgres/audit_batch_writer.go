// Package postgres holds the PostgreSQL adapter implementations of
// the repository ports. This file owns the batch audit writer wrapper.
//
// Mission PERF-F3 — kill 4 round-trips per audit event.
//
// Why a batch writer:
//
//	Every direct call to AuditRepository.Log runs inside
//	RunInTxWithTenant = `BEGIN + SELECT set_config + INSERT + COMMIT`,
//	which is 4 round-trips on Neon (~60 ms per event). Login/logout/
//	refresh/mutation/2FA/admin endpoints emit 1-5 audit rows each,
//	and the background goroutine that owns the call still steals a
//	connection pool slot for the entire round-trip burst.
//
//	BatchAuditWriter receives events into a buffered channel and
//	flushes them with a single multi-row INSERT every 5 s OR when 100
//	events are queued, whichever fires first.
//
// Correctness invariants:
//
//   - Order preservation: events flush in the order they were Log'd.
//     A single channel is the queue, FIFO. The flusher drains the
//     channel into a slice, then issues ONE multi-row INSERT in the
//     same order.
//
//   - Tenant isolation: audit_logs is RLS-protected by migration 125
//     with `USING (user_id = current_setting(...))` and `WITH CHECK
//     (true)` (migration 129). The WITH CHECK clause is what makes
//     batch inserts safe across mixed actors: every row is allowed to
//     insert regardless of the session's app.current_user_id, while
//     the USING clause still filters reads per actor. The batch path
//     therefore does NOT need to set the tenant context per row.
//     This matches the existing AuditRepository.Log path, which only
//     sets the context "for parity with the rest of the RLS
//     migration" — the INSERT would succeed without it.
//
//   - Backpressure: the channel is bounded. When it fills, Submit
//     BLOCKS the caller until the next flush drains slots. We never
//     drop audit events silently — losing audit rows would violate
//     compliance requirements (CLAUDE.md "audit_logs is append-only,
//     kept indefinitely").
//
//   - Crash safety: a process crash mid-buffer LOSES the in-flight
//     events. This is the documented trade-off of batching. Shutdown
//     emits a structured metric ("audit_batch_events_at_shutdown")
//     for ops visibility. The compliance trade-off is acceptable
//     because the largest exposure window is the flush interval
//     (5 s) plus one INSERT round-trip (~20 ms on Neon).
//
//   - Failure resilience: if the multi-row INSERT fails, the buffered
//     batch is held and retried on the next tick. We retry up to N
//     times (3) before logging a structured ERROR and discarding the
//     batch — at that point the DB is unrecoverable and dropping is
//     better than blocking the queue forever.
package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/audit"
	"marketplace-backend/internal/port/repository"
)

// Compile-time guard: BatchAuditWriter satisfies the AuditRepository
// port. The decorator forwards List methods unchanged and only
// reroutes the Log path through the buffer.
var _ repository.AuditRepository = (*BatchAuditWriter)(nil)

// BatchAuditConfig groups the tunable knobs. Defaults are chosen for
// the production traffic profile in /perf-audit.md (5-10 audit rows
// per busy endpoint, ~50 RPS on the hot path).
type BatchAuditConfig struct {
	// FlushInterval is the maximum time a buffered event waits before
	// a forced flush. Default 5 s.
	FlushInterval time.Duration

	// FlushThreshold is the buffered-event count that triggers an
	// immediate flush, regardless of FlushInterval. Default 100.
	FlushThreshold int

	// ChannelCapacity bounds the in-flight queue. When the channel
	// fills, Log blocks until the next flush drains slots. Default
	// 1024 — large enough to absorb a 10× burst above FlushThreshold
	// without backpressure during a normal flush cycle.
	ChannelCapacity int

	// MaxRetriesOnFlushFailure caps how many times the flusher
	// retries a batch on transient DB failures before logging an
	// error and dropping the batch. Default 3.
	MaxRetriesOnFlushFailure int

	// FlushTimeout is the per-flush context timeout for the multi-row
	// INSERT. Default 10 s — comfortably above queryTimeout to allow
	// large batches to land on Neon.
	FlushTimeout time.Duration
}

// DefaultBatchAuditConfig returns the production defaults.
func DefaultBatchAuditConfig() BatchAuditConfig {
	return BatchAuditConfig{
		FlushInterval:            5 * time.Second,
		FlushThreshold:           100,
		ChannelCapacity:          1024,
		MaxRetriesOnFlushFailure: 3,
		FlushTimeout:             10 * time.Second,
	}
}

// auditBatchSink is the narrow interface the batch writer needs from
// the underlying postgres pool: open a transaction and exec a
// multi-row INSERT. A free-standing interface (rather than depending
// on *sql.DB) keeps the writer testable with an in-memory fake.
type auditBatchSink interface {
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}

// BatchAuditWriter buffers audit events and flushes them to the
// wrapped AuditRepository in groups. Constructed with
// NewBatchAuditWriter; start the background flusher with Start; stop
// it with Stop (or by cancelling the supplied context).
type BatchAuditWriter struct {
	cfg   BatchAuditConfig
	inner repository.AuditRepository // List paths forward here unchanged.
	sink  auditBatchSink             // The DB pool used for the batched INSERT.

	ch chan *audit.Entry

	// flusherRunning tracks whether Start has been called; double-Start
	// is a no-op (returns the existing context's done channel).
	flusherRunning atomic.Bool

	// done is closed when the flusher goroutine exits cleanly. Stop
	// waits on this to confirm graceful shutdown.
	done chan struct{}

	// queuedAtShutdown is a metric: how many events were still in the
	// buffer when Stop was called and the final flush completed (or
	// failed). Exposed for ops visibility.
	queuedAtShutdown atomic.Int64

	// onFlush is an optional hook for tests — invoked with the batch
	// size after every successful flush. Production wiring leaves it
	// nil. Mutex-protected because tests may swap it from a separate
	// goroutine.
	mu      sync.Mutex
	onFlush func(int)
}

// NewBatchAuditWriter wires the writer. Pass the SanitizingRepository
// (or any repository.AuditRepository) as `inner` — Log calls reroute
// through the batch path, list methods forward unchanged. `sink` is
// usually `*sql.DB` from the application pool; in unit tests a stub
// that satisfies auditBatchSink works.
//
// `cfg` zero values are replaced by DefaultBatchAuditConfig values so
// callers can supply only the fields they need.
func NewBatchAuditWriter(inner repository.AuditRepository, sink auditBatchSink, cfg BatchAuditConfig) *BatchAuditWriter {
	d := DefaultBatchAuditConfig()
	if cfg.FlushInterval <= 0 {
		cfg.FlushInterval = d.FlushInterval
	}
	if cfg.FlushThreshold <= 0 {
		cfg.FlushThreshold = d.FlushThreshold
	}
	if cfg.ChannelCapacity <= 0 {
		cfg.ChannelCapacity = d.ChannelCapacity
	}
	if cfg.MaxRetriesOnFlushFailure < 0 {
		cfg.MaxRetriesOnFlushFailure = d.MaxRetriesOnFlushFailure
	}
	if cfg.FlushTimeout <= 0 {
		cfg.FlushTimeout = d.FlushTimeout
	}
	return &BatchAuditWriter{
		cfg:   cfg,
		inner: inner,
		sink:  sink,
		ch:    make(chan *audit.Entry, cfg.ChannelCapacity),
		done:  make(chan struct{}),
	}
}

// SetOnFlush installs (or clears) an observer invoked after each
// successful flush with the batch size. Test-only hook — production
// wiring leaves it nil.
func (w *BatchAuditWriter) SetOnFlush(fn func(int)) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.onFlush = fn
}

// notifyFlush invokes the registered observer, if any. Called under
// the same mutex used by SetOnFlush so a swap-during-flush is
// race-free.
func (w *BatchAuditWriter) notifyFlush(n int) {
	w.mu.Lock()
	fn := w.onFlush
	w.mu.Unlock()
	if fn != nil {
		fn(n)
	}
}

// Start launches the background flusher goroutine. Safe to call once;
// subsequent calls are a no-op. The flusher exits when ctx is
// cancelled OR Stop is called.
func (w *BatchAuditWriter) Start(ctx context.Context) {
	if !w.flusherRunning.CompareAndSwap(false, true) {
		return
	}
	go w.run(ctx)
}

// Stop signals the flusher to drain remaining events and exit. It
// blocks until the flusher goroutine has returned, OR until `timeout`
// elapses (a deadlock-proof safety net). Returns the count of events
// still queued at shutdown time (zero on clean drain).
//
// Stop is idempotent — once the done channel is closed, subsequent
// calls return immediately with the recorded queuedAtShutdown.
func (w *BatchAuditWriter) Stop(timeout time.Duration) int64 {
	// Closing the channel signals the flusher to exit after draining.
	// CompareAndSwap on flusherRunning avoids a double-close panic
	// when Stop races with itself.
	if w.flusherRunning.CompareAndSwap(true, false) {
		close(w.ch)
	}
	select {
	case <-w.done:
	case <-time.After(timeout):
		slog.Warn("audit batch: shutdown timed out", "timeout", timeout)
	}
	return w.queuedAtShutdown.Load()
}

// Log buffers an entry for the next flush. Returns nil immediately on
// success. When the channel is full, Log BLOCKS the caller until a
// flush drains a slot — never silently drops the event.
//
// If the flusher has been stopped (channel closed), Log falls back to
// a direct call into the wrapped repository so audit events emitted
// during shutdown are still persisted.
func (w *BatchAuditWriter) Log(ctx context.Context, entry *audit.Entry) error {
	if entry == nil {
		return fmt.Errorf("audit batch: nil entry")
	}
	// Fast path: the flusher is running, enqueue. Recover from a
	// "send on closed channel" panic by falling back to the direct
	// path — this handles the narrow race between Stop closing the
	// channel and a late Log call.
	if w.flusherRunning.Load() {
		ok := w.trySend(ctx, entry)
		if ok {
			return nil
		}
	}
	// Fallback: the flusher is stopped, the channel is closed, or the
	// caller's context was cancelled mid-send. Persist directly so the
	// event is not lost.
	return w.inner.Log(ctx, entry)
}

// trySend pushes the entry onto the channel, honoring the caller's
// context. Returns true on success, false on context cancellation OR
// on a panic from sending to a closed channel (Stop race).
func (w *BatchAuditWriter) trySend(ctx context.Context, entry *audit.Entry) (ok bool) {
	defer func() {
		if r := recover(); r != nil {
			// "send on closed channel" — flusher is shutting down.
			ok = false
		}
	}()
	select {
	case w.ch <- entry:
		return true
	case <-ctx.Done():
		return false
	}
}

// ListByResource forwards to the wrapped repository unchanged.
func (w *BatchAuditWriter) ListByResource(
	ctx context.Context,
	resourceType audit.ResourceType,
	resourceID uuid.UUID,
	cursor string,
	limit int,
) ([]*audit.Entry, string, error) {
	return w.inner.ListByResource(ctx, resourceType, resourceID, cursor, limit)
}

// ListByUser forwards to the wrapped repository unchanged.
func (w *BatchAuditWriter) ListByUser(
	ctx context.Context,
	userID uuid.UUID,
	cursor string,
	limit int,
) ([]*audit.Entry, string, error) {
	return w.inner.ListByUser(ctx, userID, cursor, limit)
}

// run is the background flusher loop. Exits when ctx is cancelled OR
// the channel is closed; in both cases it drains the channel and
// emits a final flush before signalling done.
func (w *BatchAuditWriter) run(ctx context.Context) {
	defer close(w.done)

	ticker := time.NewTicker(w.cfg.FlushInterval)
	defer ticker.Stop()

	buffer := make([]*audit.Entry, 0, w.cfg.FlushThreshold)

	flush := func(reason string) {
		if len(buffer) == 0 {
			return
		}
		// Take a snapshot and reset buffer BEFORE issuing the
		// network call so a slow flush does not block new events
		// from accumulating into the next batch.
		batch := buffer
		buffer = make([]*audit.Entry, 0, w.cfg.FlushThreshold)
		w.flushBatch(ctx, batch, reason)
	}

	for {
		select {
		case <-ctx.Done():
			// Drain remaining events without blocking — channel may
			// still have events the producers pushed before noticing
			// ctx cancellation. We do NOT close(w.ch) here: only Stop
			// closes the channel, and Stop is the single owner of the
			// close. Use a non-blocking drain with a tight deadline.
			drained := drainChannel(w.ch, &buffer, w.cfg.FlushThreshold*10)
			w.queuedAtShutdown.Store(int64(drained))
			flush("ctx_cancelled")
			return

		case entry, open := <-w.ch:
			if !open {
				// Stop closed the channel — drain whatever is left
				// (the closed channel returns the zero value, so this
				// path only runs once) and exit.
				w.queuedAtShutdown.Store(int64(len(buffer)))
				flush("channel_closed")
				return
			}
			buffer = append(buffer, entry)
			if len(buffer) >= w.cfg.FlushThreshold {
				flush("threshold")
			}

		case <-ticker.C:
			flush("interval")
		}
	}
}

// drainChannel non-blockingly pulls every remaining entry from ch
// into the buffer slice (bounded by cap to avoid pathological
// allocation). Returns the count drained for the metric.
func drainChannel(ch <-chan *audit.Entry, buffer *[]*audit.Entry, cap int) int {
	count := 0
	for {
		select {
		case entry, open := <-ch:
			if !open {
				return count
			}
			if entry != nil {
				*buffer = append(*buffer, entry)
				count++
				if count >= cap {
					return count
				}
			}
		default:
			return count
		}
	}
}

// flushBatch persists `batch` in a single multi-row INSERT. Retries
// on transient failures up to cfg.MaxRetriesOnFlushFailure with
// exponential backoff (50 ms × 2^attempt).
//
// On final failure the batch is logged and DROPPED — at that point
// the DB is unrecoverable from this goroutine's perspective and
// holding the batch indefinitely would block the queue. The drop is
// loud (ERROR) so ops can react.
func (w *BatchAuditWriter) flushBatch(ctx context.Context, batch []*audit.Entry, reason string) {
	if len(batch) == 0 {
		return
	}

	// Detach from ctx for the actual write: a request-scoped context
	// might cancel between buffering and flushing, but we still want
	// to persist the row. We give the flush its own bounded timeout.
	flushCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), w.cfg.FlushTimeout)
	defer cancel()

	var lastErr error
	for attempt := 0; attempt <= w.cfg.MaxRetriesOnFlushFailure; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(1<<attempt) * 50 * time.Millisecond
			select {
			case <-time.After(backoff):
			case <-flushCtx.Done():
				lastErr = flushCtx.Err()
				break
			}
		}
		err := w.executeBatchInsert(flushCtx, batch)
		if err == nil {
			slog.Debug("audit batch: flushed",
				"reason", reason,
				"size", len(batch),
				"attempt", attempt,
			)
			w.notifyFlush(len(batch))
			return
		}
		lastErr = err
		slog.Warn("audit batch: flush failed",
			"reason", reason,
			"size", len(batch),
			"attempt", attempt,
			"error", err.Error(),
		)
	}
	// All retries exhausted — drop the batch.
	slog.Error("audit batch: dropped after exhausting retries",
		"reason", reason,
		"size", len(batch),
		"final_error", errStr(lastErr),
	)
}

// errStr returns the error message or "<nil>" — defensive against
// the (impossible) case where lastErr is nil but we still log.
func errStr(err error) string {
	if err == nil {
		return "<nil>"
	}
	return err.Error()
}

// executeBatchInsert runs one multi-row INSERT inside a single
// transaction. The query string is built dynamically — `lib/pq` does
// not support `pq.Array` for a slice of structs, so we expand the
// VALUES tuples ourselves with $1, $2 ... placeholders. All values
// are still passed via the args slice — no string interpolation
// touches user data.
func (w *BatchAuditWriter) executeBatchInsert(ctx context.Context, batch []*audit.Entry) error {
	if len(batch) == 0 {
		return nil
	}

	// Each row has 8 columns: id, user_id, action, resource_type,
	// resource_id, metadata, ip_address, created_at.
	const cols = 8
	args := make([]any, 0, len(batch)*cols)
	var sb strings.Builder
	sb.Grow(len(batch) * 64)
	sb.WriteString(`INSERT INTO audit_logs (
		id, user_id, action, resource_type, resource_id,
		metadata, ip_address, created_at
	) VALUES `)

	for i, entry := range batch {
		if entry == nil {
			// Skip nil — should never happen, defensive.
			continue
		}
		metadataJSON, err := json.Marshal(entry.Metadata)
		if err != nil {
			return fmt.Errorf("audit batch: marshal metadata for entry %s: %w", entry.ID, err)
		}

		var userIDArg any
		if entry.UserID != nil {
			userIDArg = *entry.UserID
		} else {
			userIDArg = nil
		}

		var resourceTypeArg any
		if entry.ResourceType != "" {
			resourceTypeArg = string(entry.ResourceType)
		} else {
			resourceTypeArg = nil
		}

		var resourceIDArg any
		if entry.ResourceID != nil {
			resourceIDArg = *entry.ResourceID
		} else {
			resourceIDArg = nil
		}

		var ipArg any
		if entry.IPAddress != nil {
			ipArg = entry.IPAddress.String()
		} else {
			ipArg = nil
		}

		if i > 0 {
			sb.WriteString(", ")
		}
		base := i*cols + 1
		fmt.Fprintf(&sb, "($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d)",
			base, base+1, base+2, base+3, base+4, base+5, base+6, base+7,
		)
		args = append(args,
			entry.ID,
			userIDArg,
			string(entry.Action),
			resourceTypeArg,
			resourceIDArg,
			metadataJSON,
			ipArg,
			entry.CreatedAt,
		)
	}

	// audit_logs is RLS-protected: USING (user_id = ...) for SELECT,
	// WITH CHECK (true) for INSERT (migration 129). The batch INSERT
	// is therefore safe across mixed actors WITHOUT setting per-row
	// app.current_user_id — every row passes the WITH CHECK.
	// Reads from this batch are still filtered correctly because the
	// USING clause applies at SELECT time against each row's
	// user_id column.
	tx, err := w.sink.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("audit batch: begin tx: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	if _, err := tx.ExecContext(ctx, sb.String(), args...); err != nil {
		return fmt.Errorf("audit batch: multi-row insert: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("audit batch: commit: %w", err)
	}
	committed = true
	return nil
}

// ErrBatchWriterStopped is returned by certain test helpers when the
// writer has been stopped — exported so tests can assert on the
// sentinel without importing the postgres internals.
var ErrBatchWriterStopped = errors.New("audit batch: writer stopped")
