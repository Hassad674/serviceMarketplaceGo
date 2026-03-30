package response

import (
	"time"

	"marketplace-backend/internal/domain/report"
)

// ReportResponse is the API representation of a report.
type ReportResponse struct {
	ID          string    `json:"id"`
	TargetType  string    `json:"target_type"`
	TargetID    string    `json:"target_id"`
	Reason      string    `json:"reason"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

// ReportFromDomain converts a domain report to an API response.
func ReportFromDomain(r *report.Report) ReportResponse {
	return ReportResponse{
		ID:          r.ID.String(),
		TargetType:  string(r.TargetType),
		TargetID:    r.TargetID.String(),
		Reason:      string(r.Reason),
		Description: r.Description,
		Status:      string(r.Status),
		CreatedAt:   r.CreatedAt,
	}
}
