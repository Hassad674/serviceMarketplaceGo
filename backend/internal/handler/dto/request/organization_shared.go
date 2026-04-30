package request

// UpdateOrganizationLocationRequest is the request body for the
// /api/v1/organization/location endpoint. Same shape as the legacy
// profile location request — lat/lng are client-supplied when the
// client-side autocomplete resolves them, nil otherwise.
type UpdateOrganizationLocationRequest struct {
	City           string   `json:"city" validate:"omitempty,max=200"`
	CountryCode    string   `json:"country_code" validate:"omitempty,len=2"`
	Latitude       *float64 `json:"latitude" validate:"omitempty,gte=-90,lte=90"`
	Longitude      *float64 `json:"longitude" validate:"omitempty,gte=-180,lte=180"`
	WorkMode       []string `json:"work_mode" validate:"omitempty,max=10,dive,min=1,max=50"`
	TravelRadiusKm *int     `json:"travel_radius_km" validate:"omitempty,gte=0,lte=20000"`
}

// UpdateOrganizationLanguagesRequest is the request body for the
// /api/v1/organization/languages endpoint.
type UpdateOrganizationLanguagesRequest struct {
	Professional   []string `json:"professional" validate:"omitempty,max=20,dive,min=1,max=50"`
	Conversational []string `json:"conversational" validate:"omitempty,max=20,dive,min=1,max=50"`
}

// UpdateOrganizationPhotoRequest is the request body for the
// /api/v1/organization/photo endpoint — a simple URL write. The
// upstream upload flow (POST /upload/photo) is unchanged; this
// endpoint only persists the resulting URL on the org row.
type UpdateOrganizationPhotoRequest struct {
	PhotoURL string `json:"photo_url" validate:"omitempty,url,max=2048"`
}
