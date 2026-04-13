package response

import (
	"github.com/google/uuid"

	"marketplace-backend/internal/domain/profile"
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

type ProfileResponse struct {
	OrganizationID       string   `json:"organization_id"`
	Title                string   `json:"title"`
	About                string   `json:"about"`
	PhotoURL             string   `json:"photo_url"`
	PresentationVideoURL string   `json:"presentation_video_url"`
	ReferrerAbout        string   `json:"referrer_about"`
	ReferrerVideoURL     string   `json:"referrer_video_url"`
	// ExpertiseDomains is the ordered list of domain specialization
	// keys the organization has declared (see internal/domain/expertise
	// for the catalog). Empty orgs and enterprise orgs always receive
	// an empty slice — never null — so the frontend can safely render
	// `response.data.expertise_domains.map(...)` without a guard.
	ExpertiseDomains []string `json:"expertise_domains"`
	// Skills is the ordered list of skills the organization has
	// declared (see internal/domain/skill). Always a non-nil slice —
	// empty for orgs that have not declared any skills yet.
	Skills    []ProfileSkillSummary `json:"skills"`
	CreatedAt string                `json:"created_at"`
	UpdatedAt string                `json:"updated_at"`
}

// PublicProfileSummary is the shape surfaced to marketplace search /
// discovery. Since phase R2, it describes an organization (the team
// behind the offering), not an individual user — the name is the
// org's display name and the role is the org type.
type PublicProfileSummary struct {
	OrganizationID  string                `json:"organization_id"`
	Name            string                `json:"name"`
	OrgType         string                `json:"org_type"`
	Title           string                `json:"title"`
	PhotoURL        string                `json:"photo_url"`
	ReferrerEnabled bool                  `json:"referrer_enabled"`
	AverageRating   float64               `json:"average_rating"`
	ReviewCount     int                   `json:"review_count"`
	Skills          []ProfileSkillSummary `json:"skills"`
}

// NewPublicProfileSummary builds the summary DTO without decorating
// it with skills. Used by tests and code paths that have not wired
// the skills batch load yet. New code should prefer
// NewPublicProfileSummaryWithSkills.
func NewPublicProfileSummary(p *profile.PublicProfile) PublicProfileSummary {
	return PublicProfileSummary{
		OrganizationID:  p.OrganizationID.String(),
		Name:            p.Name,
		OrgType:         p.OrgType,
		Title:           p.Title,
		PhotoURL:        p.PhotoURL,
		ReferrerEnabled: p.ReferrerEnabled,
		AverageRating:   p.AverageRating,
		ReviewCount:     p.ReviewCount,
		Skills:          []ProfileSkillSummary{},
	}
}

// NewPublicProfileSummaryList maps a list of public profiles to the
// summary DTO WITHOUT skill decoration. Kept for backwards
// compatibility with older tests.
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
// guaranteed empty (non-nil) slice.
func NewPublicProfileSummaryListWithSkills(
	profiles []*profile.PublicProfile,
	skillsByOrg map[uuid.UUID][]*domainskill.ProfileSkill,
) []PublicProfileSummary {
	result := make([]PublicProfileSummary, len(profiles))
	for i, p := range profiles {
		summary := NewPublicProfileSummary(p)
		if skills, ok := skillsByOrg[p.OrganizationID]; ok {
			summary.Skills = NewProfileSkillSummaryList(skills)
		}
		result[i] = summary
	}
	return result
}

// NewProfileResponse assembles the full profile DTO, including the
// expertise domain list and the declared skills. Callers that don't
// have expertise or skills wired (legacy unit tests) can pass nil —
// the response will carry empty slices so the JSON shape is stable.
func NewProfileResponse(
	p *profile.Profile,
	expertiseDomains []string,
	skills []*domainskill.ProfileSkill,
) ProfileResponse {
	if expertiseDomains == nil {
		expertiseDomains = []string{}
	}
	return ProfileResponse{
		OrganizationID:       p.OrganizationID.String(),
		Title:                p.Title,
		About:                p.About,
		PhotoURL:             p.PhotoURL,
		PresentationVideoURL: p.PresentationVideoURL,
		ReferrerAbout:        p.ReferrerAbout,
		ReferrerVideoURL:     p.ReferrerVideoURL,
		ExpertiseDomains:     expertiseDomains,
		Skills:               NewProfileSkillSummaryList(skills),
		CreatedAt:            p.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:            p.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}
