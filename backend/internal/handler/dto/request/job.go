package request

// CreateJobRequest is the expected body for POST /api/v1/jobs.
type CreateJobRequest struct {
	Title         string   `json:"title"`
	Description   string   `json:"description"`
	Skills        []string `json:"skills"`
	ApplicantType string   `json:"applicant_type"`
	BudgetType    string   `json:"budget_type"`
	MinBudget     int      `json:"min_budget"`
	MaxBudget     int      `json:"max_budget"`
}
