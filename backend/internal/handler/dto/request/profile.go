package request

// UpdateProfileRequest is the legacy core update DTO (agency / provider
// shared fields).
type UpdateProfileRequest struct {
	Title                string `json:"title" validate:"omitempty,max=200"`
	About                string `json:"about" validate:"omitempty,max=5000"`
	PhotoURL             string `json:"photo_url" validate:"omitempty,url,max=2048"`
	PresentationVideoURL string `json:"presentation_video_url" validate:"omitempty,url,max=2048"`
	ReferrerAbout        string `json:"referrer_about" validate:"omitempty,max=5000"`
	ReferrerVideoURL     string `json:"referrer_video_url" validate:"omitempty,url,max=2048"`
}

// UpdateClientProfileRequest is the payload for
// PUT /api/v1/profile/client. Both fields are optional — a caller may
// touch only the company name or only the description and leave the
// other unchanged. Pointer semantics let the handler distinguish
// "field omitted" (nil) from "field cleared to empty string" (non-nil
// with empty value). Only CompanyName rejects the empty case; an
// empty client_description is a valid reset.
type UpdateClientProfileRequest struct {
	CompanyName       *string `json:"company_name,omitempty" validate:"omitempty,min=1,max=200"`
	ClientDescription *string `json:"client_description,omitempty" validate:"omitempty,max=5000"`
}
