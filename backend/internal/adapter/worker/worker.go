// Package worker is the background goroutine that drives the
// pending_events queue: it ticks at a configurable interval, pops
// due events with FOR UPDATE SKIP LOCKED, dispatches each one to
// its registered handler, and settles the result (done | failed +
// backoff) via the repository.
//
// The worker is safe to run on multiple instances concurrently —
// PopDue uses SKIP LOCKED so two workers never claim the same row.
//
// Lifecycle: cmd/api/main.go starts the worker via Run(ctx) in a
// goroutine. Cancelling the context triggers graceful shutdown — the
// current batch finishes processing, then the loop exits.
package worker

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"marketplace-backend/internal/domain/pendingevent"
	"marketplace-backend/internal/port/repository"
)

// EventHandler is the contract a registered handler must satisfy.
// Handlers MUST be idempotent: a transient failure may leave a row
// in processing status, which the next worker pass will retry. A
// handler that has visible side effects must either persist them
// inside its own transaction (so they roll back on error) or guard
// them with a deduplication mechanism (e.g. checking the milestone
// status before acting).
type EventHandler interface {
	Handle(ctx context.Context, event *pendingevent.PendingEvent) error
}

// HandlerFunc is a convenience adapter so callers can register a
// plain function as an EventHandler without defining a struct.
type HandlerFunc func(ctx context.Context, event *pendingevent.PendingEvent) error

// Handle implements EventHandler.
func (f HandlerFunc) Handle(ctx context.Context, event *pendingevent.PendingEvent) error {
	return f(ctx, event)
}

// Worker is the long-running background goroutine.
//
// Concurrency:
//   - One Worker instance per process is the common case. Multiple
//     instances on the same DB are safe — SKIP LOCKED partitions the
//     event queue across them automatically.
//   - The internal dispatch loop is single-threaded inside one Worker;
//     batch parallelism comes from running additional Worker instances
//     OR from increasing BatchSize so each tick processes more events
//     in sequence.
type Worker struct {
	repo         repository.PendingEventRepository
	handlers     map[pendingevent.EventType]EventHandler
	tickInterval time.Duration
	batchSize    int
	logger       *slog.Logger
}

// Config carries the worker tuning knobs. All fields have sensible
// defaults if zero.
type Config struct {
	// TickInterval is how often the worker scans for due events.
	// Default: 30 seconds. Lower values give faster reaction time
	// at the cost of more idle DB queries.
	TickInterval time.Duration
	// BatchSize caps how many events PopDue claims per tick.
	// Default: 10. The DB index keeps the scan cheap even with
	// large batches.
	BatchSize int
	// Logger is the structured logger used for per-event diagnostics.
	// Defaults to slog.Default().
	Logger *slog.Logger
}

// New builds a Worker against a repository and an optional config.
// Handlers are registered separately via Register so each phase can
// add its own type without touching this constructor.
func New(repo repository.PendingEventRepository, cfg Config) *Worker {
	tick := cfg.TickInterval
	if tick <= 0 {
		tick = 30 * time.Second
	}
	batch := cfg.BatchSize
	if batch <= 0 {
		batch = 10
	}
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}
	return &Worker{
		repo:         repo,
		handlers:     make(map[pendingevent.EventType]EventHandler),
		tickInterval: tick,
		batchSize:    batch,
		logger:       logger,
	}
}

// Register attaches a handler to an event type. Calling Register
// twice for the same event type overwrites the previous handler —
// the worker logs a warning so the duplicate registration shows up
// in startup logs.
func (w *Worker) Register(eventType pendingevent.EventType, handler EventHandler) {
	if !eventType.IsValid() {
		w.logger.Warn("worker: refusing to register handler for invalid event type", "event_type", eventType)
		return
	}
	if _, exists := w.handlers[eventType]; exists {
		w.logger.Warn("worker: duplicate handler registration overwriting previous", "event_type", eventType)
	}
	w.handlers[eventType] = handler
}

// Run starts the worker loop. Blocks until ctx is cancelled, then
// returns nil after the in-flight batch finishes processing. Safe to
// call from a goroutine in main.go:
//
//	go func() {
//	    if err := worker.Run(ctx); err != nil {
//	        slog.Error("worker exited with error", "err", err)
//	    }
//	}()
func (w *Worker) Run(ctx context.Context) error {
	w.logger.Info("worker: starting",
		"tick_interval", w.tickInterval,
		"batch_size", w.batchSize,
		"handler_count", len(w.handlers))

	ticker := time.NewTicker(w.tickInterval)
	defer ticker.Stop()

	// Run an immediate tick on startup so events that were due
	// while the process was down get processed without waiting
	// the first tick interval.
	w.tick(ctx)

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("worker: shutdown requested, exiting")
			return nil
		case <-ticker.C:
			w.tick(ctx)
		}
	}
}

// tick runs one iteration: pop a batch of due events, dispatch each
// to its handler, settle the result. Errors at the per-event level
// are logged and the row is marked failed (with backoff); errors at
// the pop level are logged and the tick is aborted — the next tick
// will retry.
func (w *Worker) tick(ctx context.Context) {
	events, err := w.repo.PopDue(ctx, w.batchSize)
	if err != nil {
		w.logger.Error("worker: pop due events failed", "error", err)
		return
	}
	if len(events) == 0 {
		return
	}

	w.logger.Debug("worker: processing batch", "count", len(events))
	for _, event := range events {
		w.processOne(ctx, event)
	}
}

// processOne dispatches a single event to its registered handler
// and settles the row according to the result. Unknown event types
// are marked failed with a clear error so the admin can see the
// missing handler in the dashboard.
func (w *Worker) processOne(ctx context.Context, event *pendingevent.PendingEvent) {
	handler, ok := w.handlers[event.EventType]
	if !ok {
		w.logger.Error("worker: no handler registered for event type",
			"event_id", event.ID,
			"event_type", event.EventType)
		w.markFailed(ctx, event, fmt.Errorf("no handler registered for event type %q", event.EventType))
		return
	}

	w.logger.Debug("worker: dispatching event",
		"event_id", event.ID,
		"event_type", event.EventType,
		"attempts", event.Attempts)

	// Defensive: a handler that panics would bring down the whole
	// worker without this recover. We log the panic, mark the event
	// failed, and continue with the next one in the batch.
	var handlerErr error
	func() {
		defer func() {
			if r := recover(); r != nil {
				handlerErr = fmt.Errorf("handler panicked: %v", r)
				w.logger.Error("worker: handler panic recovered",
					"event_id", event.ID,
					"event_type", event.EventType,
					"panic", r)
			}
		}()
		handlerErr = handler.Handle(ctx, event)
	}()

	if handlerErr != nil {
		w.markFailed(ctx, event, handlerErr)
		return
	}
	w.markDone(ctx, event)
}

// markDone is a thin wrapper that applies the domain MarkDone
// transition and persists it. Errors here are logged but not fatal —
// the next pop won't pick the row up because PopDue filters on
// status IN ('pending', 'failed').
func (w *Worker) markDone(ctx context.Context, event *pendingevent.PendingEvent) {
	if err := event.MarkDone(); err != nil {
		w.logger.Error("worker: domain MarkDone failed",
			"event_id", event.ID, "error", err)
		return
	}
	if err := w.repo.MarkDone(ctx, event); err != nil {
		w.logger.Error("worker: persist MarkDone failed",
			"event_id", event.ID, "error", err)
	}
}

// markFailed is a thin wrapper that applies the domain MarkFailed
// transition (which schedules the backoff) and persists it. The
// next pop will pick the row up after the backoff delay elapses,
// up to MaxAttempts times.
func (w *Worker) markFailed(ctx context.Context, event *pendingevent.PendingEvent, handlerErr error) {
	if err := event.MarkFailed(handlerErr); err != nil {
		w.logger.Error("worker: domain MarkFailed failed",
			"event_id", event.ID, "error", err)
		return
	}
	if err := w.repo.MarkFailed(ctx, event); err != nil {
		w.logger.Error("worker: persist MarkFailed failed",
			"event_id", event.ID, "error", err)
	}
	w.logger.Warn("worker: event handler failed",
		"event_id", event.ID,
		"event_type", event.EventType,
		"attempts", event.Attempts,
		"max_attempts", pendingevent.MaxAttempts,
		"next_fires_at", event.FiresAt,
		"error", handlerErr)
}

// Compile-time assertion: HandlerFunc satisfies EventHandler.
var _ EventHandler = HandlerFunc(nil)

// shutdownGracePeriod is the maximum time the worker will wait for
// in-flight handlers to finish after a shutdown signal. Used by
// callers that want to bound their own shutdown sequence.
const shutdownGracePeriod = 30 * time.Second

// SuggestedShutdownGracePeriod returns the recommended timeout for
// callers that wrap the worker shutdown in their own context. Made
// public so the API server can use it when constructing its
// graceful-shutdown context.
func SuggestedShutdownGracePeriod() time.Duration { return shutdownGracePeriod }

// dispatcherMutex is a no-op placeholder kept for API compatibility
// with future multi-handler-per-event-type extensions. The single
// in-process Worker is currently single-threaded.
var dispatcherMutex sync.Mutex //nolint:unused
