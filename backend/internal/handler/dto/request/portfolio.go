package request

// CreatePortfolioItemRequest is the payload for creating a portfolio item.
type CreatePortfolioItemRequest struct {
	Title       string                `json:"title"`
	Description string                `json:"description,omitempty"`
	LinkURL     string                `json:"link_url,omitempty"`
	Position    int                   `json:"position"`
	Media       []PortfolioMediaInput `json:"media,omitempty"`
}

// PortfolioMediaInput describes a single media attachment.
type PortfolioMediaInput struct {
	MediaURL     string `json:"media_url"`
	MediaType    string `json:"media_type"`
	ThumbnailURL string `json:"thumbnail_url,omitempty"`
	Position     int    `json:"position"`
}

// UpdatePortfolioItemRequest is the payload for updating a portfolio item.
type UpdatePortfolioItemRequest struct {
	Title       *string               `json:"title,omitempty"`
	Description *string               `json:"description,omitempty"`
	LinkURL     *string               `json:"link_url,omitempty"`
	Media       []PortfolioMediaInput `json:"media,omitempty"`
}

// ReorderPortfolioRequest is the payload for reordering portfolio items.
type ReorderPortfolioRequest struct {
	ItemIDs []string `json:"item_ids"`
}
