package worker

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/pendingevent"
)

// mockRepo is a hand-written stub of repository.PendingEventRepository
// using the same field-functions pattern as the rest of the codebase.
type mockRepo struct {
	mu       sync.Mutex
	scheduled []*pendingevent.PendingEvent
	popQueue  [][]*pendingevent.PendingEvent
	doneIDs   []uuid.UUID
	failedIDs []uuid.UUID
	popErr    error
}

func (m *mockRepo) Schedule(_ context.Context, e *pendingevent.PendingEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.scheduled = append(m.scheduled, e)
	return nil
}

// ScheduleTx mirrors Schedule for the worker package's mock — the
// worker only consumes Pop/Mark, never schedules, so this stub is
// purely interface-satisfaction. (BUG-05 outbox path.)
func (m *mockRepo) ScheduleTx(_ context.Context, _ *sql.Tx, e *pendingevent.PendingEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.scheduled = append(m.scheduled, e)
	return nil
}

// ScheduleStripe mirrors Schedule for interface-satisfaction; the
// worker package's tests do not exercise the Stripe-webhook path
// directly. P8 added this method to PendingEventRepository.
func (m *mockRepo) ScheduleStripe(_ context.Context, e *pendingevent.PendingEvent) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.scheduled = append(m.scheduled, e)
	return true, nil
}

func (m *mockRepo) PopDue(_ context.Context, _ int) ([]*pendingevent.PendingEvent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.popErr != nil {
		return nil, m.popErr
	}
	if len(m.popQueue) == 0 {
		return nil, nil
	}
	batch := m.popQueue[0]
	m.popQueue = m.popQueue[1:]
	// Mimic PopDue: events come back in processing status.
	for _, e := range batch {
		e.Status = pendingevent.StatusProcessing
		e.Attempts++
	}
	return batch, nil
}

func (m *mockRepo) MarkDone(_ context.Context, e *pendingevent.PendingEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.doneIDs = append(m.doneIDs, e.ID)
	return nil
}

func (m *mockRepo) MarkFailed(_ context.Context, e *pendingevent.PendingEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.failedIDs = append(m.failedIDs, e.ID)
	return nil
}

func (m *mockRepo) GetByID(_ context.Context, _ uuid.UUID) (*pendingevent.PendingEvent, error) {
	return nil, pendingevent.ErrEventNotFound
}

// helper builds a fresh in-memory pending event in processing status
// (mimicking what PopDue returns).
func newProcessingEvent(t *testing.T, eventType pendingevent.EventType) *pendingevent.PendingEvent {
	t.Helper()
	payload, _ := json.Marshal(map[string]string{"k": "v"})
	e, err := pendingevent.NewPendingEvent(pendingevent.NewPendingEventInput{
		EventType: eventType,
		Payload:   payload,
		FiresAt:   time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}
	return e
}

func TestWorker_ProcessOne_Success(t *testing.T) {
	repo := &mockRepo{}
	w := New(repo, Config{})

	var called atomic.Int32
	w.Register(pendingevent.TypeMilestoneAutoApprove, HandlerFunc(func(_ context.Context, _ *pendingevent.PendingEvent) error {
		called.Add(1)
		return nil
	}))

	e := newProcessingEvent(t, pendingevent.TypeMilestoneAutoApprove)
	repo.popQueue = [][]*pendingevent.PendingEvent{{e}}

	w.tick(context.Background())

	if called.Load() != 1 {
		t.Errorf("handler called %d times, want 1", called.Load())
	}
	if len(repo.doneIDs) != 1 || repo.doneIDs[0] != e.ID {
		t.Errorf("MarkDone calls = %+v, want [%s]", repo.doneIDs, e.ID)
	}
	if len(repo.failedIDs) != 0 {
		t.Errorf("expected no failures, got %d", len(repo.failedIDs))
	}
}

func TestWorker_ProcessOne_HandlerError_MarksFailed(t *testing.T) {
	repo := &mockRepo{}
	w := New(repo, Config{})

	w.Register(pendingevent.TypeMilestoneAutoApprove, HandlerFunc(func(_ context.Context, _ *pendingevent.PendingEvent) error {
		return errors.New("boom")
	}))

	e := newProcessingEvent(t, pendingevent.TypeMilestoneAutoApprove)
	repo.popQueue = [][]*pendingevent.PendingEvent{{e}}

	w.tick(context.Background())

	if len(repo.failedIDs) != 1 {
		t.Errorf("expected 1 failure, got %d", len(repo.failedIDs))
	}
	if len(repo.doneIDs) != 0 {
		t.Errorf("expected no done, got %d", len(repo.doneIDs))
	}
	// Domain transition should have set last_error and status.
	if e.LastError == nil || *e.LastError != "boom" {
		t.Errorf("LastError = %v, want 'boom'", e.LastError)
	}
}

func TestWorker_ProcessOne_HandlerPanic_RecoveredAndMarkedFailed(t *testing.T) {
	repo := &mockRepo{}
	w := New(repo, Config{})

	w.Register(pendingevent.TypeStripeTransfer, HandlerFunc(func(_ context.Context, _ *pendingevent.PendingEvent) error {
		var s *string
		_ = *s // deliberate nil deref to trigger panic
		return nil
	}))

	e := newProcessingEvent(t, pendingevent.TypeStripeTransfer)
	repo.popQueue = [][]*pendingevent.PendingEvent{{e}}

	// The worker MUST recover from the panic and mark the event
	// failed without crashing the whole tick.
	w.tick(context.Background())

	if len(repo.failedIDs) != 1 {
		t.Errorf("expected 1 failure after panic, got %d", len(repo.failedIDs))
	}
	if e.LastError == nil {
		t.Error("LastError should be set after panic")
	}
}

func TestWorker_ProcessOne_NoHandler_MarksFailed(t *testing.T) {
	repo := &mockRepo{}
	w := New(repo, Config{})

	// No handler registered for the event type.
	e := newProcessingEvent(t, pendingevent.TypeProposalAutoClose)
	repo.popQueue = [][]*pendingevent.PendingEvent{{e}}

	w.tick(context.Background())

	if len(repo.failedIDs) != 1 {
		t.Errorf("expected 1 failure, got %d", len(repo.failedIDs))
	}
	if e.LastError == nil {
		t.Error("LastError should describe the missing handler")
	}
}

func TestWorker_Tick_BatchProcessing(t *testing.T) {
	repo := &mockRepo{}
	w := New(repo, Config{BatchSize: 5})

	var processed atomic.Int32
	w.Register(pendingevent.TypeMilestoneFundReminder, HandlerFunc(func(_ context.Context, _ *pendingevent.PendingEvent) error {
		processed.Add(1)
		return nil
	}))

	// Queue 5 events for one tick.
	var batch []*pendingevent.PendingEvent
	for i := 0; i < 5; i++ {
		batch = append(batch, newProcessingEvent(t, pendingevent.TypeMilestoneFundReminder))
	}
	repo.popQueue = [][]*pendingevent.PendingEvent{batch}

	w.tick(context.Background())

	if processed.Load() != 5 {
		t.Errorf("processed = %d, want 5", processed.Load())
	}
	if len(repo.doneIDs) != 5 {
		t.Errorf("done count = %d, want 5", len(repo.doneIDs))
	}
}

func TestWorker_Run_StopsOnContextCancel(t *testing.T) {
	repo := &mockRepo{}
	// Use a fast tick so the test completes quickly.
	w := New(repo, Config{TickInterval: 10 * time.Millisecond})

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- w.Run(ctx) }()

	// Let the worker tick a few times.
	time.Sleep(30 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Run returned error: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("worker did not exit within 1s of cancel")
	}
}

func TestWorker_Tick_PopErrorIsNonFatal(t *testing.T) {
	repo := &mockRepo{popErr: errors.New("db unreachable")}
	w := New(repo, Config{})

	// A pop error should be logged and the tick should return
	// without panicking — the next tick will retry.
	w.tick(context.Background())

	if len(repo.doneIDs) != 0 || len(repo.failedIDs) != 0 {
		t.Error("expected no settle calls after pop error")
	}
}

func TestWorker_Register_InvalidEventType(t *testing.T) {
	repo := &mockRepo{}
	w := New(repo, Config{})
	w.Register("bogus", HandlerFunc(func(_ context.Context, _ *pendingevent.PendingEvent) error {
		return nil
	}))
	if _, ok := w.handlers["bogus"]; ok {
		t.Error("invalid event type should not be registered")
	}
}
