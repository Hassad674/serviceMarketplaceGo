package job

import (
	"time"

	"github.com/google/uuid"
)

type JobStatus string

const (
	StatusOpen   JobStatus = "open"
	StatusClosed JobStatus = "closed"
)

func (s JobStatus) IsValid() bool {
	return s == StatusOpen || s == StatusClosed
}

type BudgetType string

const (
	BudgetOneShot  BudgetType = "one_shot"
	BudgetLongTerm BudgetType = "long_term"
)

func (b BudgetType) IsValid() bool {
	return b == BudgetOneShot || b == BudgetLongTerm
}

type ApplicantType string

const (
	ApplicantAll         ApplicantType = "all"
	ApplicantFreelancers ApplicantType = "freelancers"
	ApplicantAgencies    ApplicantType = "agencies"
)

func (a ApplicantType) IsValid() bool {
	return a == ApplicantAll || a == ApplicantFreelancers || a == ApplicantAgencies
}

type PaymentFrequency string

const (
	FrequencyWeekly  PaymentFrequency = "weekly"
	FrequencyMonthly PaymentFrequency = "monthly"
)

func (f PaymentFrequency) IsValid() bool {
	return f == FrequencyWeekly || f == FrequencyMonthly
}

type DescriptionType string

const (
	DescriptionText  DescriptionType = "text"
	DescriptionVideo DescriptionType = "video"
	DescriptionBoth  DescriptionType = "both"
)

func (d DescriptionType) IsValid() bool {
	return d == DescriptionText || d == DescriptionVideo || d == DescriptionBoth
}

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

	PaymentFrequency *PaymentFrequency
	DurationWeeks    *int
	IsIndefinite     bool

	DescriptionType DescriptionType
	VideoURL        *string
}

type NewJobInput struct {
	CreatorID     uuid.UUID
	Title         string
	Description   string
	Skills        []string
	ApplicantType ApplicantType
	BudgetType    BudgetType
	MinBudget     int
	MaxBudget     int

	PaymentFrequency *PaymentFrequency
	DurationWeeks    *int
	IsIndefinite     bool

	DescriptionType DescriptionType
	VideoURL        *string
}

const titleMaxLength = 100
const skillsMaxCount = 5

func NewJob(input NewJobInput) (*Job, error) {
	if err := validateJobInput(input); err != nil {
		return nil, err
	}

	now := time.Now()
	skills := input.Skills
	if skills == nil {
		skills = []string{}
	}

	descType := input.DescriptionType
	if descType == "" {
		descType = DescriptionText
	}

	return &Job{
		ID:               uuid.New(),
		CreatorID:        input.CreatorID,
		Title:            input.Title,
		Description:      input.Description,
		Skills:           skills,
		ApplicantType:    input.ApplicantType,
		BudgetType:       input.BudgetType,
		MinBudget:        input.MinBudget,
		MaxBudget:        input.MaxBudget,
		Status:           StatusOpen,
		CreatedAt:        now,
		UpdatedAt:        now,
		PaymentFrequency: input.PaymentFrequency,
		DurationWeeks:    input.DurationWeeks,
		IsIndefinite:     input.IsIndefinite,
		DescriptionType:  descType,
		VideoURL:         input.VideoURL,
	}, nil
}

func validateJobInput(input NewJobInput) error {
	if input.Title == "" {
		return ErrEmptyTitle
	}
	if len(input.Title) > titleMaxLength {
		return ErrTitleTooLong
	}
	dt := input.DescriptionType
	if dt == "" {
		dt = DescriptionText
	}
	if dt != DescriptionVideo && input.Description == "" {
		return ErrEmptyDescription
	}
	if len(input.Skills) > skillsMaxCount {
		return ErrTooManySkills
	}
	if !input.ApplicantType.IsValid() {
		return ErrInvalidApplicantType
	}
	if !input.BudgetType.IsValid() {
		return ErrInvalidBudgetType
	}
	if input.MinBudget <= 0 || input.MaxBudget <= 0 {
		return ErrInvalidBudget
	}
	if input.MinBudget > input.MaxBudget {
		return ErrMinExceedsMax
	}
	if input.BudgetType == BudgetLongTerm && input.PaymentFrequency != nil && !input.PaymentFrequency.IsValid() {
		return ErrInvalidPaymentFrequency
	}
	if dt != "" && dt.IsValid() && (dt == DescriptionVideo || dt == DescriptionBoth) && input.VideoURL == nil {
		return ErrVideoURLRequired
	}
	if dt != "" && !dt.IsValid() {
		return ErrInvalidDescriptionType
	}
	return nil
}

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

func (j *Job) Reopen(userID uuid.UUID) error {
	if j.CreatorID != userID {
		return ErrNotOwner
	}
	if j.Status != StatusClosed {
		return ErrAlreadyOpen
	}
	j.Status = StatusOpen
	j.ClosedAt = nil
	j.UpdatedAt = time.Now()
	return nil
}
