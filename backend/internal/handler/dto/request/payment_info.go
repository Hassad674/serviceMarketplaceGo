package request

type SavePaymentInfoRequest struct {
	Email       string `json:"email"` // user email for Stripe KYC
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	DateOfBirth string `json:"date_of_birth"`
	Nationality string `json:"nationality"`
	Address     string `json:"address"`
	City        string `json:"city"`
	PostalCode  string `json:"postal_code"`

	IsBusiness         bool   `json:"is_business"`
	BusinessName       string `json:"business_name"`
	BusinessAddress    string `json:"business_address"`
	BusinessCity       string `json:"business_city"`
	BusinessPostalCode string `json:"business_postal_code"`
	BusinessCountry    string `json:"business_country"`
	TaxID              string `json:"tax_id"`
	VATNumber          string `json:"vat_number"`
	RoleInCompany      string `json:"role_in_company"`

	Phone          string `json:"phone"`
	ActivitySector string `json:"activity_sector"`

	IBAN          string `json:"iban"`
	BIC           string `json:"bic"`
	AccountNumber string `json:"account_number"`
	RoutingNumber string `json:"routing_number"`
	AccountHolder string `json:"account_holder"`
	BankCountry   string `json:"bank_country"`
}
