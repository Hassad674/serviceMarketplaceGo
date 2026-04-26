package handler

import (
	"context"
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
	subscriptiondomain "marketplace-backend/internal/domain/subscription"
	"marketplace-backend/internal/handler/dto/response"
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

	// Invoicing wiring. Optional — when nil the invoice.paid hook is a
	// no-op. Removing the invoicing module from main.go disables the
	// hook cleanly.
	invoicingSvc *invoicingapp.Service
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
	case "invoice.paid":
		// invoice.paid fires on EVERY successful subscription
		// payment (initial + renewals), and crucially carries the
		// authoritative AmountPaid + service period. We use it as
		// the trigger for issuing our own customer-facing invoice
		// (FAC-NNNNNN), independent of the subscription state
		// reflection that customer.subscription.updated drives.
		h.handleInvoicePaid(r, event)
	case "charge.refunded":
		// charge.refunded triggers a credit note (AV-NNNNNN) for the
		// refunded amount. The handler short-circuits when the refund
		// can't be matched to one of our subscription invoices —
		// non-invoiced charges are out of scope.
		h.handleChargeRefunded(r, event)
	default:
		slog.Debug("unhandled stripe event", "type", event.Type)
	}

	// Always return 200 to Stripe to acknowledge receipt
	w.WriteHeader(http.StatusOK)
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
func (h *StripeHandler) handleSubscriptionCreated(r *http.Request, event *portservice.StripeWebhookEvent) {
	if h.subscriptionSvc == nil || event.SubscriptionSnapshot == nil {
		return
	}
	orgID, cacheUserID, err := h.resolveSubscriptionOwner(r.Context(), event)
	if err != nil {
		slog.Warn("stripe webhook: subscription.created owner resolution failed",
			"event_id", event.EventID,
			"organization_id_raw", event.SubscriptionOrganizationID,
			"user_id_raw", event.SubscriptionUserID,
			"error", err)
		return
	}
	if event.SubscriptionPlan == "" || event.SubscriptionCycle == "" {
		slog.Warn("stripe webhook: subscription.created could not parse plan/cycle from lookup_key",
			"event_id", event.EventID, "stripe_sub_id", event.SubscriptionSnapshot.ID)
		return
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
		if uErr := h.subscriptionSvc.EnforceCancelAtPeriodEnd(r.Context(), snap.ID, true); uErr != nil {
			slog.Warn("stripe webhook: enforce cancel_at_period_end failed, persisting Stripe default",
				"event_id", event.EventID, "stripe_sub_id", snap.ID, "error", uErr)
		} else {
			snap.CancelAtPeriodEnd = true
		}
	}

	if err := h.subscriptionSvc.RegisterFromCheckout(
		r.Context(),
		orgID,
		subscriptiondomain.Plan(event.SubscriptionPlan),
		subscriptiondomain.BillingCycle(event.SubscriptionCycle),
		snap.CustomerID,
		snap,
	); err != nil {
		slog.Error("stripe webhook: register subscription failed",
			"event_id", event.EventID, "organization_id", orgID, "error", err)
		return
	}

	// Cache is still keyed by user id (billing is per-provider). Only the
	// legacy path gives us a direct user id; new metadata carries only
	// org_id, and invalidation falls back to TTL — acceptable given the
	// 60s window.
	if cacheUserID != uuid.Nil {
		h.invalidateSubscriptionCache(r.Context(), cacheUserID)
	}
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
func (h *StripeHandler) handleSubscriptionSnapshot(r *http.Request, event *portservice.StripeWebhookEvent) {
	if h.subscriptionSvc == nil || event.SubscriptionSnapshot == nil {
		return
	}
	if err := h.subscriptionSvc.HandleSubscriptionSnapshot(r.Context(), *event.SubscriptionSnapshot, event.SubscriptionDeleted); err != nil {
		slog.Error("stripe webhook: subscription snapshot update failed",
			"event_id", event.EventID, "stripe_sub_id", event.SubscriptionSnapshot.ID, "error", err)
		return
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
			h.invalidateSubscriptionCache(r.Context(), uid)
		}
	}
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
func (h *StripeHandler) handleInvoicePaid(r *http.Request, event *portservice.StripeWebhookEvent) {
	if h.invoicingSvc == nil {
		return
	}
	if !event.InvoicePaid || event.InvoiceSubscriptionID == "" {
		// Either not the projection we expect, or a non-subscription
		// invoice (manual / one-off). The latter is out of scope
		// for the invoice.paid -> FAC pipeline.
		return
	}

	orgID, err := h.resolveInvoicePaidOwner(r.Context(), event)
	if err != nil {
		slog.Warn("stripe webhook: invoice.paid owner resolution failed",
			"event_id", event.EventID,
			"stripe_invoice_id", event.InvoiceID,
			"organization_id_raw", event.InvoiceSubscriptionOrgID,
			"user_id_raw", event.InvoiceSubscriptionUserID,
			"error", err)
		return
	}

	planLabel := event.InvoiceLineDescription
	if planLabel == "" {
		planLabel = "Premium subscription"
	}

	if _, err := h.invoicingSvc.IssueFromSubscription(r.Context(), invoicingapp.IssueFromSubscriptionInput{
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
	}
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
func (h *StripeHandler) handleChargeRefunded(r *http.Request, event *portservice.StripeWebhookEvent) {
	if h.invoicingSvc == nil {
		return
	}
	if !event.ChargeRefunded || event.ChargePaymentIntentID == "" {
		slog.Debug("stripe webhook: charge.refunded without payment intent — skipping",
			"event_id", event.EventID, "charge_id", event.ChargeID)
		return
	}

	inv, err := h.invoicingSvc.FindInvoiceByPaymentIntentID(r.Context(), event.ChargePaymentIntentID)
	if err != nil {
		// Not all charges produce one of OUR invoices (early
		// test data, non-subscription payments, etc.). A miss is
		// not an error condition — log and bail.
		slog.Info("stripe webhook: charge.refunded has no matching invoice — skipping",
			"event_id", event.EventID,
			"payment_intent_id", event.ChargePaymentIntentID,
			"error", err)
		return
	}

	if _, err := h.invoicingSvc.IssueCreditNote(r.Context(), invoicingapp.IssueCreditNoteInput{
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

