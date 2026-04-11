package job

import (
	"time"

	"github.com/google/uuid"
)

const applicationMessageMaxLength = 5000

// JobApplication represents a provider's application to a job posting.
// Since phase R3 extended, every application carries the applicant's
// current organization id so any operator of the org sees the
// application in their "my applications" list — the Stripe Dashboard
// shared-workspace model.
type JobApplication struct {
	ID                      uuid.UUID
	JobID                   uuid.UUID
	ApplicantID             uuid.UUID
	ApplicantOrganizationID uuid.UUID
	Message                 string
	VideoURL                *string
	CreatedAt               time.Time
	UpdatedAt               time.Time
}

// NewApplicationInput contains the data required to create a job application.
type NewApplicationInput struct {
	JobID                   uuid.UUID
	ApplicantID             uuid.UUID
	ApplicantOrganizationID uuid.UUID
	Message                 string
	VideoURL                *string
}

// NewJobApplication creates a validated JobApplication from the given input.
func NewJobApplication(input NewApplicationInput) (*JobApplication, error) {
	if input.JobID == uuid.Nil {
		return nil, ErrCannotApplyToClosed // job ID is required
	}
	if input.ApplicantID == uuid.Nil {
		return nil, ErrNotApplicant // applicant ID is required
	}
	if input.ApplicantOrganizationID == uuid.Nil {
		return nil, ErrNotApplicant
	}
	if len([]rune(input.Message)) > applicationMessageMaxLength {
		return nil, ErrApplicationMessageTooLong
	}

	now := time.Now()
	return &JobApplication{
		ID:                      uuid.New(),
		JobID:                   input.JobID,
		ApplicantID:             input.ApplicantID,
		ApplicantOrganizationID: input.ApplicantOrganizationID,
		Message:                 input.Message,
		VideoURL:                input.VideoURL,
		CreatedAt:               now,
		UpdatedAt:               now,
	}, nil
}
