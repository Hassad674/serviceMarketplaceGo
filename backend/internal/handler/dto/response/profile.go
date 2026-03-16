package response

import (
	"marketplace-backend/internal/domain/profile"
)

type ProfileResponse struct {
	UserID               string `json:"user_id"`
	Title                string `json:"title"`
	PhotoURL             string `json:"photo_url"`
	PresentationVideoURL string `json:"presentation_video_url"`
	ReferrerVideoURL     string `json:"referrer_video_url"`
	CreatedAt            string `json:"created_at"`
	UpdatedAt            string `json:"updated_at"`
}

func NewProfileResponse(p *profile.Profile) ProfileResponse {
	return ProfileResponse{
		UserID:               p.UserID.String(),
		Title:                p.Title,
		PhotoURL:             p.PhotoURL,
		PresentationVideoURL: p.PresentationVideoURL,
		ReferrerVideoURL:     p.ReferrerVideoURL,
		CreatedAt:            p.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:            p.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}
