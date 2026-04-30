package request

// CreateReviewRequest is the payload for POST /api/v1/reviews.
type CreateReviewRequest struct {
	ProposalID    string `json:"proposal_id" validate:"required,uuid"`
	GlobalRating  int    `json:"global_rating" validate:"gte=1,lte=5"`
	Timeliness    *int   `json:"timeliness,omitempty" validate:"omitempty,gte=1,lte=5"`
	Communication *int   `json:"communication,omitempty" validate:"omitempty,gte=1,lte=5"`
	Quality       *int   `json:"quality,omitempty" validate:"omitempty,gte=1,lte=5"`
	Comment       string `json:"comment,omitempty" validate:"omitempty,max=2000"`
	VideoURL      string `json:"video_url,omitempty" validate:"omitempty,url,max=2048"`
	// TitleVisible toggles whether the mission title can be displayed alongside
	// the review on the provider's public project history. Optional in the JSON
	// payload: when omitted, the handler defaults it to true (visible).
	TitleVisible *bool `json:"title_visible,omitempty"`
}
