package stripe

import (
	"context"
	"fmt"

	stripe "github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/countryspec"

	"marketplace-backend/internal/domain/payment"
)

// GetCountrySpec retrieves the field requirements for a single country.
func (s *Service) GetCountrySpec(_ context.Context, country string) (*payment.CountryFieldSpec, error) {
	spec, err := countryspec.Get(country, nil)
	if err != nil {
		return nil, fmt.Errorf("get country spec %s: %w", country, err)
	}
	return mapCountrySpec(spec), nil
}

// ListAllCountrySpecs retrieves specs for all Stripe-supported countries.
func (s *Service) ListAllCountrySpecs(_ context.Context) ([]*payment.CountryFieldSpec, error) {
	params := &stripe.CountrySpecListParams{}
	params.Filters.AddFilter("limit", "", "100")

	var specs []*payment.CountryFieldSpec
	iter := countryspec.List(params)
	for iter.Next() {
		specs = append(specs, mapCountrySpec(iter.CountrySpec()))
	}
	if err := iter.Err(); err != nil {
		return nil, fmt.Errorf("list country specs: %w", err)
	}
	return specs, nil
}

// mapCountrySpec converts a Stripe CountrySpec to our domain type.
func mapCountrySpec(spec *stripe.CountrySpec) *payment.CountryFieldSpec {
	result := &payment.CountryFieldSpec{
		Country:         spec.ID,
		DefaultCurrency: string(spec.DefaultCurrency),
	}

	if indiv, ok := spec.VerificationFields[stripe.AccountBusinessTypeIndividual]; ok && indiv != nil {
		result.IndividualMinimum = extractFields(indiv.Minimum)
		result.IndividualAdditional = extractFields(indiv.AdditionalFields)
	}
	if company, ok := spec.VerificationFields[stripe.AccountBusinessTypeCompany]; ok && company != nil {
		result.CompanyMinimum = extractFields(company.Minimum)
		result.CompanyAdditional = extractFields(company.AdditionalFields)
	}

	return result
}

// extractFields converts Stripe's field slice to a string slice.
func extractFields(fields []string) []string {
	if fields == nil {
		return []string{}
	}
	return fields
}
