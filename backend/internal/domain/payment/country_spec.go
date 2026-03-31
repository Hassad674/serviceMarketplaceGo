package payment

// CountryFieldSpec holds the Stripe field requirements for a specific country.
type CountryFieldSpec struct {
	Country              string
	IndividualMinimum    []string
	IndividualAdditional []string
	CompanyMinimum       []string
	CompanyAdditional    []string
	DefaultCurrency      string
}
