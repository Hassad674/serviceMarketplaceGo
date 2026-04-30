package request

// CreatePortfolioItemRequest is the payload for creating a portfolio item.
type CreatePortfolioItemRequest struct {
	Title       string                `json:"title" validate:"required,min=1,max=200"`
	Description string                `json:"description,omitempty" validate:"omitempty,max=5000"`
	LinkURL     string                `json:"link_url,omitempty" validate:"omitempty,url,max=2048"`
	Position    int                   `json:"position" validate:"gte=0,lte=10000"`
	Media       []PortfolioMediaInput `json:"media,omitempty" validate:"omitempty,max=20,dive"`
}

// PortfolioMediaInput describes a single media attachment.
type PortfolioMediaInput struct {
	MediaURL     string `json:"media_url" validate:"required,url,max=2048"`
	MediaType    string `json:"media_type" validate:"required,min=1,max=50"`
	ThumbnailURL string `json:"thumbnail_url,omitempty" validate:"omitempty,url,max=2048"`
	Position     int    `json:"position" validate:"gte=0,lte=10000"`
}

// UpdatePortfolioItemRequest is the payload for updating a portfolio item.
type UpdatePortfolioItemRequest struct {
	Title       *string               `json:"title,omitempty" validate:"omitempty,min=1,max=200"`
	Description *string               `json:"description,omitempty" validate:"omitempty,max=5000"`
	LinkURL     *string               `json:"link_url,omitempty" validate:"omitempty,url,max=2048"`
	Media       []PortfolioMediaInput `json:"media,omitempty" validate:"omitempty,max=20,dive"`
}

// ReorderPortfolioRequest is the payload for reordering portfolio items.
type ReorderPortfolioRequest struct {
	ItemIDs []string `json:"item_ids" validate:"required,min=1,max=100,dive,uuid"`
}
