package stripe

import (
	"context"
	"fmt"

	stripe "github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/account"
	"github.com/stripe/stripe-go/v82/accountlink"

	"marketplace-backend/internal/domain/payment"
	portservice "marketplace-backend/internal/port/service"
)

// GetAccount retrieves a connected account's capability status.
func (s *Service) GetAccount(ctx context.Context, accountID string) (*portservice.StripeAccountInfo, error) {
	acct, err := account.GetByID(accountID, nil)
	if err != nil {
		return nil, fmt.Errorf("get stripe account: %w", err)
	}
	return &portservice.StripeAccountInfo{
		ChargesEnabled: acct.ChargesEnabled,
		PayoutsEnabled: acct.PayoutsEnabled,
	}, nil
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

	// For individual accounts: check individual.verification
	if acct.Individual != nil && acct.Individual.Verification != nil {
		ver := acct.Individual.Verification
		frontID := ""
		if ver.Document != nil && ver.Document.Front != nil {
			frontID = ver.Document.Front.ID
		}
		return string(ver.Status), frontID, nil
	}

	// For company accounts: if charges+payouts enabled, consider verified
	if acct.ChargesEnabled && acct.PayoutsEnabled {
		return "verified", "", nil
	}

	// Company account not yet fully active — don't mark as rejected, keep pending
	if acct.BusinessType == stripe.AccountBusinessTypeCompany {
		return "pending", "", nil
	}

	return "unverified", "", nil
}

// GetAccountFullStatus returns verification status, charges_enabled, and payouts_enabled in one API call.
func (s *Service) GetAccountFullStatus(ctx context.Context, accountID string) (*portservice.AccountFullStatus, error) {
	acct, err := account.GetByID(accountID, nil)
	if err != nil {
		return nil, fmt.Errorf("get stripe account: %w", err)
	}

	result := &portservice.AccountFullStatus{
		ChargesEnabled: acct.ChargesEnabled,
		PayoutsEnabled: acct.PayoutsEnabled,
	}

	// For individual accounts: check individual.verification
	if acct.Individual != nil && acct.Individual.Verification != nil {
		ver := acct.Individual.Verification
		result.VerificationStatus = string(ver.Status)
		if ver.Document != nil && ver.Document.Front != nil {
			result.VerifiedFileID = ver.Document.Front.ID
		}
	} else if acct.ChargesEnabled && acct.PayoutsEnabled {
		result.VerificationStatus = "verified"
	} else if acct.BusinessType == stripe.AccountBusinessTypeCompany {
		result.VerificationStatus = "pending"
	} else {
		result.VerificationStatus = "unverified"
	}

	return result, nil
}

// GetAccountRequirements returns the full account requirements for a connected account.
func (s *Service) GetAccountRequirements(ctx context.Context, accountID string) (*payment.AccountRequirements, error) {
	acct, err := account.GetByID(accountID, nil)
	if err != nil {
		return nil, fmt.Errorf("get stripe account: %w", err)
	}
	reqs := acct.Requirements
	var errors []payment.RequirementError
	for _, e := range reqs.Errors {
		errors = append(errors, payment.RequirementError{
			Code:        string(e.Code),
			Reason:      e.Reason,
			Requirement: e.Requirement,
		})
	}
	return &payment.AccountRequirements{
		CurrentlyDue:        reqs.CurrentlyDue,
		EventuallyDue:       reqs.EventuallyDue,
		PastDue:             reqs.PastDue,
		PendingVerification: reqs.PendingVerification,
		CurrentDeadline:     reqs.CurrentDeadline,
		Errors:              errors,
	}, nil
}

// UpdatePayoutSchedule mutates only the payout schedule on a connected
// account, leaving every other account setting (KYC, business profile,
// external bank account, capabilities) untouched. Used by the
// stripe-payout-schedule-backfill command and by any defensive code path
// that wants to re-assert the manual policy without rebuilding the full
// AccountParams payload.
//
// The Stripe SDK call is idempotent at the field level: posting the
// same interval value again is a no-op for billing.
func (s *Service) UpdatePayoutSchedule(_ context.Context, accountID, interval string) error {
	if accountID == "" {
		return fmt.Errorf("update payout schedule: empty account id")
	}
	if interval == "" {
		return fmt.Errorf("update payout schedule: empty interval")
	}
	params := &stripe.AccountParams{
		Settings: &stripe.AccountSettingsParams{
			Payouts: &stripe.AccountSettingsPayoutsParams{
				Schedule: &stripe.AccountSettingsPayoutsScheduleParams{
					Interval: stripe.String(interval),
				},
			},
		},
	}
	if _, err := account.Update(accountID, params); err != nil {
		return fmt.Errorf("update payout schedule on %s: %w", accountID, err)
	}
	return nil
}

// CreateAccountLink generates a Stripe-hosted link for the provider to complete requirements.
func (s *Service) CreateAccountLink(ctx context.Context, accountID, returnURL, refreshURL string) (string, error) {
	params := &stripe.AccountLinkParams{
		Account:    stripe.String(accountID),
		Type:       stripe.String(string(stripe.AccountLinkTypeAccountUpdate)),
		ReturnURL:  stripe.String(returnURL),
		RefreshURL: stripe.String(refreshURL),
		CollectionOptions: &stripe.AccountLinkCollectionOptionsParams{
			Fields: stripe.String(string(stripe.AccountLinkCollectCurrentlyDue)),
		},
	}

	link, err := accountlink.New(params)
	if err != nil {
		return "", fmt.Errorf("create account link: %w", err)
	}
	return link.URL, nil
}

// applyExtraFieldsToIndividual sets country-specific extra fields on the Stripe individual params.
// Extra fields may be stored with short keys ("state") or full Stripe paths ("individual.address.state").
func applyExtraFieldsToIndividual(p *stripe.PersonParams, extra map[string]string) {
	if extra == nil {
		return
	}
	get := func(keys ...string) string {
		for _, k := range keys {
			if v, ok := extra[k]; ok && v != "" {
				return v
			}
		}
		return ""
	}

	if v := get("id_number", "individual.id_number"); v != "" {
		p.IDNumber = stripe.String(v)
	}
	if v := get("ssn_last_4", "individual.ssn_last_4"); v != "" {
		p.SSNLast4 = stripe.String(v)
	}
	if v := get("state", "individual.address.state"); v != "" && p.Address != nil {
		p.Address.State = stripe.String(v)
	}
	if v := get("first_name_kana", "individual.first_name_kana"); v != "" {
		p.FirstNameKana = stripe.String(v)
	}
	if v := get("last_name_kana", "individual.last_name_kana"); v != "" {
		p.LastNameKana = stripe.String(v)
	}
	if v := get("first_name_kanji", "individual.first_name_kanji"); v != "" {
		p.FirstNameKanji = stripe.String(v)
	}
	if v := get("last_name_kanji", "individual.last_name_kanji"); v != "" {
		p.LastNameKanji = stripe.String(v)
	}
}

// getExtraField looks up a value in extra_fields by multiple possible keys.
func getExtraField(extra map[string]string, keys ...string) string {
	for _, k := range keys {
		if v, ok := extra[k]; ok && v != "" {
			return v
		}
	}
	return ""
}

// resolveCountryCode returns a 2-letter country code from the payment info.
// Prefers the explicit Country field, then BankCountry, then Nationality.
func resolveCountryCode(info *payment.PaymentInfo) string {
	if info.Country != "" && len(info.Country) == 2 {
		return info.Country
	}
	if info.BankCountry != "" && len(info.BankCountry) == 2 {
		return info.BankCountry
	}
	if info.Nationality != "" && len(info.Nationality) == 2 {
		return info.Nationality
	}
	return "FR"
}

// countryToCurrency maps ISO country codes to their default Stripe currencies.
func countryToCurrency(country string) string {
	currencies := map[string]string{
		"FR": "eur", "DE": "eur", "IT": "eur", "ES": "eur", "NL": "eur",
		"BE": "eur", "AT": "eur", "PT": "eur", "FI": "eur", "IE": "eur",
		"LU": "eur", "GR": "eur", "SK": "eur", "SI": "eur", "EE": "eur",
		"LV": "eur", "LT": "eur", "CY": "eur", "MT": "eur",
		"US": "usd", "GB": "gbp", "JP": "jpy", "BR": "brl", "CA": "cad",
		"AU": "aud", "NZ": "nzd", "CH": "chf", "SE": "sek", "NO": "nok",
		"DK": "dkk", "PL": "pln", "CZ": "czk", "HU": "huf", "RO": "ron",
		"BG": "bgn", "HR": "eur", "SG": "sgd", "HK": "hkd", "MY": "myr",
		"TH": "thb", "MX": "mxn", "IN": "inr", "AE": "aed",
	}
	if cur, ok := currencies[country]; ok {
		return cur
	}
	return "eur"
}
