package request

// UpdateReferrerProfileRequest is the request body for the core
// update endpoint on a referrer profile.
type UpdateReferrerProfileRequest struct {
	Title    string `json:"title"`
	About    string `json:"about"`
	VideoURL string `json:"video_url"`
}

// UpdateReferrerAvailabilityRequest is the request body for the
// availability endpoint on a referrer profile.
type UpdateReferrerAvailabilityRequest struct {
	AvailabilityStatus string `json:"availability_status"`
}

// UpdateReferrerExpertiseRequest is the request body for the
// expertise endpoint on a referrer profile.
type UpdateReferrerExpertiseRequest struct {
	Domains []string `json:"domains"`
}

// UpsertReferrerPricingRequest is the request body for the pricing
// upsert endpoint on a referrer profile.
type UpsertReferrerPricingRequest struct {
	Type       string `json:"type"`
	MinAmount  int64  `json:"min_amount"`
	MaxAmount  *int64 `json:"max_amount"`
	Currency   string `json:"currency"`
	Note       string `json:"note"`
	Negotiable bool   `json:"negotiable"`
}
