package handler

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	invoicingapp "marketplace-backend/internal/app/invoicing"
	subscriptiondomain "marketplace-backend/internal/domain/subscription"
	portservice "marketplace-backend/internal/port/service"
	"marketplace-backend/internal/system"
)

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

// handlePaymentSucceededWithEvent is the entry point used by Dispatch
// when the projected event carries optional billing-details (client name
// + address) that Stripe collected inline on the Payment Element.
// Forwards to handlePaymentSucceeded for the canonical reconciliation
// (mark record paid + activate proposal); afterwards, hydrates the
// client_billing_profile so the receipt snapshot for FUTURE charges and
// the standalone "Mes infos de facturation" page reflect the inline
// capture.
//
// Hydration is a best-effort side-effect: any error inside the hook is
// logged at WARN and never propagated. The receipt snapshot for the
// charge that just succeeded was captured at PaymentIntent CREATION
// time, so this hydration cannot retroactively fix that snapshot — its
// purpose is to enrich the org's billing identity for the next charge
// and for the user's settings page.
func (h *StripeHandler) handlePaymentSucceededWithEvent(ctx context.Context, event *portservice.StripeWebhookEvent) error {
	if err := h.handlePaymentSucceeded(ctx, event.PaymentIntentID); err != nil {
		return err
	}
	// Server-side analytics — captures payment success even if the
	// browser closed before the redirect. Stripe's event id doubles
	// as the PostHog message_id so a duplicated webhook delivery
	// (Stripe retries on 5xx) does not double-count the conversion.
	h.captureProposalPaymentSucceeded(ctx, event)
	if event.PaymentBillingDetails == nil || h.invoicingSvc == nil {
		return nil
	}
	h.hydrateClientBillingProfile(ctx, event.PaymentIntentID, *event.PaymentBillingDetails)
	return nil
}

// captureProposalPaymentSucceeded ships the conversion event to
// PostHog. Best-effort: any failure inside the analytics pipeline is
// already swallowed by the adapter, so we only need to guard against
// the optional analytics dep being nil and against missing
// payment-record metadata.
func (h *StripeHandler) captureProposalPaymentSucceeded(ctx context.Context, event *portservice.StripeWebhookEvent) {
	if h.analytics == nil || event == nil || h.paymentSvc == nil {
		return
	}
	record, err := h.paymentSvc.FindRecordByPaymentIntentID(ctx, event.PaymentIntentID)
	if err != nil || record == nil {
		// Silent fallback: ship a minimal event so the conversion
		// still shows up in the dashboard, just without amount/owner
		// dimensions.
		h.analytics.Capture(ctx, portservice.AnalyticsEvent{
			DistinctID: "stripe-anon",
			EventName:  "proposal.payment_succeeded",
			Properties: map[string]any{
				"payment_intent_id": event.PaymentIntentID,
				"source":            "stripe_webhook",
			},
			MessageID: event.EventID,
		})
		return
	}
	h.analytics.Capture(ctx, portservice.AnalyticsEvent{
		DistinctID: record.ClientID.String(),
		EventName:  "proposal.payment_succeeded",
		Properties: map[string]any{
			"payment_intent_id": event.PaymentIntentID,
			"proposal_id":       record.ProposalID.String(),
			"amount":            record.ProposalAmount,
			"client_total":      record.ClientTotalAmount,
			"currency":          record.Currency,
			"provider_id":       record.ProviderID.String(),
			"source":            "stripe_webhook",
		},
		MessageID: event.EventID,
	})
}

// hydrateClientBillingProfile resolves the client organization that
// owns the just-paid PaymentIntent and merges Stripe billing_details
// into its profile row. Best-effort: every failure is logged and
// swallowed so a glitch in the hydration pipeline never breaks the
// money-movement contract.
func (h *StripeHandler) hydrateClientBillingProfile(
	ctx context.Context,
	paymentIntentID string,
	bd portservice.PaymentBillingDetails,
) {
	if h.paymentSvc == nil || h.invoicingSvc == nil {
		return
	}
	record, err := h.paymentSvc.FindRecordByPaymentIntentID(ctx, paymentIntentID)
	if err != nil || record == nil {
		slog.Warn("stripe webhook: hydrate billing profile — record lookup failed",
			"payment_intent_id", paymentIntentID, "error", err)
		return
	}
	orgID, err := h.resolveClientOrg(ctx, record.ClientID)
	if err != nil || orgID == uuid.Nil {
		slog.Warn("stripe webhook: hydrate billing profile — org resolution failed",
			"payment_intent_id", paymentIntentID, "client_user_id", record.ClientID, "error", err)
		return
	}
	if err := h.invoicingSvc.HydrateFromPaymentBillingDetails(ctx, orgID, bd); err != nil {
		slog.Warn("stripe webhook: hydrate billing profile — persist failed",
			"payment_intent_id", paymentIntentID, "organization_id", orgID, "error", err)
		return
	}
	slog.Info("stripe webhook: client billing profile hydrated from inline payment details",
		"payment_intent_id", paymentIntentID,
		"organization_id", orgID,
		"has_legal_name", bd.Name != "",
		"has_address", bd.AddressLine1 != "",
	)
}

// resolveClientOrg returns the organization id that owns userID. nil
// userOrgResolver is a no-op (returns uuid.Nil with nil err).
func (h *StripeHandler) resolveClientOrg(ctx context.Context, userID uuid.UUID) (uuid.UUID, error) {
	if h.userOrgResolver == nil {
		return uuid.Nil, nil
	}
	return h.userOrgResolver(ctx, userID)
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
