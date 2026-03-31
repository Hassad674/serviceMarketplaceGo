package response

import (
	"marketplace-backend/internal/domain/payment"
)

type PaymentInfoResponse struct {
	ID              string `json:"id"`
	UserID          string `json:"user_id"`
	StripeAccountID string `json:"stripe_account_id"`
	StripeVerified  bool   `json:"stripe_verified"`
	ChargesEnabled  bool   `json:"charges_enabled"`
	PayoutsEnabled  bool   `json:"payouts_enabled"`
	BusinessType    string `json:"stripe_business_type"`
	Country         string `json:"stripe_country"`
	DisplayName     string `json:"stripe_display_name"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
}

type PaymentInfoStatusResponse struct {
	Complete bool `json:"complete"`
}

func NewPaymentInfoResponse(p *payment.PaymentInfo) PaymentInfoResponse {
	return PaymentInfoResponse{
		ID:              p.ID.String(),
		UserID:          p.UserID.String(),
		StripeAccountID: p.StripeAccountID,
		StripeVerified:  p.StripeVerified,
		ChargesEnabled:  p.ChargesEnabled,
		PayoutsEnabled:  p.PayoutsEnabled,
		BusinessType:    p.StripeBusinessType,
		Country:         p.StripeCountry,
		DisplayName:     p.StripeDisplayName,
		CreatedAt:       p.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:       p.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}
