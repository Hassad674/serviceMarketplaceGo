package response

import (
	"github.com/google/uuid"

	"marketplace-backend/internal/domain/profile"
	domainpricing "marketplace-backend/internal/domain/profilepricing"
	domainskill "marketplace-backend/internal/domain/skill"
)

// ProfileSkillSummary is the compact DTO for one skill attached to a
// profile. We only surface the pair (skill_text, display_text) —
// usage_count and is_curated are internal bookkeeping the public
// profile viewer does not need. The text is always normalized
// lowercase; the display_text preserves the original casing so the
// UI can render "React" or "Next.js" correctly.
type ProfileSkillSummary struct {
	SkillText   string `json:"skill_text"`
	DisplayText string `json:"display_text"`
}

// NewProfileSkillSummaryList maps domain profile skills to the
// compact DTO, skipping any nil entries defensively.
func NewProfileSkillSummaryList(skills []*domainskill.ProfileSkill) []ProfileSkillSummary {
	out := make([]ProfileSkillSummary, 0, len(skills))
	for _, s := range skills {
		if s == nil {
			continue
		}
		// DisplayText on ProfileSkill is not persisted on the
		// profile_skills row — only skill_text lives there. The
		// public profile endpoints enrich the slice upstream by
		// falling back to skill_text when display is missing.
		display := s.DisplayText
		if display == "" {
			display = s.SkillText
		}
		out = append(out, ProfileSkillSummary{
			SkillText:   s.SkillText,
			DisplayText: display,
		})
	}
	return out
}

// PricingSummary is the public DTO for one pricing row. Mirrors
// the profile_pricing table (migration 083) one-to-one: user-facing
// fields plus the composite key (kind). Timestamps are internal
// bookkeeping and not exposed in the response.
type PricingSummary struct {
	Kind       string `json:"kind"`
	Type       string `json:"type"`
	MinAmount  int64  `json:"min_amount"`
	MaxAmount  *int64 `json:"max_amount"`
	Currency   string `json:"currency"`
	Note       string `json:"note"`
	Negotiable bool   `json:"negotiable"`
}

// NewPricingSummary converts a domain Pricing to its DTO. Nil
// input returns a zero-value summary — callers are expected to
// pre-filter nils from their slices.
func NewPricingSummary(p *domainpricing.Pricing) PricingSummary {
	return PricingSummary{
		Kind:       string(p.Kind),
		Type:       string(p.Type),
		MinAmount:  p.MinAmount,
		MaxAmount:  p.MaxAmount,
		Currency:   p.Currency,
		Note:       p.Note,
		Negotiable: p.Negotiable,
	}
}

// NewPricingSummaryList maps a slice of domain pricings to DTOs.
// Returns a guaranteed non-nil slice so the JSON shape stays
// `[]` for orgs with no declared pricing.
func NewPricingSummaryList(pricings []*domainpricing.Pricing) []PricingSummary {
	out := make([]PricingSummary, 0, len(pricings))
	for _, p := range pricings {
		if p == nil {
			continue
		}
		out = append(out, NewPricingSummary(p))
	}
	return out
}

// ProfileResponse is the full public profile payload: identity +
// classic fields + expertise + skills + Tier 1 completion blocks
// (location / languages / availability / pricing). Every slice and
// every nullable is explicitly modeled so the JSON shape is stable
// across requests — frontend code can rely on the keys existing
// without defensive optional chaining.
type ProfileResponse struct {
	OrganizationID       string `json:"organization_id"`
	Title                string `json:"title"`
	About                string `json:"about"`
	PhotoURL             string `json:"photo_url"`
	PresentationVideoURL string `json:"presentation_video_url"`
	ReferrerAbout        string `json:"referrer_about"`
	ReferrerVideoURL     string `json:"referrer_video_url"`

	// ExpertiseDomains is the ordered list of domain specialization
	// keys the organization has declared (see internal/domain/expertise
	// for the catalog). Empty orgs and enterprise orgs always receive
	// an empty slice — never null — so the frontend can safely render
	// `response.data.expertise_domains.map(...)` without a guard.
	ExpertiseDomains []string `json:"expertise_domains"`

	// Skills is the ordered list of skills the organization has
	// declared (see internal/domain/skill). Always a non-nil slice —
	// empty for orgs that have not declared any skills yet.
	Skills []ProfileSkillSummary `json:"skills"`

	// ---- Tier 1 completion (migration 083) ----
	City                       string           `json:"city"`
	CountryCode                string           `json:"country_code"`
	Latitude                   *float64         `json:"latitude"`
	Longitude                  *float64         `json:"longitude"`
	WorkMode                   []string         `json:"work_mode"`
	TravelRadiusKm             *int             `json:"travel_radius_km"`
	LanguagesProfessional      []string         `json:"languages_professional"`
	LanguagesConversational    []string         `json:"languages_conversational"`
	AvailabilityStatus         string           `json:"availability_status"`
	ReferrerAvailabilityStatus *string          `json:"referrer_availability_status"`
	Pricing                    []PricingSummary `json:"pricing"` // 0..2 rows

	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// PublicProfileSummary is the shape surfaced to marketplace search /
// discovery. Since phase R2, it describes an organization (the team
// behind the offering), not an individual user — the name is the
// org's display name and the role is the org type.
//
// The listing shape is intentionally leaner than ProfileResponse: it
// exposes only the fields useful on a search card. The Tier 1 signal
// fields (city, country_code, languages, availability) and the
// aggregate fields (total_earned, completed_projects) are lit up by
// the SearchPublic query; detail-view read paths leave them at their
// zero values and the JSON marshals them as empty strings / empty
// slices / zeros so the envelope shape stays stable across callers.
type PublicProfileSummary struct {
	OrganizationID string `json:"organization_id"`
	// OwnerUserID is the id of the user at the top of the org — the
	// "party id" the business-referral feature consumes when the
	// apporteur picks a provider from the search results.
	OwnerUserID    string `json:"owner_user_id"`
	Name            string                `json:"name"`
	OrgType         string                `json:"org_type"`
	Title           string                `json:"title"`
	PhotoURL        string                `json:"photo_url"`
	ReferrerEnabled bool                  `json:"referrer_enabled"`
	AverageRating   float64               `json:"average_rating"`
	ReviewCount     int                   `json:"review_count"`
	Skills          []ProfileSkillSummary `json:"skills"`
	Pricing         []PricingSummary      `json:"pricing"`

	// ---- Tier 1 signal fields lit by SearchPublic ----
	City                  string   `json:"city"`
	CountryCode           string   `json:"country_code"`
	LanguagesProfessional []string `json:"languages_professional"`
	AvailabilityStatus    string   `json:"availability_status"`

	// ---- Aggregate fields lit by SearchPublic ----
	// TotalEarned is in the smallest currency unit (centimes) — matches
	// the proposal_milestones.amount scale. 0 when no released payments.
	TotalEarned int64 `json:"total_earned"`
	// CompletedProjects counts distinct proposals with at least one
	// released milestone attributed to this org's owner.
	CompletedProjects int `json:"completed_projects"`
}

// NewPublicProfileSummary builds the summary DTO without decorating
// it with skills or pricing. Used by tests and code paths that have
// not wired the batch loaders yet. New code should prefer
// NewPublicProfileSummaryListWithExtras.
func NewPublicProfileSummary(p *profile.PublicProfile) PublicProfileSummary {
	languages := p.LanguagesProfessional
	if languages == nil {
		languages = []string{}
	}
	return PublicProfileSummary{
		OrganizationID:        p.OrganizationID.String(),
		OwnerUserID:           p.OwnerUserID.String(),
		Name:                  p.Name,
		OrgType:               p.OrgType,
		Title:                 p.Title,
		PhotoURL:              p.PhotoURL,
		ReferrerEnabled:       p.ReferrerEnabled,
		AverageRating:         p.AverageRating,
		ReviewCount:           p.ReviewCount,
		Skills:                []ProfileSkillSummary{},
		Pricing:               []PricingSummary{},
		City:                  p.City,
		CountryCode:           p.CountryCode,
		LanguagesProfessional: languages,
		AvailabilityStatus:    p.AvailabilityStatus,
		TotalEarned:           p.TotalEarned,
		CompletedProjects:     p.CompletedProjects,
	}
}

// NewPublicProfileSummaryList maps a list of public profiles to the
// summary DTO WITHOUT skill or pricing decoration. Kept for
// backwards compatibility with older tests and with code paths that
// do not need the decorated shape.
func NewPublicProfileSummaryList(profiles []*profile.PublicProfile) []PublicProfileSummary {
	result := make([]PublicProfileSummary, len(profiles))
	for i, p := range profiles {
		result[i] = NewPublicProfileSummary(p)
	}
	return result
}

// NewPublicProfileSummaryListWithSkills maps a list of public
// profiles AND decorates each entry with the batched skill list
// fetched by the handler. Profiles with no declared skills get a
// guaranteed empty (non-nil) slice. Kept for backwards compatibility
// with call sites that have not wired pricing yet.
func NewPublicProfileSummaryListWithSkills(
	profiles []*profile.PublicProfile,
	skillsByOrg map[uuid.UUID][]*domainskill.ProfileSkill,
) []PublicProfileSummary {
	return NewPublicProfileSummaryListWithExtras(profiles, skillsByOrg, nil)
}

// NewPublicProfileSummaryListWithExtras maps a list of public
// profiles AND decorates each entry with the batched skill and
// pricing lists fetched by the handler. Profiles with no declared
// skills / pricing get a guaranteed empty (non-nil) slice, so the
// JSON shape stays consistent. Nil maps are tolerated — the
// corresponding decoration becomes an empty slice across the board.
func NewPublicProfileSummaryListWithExtras(
	profiles []*profile.PublicProfile,
	skillsByOrg map[uuid.UUID][]*domainskill.ProfileSkill,
	pricingByOrg map[uuid.UUID][]*domainpricing.Pricing,
) []PublicProfileSummary {
	result := make([]PublicProfileSummary, len(profiles))
	for i, p := range profiles {
		summary := NewPublicProfileSummary(p)
		if skills, ok := skillsByOrg[p.OrganizationID]; ok {
			summary.Skills = NewProfileSkillSummaryList(skills)
		}
		if pricings, ok := pricingByOrg[p.OrganizationID]; ok {
			summary.Pricing = NewPricingSummaryList(pricings)
		}
		result[i] = summary
	}
	return result
}

// NewProfileResponse assembles the full profile DTO, including the
// expertise domain list and the declared skills. Callers that don't
// have expertise or skills wired (legacy unit tests) can pass nil —
// the response will carry empty slices so the JSON shape is stable.
//
// Backwards-compatible wrapper around NewProfileResponseWithExtras
// that defaults pricing to nil (empty slice in the response).
func NewProfileResponse(
	p *profile.Profile,
	expertiseDomains []string,
	skills []*domainskill.ProfileSkill,
) ProfileResponse {
	return NewProfileResponseWithExtras(p, expertiseDomains, skills, nil)
}

// NewProfileResponseWithExtras assembles the full profile DTO,
// including the optional pricing list decorated by the handler.
// Nil pricing yields an empty slice so the JSON shape stays stable
// across requests.
func NewProfileResponseWithExtras(
	p *profile.Profile,
	expertiseDomains []string,
	skills []*domainskill.ProfileSkill,
	pricing []*domainpricing.Pricing,
) ProfileResponse {
	if expertiseDomains == nil {
		expertiseDomains = []string{}
	}
	workMode := p.WorkMode
	if workMode == nil {
		workMode = []string{}
	}
	langPro := p.LanguagesProfessional
	if langPro == nil {
		langPro = []string{}
	}
	langConv := p.LanguagesConversational
	if langConv == nil {
		langConv = []string{}
	}
	var referrerAvail *string
	if p.ReferrerAvailabilityStatus != nil {
		s := string(*p.ReferrerAvailabilityStatus)
		referrerAvail = &s
	}
	availability := string(p.AvailabilityStatus)
	if availability == "" {
		// Defensive default — the domain guarantees this is set on
		// new profiles, but a hand-crafted *Profile in a legacy test
		// may leave it empty. Fall back to the same default used by
		// NewProfile so the JSON never surfaces "".
		availability = string(profile.AvailabilityNow)
	}
	return ProfileResponse{
		OrganizationID:             p.OrganizationID.String(),
		Title:                      p.Title,
		About:                      p.About,
		PhotoURL:                   p.PhotoURL,
		PresentationVideoURL:       p.PresentationVideoURL,
		ReferrerAbout:              p.ReferrerAbout,
		ReferrerVideoURL:           p.ReferrerVideoURL,
		ExpertiseDomains:           expertiseDomains,
		Skills:                     NewProfileSkillSummaryList(skills),
		City:                       p.City,
		CountryCode:                p.CountryCode,
		Latitude:                   p.Latitude,
		Longitude:                  p.Longitude,
		WorkMode:                   workMode,
		TravelRadiusKm:             p.TravelRadiusKm,
		LanguagesProfessional:      langPro,
		LanguagesConversational:    langConv,
		AvailabilityStatus:         availability,
		ReferrerAvailabilityStatus: referrerAvail,
		Pricing:                    NewPricingSummaryList(pricing),
		CreatedAt:                  p.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:                  p.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}
