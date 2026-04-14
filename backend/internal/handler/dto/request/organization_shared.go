package request

// UpdateOrganizationLocationRequest is the request body for the
// /api/v1/organization/location endpoint. Same shape as the legacy
// profile location request — lat/lng are client-supplied when the
// client-side autocomplete resolves them, nil otherwise.
type UpdateOrganizationLocationRequest struct {
	City           string   `json:"city"`
	CountryCode    string   `json:"country_code"`
	Latitude       *float64 `json:"latitude"`
	Longitude      *float64 `json:"longitude"`
	WorkMode       []string `json:"work_mode"`
	TravelRadiusKm *int     `json:"travel_radius_km"`
}

// UpdateOrganizationLanguagesRequest is the request body for the
// /api/v1/organization/languages endpoint.
type UpdateOrganizationLanguagesRequest struct {
	Professional   []string `json:"professional"`
	Conversational []string `json:"conversational"`
}

// UpdateOrganizationPhotoRequest is the request body for the
// /api/v1/organization/photo endpoint — a simple URL write. The
// upstream upload flow (POST /upload/photo) is unchanged; this
// endpoint only persists the resulting URL on the org row.
type UpdateOrganizationPhotoRequest struct {
	PhotoURL string `json:"photo_url"`
}
