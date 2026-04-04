package response

import (
	"time"

	adminapp "marketplace-backend/internal/app/admin"
)

// AdminJobResponse is the JSON response for admin job listing/detail.
type AdminJobResponse struct {
	ID               string        `json:"id"`
	Title            string        `json:"title"`
	Description      string        `json:"description"`
	Skills           []string      `json:"skills"`
	ApplicantType    string        `json:"applicant_type"`
	BudgetType       string        `json:"budget_type"`
	MinBudget        int           `json:"min_budget"`
	MaxBudget        int           `json:"max_budget"`
	Status           string        `json:"status"`
	CreatedAt        string        `json:"created_at"`
	UpdatedAt        string        `json:"updated_at"`
	ClosedAt         *string       `json:"closed_at,omitempty"`
	PaymentFrequency *string       `json:"payment_frequency,omitempty"`
	DurationWeeks    *int          `json:"duration_weeks,omitempty"`
	IsIndefinite     bool          `json:"is_indefinite"`
	DescriptionType  string        `json:"description_type"`
	VideoURL         *string       `json:"video_url,omitempty"`
	ApplicationCount int           `json:"application_count"`
	Author           AdminJobAuthor `json:"author"`
}

// AdminJobAuthor is the author info embedded in an admin job response.
type AdminJobAuthor struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
	Role        string `json:"role"`
}

// NewAdminJobResponse converts an admin job to its JSON response.
func NewAdminJobResponse(j adminapp.AdminJob) AdminJobResponse {
	skills := j.Skills
	if skills == nil {
		skills = []string{}
	}

	resp := AdminJobResponse{
		ID:               j.ID.String(),
		Title:            j.Title,
		Description:      j.Description,
		Skills:           skills,
		ApplicantType:    j.ApplicantType,
		BudgetType:       j.BudgetType,
		MinBudget:        j.MinBudget,
		MaxBudget:        j.MaxBudget,
		Status:           j.Status,
		CreatedAt:        j.CreatedAt.Format(time.RFC3339),
		UpdatedAt:        j.UpdatedAt.Format(time.RFC3339),
		PaymentFrequency: j.PaymentFrequency,
		DurationWeeks:    j.DurationWeeks,
		IsIndefinite:     j.IsIndefinite,
		DescriptionType:  j.DescriptionType,
		VideoURL:         j.VideoURL,
		ApplicationCount: j.ApplicationCount,
		Author: AdminJobAuthor{
			ID:          j.CreatorID.String(),
			DisplayName: j.AuthorDisplayName,
			Email:       j.AuthorEmail,
			Role:        j.AuthorRole,
		},
	}

	if j.ClosedAt != nil {
		s := j.ClosedAt.Format(time.RFC3339)
		resp.ClosedAt = &s
	}

	return resp
}

// AdminJobApplicationResponse is the JSON response for admin job application listing.
type AdminJobApplicationResponse struct {
	ID        string                      `json:"id"`
	Message   string                      `json:"message"`
	VideoURL  *string                     `json:"video_url,omitempty"`
	CreatedAt string                      `json:"created_at"`
	UpdatedAt string                      `json:"updated_at"`
	Candidate AdminJobApplicationCandidate `json:"candidate"`
	Job       AdminJobApplicationJob       `json:"job"`
}

// AdminJobApplicationCandidate is the candidate info embedded in an application response.
type AdminJobApplicationCandidate struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
	Role        string `json:"role"`
}

// AdminJobApplicationJob is the job info embedded in an application response.
type AdminJobApplicationJob struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Status string `json:"status"`
}

// NewAdminJobApplicationResponse converts an admin job application to its JSON response.
func NewAdminJobApplicationResponse(a adminapp.AdminJobApplication) AdminJobApplicationResponse {
	return AdminJobApplicationResponse{
		ID:        a.ID.String(),
		Message:   a.Message,
		VideoURL:  a.VideoURL,
		CreatedAt: a.CreatedAt.Format(time.RFC3339),
		UpdatedAt: a.UpdatedAt.Format(time.RFC3339),
		Candidate: AdminJobApplicationCandidate{
			ID:          a.ApplicantID.String(),
			DisplayName: a.CandidateDisplayName,
			Email:       a.CandidateEmail,
			Role:        a.CandidateRole,
		},
		Job: AdminJobApplicationJob{
			ID:     a.JobID.String(),
			Title:  a.JobTitle,
			Status: a.JobStatus,
		},
	}
}
