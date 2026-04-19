package response

import (
	"time"

	appreferrer "marketplace-backend/internal/app/referrerprofile"
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

// ReferrerReputationResponse is the JSON shape returned by
// GET /api/v1/referrer-profile/{user_id}/reputation.
//
// rating_avg and review_count are summary stats computed once across
// every reviewed, completed attribution — they are NOT repaginated
// across history pages. history[] rotates as the caller walks through
// next_cursor.
type ReferrerReputationResponse struct {
	RatingAvg   float64                        `json:"rating_avg"`
	ReviewCount int                            `json:"review_count"`
	History     []ProjectHistoryEntryResponse  `json:"history"`
	NextCursor  string                         `json:"next_cursor"`
	HasMore     bool                           `json:"has_more"`
}

// ProjectHistoryEntryResponse is one attributed mission on the public
// apporteur reputation surface. BOTH the client and the provider
// identities are intentionally absent:
//
//   - client identity: B2B working-relationship confidentiality (Modèle A)
//   - provider identity: the apporteur's recommendation graph is private
//
// The clients of this DTO render the introduced provider as a static
// "Prestataire introduit" label. The review, when present, carries the
// full double-blind client→provider feedback (sub-criteria + video)
// so clients can reuse the shared ReviewCard primitive — same shape
// as the freelance project history surface.
type ProjectHistoryEntryResponse struct {
	ProposalID     string          `json:"proposal_id"`
	ProposalTitle  string          `json:"proposal_title"`
	ProposalStatus string          `json:"proposal_status"`
	Review         *ReviewResponse `json:"review"`
	CompletedAt    *string         `json:"completed_at"`
	AttributedAt   string          `json:"attributed_at"`
}

// NewReferrerReputationResponse maps the service aggregate to the DTO.
func NewReferrerReputationResponse(rep appreferrer.ReferrerReputation) ReferrerReputationResponse {
	history := make([]ProjectHistoryEntryResponse, 0, len(rep.History))
	for _, e := range rep.History {
		history = append(history, newProjectHistoryEntryResponse(e))
	}
	return ReferrerReputationResponse{
		RatingAvg:   rep.RatingAvg,
		ReviewCount: rep.ReviewCount,
		History:     history,
		NextCursor:  rep.NextCursor,
		HasMore:     rep.NextCursor != "",
	}
}

func newProjectHistoryEntryResponse(e appreferrer.ProjectHistoryEntry) ProjectHistoryEntryResponse {
	out := ProjectHistoryEntryResponse{
		ProposalID:     e.ProposalID.String(),
		ProposalTitle:  e.ProposalTitle,
		ProposalStatus: e.ProposalStatus,
		CompletedAt:    formatOptionalTime(e.CompletedAt),
		AttributedAt:   e.AttributedAt.Format(time.RFC3339),
	}
	if e.Review != nil {
		r := ReviewFromDomain(e.Review)
		out.Review = &r
	}
	return out
}

func formatOptionalTime(t *time.Time) *string {
	if t == nil {
		return nil
	}
	formatted := t.Format(time.RFC3339)
	return &formatted
}

// Compile-time check.
var _ = referrerprofile.ErrProfileNotFound
