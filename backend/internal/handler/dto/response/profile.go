package response

import (
	"marketplace-backend/internal/domain/profile"
)

type ProfileResponse struct {
	OrganizationID       string `json:"organization_id"`
	Title                string `json:"title"`
	About                string `json:"about"`
	PhotoURL             string `json:"photo_url"`
	PresentationVideoURL string `json:"presentation_video_url"`
	ReferrerAbout        string `json:"referrer_about"`
	ReferrerVideoURL     string `json:"referrer_video_url"`
	CreatedAt            string `json:"created_at"`
	UpdatedAt            string `json:"updated_at"`
}

// PublicProfileSummary is the shape surfaced to marketplace search /
// discovery. Since phase R2, it describes an organization (the team
// behind the offering), not an individual user — the name is the
// org's display name and the role is the org type.
type PublicProfileSummary struct {
	OrganizationID  string  `json:"organization_id"`
	Name            string  `json:"name"`
	OrgType         string  `json:"org_type"`
	Title           string  `json:"title"`
	PhotoURL        string  `json:"photo_url"`
	ReferrerEnabled bool    `json:"referrer_enabled"`
	AverageRating   float64 `json:"average_rating"`
	ReviewCount     int     `json:"review_count"`
}

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
	}
}

func NewPublicProfileSummaryList(profiles []*profile.PublicProfile) []PublicProfileSummary {
	result := make([]PublicProfileSummary, len(profiles))
	for i, p := range profiles {
		result[i] = NewPublicProfileSummary(p)
	}
	return result
}

func NewProfileResponse(p *profile.Profile) ProfileResponse {
	return ProfileResponse{
		OrganizationID:       p.OrganizationID.String(),
		Title:                p.Title,
		About:                p.About,
		PhotoURL:             p.PhotoURL,
		PresentationVideoURL: p.PresentationVideoURL,
		ReferrerAbout:        p.ReferrerAbout,
		ReferrerVideoURL:     p.ReferrerVideoURL,
		CreatedAt:            p.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:            p.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}
