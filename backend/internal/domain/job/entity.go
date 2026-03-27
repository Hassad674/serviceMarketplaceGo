package job

import (
	"time"

	"github.com/google/uuid"
)

// JobStatus represents the lifecycle state of a job posting.
type JobStatus string

const (
	StatusOpen   JobStatus = "open"
	StatusClosed JobStatus = "closed"
)

func (s JobStatus) IsValid() bool {
	return s == StatusOpen || s == StatusClosed
}

// BudgetType distinguishes project engagement models.
type BudgetType string

const (
	BudgetOneShot  BudgetType = "one_shot"
	BudgetLongTerm BudgetType = "long_term"
)

func (b BudgetType) IsValid() bool {
	return b == BudgetOneShot || b == BudgetLongTerm
}

// ApplicantType restricts who may apply to a job.
type ApplicantType string

const (
	ApplicantAll         ApplicantType = "all"
	ApplicantFreelancers ApplicantType = "freelancers"
	ApplicantAgencies    ApplicantType = "agencies"
)

func (a ApplicantType) IsValid() bool {
	return a == ApplicantAll || a == ApplicantFreelancers || a == ApplicantAgencies
}

// Job represents a public job posting.
type Job struct {
	ID            uuid.UUID
	CreatorID     uuid.UUID
	Title         string
	Description   string
	Skills        []string
	ApplicantType ApplicantType
	BudgetType    BudgetType
	MinBudget     int
	MaxBudget     int
	Status        JobStatus
	CreatedAt     time.Time
	UpdatedAt     time.Time
	ClosedAt      *time.Time
}

// NewJobInput contains the fields required to create a new Job.
type NewJobInput struct {
	CreatorID     uuid.UUID
	Title         string
	Description   string
	Skills        []string
	ApplicantType ApplicantType
	BudgetType    BudgetType
	MinBudget     int
	MaxBudget     int
}

const titleMaxLength = 100
const skillsMaxCount = 5

// NewJob creates a validated Job from the given input.
func NewJob(input NewJobInput) (*Job, error) {
	title := input.Title
	if title == "" {
		return nil, ErrEmptyTitle
	}
	if len(title) > titleMaxLength {
		return nil, ErrTitleTooLong
	}
	if input.Description == "" {
		return nil, ErrEmptyDescription
	}
	if len(input.Skills) > skillsMaxCount {
		return nil, ErrTooManySkills
	}
	if !input.ApplicantType.IsValid() {
		return nil, ErrInvalidApplicantType
	}
	if !input.BudgetType.IsValid() {
		return nil, ErrInvalidBudgetType
	}
	if input.MinBudget <= 0 {
		return nil, ErrInvalidBudget
	}
	if input.MaxBudget <= 0 {
		return nil, ErrInvalidBudget
	}
	if input.MinBudget > input.MaxBudget {
		return nil, ErrMinExceedsMax
	}

	now := time.Now()
	skills := input.Skills
	if skills == nil {
		skills = []string{}
	}

	return &Job{
		ID:            uuid.New(),
		CreatorID:     input.CreatorID,
		Title:         title,
		Description:   input.Description,
		Skills:        skills,
		ApplicantType: input.ApplicantType,
		BudgetType:    input.BudgetType,
		MinBudget:     input.MinBudget,
		MaxBudget:     input.MaxBudget,
		Status:        StatusOpen,
		CreatedAt:     now,
		UpdatedAt:     now,
	}, nil
}

// Close transitions an open job to closed. Only the creator may close it.
func (j *Job) Close(userID uuid.UUID) error {
	if j.CreatorID != userID {
		return ErrNotOwner
	}
	if j.Status != StatusOpen {
		return ErrAlreadyClosed
	}
	now := time.Now()
	j.Status = StatusClosed
	j.ClosedAt = &now
	j.UpdatedAt = now
	return nil
}
