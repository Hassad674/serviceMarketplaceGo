package request

// ApplyToJobRequest is the body of POST /api/v1/jobs/{id}/apply.
type ApplyToJobRequest struct {
	Message  string  `json:"message" validate:"required,min=1,max=5000"`
	VideoURL *string `json:"video_url,omitempty" validate:"omitempty,url,max=2048"`
}

// CreateJobRequest is the body of POST /api/v1/jobs.
//
// MinBudget / MaxBudget are in centimes — capped at 1 000 000 000 (= 10M €)
// to keep Stripe's int32 amount surface safe even when summed across many
// milestones in a single proposal.
type CreateJobRequest struct {
	Title            string   `json:"title" validate:"required,min=1,max=200"`
	Description      string   `json:"description" validate:"required,min=1,max=10000"`
	Skills           []string `json:"skills" validate:"omitempty,max=30,dive,min=1,max=100"`
	ApplicantType    string   `json:"applicant_type" validate:"required,min=1,max=50"`
	BudgetType       string   `json:"budget_type" validate:"required,min=1,max=50"`
	MinBudget        int      `json:"min_budget" validate:"gte=0,lte=1000000000"`
	MaxBudget        int      `json:"max_budget" validate:"gte=0,lte=1000000000"`
	PaymentFrequency *string  `json:"payment_frequency,omitempty" validate:"omitempty,max=50"`
	DurationWeeks    *int     `json:"duration_weeks,omitempty" validate:"omitempty,gte=0,lte=520"`
	IsIndefinite     bool     `json:"is_indefinite"`
	DescriptionType  string   `json:"description_type" validate:"required,min=1,max=50"`
	VideoURL         *string  `json:"video_url,omitempty" validate:"omitempty,url,max=2048"`
}
