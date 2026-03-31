package stripe

import (
	"context"
	"fmt"

	stripe "github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/account"

	portservice "marketplace-backend/internal/port/service"
)

// CreateMinimalAccount creates a minimal Stripe Custom account for embedded onboarding.
func (s *Service) CreateMinimalAccount(_ context.Context, country, email string) (string, error) {
	if country == "" {
		country = "FR"
	}

	params := &stripe.AccountParams{
		Type:    stripe.String(string(stripe.AccountTypeCustom)),
		Country: stripe.String(country),
		Email:   stripe.String(email),
		Capabilities: &stripe.AccountCapabilitiesParams{
			CardPayments: &stripe.AccountCapabilitiesCardPaymentsParams{
				Requested: stripe.Bool(true),
			},
			Transfers: &stripe.AccountCapabilitiesTransfersParams{
				Requested: stripe.Bool(true),
			},
		},
		BusinessProfile: &stripe.AccountBusinessProfileParams{
			URL: stripe.String("https://service-marketplace-go.vercel.app"),
		},
	}

	acct, err := account.New(params)
	if err != nil {
		return "", fmt.Errorf("create minimal stripe account: %w", err)
	}

	return acct.ID, nil
}

// GetAccountStatus checks whether a connected account is verified.
func (s *Service) GetAccountStatus(_ context.Context, accountID string) (bool, error) {
	acct, err := account.GetByID(accountID, nil)
	if err != nil {
		return false, fmt.Errorf("get stripe account: %w", err)
	}
	return acct.ChargesEnabled && acct.PayoutsEnabled, nil
}

// GetFullAccount retrieves detailed account info for syncing to the database.
func (s *Service) GetFullAccount(_ context.Context, accountID string) (*portservice.StripeAccountInfo, error) {
	acct, err := account.GetByID(accountID, nil)
	if err != nil {
		return nil, fmt.Errorf("get stripe account: %w", err)
	}

	info := &portservice.StripeAccountInfo{
		ChargesEnabled: acct.ChargesEnabled,
		PayoutsEnabled: acct.PayoutsEnabled,
		Country:        acct.Country,
		BusinessType:   string(acct.BusinessType),
	}

	if acct.Requirements != nil {
		info.CurrentlyDue = acct.Requirements.CurrentlyDue
	}

	info.DisplayName = resolveDisplayName(acct)

	return info, nil
}

// resolveDisplayName extracts a human-readable name from the account.
func resolveDisplayName(acct *stripe.Account) string {
	if acct.Company != nil && acct.Company.Name != "" {
		return acct.Company.Name
	}
	if acct.Individual != nil {
		name := ""
		if acct.Individual.FirstName != "" {
			name = acct.Individual.FirstName
		}
		if acct.Individual.LastName != "" {
			if name != "" {
				name += " "
			}
			name += acct.Individual.LastName
		}
		if name != "" {
			return name
		}
	}
	return ""
}
