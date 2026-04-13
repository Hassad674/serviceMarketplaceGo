package response

import (
	"time"

	"marketplace-backend/internal/domain/review"
)

// ReviewResponse is the API representation of a review.
//
// Since phase R18 (double-blind reviews) the response carries the review
// Side and its PublishedAt timestamp. PublishedAt is nil when the review
// is still hidden — the public GET endpoints filter those out, but admin
// and authenticated owner-side reads may return them.
type ReviewResponse struct {
	ID            string     `json:"id"`
	ProposalID    string     `json:"proposal_id"`
	ReviewerID    string     `json:"reviewer_id"`
	ReviewedID    string     `json:"reviewed_id"`
	Side          string     `json:"side"`
	GlobalRating  int        `json:"global_rating"`
	Timeliness    *int       `json:"timeliness"`
	Communication *int       `json:"communication"`
	Quality       *int       `json:"quality"`
	Comment       string     `json:"comment"`
	VideoURL      string     `json:"video_url"`
	TitleVisible  bool       `json:"title_visible"`
	CreatedAt     time.Time  `json:"created_at"`
	PublishedAt   *time.Time `json:"published_at"`
}

// ReviewFromDomain converts a domain review to an API response.
func ReviewFromDomain(r *review.Review) ReviewResponse {
	return ReviewResponse{
		ID:            r.ID.String(),
		ProposalID:    r.ProposalID.String(),
		ReviewerID:    r.ReviewerID.String(),
		ReviewedID:    r.ReviewedID.String(),
		Side:          r.Side,
		GlobalRating:  r.GlobalRating,
		Timeliness:    r.Timeliness,
		Communication: r.Communication,
		Quality:       r.Quality,
		Comment:       r.Comment,
		VideoURL:      r.VideoURL,
		TitleVisible:  r.TitleVisible,
		CreatedAt:     r.CreatedAt,
		PublishedAt:   r.PublishedAt,
	}
}

// AverageRatingResponse is the API representation of an average rating.
type AverageRatingResponse struct {
	Average float64 `json:"average"`
	Count   int     `json:"count"`
}
