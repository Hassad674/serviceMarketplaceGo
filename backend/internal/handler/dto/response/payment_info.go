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

	Phone          string `json:"phone"`
	ActivitySector string `json:"activity_sector"`

	IsSelfRepresentative bool `json:"is_self_representative"`
	IsSelfDirector       bool `json:"is_self_director"`
	NoMajorOwners        bool `json:"no_major_owners"`
	IsSelfExecutive      bool `json:"is_self_executive"`

	BusinessPersons []BusinessPersonResponse `json:"business_persons"`

	IBAN          string `json:"iban"`
	BIC           string `json:"bic"`
	AccountNumber string `json:"account_number"`
	RoutingNumber string `json:"routing_number"`
	AccountHolder string `json:"account_holder"`
	BankCountry   string `json:"bank_country"`

	StripeAccountID string `json:"stripe_account_id"`
	StripeVerified  bool   `json:"stripe_verified"`

	Country     string            `json:"country"`
	ExtraFields map[string]string `json:"extra_fields"`

	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type BusinessPersonResponse struct {
	Role        string `json:"role"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	DateOfBirth string `json:"date_of_birth"`
	Email       string `json:"email"`
	Phone       string `json:"phone"`
	Address     string `json:"address"`
	City        string `json:"city"`
	PostalCode  string `json:"postal_code"`
	Title       string `json:"title"`
}

type PaymentInfoStatusResponse struct {
	Complete bool `json:"complete"`
}

func NewPaymentInfoResponse(p *payment.PaymentInfo, persons []*payment.BusinessPerson) PaymentInfoResponse {
	bpList := make([]BusinessPersonResponse, 0, len(persons))
	for _, bp := range persons {
		dob := ""
		if !bp.DateOfBirth.IsZero() {
			dob = bp.DateOfBirth.Format("2006-01-02")
		}
		bpList = append(bpList, BusinessPersonResponse{
			Role:        string(bp.Role),
			FirstName:   bp.FirstName,
			LastName:    bp.LastName,
			DateOfBirth: dob,
			Email:       bp.Email,
			Phone:       bp.Phone,
			Address:     bp.Address,
			City:        bp.City,
			PostalCode:  bp.PostalCode,
			Title:       bp.Title,
		})
	}

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
		Phone:                p.Phone,
		ActivitySector:       p.ActivitySector,
		IsSelfRepresentative: p.IsSelfRepresentative,
		IsSelfDirector:       p.IsSelfDirector,
		NoMajorOwners:        p.NoMajorOwners,
		IsSelfExecutive:      p.IsSelfExecutive,
		BusinessPersons:      bpList,
		IBAN:               p.IBAN,
		BIC:                p.BIC,
		AccountNumber:      p.AccountNumber,
		RoutingNumber:      p.RoutingNumber,
		AccountHolder:      p.AccountHolder,
		BankCountry:        p.BankCountry,
		StripeAccountID:    p.StripeAccountID,
		StripeVerified:     p.StripeVerified,
		Country:            p.Country,
		ExtraFields:        p.ExtraFields,
		CreatedAt:          p.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:          p.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}
