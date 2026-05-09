package job

import (
	"time"

	"github.com/google/uuid"
)

const applicationMessageMaxLength = 5000

// ApplicantKind names the role under which the applicant submits the
// application. A provider with referrer_enabled=true picks between
// freelance (do the work themselves) and referrer (broker the deal).
// Pure agencies always submit as 'agency'. The kind is decided at
// apply time and persisted alongside the application so the employer's
// candidates list can filter without recomputing it.
type ApplicantKind string

const (
	ApplicantKindFreelance ApplicantKind = "freelance"
	ApplicantKindAgency    ApplicantKind = "agency"
	ApplicantKindReferrer  ApplicantKind = "referrer"
)

// IsValid returns true when the kind is one of the three persisted values.
func (k ApplicantKind) IsValid() bool {
	switch k {
	case ApplicantKindFreelance, ApplicantKindAgency, ApplicantKindReferrer:
		return true
	default:
		return false
	}
}

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
	ApplicantKind           ApplicantKind
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
	ApplicantKind           ApplicantKind
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
	if !input.ApplicantKind.IsValid() {
		return nil, ErrInvalidApplicantKind
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
		ApplicantKind:           input.ApplicantKind,
		Message:                 input.Message,
		VideoURL:                input.VideoURL,
		CreatedAt:               now,
		UpdatedAt:               now,
	}, nil
}
