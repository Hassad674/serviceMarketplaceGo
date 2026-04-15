package response

import (
	"marketplace-backend/internal/domain/profile"
)

// SocialLinkResponse is the API representation of a social link.
type SocialLinkResponse struct {
	ID        string `json:"id"`
	Persona   string `json:"persona"`
	Platform  string `json:"platform"`
	URL       string `json:"url"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// NewSocialLinkResponse maps a domain SocialLink to its API response.
func NewSocialLinkResponse(link *profile.SocialLink) SocialLinkResponse {
	return SocialLinkResponse{
		ID:        link.ID.String(),
		Persona:   string(link.Persona),
		Platform:  link.Platform,
		URL:       link.URL,
		CreatedAt: link.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt: link.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

// NewSocialLinkListResponse maps a slice of domain SocialLinks to API responses.
func NewSocialLinkListResponse(links []*profile.SocialLink) []SocialLinkResponse {
	result := make([]SocialLinkResponse, len(links))
	for i, link := range links {
		result[i] = NewSocialLinkResponse(link)
	}
	return result
}
