package request

type UpdateProfileRequest struct {
	Title                string `json:"title"`
	About                string `json:"about"`
	PhotoURL             string `json:"photo_url"`
	PresentationVideoURL string `json:"presentation_video_url"`
	ReferrerAbout        string `json:"referrer_about"`
	ReferrerVideoURL     string `json:"referrer_video_url"`
}

// UpdateClientProfileRequest is the payload for
// PUT /api/v1/profile/client. Both fields are optional — a caller may
// touch only the company name or only the description and leave the
// other unchanged. Pointer semantics let the handler distinguish
// "field omitted" (nil) from "field cleared to empty string" (non-nil
// with empty value). Only CompanyName rejects the empty case; an
// empty client_description is a valid reset.
type UpdateClientProfileRequest struct {
	CompanyName       *string `json:"company_name,omitempty"`
	ClientDescription *string `json:"client_description,omitempty"`
}
