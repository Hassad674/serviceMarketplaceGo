package request

type CreateJobRequest struct {
	Title            string   `json:"title"`
	Description      string   `json:"description"`
	Skills           []string `json:"skills"`
	ApplicantType    string   `json:"applicant_type"`
	BudgetType       string   `json:"budget_type"`
	MinBudget        int      `json:"min_budget"`
	MaxBudget        int      `json:"max_budget"`
	PaymentFrequency *string  `json:"payment_frequency,omitempty"`
	DurationWeeks    *int     `json:"duration_weeks,omitempty"`
	IsIndefinite     bool     `json:"is_indefinite"`
	DescriptionType  string   `json:"description_type"`
	VideoURL         *string  `json:"video_url,omitempty"`
}
