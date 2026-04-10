package request

// CreateReviewRequest is the payload for POST /api/v1/reviews.
type CreateReviewRequest struct {
	ProposalID    string `json:"proposal_id"`
	GlobalRating  int    `json:"global_rating"`
	Timeliness    *int   `json:"timeliness,omitempty"`
	Communication *int   `json:"communication,omitempty"`
	Quality       *int   `json:"quality,omitempty"`
	Comment       string `json:"comment,omitempty"`
	VideoURL      string `json:"video_url,omitempty"`
	// TitleVisible toggles whether the mission title can be displayed alongside
	// the review on the provider's public project history. Optional in the JSON
	// payload: when omitted, the handler defaults it to true (visible).
	TitleVisible *bool `json:"title_visible,omitempty"`
}
