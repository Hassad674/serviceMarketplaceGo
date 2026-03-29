package stripe

import (
	"context"
	"fmt"

	stripe "github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/account"
	"github.com/stripe/stripe-go/v82/token"

	"marketplace-backend/internal/domain/payment"
)

func (s *Service) CreateConnectedAccount(ctx context.Context, info *payment.PaymentInfo, tosIP string, email string) (string, error) {
	// Step 1: Create an account token with the person/company data
	accountToken, err := createAccountToken(info, tosIP, email)
	if err != nil {
		return "", fmt.Errorf("create account token: %w", err)
	}

	// Step 2: Create the connected account using the token
	country := resolveCountryCode(info)
	mcc := info.ActivitySector
	if mcc == "" {
		mcc = "8999"
	}
	acctParams := &stripe.AccountParams{
		Type:         stripe.String(string(stripe.AccountTypeCustom)),
		Country:      stripe.String(country),
		AccountToken: stripe.String(accountToken),
		BusinessProfile: &stripe.AccountBusinessProfileParams{
			MCC: stripe.String(mcc),
			URL: stripe.String("https://service-marketplace-go.vercel.app"),
		},
		Capabilities: &stripe.AccountCapabilitiesParams{
			CardPayments: &stripe.AccountCapabilitiesCardPaymentsParams{
				Requested: stripe.Bool(true),
			},
			Transfers: &stripe.AccountCapabilitiesTransfersParams{
				Requested: stripe.Bool(true),
			},
		},
	}

	// External bank account via IBAN
	if info.IBAN != "" {
		acctParams.AddExtra("external_account[object]", "bank_account")
		acctParams.AddExtra("external_account[country]", country)
		acctParams.AddExtra("external_account[currency]", "eur")
		acctParams.AddExtra("external_account[account_holder_name]", info.AccountHolder)
		acctParams.AddExtra("external_account[account_number]", info.IBAN)
	}

	acct, err := account.New(acctParams)
	if err != nil {
		return "", fmt.Errorf("create stripe account: %w", err)
	}

	return acct.ID, nil
}

func createAccountToken(info *payment.PaymentInfo, tosIP string, email string) (string, error) {
	params := &stripe.TokenParams{
		Account: &stripe.TokenAccountParams{
			TOSShownAndAccepted: stripe.Bool(true),
		},
	}

	country := resolveCountryCode(info)

	if info.IsBusiness {
		params.Account.BusinessType = stripe.String("company")
		params.Account.Company = &stripe.AccountCompanyParams{
			Name:  stripe.String(info.BusinessName),
			Phone: stripe.String(info.Phone),
			Address: &stripe.AddressParams{
				Line1:      stripe.String(info.BusinessAddress),
				City:       stripe.String(info.BusinessCity),
				PostalCode: stripe.String(info.BusinessPostalCode),
				Country:    stripe.String(country),
			},
			TaxID: stripe.String(info.TaxID),
		}
	} else {
		params.Account.BusinessType = stripe.String("individual")
		params.Account.Individual = &stripe.PersonParams{
			FirstName: stripe.String(info.FirstName),
			LastName:  stripe.String(info.LastName),
			Email:     stripe.String(email),
			Phone:     stripe.String(info.Phone),
			DOB: &stripe.PersonDOBParams{
				Day:   stripe.Int64(int64(info.DateOfBirth.Day())),
				Month: stripe.Int64(int64(info.DateOfBirth.Month())),
				Year:  stripe.Int64(int64(info.DateOfBirth.Year())),
			},
			Address: &stripe.AddressParams{
				Line1:      stripe.String(info.Address),
				City:       stripe.String(info.City),
				PostalCode: stripe.String(info.PostalCode),
				Country:    stripe.String(country),
			},
		}
	}

	tok, err := token.New(params)
	if err != nil {
		return "", err
	}

	return tok.ID, nil
}

func (s *Service) GetAccountStatus(ctx context.Context, accountID string) (bool, error) {
	acct, err := account.GetByID(accountID, nil)
	if err != nil {
		return false, fmt.Errorf("get stripe account: %w", err)
	}
	return acct.ChargesEnabled && acct.PayoutsEnabled, nil
}

// GetIdentityVerificationStatus returns the verification status and the verified file ID.
func (s *Service) GetIdentityVerificationStatus(ctx context.Context, accountID string) (status string, verifiedFileID string, err error) {
	acct, err := account.GetByID(accountID, nil)
	if err != nil {
		return "", "", fmt.Errorf("get stripe account: %w", err)
	}
	if acct.Individual != nil && acct.Individual.Verification != nil {
		ver := acct.Individual.Verification
		frontID := ""
		if ver.Document != nil && ver.Document.Front != nil {
			frontID = ver.Document.Front.ID
		}
		return string(ver.Status), frontID, nil
	}
	return "unverified", "", nil
}

// resolveCountryCode returns a 2-letter country code from the payment info.
func resolveCountryCode(info *payment.PaymentInfo) string {
	if info.BankCountry != "" && len(info.BankCountry) == 2 {
		return info.BankCountry
	}
	if info.Nationality != "" && len(info.Nationality) == 2 {
		return info.Nationality
	}
	return "FR"
}
