package stripe

import (
	"context"
	"fmt"

	stripe "github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/accountsession"
)

// CreateAccountSession creates an Account Session for Stripe Connect Embedded Components.
func (s *Service) CreateAccountSession(_ context.Context, accountID string) (string, error) {
	params := &stripe.AccountSessionParams{
		Account: stripe.String(accountID),
		Components: &stripe.AccountSessionComponentsParams{
			AccountOnboarding: &stripe.AccountSessionComponentsAccountOnboardingParams{
				Enabled: stripe.Bool(true),
				Features: &stripe.AccountSessionComponentsAccountOnboardingFeaturesParams{
					ExternalAccountCollection: stripe.Bool(true),
				},
			},
		},
	}

	session, err := accountsession.New(params)
	if err != nil {
		return "", fmt.Errorf("create account session: %w", err)
	}

	return session.ClientSecret, nil
}
