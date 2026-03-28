package response

import (
	"marketplace-backend/internal/domain/payment"
)

type PaymentInfoResponse struct {
	ID     string `json:"id"`
	UserID string `json:"user_id"`

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

	IBAN          string `json:"iban"`
	BIC           string `json:"bic"`
	AccountNumber string `json:"account_number"`
	RoutingNumber string `json:"routing_number"`
	AccountHolder string `json:"account_holder"`
	BankCountry   string `json:"bank_country"`

	StripeAccountID string `json:"stripe_account_id"`
	StripeVerified  bool   `json:"stripe_verified"`

	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type PaymentInfoStatusResponse struct {
	Complete bool `json:"complete"`
}

func NewPaymentInfoResponse(p *payment.PaymentInfo) PaymentInfoResponse {
	return PaymentInfoResponse{
		ID:                 p.ID.String(),
		UserID:             p.UserID.String(),
		FirstName:          p.FirstName,
		LastName:           p.LastName,
		DateOfBirth:        p.DateOfBirth.Format("2006-01-02"),
		Nationality:        p.Nationality,
		Address:            p.Address,
		City:               p.City,
		PostalCode:         p.PostalCode,
		IsBusiness:         p.IsBusiness,
		BusinessName:       p.BusinessName,
		BusinessAddress:    p.BusinessAddress,
		BusinessCity:       p.BusinessCity,
		BusinessPostalCode: p.BusinessPostalCode,
		BusinessCountry:    p.BusinessCountry,
		TaxID:              p.TaxID,
		VATNumber:          p.VATNumber,
		RoleInCompany:      p.RoleInCompany,
		IBAN:               p.IBAN,
		BIC:                p.BIC,
		AccountNumber:      p.AccountNumber,
		RoutingNumber:      p.RoutingNumber,
		AccountHolder:      p.AccountHolder,
		BankCountry:        p.BankCountry,
		StripeAccountID:    p.StripeAccountID,
		StripeVerified:     p.StripeVerified,
		CreatedAt:          p.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:          p.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}
