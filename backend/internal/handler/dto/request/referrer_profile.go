package request

// UpdateReferrerProfileRequest is the request body for the core
// update endpoint on a referrer profile.
type UpdateReferrerProfileRequest struct {
	Title    string `json:"title" validate:"omitempty,max=200"`
	About    string `json:"about" validate:"omitempty,max=5000"`
	VideoURL string `json:"video_url" validate:"omitempty,url,max=2048"`
}

// UpdateReferrerAvailabilityRequest is the request body for the
// availability endpoint on a referrer profile.
type UpdateReferrerAvailabilityRequest struct {
	AvailabilityStatus string `json:"availability_status" validate:"required,min=1,max=50"`
}

// UpdateReferrerExpertiseRequest is the request body for the
// expertise endpoint on a referrer profile.
type UpdateReferrerExpertiseRequest struct {
	Domains []string `json:"domains" validate:"omitempty,max=20,dive,min=1,max=100"`
}

// UpsertReferrerPricingRequest is the request body for the pricing
// upsert endpoint on a referrer profile.
type UpsertReferrerPricingRequest struct {
	Type       string `json:"type" validate:"required,min=1,max=50"`
	MinAmount  int64  `json:"min_amount" validate:"gte=0,lte=999999999"`
	MaxAmount  *int64 `json:"max_amount" validate:"omitempty,gte=0,lte=999999999"`
	Currency   string `json:"currency" validate:"required,len=3"`
	Note       string `json:"note" validate:"omitempty,max=2000"`
	Negotiable bool   `json:"negotiable"`
}
