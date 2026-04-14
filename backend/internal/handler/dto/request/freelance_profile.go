package request

// UpdateFreelanceProfileRequest is the request body for the "update
// core" endpoint on a freelance profile. Empty strings are valid —
// they clear the corresponding column.
type UpdateFreelanceProfileRequest struct {
	Title    string `json:"title"`
	About    string `json:"about"`
	VideoURL string `json:"video_url"`
}

// UpdateFreelanceAvailabilityRequest is the request body for the
// availability endpoint on a freelance profile. Single required
// field so the JSON shape stays symmetrical with the referrer side.
type UpdateFreelanceAvailabilityRequest struct {
	AvailabilityStatus string `json:"availability_status"`
}

// UpdateFreelanceExpertiseRequest is the request body for the
// expertise endpoint on a freelance profile. Empty slice clears
// the declared expertise.
type UpdateFreelanceExpertiseRequest struct {
	Domains []string `json:"domains"`
}

// UpsertFreelancePricingRequest is the request body for the
// pricing upsert endpoint on a freelance profile. MaxAmount is a
// pointer so a missing field stays nil (scalar types) vs a present
// integer (range types).
type UpsertFreelancePricingRequest struct {
	Type       string `json:"type"`
	MinAmount  int64  `json:"min_amount"`
	MaxAmount  *int64 `json:"max_amount"`
	Currency   string `json:"currency"`
	Note       string `json:"note"`
	Negotiable bool   `json:"negotiable"`
}
