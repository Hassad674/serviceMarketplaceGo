// Package handlers — Stripe webhook async dispatch.
//
// The webhook HTTP entry point (handler.StripeHandler.HandleWebhook)
// verifies the signature, marshals the projected StripeWebhookEvent,
// and enqueues it on pending_events with TypeStripeWebhook. This file
// hosts the worker-side handler that closes the loop: it decodes the
// event payload, calls into the per-event dispatcher, and lets the
// worker's MarkDone / MarkFailed lifecycle settle the row.
//
// Idempotency: every per-event handler downstream of Dispatch is
// already idempotent for its own state — they were designed that way
// for Stripe re-deliveries on the synchronous path. Re-dispatch on a
// worker retry is therefore safe:
//
//   - handlePaymentSucceeded → loadProposalForActor short-circuits
//     when the proposal is already active.
//   - handleSubscriptionCreated / Snapshot → app service is
//     UPSERT-shaped (RegisterFromCheckout, HandleSubscriptionSnapshot).
//   - handleInvoicePaid → invoicingSvc.IssueFromSubscription guards
//     on (StripeEventID, StripeInvoiceID); a duplicate invoice insert
//     fails on the unique constraint and the handler logs + returns
//     nil so the worker marks the event done without retrying.
//   - handleChargeRefunded → IssueCreditNote guards on StripeEventID.
//
// The pending_events partial unique index on stripe_event_id already
// dedupes Stripe re-deliveries before enqueue, so the worker only
// processes each event_id at most once per first-delivery — but if a
// worker process crashes mid-dispatch, the BUG-NEW-03 stale recovery
// will re-pop the row and re-run the handler, which is why the
// per-handler idempotency above matters.
package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"marketplace-backend/internal/domain/pendingevent"
	portservice "marketplace-backend/internal/port/service"
)

// StripeDispatcher is the narrow interface the worker handler depends
// on. The production binding is *handler.StripeHandler from the HTTP
// layer — the same object that owns the per-event helpers (the dispatch
// is identical, only the entry point changes). Defining an interface
// here keeps adapter/worker/handlers free of the upstream handler
// package import and lets tests stub the dispatcher with a one-liner.
type StripeDispatcher interface {
	Dispatch(ctx context.Context, event *portservice.StripeWebhookEvent) error
}

// StripeWebhookHandler is the worker-side EventHandler for the
// TypeStripeWebhook outbox row. Decodes the persisted projection back
// into a StripeWebhookEvent and hands it to the dispatcher.
type StripeWebhookHandler struct {
	dispatcher StripeDispatcher
}

// NewStripeWebhookHandler builds the worker handler against a
// dispatcher (concrete *handler.StripeHandler in production wiring).
// The worker registers it under TypeStripeWebhook in
// cmd/api/wire_pending_events.go.
func NewStripeWebhookHandler(dispatcher StripeDispatcher) *StripeWebhookHandler {
	return &StripeWebhookHandler{dispatcher: dispatcher}
}

// Handle decodes the queue row's JSON payload into the projected
// StripeWebhookEvent and invokes the dispatcher. A decode error is
// surfaced (the row will be retried with backoff) — the payload was
// produced by our own marshaller in HandleWebhook so a decode failure
// signals a corrupted row, not a transient issue, but retry-with-
// backoff is still the safest behaviour: it puts the row in the
// admin-visible failed bucket after MaxAttempts where ops can triage
// it manually.
//
// Emits structured logs with `event_id`, `event_type`, and
// `process_ms` so the dashboard can correlate enqueue→dispatch
// latency end-to-end (the enqueue side logs `enqueue_ms` from the
// HTTP handler).
func (h *StripeWebhookHandler) Handle(ctx context.Context, event *pendingevent.PendingEvent) error {
	if h.dispatcher == nil {
		// Defensive: a wiring mistake (queue registered, dispatcher
		// nil) would otherwise panic. Fail loud so the worker logs
		// the missing wiring and the row backs off rather than
		// crashing the process.
		return fmt.Errorf("stripe webhook worker: dispatcher is not wired")
	}
	var stripeEvent portservice.StripeWebhookEvent
	if err := json.Unmarshal(event.Payload, &stripeEvent); err != nil {
		return fmt.Errorf("decode stripe webhook payload: %w", err)
	}

	start := time.Now()
	dispatchErr := h.dispatcher.Dispatch(ctx, &stripeEvent)
	processMS := time.Since(start).Milliseconds()

	if dispatchErr != nil {
		slog.Warn("stripe webhook worker: dispatch failed — row will retry with backoff",
			"event_id", stripeEvent.EventID,
			"event_type", stripeEvent.Type,
			"process_ms", processMS,
			"attempts", event.Attempts,
			"error", dispatchErr)
		return dispatchErr
	}
	slog.Info("stripe webhook worker: dispatch succeeded",
		"event_id", stripeEvent.EventID,
		"event_type", stripeEvent.Type,
		"process_ms", processMS,
		"attempts", event.Attempts)
	return nil
}
