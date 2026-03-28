package response

type StripeConfigResponse struct {
	PublishableKey string `json:"publishable_key"`
}

type PaymentIntentResponse struct {
	ClientSecret    string         `json:"client_secret"`
	PaymentRecordID string         `json:"payment_record_id"`
	Amounts         PaymentAmounts `json:"amounts"`
}

type PaymentAmounts struct {
	ProposalAmount int64 `json:"proposal_amount"`
	StripeFee      int64 `json:"stripe_fee"`
	PlatformFee    int64 `json:"platform_fee"`
	ClientTotal    int64 `json:"client_total"`
	ProviderPayout int64 `json:"provider_payout"`
}
