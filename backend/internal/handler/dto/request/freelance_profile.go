package request

// UpdateFreelanceProfileRequest is the request body for the "update
// core" endpoint on a freelance profile. Empty strings are valid —
// they clear the corresponding column.
type UpdateFreelanceProfileRequest struct {
	Title    string `json:"title" validate:"omitempty,max=200"`
	About    string `json:"about" validate:"omitempty,max=5000"`
	VideoURL string `json:"video_url" validate:"omitempty,url,max=2048"`
}

// UpdateFreelanceAvailabilityRequest is the request body for the
// availability endpoint on a freelance profile. Single required
// field so the JSON shape stays symmetrical with the referrer side.
type UpdateFreelanceAvailabilityRequest struct {
	AvailabilityStatus string `json:"availability_status" validate:"required,min=1,max=50"`
}

// UpdateFreelanceExpertiseRequest is the request body for the
// expertise endpoint on a freelance profile. Empty slice clears
// the declared expertise.
type UpdateFreelanceExpertiseRequest struct {
	Domains []string `json:"domains" validate:"omitempty,max=20,dive,min=1,max=100"`
}

// UpsertFreelancePricingRequest is the request body for the
// pricing upsert endpoint on a freelance profile. MaxAmount is a
// pointer so a missing field stays nil (scalar types) vs a present
// integer (range types).
type UpsertFreelancePricingRequest struct {
	Type       string `json:"type" validate:"required,min=1,max=50"`
	MinAmount  int64  `json:"min_amount" validate:"gte=0,lte=999999999"`
	MaxAmount  *int64 `json:"max_amount" validate:"omitempty,gte=0,lte=999999999"`
	Currency   string `json:"currency" validate:"required,len=3"`
	Note       string `json:"note" validate:"omitempty,max=2000"`
	Negotiable bool   `json:"negotiable"`
}
