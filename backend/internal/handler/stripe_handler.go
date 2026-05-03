package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"

	embeddedapp "marketplace-backend/internal/app/embedded"
	invoicingapp "marketplace-backend/internal/app/invoicing"
	paymentapp "marketplace-backend/internal/app/payment"
	proposalapp "marketplace-backend/internal/app/proposal"
	subscriptionapp "marketplace-backend/internal/app/subscription"
	"marketplace-backend/internal/domain/pendingevent"
	"marketplace-backend/internal/handler/dto/response"
	"marketplace-backend/internal/port/repository"
	portservice "marketplace-backend/internal/port/service"
	res "marketplace-backend/pkg/response"
)

// errMissingOwnerMetadata is returned when a Stripe subscription event
// carries no resolvable owner. The canonical key is organization_id —
// user_id is only checked as a legacy fallback for subscriptions
// created before migration 119, while the metadata backfill script
// hasn't run on them yet. Once that script has been run in every env
// the user_id branch is dead code and can be dropped.
var errMissingOwnerMetadata = errors.New("stripe webhook: subscription metadata is missing organization_id (legacy user_id fallback also empty)")

// IdempotencyClaimer is the narrow interface the Stripe webhook handler
// consumes to dedupe replays. Implemented by the composite claimer in
// app/webhookidempotency, which combines a Redis fast-path with a
// durable Postgres source of truth (see BUG-10). Kept as a local
// interface so tests can stub it without pulling either backend.
//
// The eventType argument lets the durable adapter populate the
// `stripe_webhook_events.event_type` column without a second fetch —
// useful for ad-hoc analytics (which event types we replay most).
//
// Contract:
//   - (true, nil)  → first delivery, caller MUST process the event.
//   - (false, nil) → already processed, caller MUST skip.
//   - (_, err)     → both fast-path and durable layer failed, caller
//     MUST reply non-2xx so Stripe retries.
//
// BUG-NEW-06 — Release reverses a successful TryClaim so Stripe's next
// retry delivers the same event_id to a fresh dispatcher attempt. The
// dispatcher calls it ONLY when a downstream handler returned an error
// AFTER the claim succeeded — without it, the durable claim would be
// permanent and Stripe's retry would be silently deduped, leaving the
// state change un-applied forever.
type IdempotencyClaimer interface {
	TryClaim(ctx context.Context, eventID, eventType string) (bool, error)
	// Release reverses a prior successful TryClaim by deleting the
	// durable record AND clearing the fast-path cache so a subsequent
	// retry is treated as a fresh delivery. Returns nil even when the
	// row is absent — making it safe to call on the cleanup path even
	// if a parallel claim succeeded after our attempt.
	Release(ctx context.Context, eventID string) error
}

// SubscriptionCacheInvalidator is the narrow interface used after a
// subscription state change to flush the cached IsActive answer.
// Implemented by adapter/redis.CachedSubscriptionReader.
type SubscriptionCacheInvalidator interface {
	Invalidate(ctx context.Context, userID uuid.UUID) error
}

type StripeHandler struct {
	paymentSvc       *paymentapp.Service
	proposalSvc      *proposalapp.Service
	embeddedNotifier *embeddedapp.Notifier // optional, nil = classic path only
	publishableKey   string

	// Subscription wiring. All three are optional and only populated
	// when the subscription feature is wired in main.go.
	subscriptionSvc   *subscriptionapp.Service
	subscriptionCache SubscriptionCacheInvalidator
	idempotencyStore  IdempotencyClaimer

	// Invoicing wiring. Optional — when nil the invoice.paid hook is a
	// no-op. Removing the invoicing module from main.go disables the
	// hook cleanly.
	invoicingSvc *invoicingapp.Service

	// Async pipeline (P8). pendingEvents is the queue the webhook
	// HTTP handler enqueues onto after signature verification — the
	// dispatch chain (PDF generation, email sends, multi-row DB
	// writes) runs in the background worker so HandleWebhook can
	// reply 200 to Stripe in <50ms. nil disables the async path
	// entirely; HandleWebhook then dispatches inline (legacy
	// behaviour, kept so unit tests that don't wire a queue still
	// drive the dispatcher).
	pendingEvents repository.PendingEventRepository
}

func NewStripeHandler(paymentSvc *paymentapp.Service, proposalSvc *proposalapp.Service, publishableKey string) *StripeHandler {
	return &StripeHandler{
		paymentSvc:     paymentSvc,
		proposalSvc:    proposalSvc,
		publishableKey: publishableKey,
	}
}

// WithEmbeddedNotifier attaches an embedded notifier to the handler so
// account.* webhooks emit rich notifications via the shared notification
// service. Call once at wiring time.
func (h *StripeHandler) WithEmbeddedNotifier(n *embeddedapp.Notifier) *StripeHandler {
	h.embeddedNotifier = n
	return h
}

// WithInvoicing wires the invoicing app service into the webhook
// dispatcher. Once attached, every successful subscription invoice
// (Stripe invoice.paid) triggers an outbound customer-facing invoice.
// Pass svc=nil to leave the feature off — the hook degrades to a no-op
// and the rest of the dispatcher continues to work unchanged.
func (h *StripeHandler) WithInvoicing(svc *invoicingapp.Service) *StripeHandler {
	h.invoicingSvc = svc
	return h
}

// WithSubscription wires the subscription feature into the webhook
// dispatcher. Pass all three together — the cache invalidator and the
// idempotency store are required for correctness (stale cache after a
// webhook-driven change, double-processing on replays). Calling this
// method with svc=nil leaves the feature disabled.
func (h *StripeHandler) WithSubscription(svc *subscriptionapp.Service, cache SubscriptionCacheInvalidator, idempotency IdempotencyClaimer) *StripeHandler {
	h.subscriptionSvc = svc
	h.subscriptionCache = cache
	h.idempotencyStore = idempotency
	return h
}

// WithPendingEventsQueue wires the async-dispatch queue into the
// webhook HTTP handler. After this setter is called, HandleWebhook
// verifies the signature and immediately enqueues a TypeStripeWebhook
// row on pending_events — the registered worker handler in
// adapter/worker/handlers/stripe_handlers.go decodes the projected
// event from the queue and calls Dispatch in a background goroutine.
//
// Pass repo=nil to disable the async path: HandleWebhook then falls
// back to inline dispatch (the legacy synchronous behaviour, kept so
// existing unit tests that don't wire a queue still exercise the
// dispatcher directly).
func (h *StripeHandler) WithPendingEventsQueue(repo repository.PendingEventRepository) *StripeHandler {
	h.pendingEvents = repo
	return h
}

// GetConfig returns the Stripe publishable key for frontend initialization.
func (h *StripeHandler) GetConfig(w http.ResponseWriter, r *http.Request) {
	res.JSON(w, http.StatusOK, response.StripeConfigResponse{
		PublishableKey: h.publishableKey,
	})
}

// HandleWebhook processes Stripe webhook events.
// No auth middleware — Stripe sends directly, verified by signature.
//
// P8 — Async dispatch. The webhook handler used to dispatch every
// event inline, which on invoice.paid (PDF generation via headless
// chrome, ~2-5s) routinely came within 1-2 seconds of Stripe's 10s
// timeout. P8 moved dispatch to a pending_events worker:
//
//  1. Verify the Stripe signature (fast — pure crypto, no DB).
//  2. If the async queue is wired, marshal the projected event and
//     enqueue it via ScheduleStripe (ON CONFLICT DO NOTHING on the
//     evt_* id — Stripe re-deliveries are silent no-ops).
//  3. Reply 200 OK in <50ms regardless of dispatch outcome.
//  4. The worker handler picks up the row in the next tick (default
//     30s, but 0s on cold-start since the worker runs an immediate
//     tick) and calls Dispatch.
//
// When pendingEvents is nil (test wiring or local debugging), the
// handler falls back to the legacy inline-dispatch path with the
// IdempotencyClaimer + Release-on-error semantics so existing tests
// continue to drive the dispatcher directly.
func (h *StripeHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	enqueueStart := time.Now()
	body, err := io.ReadAll(io.LimitReader(r.Body, 65536))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "read_error", "cannot read request body")
		return
	}

	signature := r.Header.Get("Stripe-Signature")
	if signature == "" {
		res.Error(w, http.StatusBadRequest, "missing_signature", "Stripe-Signature header required")
		return
	}

	event, err := h.paymentSvc.VerifyWebhook(body, signature)
	if err != nil {
		slog.Error("stripe webhook signature verification failed", "error", err)
		res.Error(w, http.StatusBadRequest, "invalid_signature", "webhook signature verification failed")
		return
	}

	// Async path: enqueue and return 200 immediately. Skips the
	// IdempotencyClaimer entirely — the partial unique index on
	// pending_events.stripe_event_id provides at-most-once
	// semantics directly at the DB layer, and the worker handlers
	// are already idempotent so a re-dispatch on retry is safe.
	if h.pendingEvents != nil && event.EventID != "" {
		if h.enqueueAsync(r.Context(), event, enqueueStart, w) {
			return
		}
		// enqueueAsync wrote a 5xx already — fall through is not
		// safe; bail out so we don't double-write the response.
		return
	}

	// Legacy inline-dispatch path. Used only when pendingEvents
	// is not wired (e.g. in unit tests that drive the dispatcher
	// directly and do not need the async pipeline).
	h.handleWebhookInline(w, r, event)
}

// enqueueAsync runs the P8 async path: marshal the event projection,
// schedule it via ScheduleStripe (ON CONFLICT DO NOTHING), reply 200.
// Returns true when the response has been written to w (success or
// silent dedup); returns false when a fall-through is desired.
//
// On a database error (Postgres down, conflict on a non-evt index,
// etc.) the caller responds 5xx to signal retry — Stripe will re-send
// the same evt_* id and the next attempt will land on a freshly
// reachable database.
func (h *StripeHandler) enqueueAsync(
	ctx context.Context,
	event *portservice.StripeWebhookEvent,
	enqueueStart time.Time,
	w http.ResponseWriter,
) bool {
	payload, err := json.Marshal(event)
	if err != nil {
		// Marshalling our own struct shouldn't ever fail; if it
		// does, treat it as a 5xx so Stripe retries — by the next
		// retry the deploy may have rolled back the offending
		// shape.
		slog.Error("stripe webhook: marshal event for queue failed",
			"event_id", event.EventID, "event_type", event.Type, "error", err)
		res.Error(w, http.StatusServiceUnavailable, "enqueue_error",
			"failed to encode event for async dispatch")
		return true
	}

	pe, err := pendingevent.NewPendingEvent(pendingevent.NewPendingEventInput{
		EventType:     pendingevent.TypeStripeWebhook,
		Payload:       payload,
		FiresAt:       time.Now(),
		StripeEventID: event.EventID,
	})
	if err != nil {
		slog.Error("stripe webhook: build pending event failed",
			"event_id", event.EventID, "event_type", event.Type, "error", err)
		res.Error(w, http.StatusServiceUnavailable, "enqueue_error",
			"failed to construct queue row")
		return true
	}

	inserted, err := h.pendingEvents.ScheduleStripe(ctx, pe)
	if err != nil {
		slog.Error("stripe webhook: enqueue failed — Stripe will retry",
			"event_id", event.EventID, "event_type", event.Type, "error", err)
		res.Error(w, http.StatusServiceUnavailable, "enqueue_error",
			"failed to enqueue event for async dispatch")
		return true
	}

	enqueueMS := time.Since(enqueueStart).Milliseconds()
	if inserted {
		slog.Info("stripe webhook: enqueued for async dispatch",
			"event_id", event.EventID,
			"event_type", event.Type,
			"enqueue_ms", enqueueMS)
	} else {
		// Duplicate Stripe delivery — caught by ON CONFLICT
		// DO NOTHING. Silent no-op is the contract.
		slog.Info("stripe webhook: duplicate delivery deduplicated by ON CONFLICT",
			"event_id", event.EventID,
			"event_type", event.Type,
			"enqueue_ms", enqueueMS)
	}

	w.WriteHeader(http.StatusOK)
	return true
}

// handleWebhookInline is the legacy synchronous dispatcher kept for
// the test wiring that does not provide a pending_events queue. The
// production path goes through enqueueAsync; this function exists so
// the existing unit tests in stripe_handler_*_test.go continue to
// drive the dispatcher without rewiring every fixture.
func (h *StripeHandler) handleWebhookInline(w http.ResponseWriter, r *http.Request, event *portservice.StripeWebhookEvent) {
	// Idempotency guard. Stripe retries on 5xx, and a transient DB blip
	// could apply the same subscription transition twice (reactivate,
	// re-bump StartedAt) — or worse, fund a milestone twice. The
	// composite claimer first checks the Redis fast-path (5-min TTL)
	// then falls through to the durable `stripe_webhook_events` table
	// (BUG-10 fix).
	//
	// On a hard failure (both layers down) we explicitly reply 503 so
	// Stripe retries the webhook. The pre-fix code returned 200 here,
	// which let Stripe drop the event entirely and we silently lost
	// the state transition. 503 is the right answer: it preserves
	// the at-least-once delivery contract.
	if h.idempotencyStore != nil && event.EventID != "" {
		claimed, cErr := h.idempotencyStore.TryClaim(r.Context(), event.EventID, event.Type)
		if cErr != nil {
			slog.Error("stripe webhook: idempotency claim failed on both layers, refusing to process",
				"event_id", event.EventID, "type", event.Type, "error", cErr)
			res.Error(w, http.StatusServiceUnavailable, "idempotency_unavailable",
				"both fast-path and durable idempotency layers are down")
			return
		}
		if !claimed {
			slog.Info("stripe webhook: replay ignored", "event_id", event.EventID, "type", event.Type)
			w.WriteHeader(http.StatusOK)
			return
		}
	}

	// BUG-NEW-06 — dispatch errors must trigger a Release of the
	// idempotency claim AND a 5xx response so Stripe retries the
	// webhook. The pre-fix code logged errors inside each handler and
	// then unconditionally returned 200, leaving the durable claim
	// permanent — Stripe's next retry was silently deduped and the
	// state change was lost forever.
	dispatchErr := h.Dispatch(r.Context(), event)

	if dispatchErr != nil {
		slog.Error("stripe webhook: handler returned error, releasing idempotency claim and replying 5xx",
			"event_id", event.EventID,
			"event_type", event.Type,
			"error", dispatchErr)
		if h.idempotencyStore != nil && event.EventID != "" {
			if relErr := h.idempotencyStore.Release(r.Context(), event.EventID); relErr != nil {
				// Release failed — log loud but still respond 5xx.
				// Stripe will retry and the next attempt will see
				// the (still-present) claim as a replay; we'll log a
				// "replay ignored" line and the state change will
				// be permanently lost. There's no remediation here
				// that doesn't involve manual ops intervention.
				slog.Error("stripe webhook: idempotency release FAILED — state change will be permanently lost on Stripe retry",
					"event_id", event.EventID,
					"event_type", event.Type,
					"release_error", relErr,
					"original_error", dispatchErr)
			}
		}
		res.Error(w, http.StatusServiceUnavailable, "handler_error",
			"handler failed, retry expected")
		return
	}

	// All handlers succeeded — return 200 to Stripe to acknowledge receipt
	w.WriteHeader(http.StatusOK)
}

// Dispatch routes a verified Stripe event to its type-specific
// handler and returns the first non-nil error. Used by the
// pending_events worker handler (registered in
// adapter/worker/handlers/stripe_handlers.go) — the handler decodes
// the persisted StripeWebhookEvent payload from the queue row and
// hands it to this method.
//
// Public so the worker handler can call it from outside the handler
// package without exposing every per-event method individually. The
// HTTP webhook entry point (HandleWebhook) no longer calls Dispatch —
// it enqueues and returns 200 — but tests still use the per-event
// helpers directly to assert outcomes without going through the
// queue.
func (h *StripeHandler) Dispatch(ctx context.Context, event *portservice.StripeWebhookEvent) error {
	switch event.Type {
	case "payment_intent.succeeded":
		return h.handlePaymentSucceeded(ctx, event.PaymentIntentID)
	case "payment_intent.payment_failed":
		slog.Warn("payment intent failed", "payment_intent_id", event.PaymentIntentID)
		return nil
	case "account.updated":
		return h.dispatchEmbeddedNotif(ctx, event)
	case "capability.updated",
		"account.application.authorized",
		"account.application.deauthorized",
		"account.external_account.created",
		"account.external_account.updated",
		"account.external_account.deleted":
		return h.dispatchEmbeddedNotif(ctx, event)
	case "customer.subscription.created":
		return h.handleSubscriptionCreated(ctx, event)
	case "customer.subscription.updated",
		"customer.subscription.deleted":
		return h.handleSubscriptionSnapshot(ctx, event)
	case "invoice.payment_failed":
		return h.handleInvoicePaymentFailed(ctx, event)
	case "invoice.payment_succeeded":
		// Handled by the customer.subscription.updated that follows.
		// Stripe fires both on a successful renewal, but the
		// subscription.updated carries the new period dates — the
		// invoice event is informational.
		return nil
	case "invoice.paid":
		// invoice.paid fires on EVERY successful subscription
		// payment (initial + renewals), and crucially carries the
		// authoritative AmountPaid + service period. We use it as
		// the trigger for issuing our own customer-facing invoice
		// (FAC-NNNNNN), independent of the subscription state
		// reflection that customer.subscription.updated drives.
		return h.handleInvoicePaid(ctx, event)
	case "charge.refunded":
		// charge.refunded triggers a credit note (AV-NNNNNN) for the
		// refunded amount. The handler short-circuits when the refund
		// can't be matched to one of our subscription invoices —
		// non-invoiced charges are out of scope.
		return h.handleChargeRefunded(ctx, event)
	default:
		slog.Debug("unhandled stripe event", "type", event.Type)
		return nil
	}
}
