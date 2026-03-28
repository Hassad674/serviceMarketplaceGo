package response

import (
	"time"

	"marketplace-backend/internal/domain/review"
)

// ReviewResponse is the API representation of a review.
type ReviewResponse struct {
	ID            string    `json:"id"`
	ProposalID    string    `json:"proposal_id"`
	ReviewerID    string    `json:"reviewer_id"`
	ReviewedID    string    `json:"reviewed_id"`
	GlobalRating  int       `json:"global_rating"`
	Timeliness    *int      `json:"timeliness"`
	Communication *int      `json:"communication"`
	Quality       *int      `json:"quality"`
	Comment       string    `json:"comment"`
	VideoURL      string    `json:"video_url"`
	CreatedAt     time.Time `json:"created_at"`
}

// ReviewFromDomain converts a domain review to an API response.
func ReviewFromDomain(r *review.Review) ReviewResponse {
	return ReviewResponse{
		ID:            r.ID.String(),
		ProposalID:    r.ProposalID.String(),
		ReviewerID:    r.ReviewerID.String(),
		ReviewedID:    r.ReviewedID.String(),
		GlobalRating:  r.GlobalRating,
		Timeliness:    r.Timeliness,
		Communication: r.Communication,
		Quality:       r.Quality,
		Comment:       r.Comment,
		VideoURL:      r.VideoURL,
		CreatedAt:     r.CreatedAt,
	}
}

// AverageRatingResponse is the API representation of an average rating.
type AverageRatingResponse struct {
	Average float64 `json:"average"`
	Count   int     `json:"count"`
}
