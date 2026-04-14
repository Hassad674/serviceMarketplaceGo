package response

import (
	"marketplace-backend/internal/domain/freelanceprofile"
	domainpricing "marketplace-backend/internal/domain/freelancepricing"
	"marketplace-backend/internal/domain/profile"
	"marketplace-backend/internal/port/repository"
)

// FreelanceProfileResponse is the public JSON shape for one
// freelance profile — persona-specific fields from freelance_profiles
// plus the shared fields from organizations. Every slice is
// guaranteed non-nil so the client never has to write defensive
// optional-chaining.
//
// Skills are decorated by the handler using a batch reader (see
// FreelanceProfileHandler) so the shape matches the frontend's
// pre-split expectation that a profile carries its skills inline.
// Pricing is a single row (or nil) on the freelance side — unlike
// the legacy profile which carried an array of 0..2 rows.
type FreelanceProfileResponse struct {
	ID                 string   `json:"id"`
	OrganizationID     string   `json:"organization_id"`
	Title              string   `json:"title"`
	About              string   `json:"about"`
	VideoURL           string   `json:"video_url"`
	AvailabilityStatus string   `json:"availability_status"`
	ExpertiseDomains   []string `json:"expertise_domains"`

	// ---- Shared block (joined from organizations) ----
	PhotoURL                string   `json:"photo_url"`
	City                    string   `json:"city"`
	CountryCode             string   `json:"country_code"`
	Latitude                *float64 `json:"latitude"`
	Longitude               *float64 `json:"longitude"`
	WorkMode                []string `json:"work_mode"`
	TravelRadiusKm          *int     `json:"travel_radius_km"`
	LanguagesProfessional   []string `json:"languages_professional"`
	LanguagesConversational []string `json:"languages_conversational"`

	// ---- Decorations ----
	Skills  []ProfileSkillSummary   `json:"skills"`
	Pricing *FreelancePricingSummary `json:"pricing"`

	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// FreelancePricingSummary is the JSON shape for the pricing row
// attached to a freelance profile. Nil MaxAmount collapses to a
// JSON null, honoring the scalar-vs-range distinction.
type FreelancePricingSummary struct {
	Type       string `json:"type"`
	MinAmount  int64  `json:"min_amount"`
	MaxAmount  *int64 `json:"max_amount"`
	Currency   string `json:"currency"`
	Note       string `json:"note"`
	Negotiable bool   `json:"negotiable"`
}

// NewFreelancePricingSummary converts a domain pricing to its DTO.
// Nil input returns nil — callers that hold a *Pricing value and
// want to omit the field can simply pass nil.
func NewFreelancePricingSummary(p *domainpricing.Pricing) *FreelancePricingSummary {
	if p == nil {
		return nil
	}
	return &FreelancePricingSummary{
		Type:       string(p.Type),
		MinAmount:  p.MinAmount,
		MaxAmount:  p.MaxAmount,
		Currency:   p.Currency,
		Note:       p.Note,
		Negotiable: p.Negotiable,
	}
}

// NewFreelanceProfileResponse assembles the full response DTO from
// a FreelanceProfileView (persona + shared) plus optional pricing
// and optional skills decoration. A nil skills slice yields an
// empty (non-nil) array so the JSON shape is stable.
func NewFreelanceProfileResponse(
	view *repository.FreelanceProfileView,
	pricing *domainpricing.Pricing,
	skills []ProfileSkillSummary,
) FreelanceProfileResponse {
	p := view.Profile
	if skills == nil {
		skills = []ProfileSkillSummary{}
	}
	availability := string(p.AvailabilityStatus)
	if availability == "" {
		availability = string(profile.AvailabilityNow)
	}
	return FreelanceProfileResponse{
		ID:                      p.ID.String(),
		OrganizationID:          p.OrganizationID.String(),
		Title:                   p.Title,
		About:                   p.About,
		VideoURL:                p.VideoURL,
		AvailabilityStatus:      availability,
		ExpertiseDomains:        nilToEmptyStrings(p.ExpertiseDomains),
		PhotoURL:                view.Shared.PhotoURL,
		City:                    view.Shared.City,
		CountryCode:             view.Shared.CountryCode,
		Latitude:                view.Shared.Latitude,
		Longitude:               view.Shared.Longitude,
		WorkMode:                nilToEmptyStrings(view.Shared.WorkMode),
		TravelRadiusKm:          view.Shared.TravelRadiusKm,
		LanguagesProfessional:   nilToEmptyStrings(view.Shared.LanguagesProfessional),
		LanguagesConversational: nilToEmptyStrings(view.Shared.LanguagesConversational),
		Skills:                  skills,
		Pricing:                 NewFreelancePricingSummary(pricing),
		CreatedAt:               p.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:               p.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

// Compile-time check that the freelanceprofile domain package still
// exposes the symbols this DTO references. Guards against silent
// drift in the domain layer.
var _ = freelanceprofile.ErrProfileNotFound
