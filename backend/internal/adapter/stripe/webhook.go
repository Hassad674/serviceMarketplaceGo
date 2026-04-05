package stripe

import (
	"encoding/json"
	"fmt"

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
		Type: string(event.Type),
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
	}

	return result, nil
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
