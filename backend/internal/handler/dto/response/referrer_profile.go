package response

import (
	"marketplace-backend/internal/domain/profile"
	domainpricing "marketplace-backend/internal/domain/referrerpricing"
	"marketplace-backend/internal/domain/referrerprofile"
	"marketplace-backend/internal/port/repository"
)

// ReferrerProfileResponse is the public JSON shape for one referrer
// profile. Mirrors FreelanceProfileResponse structurally — the only
// differences are the semantic types (ReferrerPricingSummary, no
// skills field). Skills stay on the freelance persona because skill
// vocabularies (e.g. Go, Kubernetes) describe what a person does
// themselves, not what deals they bring in.
type ReferrerProfileResponse struct {
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

	Pricing *ReferrerPricingSummary `json:"pricing"`

	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// ReferrerPricingSummary is the JSON shape for the pricing row
// attached to a referrer profile. commission_pct uses "pct" as
// currency with basis points; commission_flat uses an ISO 4217
// code with cents.
type ReferrerPricingSummary struct {
	Type       string `json:"type"`
	MinAmount  int64  `json:"min_amount"`
	MaxAmount  *int64 `json:"max_amount"`
	Currency   string `json:"currency"`
	Note       string `json:"note"`
	Negotiable bool   `json:"negotiable"`
}

// NewReferrerPricingSummary converts a domain pricing to its DTO.
// Nil input returns nil.
func NewReferrerPricingSummary(p *domainpricing.Pricing) *ReferrerPricingSummary {
	if p == nil {
		return nil
	}
	return &ReferrerPricingSummary{
		Type:       string(p.Type),
		MinAmount:  p.MinAmount,
		MaxAmount:  p.MaxAmount,
		Currency:   p.Currency,
		Note:       p.Note,
		Negotiable: p.Negotiable,
	}
}

// NewReferrerProfileResponse assembles the full response DTO from a
// ReferrerProfileView plus optional pricing. Every slice is
// guaranteed non-nil.
func NewReferrerProfileResponse(
	view *repository.ReferrerProfileView,
	pricing *domainpricing.Pricing,
) ReferrerProfileResponse {
	p := view.Profile
	availability := string(p.AvailabilityStatus)
	if availability == "" {
		availability = string(profile.AvailabilityNow)
	}
	return ReferrerProfileResponse{
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
		Pricing:                 NewReferrerPricingSummary(pricing),
		CreatedAt:               p.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:               p.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

// Compile-time check.
var _ = referrerprofile.ErrProfileNotFound
