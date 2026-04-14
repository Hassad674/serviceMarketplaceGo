package response

import "marketplace-backend/internal/port/repository"

// OrganizationSharedProfileResponse is the JSON shape returned by
// the /api/v1/organization/shared read endpoint and by the mutation
// endpoints (/api/v1/organization/location, /languages, /photo) as
// their response body. Mirrors the shared-profile columns on the
// organizations row verbatim — every slice is guaranteed non-nil.
type OrganizationSharedProfileResponse struct {
	PhotoURL                string   `json:"photo_url"`
	City                    string   `json:"city"`
	CountryCode             string   `json:"country_code"`
	Latitude                *float64 `json:"latitude"`
	Longitude               *float64 `json:"longitude"`
	WorkMode                []string `json:"work_mode"`
	TravelRadiusKm          *int     `json:"travel_radius_km"`
	LanguagesProfessional   []string `json:"languages_professional"`
	LanguagesConversational []string `json:"languages_conversational"`
}

// NewOrganizationSharedProfileResponse converts the port-level
// OrganizationSharedProfile bundle into the DTO. Nil slices are
// coerced to empty slices so the JSON shape is stable.
func NewOrganizationSharedProfileResponse(shared *repository.OrganizationSharedProfile) OrganizationSharedProfileResponse {
	return OrganizationSharedProfileResponse{
		PhotoURL:                shared.PhotoURL,
		City:                    shared.City,
		CountryCode:             shared.CountryCode,
		Latitude:                shared.Latitude,
		Longitude:               shared.Longitude,
		WorkMode:                nilToEmptyStrings(shared.WorkMode),
		TravelRadiusKm:          shared.TravelRadiusKm,
		LanguagesProfessional:   nilToEmptyStrings(shared.LanguagesProfessional),
		LanguagesConversational: nilToEmptyStrings(shared.LanguagesConversational),
	}
}
