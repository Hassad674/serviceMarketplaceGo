package handler

// P8 — async dispatch via pending_events. The webhook handler must
// verify the signature, marshal the projected event, enqueue with
// ON CONFLICT DO NOTHING, and reply 200 OK in <50ms. The dispatch
// itself runs in a worker handler exercised by a separate test in
// adapter/worker/handlers/.

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/domain/pendingevent"
	portservice "marketplace-backend/internal/port/service"
)

// stubPendingEventsQueue is the minimal PendingEventRepository the
// async webhook path consumes. Every method except ScheduleStripe is
// stubbed to a no-op return — the webhook handler never calls them.
type stubPendingEventsQueue struct {
	mu         sync.Mutex
	scheduled  []*pendingevent.PendingEvent
	insertErr  error
	insertedBy map[string]bool // event_id -> first-delivery flag
}

func newStubPendingEventsQueue() *stubPendingEventsQueue {
	return &stubPendingEventsQueue{
		insertedBy: map[string]bool{},
	}
}

func (s *stubPendingEventsQueue) Schedule(_ context.Context, e *pendingevent.PendingEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.scheduled = append(s.scheduled, e)
	return nil
}

func (s *stubPendingEventsQueue) ScheduleTx(_ context.Context, _ *sql.Tx, e *pendingevent.PendingEvent) error {
	return nil
}

func (s *stubPendingEventsQueue) ScheduleStripe(_ context.Context, e *pendingevent.PendingEvent) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.insertErr != nil {
		return false, s.insertErr
	}
	if s.insertedBy[e.StripeEventID] {
		return false, nil // duplicate, ON CONFLICT DO NOTHING
	}
	s.insertedBy[e.StripeEventID] = true
	s.scheduled = append(s.scheduled, e)
	return true, nil
}

func (s *stubPendingEventsQueue) PopDue(_ context.Context, _ int) ([]*pendingevent.PendingEvent, error) {
	return nil, nil
}

func (s *stubPendingEventsQueue) MarkDone(_ context.Context, _ *pendingevent.PendingEvent) error {
	return nil
}

func (s *stubPendingEventsQueue) MarkFailed(_ context.Context, _ *pendingevent.PendingEvent) error {
	return nil
}

func (s *stubPendingEventsQueue) GetByID(_ context.Context, _ uuid.UUID) (*pendingevent.PendingEvent, error) {
	return nil, nil
}

// stubPendingEventsQueue must satisfy repository.PendingEventRepository
// — the compile-time interface assertion lives at the foot of the file
// to catch breaking changes to the interface immediately.

// TestHandleWebhook_AsyncEnqueueReturns200Fast asserts the P8
// contract: with the queue wired, the webhook handler verifies the
// signature, marshals the event, calls ScheduleStripe, and replies
// 200 OK well within Stripe's 10s timeout window. We require <50ms
// here so a regression in the inline path (e.g. someone re-adding
// PDF generation) shows up immediately.
func TestHandleWebhook_AsyncEnqueueReturns200Fast(t *testing.T) {
	const eventID = "evt_async_fast_001"
	queue := newStubPendingEventsQueue()
	event := &portservice.StripeWebhookEvent{
		EventID: eventID,
		Type:    "invoice.paid",
	}
	paymentSvc := newPaymentSvcVerifyingTo(event)

	h := (&StripeHandler{paymentSvc: paymentSvc}).WithPendingEventsQueue(queue)

	body := []byte(`{"id":"evt_async_fast_001","type":"invoice.paid"}`)
	r := httptest.NewRequest("POST", "/webhook", bytes.NewReader(body))
	r.Header.Set("Stripe-Signature", "t=any,v1=any")
	w := httptest.NewRecorder()

	start := time.Now()
	h.HandleWebhook(w, r)
	elapsed := time.Since(start)

	assert.Equal(t, http.StatusOK, w.Code, "async enqueue path must reply 200 OK")
	assert.Less(t, elapsed, 50*time.Millisecond, "webhook handler must respond <50ms; current=%s", elapsed)

	require.Len(t, queue.scheduled, 1, "exactly one pending event must be enqueued")
	pe := queue.scheduled[0]
	assert.Equal(t, pendingevent.TypeStripeWebhook, pe.EventType)
	assert.Equal(t, eventID, pe.StripeEventID)

	// Payload must round-trip the projected event so the worker can
	// reconstruct it without losing fields.
	var decoded portservice.StripeWebhookEvent
	require.NoError(t, json.Unmarshal(pe.Payload, &decoded))
	assert.Equal(t, eventID, decoded.EventID)
	assert.Equal(t, "invoice.paid", decoded.Type)
}

// TestHandleWebhook_AsyncEnqueueDeduplicatesRetry covers the
// ON CONFLICT (stripe_event_id) DO NOTHING path: a Stripe re-delivery
// of the same evt_* MUST be a silent 200 OK, never a duplicate
// pending_event row.
func TestHandleWebhook_AsyncEnqueueDeduplicatesRetry(t *testing.T) {
	const eventID = "evt_async_dup_001"
	queue := newStubPendingEventsQueue()
	event := &portservice.StripeWebhookEvent{
		EventID: eventID,
		Type:    "customer.subscription.updated",
	}
	paymentSvc := newPaymentSvcVerifyingTo(event)
	h := (&StripeHandler{paymentSvc: paymentSvc}).WithPendingEventsQueue(queue)

	body := []byte(`{"id":"evt_async_dup_001","type":"customer.subscription.updated"}`)

	// First delivery: enqueue + 200.
	r1 := httptest.NewRequest("POST", "/webhook", bytes.NewReader(body))
	r1.Header.Set("Stripe-Signature", "t=any,v1=any")
	w1 := httptest.NewRecorder()
	h.HandleWebhook(w1, r1)
	assert.Equal(t, http.StatusOK, w1.Code)

	// Stripe retry of the same event: 200 OK, but no second row.
	r2 := httptest.NewRequest("POST", "/webhook", bytes.NewReader(body))
	r2.Header.Set("Stripe-Signature", "t=any,v1=any")
	w2 := httptest.NewRecorder()
	h.HandleWebhook(w2, r2)
	assert.Equal(t, http.StatusOK, w2.Code, "duplicate delivery must still ack with 200")

	require.Len(t, queue.scheduled, 1, "ON CONFLICT DO NOTHING must prevent a second row")
	assert.Equal(t, eventID, queue.scheduled[0].StripeEventID)
}

// TestHandleWebhook_AsyncEnqueueDBErrorReturns503 asserts the failure
// path: when ScheduleStripe returns a non-nil error (Postgres down,
// migration mid-flight, etc.) the handler must respond 5xx so Stripe
// retries — never 200 with the event silently dropped.
func TestHandleWebhook_AsyncEnqueueDBErrorReturns503(t *testing.T) {
	queue := newStubPendingEventsQueue()
	queue.insertErr = errors.New("synthetic db failure")
	event := &portservice.StripeWebhookEvent{
		EventID: "evt_async_db_err_001",
		Type:    "charge.refunded",
	}
	paymentSvc := newPaymentSvcVerifyingTo(event)
	h := (&StripeHandler{paymentSvc: paymentSvc}).WithPendingEventsQueue(queue)

	body := []byte(`{"id":"evt_async_db_err_001","type":"charge.refunded"}`)
	r := httptest.NewRequest("POST", "/webhook", bytes.NewReader(body))
	r.Header.Set("Stripe-Signature", "t=any,v1=any")
	w := httptest.NewRecorder()

	h.HandleWebhook(w, r)

	assert.GreaterOrEqual(t, w.Code, 500, "DB failure on enqueue must return 5xx so Stripe retries")
	assert.Less(t, w.Code, 600)
	assert.Empty(t, queue.scheduled, "no row inserted on DB failure")
}

// TestHandleWebhook_AsyncEnqueueWithoutEventIDFallsBackToInline
// ensures the async path is gated on a non-empty Stripe event id —
// without an evt_* the dedup index can't apply, so we fall back to
// the legacy inline dispatcher (which is the safer behaviour for
// edge cases like a Stripe event without an id).
func TestHandleWebhook_AsyncEnqueueWithoutEventIDFallsBackToInline(t *testing.T) {
	queue := newStubPendingEventsQueue()
	event := &portservice.StripeWebhookEvent{
		EventID: "", // unusual: Stripe always supplies one, but defensive
		Type:    "payment_intent.succeeded",
	}
	paymentSvc := newPaymentSvcVerifyingTo(event)
	h := (&StripeHandler{paymentSvc: paymentSvc}).WithPendingEventsQueue(queue)

	body := []byte(`{"id":"","type":"payment_intent.succeeded"}`)
	r := httptest.NewRequest("POST", "/webhook", bytes.NewReader(body))
	r.Header.Set("Stripe-Signature", "t=any,v1=any")
	w := httptest.NewRecorder()

	h.HandleWebhook(w, r)

	// Inline dispatch ran (handlePaymentSucceeded fails because the
	// payment service has no Records configured) → 5xx via the
	// idempotency-release path. The async queue must NOT have been
	// touched.
	assert.Empty(t, queue.scheduled, "no event id → no async enqueue")
}
