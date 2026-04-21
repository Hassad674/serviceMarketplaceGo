package stripe

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	stripe "github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/webhook"

	portservice "marketplace-backend/internal/port/service"
)

func (s *Service) ConstructWebhookEvent(payload []byte, signature string) (*portservice.StripeWebhookEvent, error) {
	event, err := webhook.ConstructEventWithOptions(payload, signature, s.webhookSecret, webhook.ConstructEventOptions{
		IgnoreAPIVersionMismatch: true,
	})
	if err != nil {
		return nil, fmt.Errorf("verify webhook signature: %w", err)
	}

	result := &portservice.StripeWebhookEvent{
		EventID: event.ID,
		Type:    string(event.Type),
	}

	switch event.Type {
	case "payment_intent.succeeded", "payment_intent.payment_failed":
		var pi stripe.PaymentIntent
		if err := json.Unmarshal(event.Data.Raw, &pi); err != nil {
			return nil, fmt.Errorf("unmarshal payment intent: %w", err)
		}
		result.PaymentIntentID = pi.ID

	case "account.updated", "account.application.authorized",
		"account.application.deauthorized", "account.external_account.created",
		"account.external_account.updated", "account.external_account.deleted",
		"capability.updated":
		var acct stripe.Account
		if err := json.Unmarshal(event.Data.Raw, &acct); err != nil {
			return nil, fmt.Errorf("unmarshal account: %w", err)
		}
		result.AccountID = acct.ID
		result.AccountSnapshot = buildAccountSnapshot(&acct)

	case "customer.subscription.created",
		"customer.subscription.updated",
		"customer.subscription.deleted":
		var sub stripe.Subscription
		if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
			return nil, fmt.Errorf("unmarshal subscription: %w", err)
		}
		snap := toSubscriptionSnapshot(&sub)
		result.SubscriptionSnapshot = &snap
		result.SubscriptionDeleted = event.Type == "customer.subscription.deleted"
		if sub.Metadata != nil {
			result.SubscriptionUserID = sub.Metadata["user_id"]
			// The UI sends "auto-renew off" by default via this metadata
			// key (Stripe Checkout does not support cancel_at_period_end
			// at creation time, see CreateCheckoutSession).
			result.SubscriptionCancelAtPeriodEndIntent = sub.Metadata["cancel_at_period_end"] == "true"
		}
		plan, cycle := parsePlanCycleFromSubscription(&sub)
		result.SubscriptionPlan = plan
		result.SubscriptionCycle = cycle

	case "invoice.payment_succeeded", "invoice.payment_failed":
		var inv stripe.Invoice
		if err := json.Unmarshal(event.Data.Raw, &inv); err != nil {
			return nil, fmt.Errorf("unmarshal invoice: %w", err)
		}
		if inv.Parent != nil && inv.Parent.SubscriptionDetails != nil && inv.Parent.SubscriptionDetails.Subscription != nil {
			result.InvoiceSubscriptionID = inv.Parent.SubscriptionDetails.Subscription.ID
		}
		result.InvoicePaymentFailed = event.Type == "invoice.payment_failed"
	}

	return result, nil
}

// toSubscriptionSnapshot projects the stripe.Subscription into a DTO the
// app layer can consume without importing the Stripe SDK. Duplicated
// here and not imported from subscription.go so the webhook adapter
// survives deletion of the subscription feature — at worst these
// SubscriptionSnapshot fields are zero-valued and ignored downstream.
func toSubscriptionSnapshot(sub *stripe.Subscription) portservice.SubscriptionSnapshot {
	snap := portservice.SubscriptionSnapshot{
		ID:                sub.ID,
		Status:            string(sub.Status),
		CancelAtPeriodEnd: sub.CancelAtPeriodEnd,
	}
	// The Stripe SDK models Subscription.Customer as either a bare id
	// or an expanded *Customer object depending on the request. For
	// webhook payloads we receive the id only, which the SDK exposes
	// on .Customer.ID (Customer is always non-nil even when not expanded).
	if sub.Customer != nil {
		snap.CustomerID = sub.Customer.ID
	}
	if sub.Items != nil && len(sub.Items.Data) > 0 {
		item := sub.Items.Data[0]
		if item.Price != nil {
			snap.PriceID = item.Price.ID
		}
		if item.CurrentPeriodStart > 0 {
			snap.CurrentPeriodStart = time.Unix(item.CurrentPeriodStart, 0).UTC()
		}
		if item.CurrentPeriodEnd > 0 {
			snap.CurrentPeriodEnd = time.Unix(item.CurrentPeriodEnd, 0).UTC()
		}
	}
	return snap
}

// parsePlanCycleFromSubscription reads the subscription's first price
// lookup_key ("premium_{plan}_{cycle}") and returns the two components.
// Unknown keys return empty strings so the handler can decide to skip
// rather than misclassify.
func parsePlanCycleFromSubscription(sub *stripe.Subscription) (plan, cycle string) {
	if sub.Items == nil || len(sub.Items.Data) == 0 {
		return "", ""
	}
	item := sub.Items.Data[0]
	if item.Price == nil || item.Price.LookupKey == "" {
		return "", ""
	}
	// Expected format: premium_{plan}_{cycle}
	parts := strings.SplitN(item.Price.LookupKey, "_", 3)
	if len(parts) != 3 || parts[0] != "premium" {
		return "", ""
	}
	return parts[1], parts[2]
}

// buildAccountSnapshot extracts a complete requirements picture from a Stripe
// Account so downstream handlers can decide what to notify without a second
// API round-trip.
func buildAccountSnapshot(acct *stripe.Account) *portservice.StripeAccountSnapshot {
	snap := &portservice.StripeAccountSnapshot{
		AccountID:        acct.ID,
		Country:          acct.Country,
		ChargesEnabled:   acct.ChargesEnabled,
		PayoutsEnabled:   acct.PayoutsEnabled,
		DetailsSubmitted: acct.DetailsSubmitted,
	}
	if acct.BusinessType != "" {
		snap.BusinessType = string(acct.BusinessType)
	}
	if acct.Requirements == nil {
		return snap
	}
	snap.CurrentlyDue = acct.Requirements.CurrentlyDue
	snap.EventuallyDue = acct.Requirements.EventuallyDue
	snap.PastDue = acct.Requirements.PastDue
	snap.PendingVerification = acct.Requirements.PendingVerification
	snap.DisabledReason = string(acct.Requirements.DisabledReason)
	for _, e := range acct.Requirements.Errors {
		snap.RequirementErrors = append(snap.RequirementErrors, portservice.StripeRequirementError{
			Requirement: e.Requirement,
			Code:        string(e.Code),
			Reason:      e.Reason,
		})
	}
	return snap
}
