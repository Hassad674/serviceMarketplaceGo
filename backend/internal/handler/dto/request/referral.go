package request

// CreateReferralRequest is the payload POST /api/v1/referrals accepts. The
// referrer_id is NEVER in the body — it comes from the JWT.
type CreateReferralRequest struct {
	ProviderID           string                 `json:"provider_id"`
	ClientID             string                 `json:"client_id"`
	RatePct              float64                `json:"rate_pct"`
	DurationMonths       int16                  `json:"duration_months"`
	IntroMessageProvider string                 `json:"intro_message_provider"`
	IntroMessageClient   string                 `json:"intro_message_client"`
	SnapshotToggles      *ReferralSnapshotToggles `json:"snapshot_toggles,omitempty"`
}

// ReferralSnapshotToggles lets the apporteur choose which auto-filled
// provider fields to reveal on the anonymised card. Mirrored 1:1 on the
// app-service SnapshotToggles struct.
type ReferralSnapshotToggles struct {
	IncludeExpertise    bool `json:"include_expertise"`
	IncludeExperience   bool `json:"include_experience"`
	IncludeRating       bool `json:"include_rating"`
	IncludePricing      bool `json:"include_pricing"`
	IncludeRegion       bool `json:"include_region"`
	IncludeLanguages    bool `json:"include_languages"`
	IncludeAvailability bool `json:"include_availability"`
}

// RespondReferralRequest is the unified body for POST /api/v1/referrals/{id}/respond.
// The server infers the actor role from the JWT user vs the referral parties.
type RespondReferralRequest struct {
	// Action is one of: "accept", "reject", "negotiate", "cancel", "terminate".
	Action string `json:"action"`

	// NewRatePct is the new commission rate for "negotiate" actions; ignored
	// otherwise.
	NewRatePct float64 `json:"new_rate_pct,omitempty"`

	// Message is an optional human-readable justification for the action.
	Message string `json:"message,omitempty"`
}
