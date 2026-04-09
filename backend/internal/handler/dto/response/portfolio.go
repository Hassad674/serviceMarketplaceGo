package response

import (
	"marketplace-backend/internal/domain/portfolio"
)

// PortfolioItemResponse is the API representation of a portfolio item.
type PortfolioItemResponse struct {
	ID          string                   `json:"id"`
	UserID      string                   `json:"user_id"`
	Title       string                   `json:"title"`
	Description string                   `json:"description"`
	LinkURL     string                   `json:"link_url"`
	CoverURL    string                   `json:"cover_url"`
	Position    int                      `json:"position"`
	Media       []PortfolioMediaResponse `json:"media"`
	CreatedAt   string                   `json:"created_at"`
	UpdatedAt   string                   `json:"updated_at"`
}

// PortfolioMediaResponse is the API representation of a portfolio media.
type PortfolioMediaResponse struct {
	ID           string `json:"id"`
	MediaURL     string `json:"media_url"`
	MediaType    string `json:"media_type"`
	ThumbnailURL string `json:"thumbnail_url"`
	Position     int    `json:"position"`
	CreatedAt    string `json:"created_at"`
}

// PortfolioItemFromDomain maps a domain entity to an API response.
func PortfolioItemFromDomain(item *portfolio.PortfolioItem) PortfolioItemResponse {
	media := make([]PortfolioMediaResponse, 0, len(item.Media))
	for _, m := range item.Media {
		media = append(media, PortfolioMediaResponse{
			ID:           m.ID.String(),
			MediaURL:     m.MediaURL,
			MediaType:    string(m.MediaType),
			ThumbnailURL: m.ThumbnailURL,
			Position:     m.Position,
			CreatedAt:    m.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	return PortfolioItemResponse{
		ID:          item.ID.String(),
		UserID:      item.UserID.String(),
		Title:       item.Title,
		Description: item.Description,
		LinkURL:     item.LinkURL,
		CoverURL:    item.CoverURL(),
		Position:    item.Position,
		Media:       media,
		CreatedAt:   item.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   item.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

// PortfolioListFromDomain maps a slice of domain entities to API responses.
func PortfolioListFromDomain(items []*portfolio.PortfolioItem) []PortfolioItemResponse {
	result := make([]PortfolioItemResponse, 0, len(items))
	for _, item := range items {
		result = append(result, PortfolioItemFromDomain(item))
	}
	return result
}
