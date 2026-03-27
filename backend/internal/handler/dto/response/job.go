package response

import (
	"time"
	"marketplace-backend/internal/domain/job"
)

type JobResponse struct {
	ID               string   `json:"id"`
	CreatorID        string   `json:"creator_id"`
	Title            string   `json:"title"`
	Description      string   `json:"description"`
	Skills           []string `json:"skills"`
	ApplicantType    string   `json:"applicant_type"`
	BudgetType       string   `json:"budget_type"`
	MinBudget        int      `json:"min_budget"`
	MaxBudget        int      `json:"max_budget"`
	Status           string   `json:"status"`
	CreatedAt        string   `json:"created_at"`
	UpdatedAt        string   `json:"updated_at"`
	ClosedAt         *string  `json:"closed_at,omitempty"`
	PaymentFrequency *string  `json:"payment_frequency,omitempty"`
	DurationWeeks    *int     `json:"duration_weeks,omitempty"`
	IsIndefinite     bool     `json:"is_indefinite"`
	DescriptionType  string   `json:"description_type"`
	VideoURL         *string  `json:"video_url,omitempty"`
}

type JobListResponse struct {
	Data       []JobResponse `json:"data"`
	NextCursor string        `json:"next_cursor"`
	HasMore    bool          `json:"has_more"`
}

func NewJobResponse(j *job.Job) JobResponse {
	resp := JobResponse{
		ID:              j.ID.String(),
		CreatorID:       j.CreatorID.String(),
		Title:           j.Title,
		Description:     j.Description,
		Skills:          j.Skills,
		ApplicantType:   string(j.ApplicantType),
		BudgetType:      string(j.BudgetType),
		MinBudget:       j.MinBudget,
		MaxBudget:       j.MaxBudget,
		Status:          string(j.Status),
		CreatedAt:       j.CreatedAt.Format(time.RFC3339),
		UpdatedAt:       j.UpdatedAt.Format(time.RFC3339),
		IsIndefinite:    j.IsIndefinite,
		DescriptionType: string(j.DescriptionType),
		DurationWeeks:   j.DurationWeeks,
		VideoURL:        j.VideoURL,
	}
	if j.PaymentFrequency != nil {
		s := string(*j.PaymentFrequency)
		resp.PaymentFrequency = &s
	}
	if j.ClosedAt != nil {
		t := j.ClosedAt.Format(time.RFC3339)
		resp.ClosedAt = &t
	}
	if resp.Skills == nil {
		resp.Skills = []string{}
	}
	if resp.DescriptionType == "" {
		resp.DescriptionType = "text"
	}
	return resp
}

func NewJobListResponse(jobs []*job.Job, nextCursor string) JobListResponse {
	data := make([]JobResponse, len(jobs))
	for i, j := range jobs {
		data[i] = NewJobResponse(j)
	}
	return JobListResponse{
		Data:       data,
		NextCursor: nextCursor,
		HasMore:    nextCursor != "",
	}
}
