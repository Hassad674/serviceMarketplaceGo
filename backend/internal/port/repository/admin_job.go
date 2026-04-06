package repository

import (
	"time"

	"github.com/google/uuid"
)

// AdminJobFilters holds query parameters for admin job listing.
type AdminJobFilters struct {
	Status string
	Search string
	Sort   string
	Filter string
	Cursor string
	Limit  int
	Page   int
}

// AdminJob represents a job with author info for admin listing.
type AdminJob struct {
	ID               uuid.UUID
	CreatorID        uuid.UUID
	Title            string
	Description      string
	Skills           []string
	ApplicantType    string
	BudgetType       string
	MinBudget        int
	MaxBudget        int
	Status           string
	CreatedAt        time.Time
	UpdatedAt        time.Time
	ClosedAt         *time.Time
	PaymentFrequency *string
	DurationWeeks    *int
	IsIndefinite     bool
	DescriptionType  string
	VideoURL         *string
	ApplicationCount   int
	PendingReportCount int

	AuthorDisplayName string
	AuthorEmail       string
	AuthorRole        string
}

// AdminApplicationFilters holds query parameters for admin application listing.
type AdminApplicationFilters struct {
	JobID  string
	Search string
	Sort   string
	Filter string
	Cursor string
	Limit  int
	Page   int
}

// AdminJobApplication represents a job application with candidate and job info.
type AdminJobApplication struct {
	ID          uuid.UUID
	JobID       uuid.UUID
	ApplicantID uuid.UUID
	Message     string
	VideoURL    *string
	CreatedAt   time.Time
	UpdatedAt   time.Time

	CandidateDisplayName string
	CandidateEmail       string
	CandidateRole        string

	JobTitle           string
	JobStatus          string
	PendingReportCount int
}
