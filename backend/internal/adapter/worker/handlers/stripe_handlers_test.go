package handlers_test

// P8 — Stripe webhook async dispatch worker handler. The handler must
// decode the persisted StripeWebhookEvent payload from a pending_event
// and forward it to the dispatcher. Failures here surface as worker
// retries (the row is already claimed in 'processing'); a successful
// dispatch lets the worker mark the row done.

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/adapter/worker/handlers"
	"marketplace-backend/internal/domain/pendingevent"
	portservice "marketplace-backend/internal/port/service"
)

// stubDispatcher captures dispatch calls so the test can assert the
// handler decoded and forwarded the payload faithfully.
type stubDispatcher struct {
	calls   []*portservice.StripeWebhookEvent
	nextErr error
}

func (s *stubDispatcher) Dispatch(_ context.Context, event *portservice.StripeWebhookEvent) error {
	s.calls = append(s.calls, event)
	return s.nextErr
}

// buildStripePendingEvent is a test helper: marshals the projected
// event and wraps it in a pending_event row identical in shape to what
// the webhook HTTP handler enqueues.
func buildStripePendingEvent(t *testing.T, e *portservice.StripeWebhookEvent) *pendingevent.PendingEvent {
	t.Helper()
	raw, err := json.Marshal(e)
	require.NoError(t, err)
	return &pendingevent.PendingEvent{
		ID:            uuid.New(),
		EventType:     pendingevent.TypeStripeWebhook,
		Payload:       raw,
		StripeEventID: e.EventID,
	}
}

// TestStripeWebhookHandler_RoundTripsEventToDispatcher asserts the
// happy path: decode, forward, propagate nil.
func TestStripeWebhookHandler_RoundTripsEventToDispatcher(t *testing.T) {
	disp := &stubDispatcher{}
	h := handlers.NewStripeWebhookHandler(disp)

	original := &portservice.StripeWebhookEvent{
		EventID: "evt_async_001",
		Type:    "invoice.paid",
		InvoicePaid:            true,
		InvoiceID:              "in_001",
		InvoiceAmountPaidCents: 4900,
	}
	row := buildStripePendingEvent(t, original)

	require.NoError(t, h.Handle(context.Background(), row))
	require.Len(t, disp.calls, 1)
	assert.Equal(t, original.EventID, disp.calls[0].EventID)
	assert.Equal(t, original.Type, disp.calls[0].Type)
	assert.Equal(t, original.InvoiceID, disp.calls[0].InvoiceID)
	assert.Equal(t, original.InvoiceAmountPaidCents, disp.calls[0].InvoiceAmountPaidCents)
}

// TestStripeWebhookHandler_PropagatesDispatcherError makes sure a
// downstream dispatch error bubbles up so the worker marks the row
// failed and schedules a backoff retry.
func TestStripeWebhookHandler_PropagatesDispatcherError(t *testing.T) {
	disp := &stubDispatcher{nextErr: errors.New("boom")}
	h := handlers.NewStripeWebhookHandler(disp)

	row := buildStripePendingEvent(t, &portservice.StripeWebhookEvent{
		EventID: "evt_async_002",
		Type:    "customer.subscription.updated",
	})

	err := h.Handle(context.Background(), row)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "boom")
}

// TestStripeWebhookHandler_DecodeErrorReturnsWrappedError covers a
// corrupted JSON payload — the handler must surface a decode error
// (the worker will retry up to MaxAttempts and then park the row in
// the failed bucket for ops to triage).
func TestStripeWebhookHandler_DecodeErrorReturnsWrappedError(t *testing.T) {
	disp := &stubDispatcher{}
	h := handlers.NewStripeWebhookHandler(disp)

	bad := &pendingevent.PendingEvent{
		ID:            uuid.New(),
		EventType:     pendingevent.TypeStripeWebhook,
		Payload:       []byte("not-json"),
		StripeEventID: "evt_async_003",
	}

	err := h.Handle(context.Background(), bad)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decode")
	assert.Empty(t, disp.calls, "dispatcher must NOT be called on decode failure")
}

// TestStripeWebhookHandler_NilDispatcherFailsLoud guards against a
// wiring mistake (Register called with a nil dispatcher). Returning
// an error makes the row back off rather than panicking the worker.
func TestStripeWebhookHandler_NilDispatcherFailsLoud(t *testing.T) {
	h := handlers.NewStripeWebhookHandler(nil)
	row := buildStripePendingEvent(t, &portservice.StripeWebhookEvent{
		EventID: "evt_async_004",
		Type:    "payment_intent.succeeded",
	})
	err := h.Handle(context.Background(), row)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "dispatcher")
}
