package stripe

import (
	"context"
	"fmt"
	"log/slog"

	stripe "github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/account"
	"github.com/stripe/stripe-go/v82/accountlink"
	"github.com/stripe/stripe-go/v82/bankaccount"
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
		Email:        stripe.String(email),
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

	// External bank account
	if info.IBAN != "" {
		// IBAN-based (EU, UK, etc.)
		acctParams.AddExtra("external_account[object]", "bank_account")
		acctParams.AddExtra("external_account[country]", country)
		acctParams.AddExtra("external_account[currency]", countryToCurrency(country))
		acctParams.AddExtra("external_account[account_holder_name]", info.AccountHolder)
		acctParams.AddExtra("external_account[account_number]", info.IBAN)
	} else if info.AccountNumber != "" && info.RoutingNumber != "" {
		// Local bank (US, SG, IN, AU, CA, etc.)
		acctParams.AddExtra("external_account[object]", "bank_account")
		acctParams.AddExtra("external_account[country]", country)
		acctParams.AddExtra("external_account[currency]", countryToCurrency(country))
		acctParams.AddExtra("external_account[account_holder_name]", info.AccountHolder)
		acctParams.AddExtra("external_account[account_number]", info.AccountNumber)
		acctParams.AddExtra("external_account[routing_number]", info.RoutingNumber)
	}

	acct, err := account.New(acctParams)
	if err != nil {
		return "", fmt.Errorf("create stripe account: %w", err)
	}

	return acct.ID, nil
}

// UpdateConnectedAccount updates an existing Stripe account with new data via Account Token.
// Also updates the external bank account if bank details have changed.
func (s *Service) UpdateConnectedAccount(_ context.Context, accountID string, info *payment.PaymentInfo, tosIP string, email string) error {
	tok, err := createAccountToken(info, tosIP, email)
	if err != nil {
		return fmt.Errorf("create account token for update: %w", err)
	}

	params := &stripe.AccountParams{
		AccountToken: stripe.String(tok),
	}

	mcc := info.ActivitySector
	if mcc == "" {
		mcc = "8999"
	}
	params.BusinessProfile = &stripe.AccountBusinessProfileParams{
		MCC: stripe.String(mcc),
	}

	_, err = account.Update(accountID, params)
	if err != nil {
		return fmt.Errorf("update stripe account: %w", err)
	}

	// Update external bank account
	if err := s.updateExternalAccount(accountID, info); err != nil {
		slog.Warn("failed to update external account", "account_id", accountID, "error", err)
	}

	return nil
}

// updateExternalAccount replaces the bank account on a Stripe connected account.
func (s *Service) updateExternalAccount(accountID string, info *payment.PaymentInfo) error {
	country := resolveCountryCode(info)
	currency := countryToCurrency(country)

	// Determine new bank details
	var newAccountNumber string
	if info.IBAN != "" {
		newAccountNumber = info.IBAN
	} else if info.AccountNumber != "" {
		newAccountNumber = info.AccountNumber
	}

	if newAccountNumber == "" {
		return nil // no bank details to update
	}

	// Delete existing external accounts first
	acct, err := account.GetByID(accountID, &stripe.AccountParams{})
	if err != nil {
		return fmt.Errorf("get account: %w", err)
	}

	for _, ea := range acct.ExternalAccounts.Data {
		delParams := &stripe.BankAccountParams{
			Account: stripe.String(accountID),
		}
		_, err := bankaccount.Del(ea.ID, delParams)
		if err != nil {
			slog.Warn("failed to delete old external account", "id", ea.ID, "error", err)
		}
	}

	// Create new external account
	newParams := &stripe.BankAccountParams{
		Account:           stripe.String(accountID),
		AccountHolderName: stripe.String(info.AccountHolder),
		Country:           stripe.String(country),
		Currency:          stripe.String(currency),
	}

	if info.IBAN != "" {
		newParams.AccountNumber = stripe.String(info.IBAN)
	} else {
		newParams.AccountNumber = stripe.String(info.AccountNumber)
		newParams.RoutingNumber = stripe.String(info.RoutingNumber)
	}

	_, err = bankaccount.New(newParams)
	if err != nil {
		return fmt.Errorf("create external account: %w", err)
	}

	return nil
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
		companyAddr := &stripe.AddressParams{
			Line1:      stripe.String(info.BusinessAddress),
			City:       stripe.String(info.BusinessCity),
			PostalCode: stripe.String(info.BusinessPostalCode),
			Country:    stripe.String(country),
		}
		// Add state for countries that require it (US, AU, IN, etc.)
		if v := getExtraField(info.ExtraFields, "company.address.state", "business_state"); v != "" {
			companyAddr.State = stripe.String(v)
		}
		params.Account.Company = &stripe.AccountCompanyParams{
			Name:    stripe.String(info.BusinessName),
			Phone:   stripe.String(info.Phone),
			Address: companyAddr,
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
		applyExtraFieldsToIndividual(params.Account.Individual, info.ExtraFields)
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

// GetAccountRequirements returns the currently_due requirements for a connected account.
func (s *Service) GetAccountRequirements(ctx context.Context, accountID string) ([]string, error) {
	acct, err := account.GetByID(accountID, nil)
	if err != nil {
		return nil, fmt.Errorf("get stripe account: %w", err)
	}
	return acct.Requirements.CurrentlyDue, nil
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
