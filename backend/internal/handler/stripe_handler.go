package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	subscriptiondomain "marketplace-backend/internal/domain/subscription"
	"marketplace-backend/internal/handler/dto/response"
	"marketplace-backend/internal/port/repository"
	portservice "marketplace-backend/internal/port/service"
	"marketplace-backend/internal/system"
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
//                    MUST reply non-2xx so Stripe retries.
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
	subscriptionSvc        *subscriptionapp.Service
	subscriptionCache      SubscriptionCacheInvalidator
	idempotencyStore       IdempotencyClaimer

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

// handleSubscriptionCreated fires on first payment of a checkout session.
// Persists the subscription row and pre-warms the cache invalidation so
// the next fee-preview hit reflects Premium immediately.
//
// Owner id is read from metadata with a dual-key strategy: since the
// org-scoped migration the canonical key is organization_id, but Stripe
// subscriptions created before the migration still carry the legacy
// user_id key — in that case the handler resolves the owning org via
// users.organization_id. The backfill script
// (cmd/stripe-backfill-metadata) removes the need for this fallback in
// Stripe once it runs; the code keeps it around for safety during the
// transition window.
func (h *StripeHandler) handleSubscriptionCreated(ctx context.Context, event *portservice.StripeWebhookEvent) error {
	if h.subscriptionSvc == nil || event.SubscriptionSnapshot == nil {
		return nil
	}
	orgID, cacheUserID, err := h.resolveSubscriptionOwner(ctx, event)
	if err != nil {
		slog.Warn("stripe webhook: subscription.created owner resolution failed",
			"event_id", event.EventID,
			"organization_id_raw", event.SubscriptionOrganizationID,
			"user_id_raw", event.SubscriptionUserID,
			"error", err)
		// Owner resolution can fail because the metadata was lost or
		// the user was deleted — these are not transient. Don't trigger
		// a retry on data we'll never be able to process.
		return nil
	}
	if event.SubscriptionPlan == "" || event.SubscriptionCycle == "" {
		slog.Warn("stripe webhook: subscription.created could not parse plan/cycle from lookup_key",
			"event_id", event.EventID, "stripe_sub_id", event.SubscriptionSnapshot.ID)
		return nil
	}

	// Enforce the actor's auto-renew choice captured at checkout. Stripe
	// Checkout doesn't support cancel_at_period_end at session creation,
	// so the flag rides in subscription metadata and we apply it here via
	// a secondary update. We mutate the snapshot BEFORE persisting so the
	// DB row reflects intent from the very first insert, then propagate
	// the change to Stripe. A follow-up customer.subscription.updated
	// will arrive and reconfirm; its idempotent snapshot handler makes
	// that a no-op.
	snap := *event.SubscriptionSnapshot
	if event.SubscriptionCancelAtPeriodEndIntent && !snap.CancelAtPeriodEnd {
		if uErr := h.subscriptionSvc.EnforceCancelAtPeriodEnd(ctx, snap.ID, true); uErr != nil {
			slog.Warn("stripe webhook: enforce cancel_at_period_end failed, persisting Stripe default",
				"event_id", event.EventID, "stripe_sub_id", snap.ID, "error", uErr)
		} else {
			snap.CancelAtPeriodEnd = true
		}
	}

	if err := h.subscriptionSvc.RegisterFromCheckout(
		ctx,
		orgID,
		subscriptiondomain.Plan(event.SubscriptionPlan),
		subscriptiondomain.BillingCycle(event.SubscriptionCycle),
		snap.CustomerID,
		snap,
	); err != nil {
		slog.Error("stripe webhook: register subscription failed",
			"event_id", event.EventID, "organization_id", orgID, "error", err)
		// BUG-NEW-06 — surface the error so the dispatcher releases
		// the idempotency claim and replies 5xx; Stripe will retry
		// and we'll get another chance to register the subscription.
		return fmt.Errorf("register subscription: %w", err)
	}

	// Cache is still keyed by user id (billing is per-provider). Only the
	// legacy path gives us a direct user id; new metadata carries only
	// org_id, and invalidation falls back to TTL — acceptable given the
	// 60s window.
	if cacheUserID != uuid.Nil {
		h.invalidateSubscriptionCache(ctx, cacheUserID)
	}
	return nil
}

// resolveSubscriptionOwner derives the organization_id that owns the
// subscription from the Stripe event metadata, using the dual-key
// strategy. cacheUserID is returned only when the legacy user_id path is
// used — new events with organization_id alone return uuid.Nil there,
// and the caller must rely on cache TTL for invalidation.
func (h *StripeHandler) resolveSubscriptionOwner(
	ctx context.Context,
	event *portservice.StripeWebhookEvent,
) (orgID, cacheUserID uuid.UUID, err error) {
	if event.SubscriptionOrganizationID != "" {
		parsed, pErr := uuid.Parse(event.SubscriptionOrganizationID)
		if pErr != nil {
			return uuid.Nil, uuid.Nil, pErr
		}
		return parsed, uuid.Nil, nil
	}
	if event.SubscriptionUserID == "" {
		return uuid.Nil, uuid.Nil, errMissingOwnerMetadata
	}
	userID, pErr := uuid.Parse(event.SubscriptionUserID)
	if pErr != nil {
		return uuid.Nil, uuid.Nil, pErr
	}
	resolved, rErr := h.subscriptionSvc.ResolveActorOrganization(ctx, userID)
	if rErr != nil {
		return uuid.Nil, uuid.Nil, rErr
	}
	return resolved, userID, nil
}

// handleSubscriptionSnapshot reflects customer.subscription.updated and
// customer.subscription.deleted into our row via the app service.
func (h *StripeHandler) handleSubscriptionSnapshot(ctx context.Context, event *portservice.StripeWebhookEvent) error {
	if h.subscriptionSvc == nil || event.SubscriptionSnapshot == nil {
		return nil
	}
	if err := h.subscriptionSvc.HandleSubscriptionSnapshot(ctx, *event.SubscriptionSnapshot, event.SubscriptionDeleted); err != nil {
		slog.Error("stripe webhook: subscription snapshot update failed",
			"event_id", event.EventID, "stripe_sub_id", event.SubscriptionSnapshot.ID, "error", err)
		// BUG-NEW-06 — surface so the dispatcher releases the
		// idempotency claim and replies 5xx; Stripe retries until we
		// land the snapshot.
		return fmt.Errorf("handle subscription snapshot: %w", err)
	}

	// Cache invalidation stays user-keyed by design — billing is a
	// per-provider concern (milestone payments are paid to individuals,
	// not to organizations) so the cache mirrors that grain. We only
	// have a user id to invalidate when the event carries the legacy
	// metadata; on the new metadata path we rely on the 60s TTL to
	// self-heal, which is acceptable on a billing decision that
	// already errs on the side of charging the standard fee on miss.
	if event.SubscriptionUserID != "" {
		if uid, err := uuid.Parse(event.SubscriptionUserID); err == nil {
			h.invalidateSubscriptionCache(ctx, uid)
		}
	}
	return nil
}

// handleInvoicePaid issues a customer-facing invoice for a Stripe
// invoice.paid event. The flow:
//
//  1. Skip when invoicing is disabled (feature not wired in main.go).
//  2. Filter to subscription-backed invoices — non-subscription
//     invoices are out of scope for this hook.
//  3. Resolve the owning organization via the subscription metadata
//     captured on the invoice's parent.subscription_details snapshot.
//     Falls back to the legacy user_id metadata for subscriptions
//     created before the org-scoped migration.
//  4. Pick a sensible plan label, defaulting to a generic string when
//     the line description is missing.
//  5. Hand off to the invoicing app service; errors are logged but the
//     webhook still returns 200 to Stripe (handled by the caller).
func (h *StripeHandler) handleInvoicePaid(ctx context.Context, event *portservice.StripeWebhookEvent) error {
	if h.invoicingSvc == nil {
		return nil
	}
	if !event.InvoicePaid || event.InvoiceSubscriptionID == "" {
		// Either not the projection we expect, or a non-subscription
		// invoice (manual / one-off). The latter is out of scope
		// for the invoice.paid -> FAC pipeline.
		return nil
	}

	orgID, err := h.resolveInvoicePaidOwner(ctx, event)
	if err != nil {
		slog.Warn("stripe webhook: invoice.paid owner resolution failed",
			"event_id", event.EventID,
			"stripe_invoice_id", event.InvoiceID,
			"organization_id_raw", event.InvoiceSubscriptionOrgID,
			"user_id_raw", event.InvoiceSubscriptionUserID,
			"error", err)
		// Owner resolution failures are permanent (lost metadata);
		// don't trigger Stripe retries we'll never satisfy.
		return nil
	}

	planLabel := event.InvoiceLineDescription
	if planLabel == "" {
		planLabel = "Premium subscription"
	}

	if _, err := h.invoicingSvc.IssueFromSubscription(ctx, invoicingapp.IssueFromSubscriptionInput{
		OrganizationID:        orgID,
		StripeEventID:         event.EventID,
		StripeInvoiceID:       event.InvoiceID,
		StripePaymentIntentID: event.InvoicePaymentIntentID,
		AmountCents:           event.InvoiceAmountPaidCents,
		Currency:              event.InvoiceCurrency,
		PeriodStart:           event.InvoicePeriodStart,
		PeriodEnd:             event.InvoicePeriodEnd,
		PlanLabel:             planLabel,
	}); err != nil {
		slog.Error("stripe webhook: invoice issuance failed",
			"event_id", event.EventID,
			"stripe_invoice_id", event.InvoiceID,
			"organization_id", orgID,
			"error", err)
		// BUG-NEW-06 — surface so the dispatcher releases the
		// idempotency claim and replies 5xx; Stripe retries and we
		// get another chance to issue the invoice.
		return fmt.Errorf("issue invoice: %w", err)
	}
	return nil
}

// resolveInvoicePaidOwner derives the org id from invoice.paid metadata.
// Mirrors resolveSubscriptionOwner's dual-key strategy but reads the
// fields the webhook adapter projects out of the invoice's parent
// snapshot (not the subscription event payload, which we don't have
// here).
func (h *StripeHandler) resolveInvoicePaidOwner(
	ctx context.Context,
	event *portservice.StripeWebhookEvent,
) (uuid.UUID, error) {
	if event.InvoiceSubscriptionOrgID != "" {
		return uuid.Parse(event.InvoiceSubscriptionOrgID)
	}
	if event.InvoiceSubscriptionUserID == "" {
		return uuid.Nil, errMissingOwnerMetadata
	}
	if h.subscriptionSvc == nil {
		return uuid.Nil, errMissingOwnerMetadata
	}
	userID, err := uuid.Parse(event.InvoiceSubscriptionUserID)
	if err != nil {
		return uuid.Nil, err
	}
	return h.subscriptionSvc.ResolveActorOrganization(ctx, userID)
}

// handleChargeRefunded issues a credit note for a Stripe charge.refunded
// event. The pipeline:
//
//  1. Skip when invoicing is disabled (feature not wired in main.go).
//  2. Look up the original invoice via the PaymentIntent — we only emit
//     credit notes for subscription invoices we issued. Charges that
//     never produced an invoice (early test data, non-subscription
//     payments) are silently skipped with a debug log.
//  3. Hand off to the invoicing app service. Errors are logged but the
//     webhook still returns 200 so Stripe doesn't burn its retry budget
//     re-running a pipeline that's never going to succeed.
func (h *StripeHandler) handleChargeRefunded(ctx context.Context, event *portservice.StripeWebhookEvent) error {
	if h.invoicingSvc == nil {
		return nil
	}
	if !event.ChargeRefunded || event.ChargePaymentIntentID == "" {
		slog.Debug("stripe webhook: charge.refunded without payment intent — skipping",
			"event_id", event.EventID, "charge_id", event.ChargeID)
		return nil
	}

	inv, err := h.invoicingSvc.FindInvoiceByPaymentIntentID(ctx, event.ChargePaymentIntentID)
	if err != nil {
		// Not all charges produce one of OUR invoices (early
		// test data, non-subscription payments, etc.). A miss is
		// not an error condition — log and bail.
		slog.Info("stripe webhook: charge.refunded has no matching invoice — skipping",
			"event_id", event.EventID,
			"payment_intent_id", event.ChargePaymentIntentID,
			"error", err)
		return nil
	}

	if _, err := h.invoicingSvc.IssueCreditNote(ctx, invoicingapp.IssueCreditNoteInput{
		OriginalInvoiceID: inv.ID,
		Reason:            "Stripe refund",
		AmountCents:       event.ChargeAmountRefundedCents,
		StripeEventID:     event.EventID,
		StripeRefundID:    event.ChargeRefundID,
	}); err != nil {
		slog.Error("stripe webhook: credit note issuance failed",
			"event_id", event.EventID,
			"original_invoice_id", inv.ID,
			"error", err)
		// BUG-NEW-06 — surface so the dispatcher releases the
		// idempotency claim and replies 5xx; Stripe retries and we
		// get another chance to issue the credit note.
		return fmt.Errorf("issue credit note: %w", err)
	}
	return nil
}

// handleInvoicePaymentFailed opens a grace window on the subscription.
func (h *StripeHandler) handleInvoicePaymentFailed(ctx context.Context, event *portservice.StripeWebhookEvent) error {
	_ = ctx // reserved for future grace-window writes
	if h.subscriptionSvc == nil || event.InvoiceSubscriptionID == "" {
		return nil
	}
	// We model past_due transitions via HandleSubscriptionSnapshot —
	// Stripe sends a customer.subscription.updated with status=past_due
	// alongside the invoice.payment_failed, so this handler is a
	// defensive no-op in the happy path. Logged for audit visibility.
	slog.Info("stripe webhook: invoice.payment_failed received",
		"event_id", event.EventID, "subscription_id", event.InvoiceSubscriptionID)
	_ = time.Now // keep "time" imported if future logic needs it
	return nil
}

// invalidateSubscriptionCache flushes the Premium cache entry for userID.
// Failure is logged but never surfaces to Stripe — the cache has a 60s
// TTL, so a missed invalidation self-heals quickly.
func (h *StripeHandler) invalidateSubscriptionCache(ctx context.Context, userID uuid.UUID) {
	if h.subscriptionCache == nil {
		return
	}
	if err := h.subscriptionCache.Invalidate(ctx, userID); err != nil {
		slog.Warn("stripe webhook: subscription cache invalidate failed",
			"user_id", userID, "error", err)
	}
}

// dispatchEmbeddedNotif fans out a Stripe account snapshot to the embedded
// notifier (when wired). Best-effort: logs errors but does NOT trigger a
// Stripe retry — pushing the same notification twice on a Stripe retry
// would spam users, which is worse than dropping the notification.
// Therefore this returns nil even on internal failure.
func (h *StripeHandler) dispatchEmbeddedNotif(ctx context.Context, event *portservice.StripeWebhookEvent) error {
	if h.embeddedNotifier == nil || event == nil || event.AccountSnapshot == nil {
		return nil
	}
	if err := h.embeddedNotifier.HandleAccountSnapshot(ctx, event.AccountSnapshot); err != nil {
		slog.Warn("embedded notifier: handle snapshot",
			"account_id", event.AccountSnapshot.AccountID,
			"event_type", event.Type,
			"error", err)
	}
	return nil
}

func (h *StripeHandler) handlePaymentSucceeded(ctx context.Context, piID string) error {
	// Stripe webhook is a system-actor caller: the request is
	// authenticated by signature, not by a user session, so the
	// per-tenant org context expected by user-facing flows is
	// absent. Mark the context explicitly so downstream services
	// (e.g. ConfirmPaymentAndActivate) take the system-actor
	// branch of loadProposalForActor instead of panicking on
	// MustGetOrgID.
	ctx = system.WithSystemActor(ctx)

	proposalID, err := h.paymentSvc.HandlePaymentSucceeded(ctx, piID)
	if err != nil {
		slog.Error("handle payment succeeded", "payment_intent", piID, "error", err)
		// BUG-NEW-06 — surface so the dispatcher releases the
		// idempotency claim and replies 5xx. A failed payment
		// reconciliation MUST be retried; otherwise the proposal
		// stays in pending_payment forever.
		return fmt.Errorf("handle payment succeeded: %w", err)
	}

	if err := h.proposalSvc.ConfirmPaymentAndActivate(ctx, proposalID); err != nil {
		slog.Error("confirm payment and activate", "proposal_id", proposalID, "error", err)
		return fmt.Errorf("confirm payment and activate: %w", err)
	}
	return nil
}

