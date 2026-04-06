package response

import (
	"time"

	"marketplace-backend/internal/port/repository"
)

// AdminReviewResponse is the JSON response for admin review listing/detail.
type AdminReviewResponse struct {
	ID                 string               `json:"id"`
	ProposalID         string               `json:"proposal_id"`
	GlobalRating       int                  `json:"global_rating"`
	Timeliness         *int                 `json:"timeliness,omitempty"`
	Communication      *int                 `json:"communication,omitempty"`
	Quality            *int                 `json:"quality,omitempty"`
	Comment            string               `json:"comment"`
	VideoURL           string               `json:"video_url,omitempty"`
	CreatedAt          string               `json:"created_at"`
	UpdatedAt          string               `json:"updated_at"`
	PendingReportCount int                  `json:"pending_report_count"`
	Reviewer           AdminReviewUserBrief `json:"reviewer"`
	Reviewed           AdminReviewUserBrief `json:"reviewed"`
}

// AdminReviewUserBrief is a lightweight user summary embedded in an admin review response.
type AdminReviewUserBrief struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
	Role        string `json:"role"`
}

// NewAdminReviewResponse converts an AdminReview to its JSON response.
func NewAdminReviewResponse(r repository.AdminReview) AdminReviewResponse {
	return AdminReviewResponse{
		ID:                 r.ID.String(),
		ProposalID:         r.ProposalID.String(),
		GlobalRating:       r.GlobalRating,
		Timeliness:         r.Timeliness,
		Communication:      r.Communication,
		Quality:            r.Quality,
		Comment:            r.Comment,
		VideoURL:           r.VideoURL,
		CreatedAt:          r.CreatedAt.Format(time.RFC3339),
		UpdatedAt:          r.UpdatedAt.Format(time.RFC3339),
		PendingReportCount: r.PendingReportCount,
		Reviewer: AdminReviewUserBrief{
			ID:          r.ReviewerID.String(),
			DisplayName: r.ReviewerDisplayName,
			Email:       r.ReviewerEmail,
			Role:        r.ReviewerRole,
		},
		Reviewed: AdminReviewUserBrief{
			ID:          r.ReviewedID.String(),
			DisplayName: r.ReviewedDisplayName,
			Email:       r.ReviewedEmail,
			Role:        r.ReviewedRole,
		},
	}
}
