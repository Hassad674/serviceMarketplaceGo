package stripe

import (
	"context"
	"fmt"
	"time"

	stripe "github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/account"

	"marketplace-backend/internal/domain/payment"
)

func (s *Service) CreateConnectedAccount(ctx context.Context, info *payment.PaymentInfo, tosIP string) (string, error) {
	params := &stripe.AccountParams{
		Type:    stripe.String(string(stripe.AccountTypeCustom)),
		Country: stripe.String(resolveCountryCode(info)),
		Capabilities: &stripe.AccountCapabilitiesParams{
			CardPayments: &stripe.AccountCapabilitiesCardPaymentsParams{
				Requested: stripe.Bool(true),
			},
			Transfers: &stripe.AccountCapabilitiesTransfersParams{
				Requested: stripe.Bool(true),
			},
		},
		TOSAcceptance: &stripe.AccountTOSAcceptanceParams{
			Date: stripe.Int64(time.Now().Unix()),
			IP:   stripe.String(tosIP),
		},
	}

	if info.IsBusiness {
		params.BusinessType = stripe.String("company")
		params.Company = &stripe.AccountCompanyParams{
			Name: stripe.String(info.BusinessName),
			Address: &stripe.AddressParams{
				Line1:      stripe.String(info.BusinessAddress),
				City:       stripe.String(info.BusinessCity),
				PostalCode: stripe.String(info.BusinessPostalCode),
				Country:    stripe.String(resolveCountryCode(info)),
			},
			TaxID: stripe.String(info.TaxID),
		}
	} else {
		params.BusinessType = stripe.String("individual")
	}

	params.Individual = &stripe.PersonParams{
		FirstName: stripe.String(info.FirstName),
		LastName:  stripe.String(info.LastName),
		DOB: &stripe.PersonDOBParams{
			Day:   stripe.Int64(int64(info.DateOfBirth.Day())),
			Month: stripe.Int64(int64(info.DateOfBirth.Month())),
			Year:  stripe.Int64(int64(info.DateOfBirth.Year())),
		},
		Address: &stripe.AddressParams{
			Line1:      stripe.String(info.Address),
			City:       stripe.String(info.City),
			PostalCode: stripe.String(info.PostalCode),
			Country:    stripe.String(resolveCountryCode(info)),
		},
	}

	// External bank account
	if info.IBAN != "" {
		params.ExternalAccount = &stripe.AccountExternalAccountParams{
			Token: stripe.String(""), // will use BankAccount params below
		}
		// Use raw params for IBAN-based bank account
		params.AddExtra("external_account[object]", "bank_account")
		params.AddExtra("external_account[country]", resolveCountryCode(info))
		params.AddExtra("external_account[currency]", "eur")
		params.AddExtra("external_account[account_holder_name]", info.AccountHolder)
		params.AddExtra("external_account[account_number]", info.IBAN)
	}

	acct, err := account.New(params)
	if err != nil {
		return "", fmt.Errorf("create stripe account: %w", err)
	}

	return acct.ID, nil
}

func (s *Service) GetAccountStatus(ctx context.Context, accountID string) (bool, error) {
	acct, err := account.GetByID(accountID, nil)
	if err != nil {
		return false, fmt.Errorf("get stripe account: %w", err)
	}
	return acct.ChargesEnabled && acct.PayoutsEnabled, nil
}

// resolveCountryCode returns a 2-letter country code from the payment info.
func resolveCountryCode(info *payment.PaymentInfo) string {
	if info.BankCountry != "" && len(info.BankCountry) == 2 {
		return info.BankCountry
	}
	if info.Nationality != "" && len(info.Nationality) == 2 {
		return info.Nationality
	}
	return "FR" // default
}
