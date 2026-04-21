package handler

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"

	embeddedapp "marketplace-backend/internal/app/embedded"
	paymentapp "marketplace-backend/internal/app/payment"
	proposalapp "marketplace-backend/internal/app/proposal"
	subscriptionapp "marketplace-backend/internal/app/subscription"
	subscriptiondomain "marketplace-backend/internal/domain/subscription"
	"marketplace-backend/internal/handler/dto/response"
	portservice "marketplace-backend/internal/port/service"
	res "marketplace-backend/pkg/response"
)

// IdempotencyClaimer is the narrow interface the Stripe webhook handler
// consumes to dedupe replays. Implemented by
// adapter/redis.WebhookIdempotencyStore; kept as a local interface so a
// test can stub it without pulling the redis SDK.
type IdempotencyClaimer interface {
	TryClaim(ctx context.Context, eventID string) (bool, error)
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

// GetConfig returns the Stripe publishable key for frontend initialization.
func (h *StripeHandler) GetConfig(w http.ResponseWriter, r *http.Request) {
	res.JSON(w, http.StatusOK, response.StripeConfigResponse{
		PublishableKey: h.publishableKey,
	})
}

// HandleWebhook processes Stripe webhook events.
// No auth middleware — Stripe sends directly, verified by signature.
func (h *StripeHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
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

	// Idempotency guard. Stripe retries on 5xx, and a transient DB blip
	// could apply the same subscription transition twice (reactivate,
	// re-bump StartedAt). TryClaim first-writes a 7-day Redis key keyed
	// by event.id; repeats return ok=false and are ACK'd without work.
	if h.idempotencyStore != nil && event.EventID != "" {
		claimed, cErr := h.idempotencyStore.TryClaim(r.Context(), event.EventID)
		if cErr != nil {
			slog.Warn("stripe webhook: idempotency claim failed, processing anyway",
				"event_id", event.EventID, "error", cErr)
		}
		if !claimed {
			slog.Info("stripe webhook: replay ignored", "event_id", event.EventID, "type", event.Type)
			w.WriteHeader(http.StatusOK)
			return
		}
	}

	switch event.Type {
	case "payment_intent.succeeded":
		h.handlePaymentSucceeded(r, event.PaymentIntentID)
	case "payment_intent.payment_failed":
		slog.Warn("payment intent failed", "payment_intent_id", event.PaymentIntentID)
	case "account.updated":
		h.dispatchEmbeddedNotif(r, event)
	case "capability.updated",
		"account.application.authorized",
		"account.application.deauthorized",
		"account.external_account.created",
		"account.external_account.updated",
		"account.external_account.deleted":
		h.dispatchEmbeddedNotif(r, event)
	case "customer.subscription.created":
		h.handleSubscriptionCreated(r, event)
	case "customer.subscription.updated",
		"customer.subscription.deleted":
		h.handleSubscriptionSnapshot(r, event)
	case "invoice.payment_failed":
		h.handleInvoicePaymentFailed(r, event)
	case "invoice.payment_succeeded":
		// Handled by the customer.subscription.updated that follows.
		// Stripe fires both on a successful renewal, but the
		// subscription.updated carries the new period dates — the
		// invoice event is informational.
	default:
		slog.Debug("unhandled stripe event", "type", event.Type)
	}

	// Always return 200 to Stripe to acknowledge receipt
	w.WriteHeader(http.StatusOK)
}

// handleSubscriptionCreated fires on first payment of a checkout session.
// Persists the subscription row and pre-warms the cache invalidation so
// the next fee-preview hit reflects Premium immediately.
func (h *StripeHandler) handleSubscriptionCreated(r *http.Request, event *portservice.StripeWebhookEvent) {
	if h.subscriptionSvc == nil || event.SubscriptionSnapshot == nil {
		return
	}
	userID, err := uuid.Parse(event.SubscriptionUserID)
	if err != nil {
		slog.Warn("stripe webhook: subscription.created with missing/invalid user_id metadata",
			"event_id", event.EventID, "user_id_raw", event.SubscriptionUserID)
		return
	}
	if event.SubscriptionPlan == "" || event.SubscriptionCycle == "" {
		slog.Warn("stripe webhook: subscription.created could not parse plan/cycle from lookup_key",
			"event_id", event.EventID, "stripe_sub_id", event.SubscriptionSnapshot.ID)
		return
	}

	// Enforce the user's auto-renew choice captured at checkout. Stripe
	// Checkout doesn't support cancel_at_period_end at session creation,
	// so the flag rides in subscription metadata and we apply it here via
	// a secondary update. We mutate the snapshot BEFORE persisting so the
	// DB row reflects the user's intent from the very first insert, then
	// propagate the change to Stripe. A follow-up
	// customer.subscription.updated will arrive and reconfirm; its
	// idempotent snapshot handler makes that a no-op.
	snap := *event.SubscriptionSnapshot
	if event.SubscriptionCancelAtPeriodEndIntent && !snap.CancelAtPeriodEnd {
		if uErr := h.subscriptionSvc.EnforceCancelAtPeriodEnd(r.Context(), snap.ID, true); uErr != nil {
			// Log and proceed with the original snapshot — the next
			// customer.subscription.updated event will sync the flag if
			// the update succeeds eventually.
			slog.Warn("stripe webhook: enforce cancel_at_period_end failed, persisting Stripe default",
				"event_id", event.EventID, "stripe_sub_id", snap.ID, "error", uErr)
		} else {
			snap.CancelAtPeriodEnd = true
		}
	}

	err = h.subscriptionSvc.RegisterFromCheckout(
		r.Context(),
		userID,
		subscriptiondomain.Plan(event.SubscriptionPlan),
		subscriptiondomain.BillingCycle(event.SubscriptionCycle),
		snap.CustomerID,
		snap,
	)
	if err != nil {
		slog.Error("stripe webhook: register subscription failed",
			"event_id", event.EventID, "user_id", userID, "error", err)
		return
	}

	h.invalidateSubscriptionCache(r.Context(), userID)
}

// handleSubscriptionSnapshot reflects customer.subscription.updated and
// customer.subscription.deleted into our row via the app service.
func (h *StripeHandler) handleSubscriptionSnapshot(r *http.Request, event *portservice.StripeWebhookEvent) {
	if h.subscriptionSvc == nil || event.SubscriptionSnapshot == nil {
		return
	}
	if err := h.subscriptionSvc.HandleSubscriptionSnapshot(r.Context(), *event.SubscriptionSnapshot, event.SubscriptionDeleted); err != nil {
		slog.Error("stripe webhook: subscription snapshot update failed",
			"event_id", event.EventID, "stripe_sub_id", event.SubscriptionSnapshot.ID, "error", err)
		return
	}

	// Invalidate cache using the user id from the subscription metadata
	// when available. On subscription.updated we don't always get the
	// metadata echoed back — in that case we fall back to invalidating
	// by the Stripe subscription id, which requires an app-level lookup.
	// For now, best-effort: only invalidate when user_id is present.
	if event.SubscriptionUserID != "" {
		if uid, err := uuid.Parse(event.SubscriptionUserID); err == nil {
			h.invalidateSubscriptionCache(r.Context(), uid)
		}
	}
}

// handleInvoicePaymentFailed opens a grace window on the subscription.
func (h *StripeHandler) handleInvoicePaymentFailed(r *http.Request, event *portservice.StripeWebhookEvent) {
	if h.subscriptionSvc == nil || event.InvoiceSubscriptionID == "" {
		return
	}
	// We model past_due transitions via HandleSubscriptionSnapshot —
	// Stripe sends a customer.subscription.updated with status=past_due
	// alongside the invoice.payment_failed, so this handler is a
	// defensive no-op in the happy path. Logged for audit visibility.
	slog.Info("stripe webhook: invoice.payment_failed received",
		"event_id", event.EventID, "subscription_id", event.InvoiceSubscriptionID)
	_ = time.Now // keep "time" imported if future logic needs it
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
// notifier (when wired). Best-effort: logs errors, never returns them to
// Stripe, otherwise Stripe retries our webhook which could spam users.
func (h *StripeHandler) dispatchEmbeddedNotif(r *http.Request, event *portservice.StripeWebhookEvent) {
	if h.embeddedNotifier == nil || event == nil || event.AccountSnapshot == nil {
		return
	}
	if err := h.embeddedNotifier.HandleAccountSnapshot(r.Context(), event.AccountSnapshot); err != nil {
		slog.Warn("embedded notifier: handle snapshot",
			"account_id", event.AccountSnapshot.AccountID,
			"event_type", event.Type,
			"error", err)
	}
}

func (h *StripeHandler) handlePaymentSucceeded(r *http.Request, piID string) {
	proposalID, err := h.paymentSvc.HandlePaymentSucceeded(r.Context(), piID)
	if err != nil {
		slog.Error("handle payment succeeded", "payment_intent", piID, "error", err)
		return
	}

	if err := h.proposalSvc.ConfirmPaymentAndActivate(r.Context(), proposalID); err != nil {
		slog.Error("confirm payment and activate", "proposal_id", proposalID, "error", err)
	}
}

