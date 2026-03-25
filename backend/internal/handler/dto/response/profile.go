package response

import (
	"marketplace-backend/internal/domain/profile"
)

type ProfileResponse struct {
	UserID               string `json:"user_id"`
	Title                string `json:"title"`
	About                string `json:"about"`
	PhotoURL             string `json:"photo_url"`
	PresentationVideoURL string `json:"presentation_video_url"`
	ReferrerAbout        string `json:"referrer_about"`
	ReferrerVideoURL     string `json:"referrer_video_url"`
	CreatedAt            string `json:"created_at"`
	UpdatedAt            string `json:"updated_at"`
}

type PublicProfileSummary struct {
	UserID          string `json:"user_id"`
	DisplayName     string `json:"display_name"`
	FirstName       string `json:"first_name"`
	LastName        string `json:"last_name"`
	Role            string `json:"role"`
	Title           string `json:"title"`
	PhotoURL        string `json:"photo_url"`
	ReferrerEnabled bool   `json:"referrer_enabled"`
}

func NewPublicProfileSummary(p *profile.PublicProfile) PublicProfileSummary {
	return PublicProfileSummary{
		UserID:          p.UserID.String(),
		DisplayName:     p.DisplayName,
		FirstName:       p.FirstName,
		LastName:        p.LastName,
		Role:            p.Role,
		Title:           p.Title,
		PhotoURL:        p.PhotoURL,
		ReferrerEnabled: p.ReferrerEnabled,
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
		UserID:               p.UserID.String(),
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
