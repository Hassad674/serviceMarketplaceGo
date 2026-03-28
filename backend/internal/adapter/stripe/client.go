package stripe

import (
	stripe "github.com/stripe/stripe-go/v82"
)

// Service implements port/service.StripeService using the Stripe API.
type Service struct {
	webhookSecret string
}

// NewService initializes the Stripe SDK and returns a new Service.
func NewService(secretKey, webhookSecret string) *Service {
	stripe.Key = secretKey
	return &Service{webhookSecret: webhookSecret}
}
