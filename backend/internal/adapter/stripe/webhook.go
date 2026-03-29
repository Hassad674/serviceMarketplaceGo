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

	case "account.updated":
		var acct stripe.Account
		if err := json.Unmarshal(event.Data.Raw, &acct); err != nil {
			return nil, fmt.Errorf("unmarshal account: %w", err)
		}
		result.AccountID = acct.ID
	}

	return result, nil
}
