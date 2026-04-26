package job

import "errors"

var (
	ErrJobNotFound             = errors.New("job not found")
	ErrEmptyTitle              = errors.New("job title cannot be empty")
	ErrTitleTooLong            = errors.New("job title exceeds 100 characters")
	ErrEmptyDescription        = errors.New("job description cannot be empty")
	ErrTooManySkills           = errors.New("job cannot have more than 5 skills")
	ErrInvalidApplicantType    = errors.New("invalid applicant type")
	ErrInvalidBudgetType       = errors.New("invalid budget type")
	ErrInvalidBudget           = errors.New("budget must be greater than zero")
	ErrMinExceedsMax           = errors.New("minimum budget cannot exceed maximum budget")
	ErrNotOwner                = errors.New("not the owner of this job")
	ErrAlreadyClosed           = errors.New("job is already closed")
	ErrAlreadyOpen             = errors.New("job is already open")
	ErrUnauthorizedRole        = errors.New("only enterprises and agencies can create jobs")
	ErrInvalidPaymentFrequency = errors.New("invalid payment frequency")
	ErrInvalidDescriptionType  = errors.New("invalid description type")
	ErrVideoURLRequired        = errors.New("video URL is required for video description")

	// Job application errors.
	ErrApplicationNotFound       = errors.New("job application not found")
	ErrAlreadyApplied            = errors.New("already applied to this job")
	ErrCannotApplyToOwnJob       = errors.New("cannot apply to your own job")
	ErrCannotApplyToClosed       = errors.New("cannot apply to a closed job")
	ErrEmptyApplicationMessage   = errors.New("application message cannot be empty")
	ErrApplicationMessageTooLong = errors.New("application message exceeds maximum length")
	ErrNotApplicant              = errors.New("not the applicant of this application")
	ErrApplicantTypeMismatch     = errors.New("your role does not match the required applicant type")
	ErrNoCreditsLeft             = errors.New("no application credits remaining")

	// Phase 2 moderation gate. Returned when the synchronous text
	// moderation refuses a job title (strict 0.50 threshold) or
	// description (looser 0.85 threshold). HTTP 422.
	ErrJobTitleInappropriate       = errors.New("job title inappropriate")
	ErrJobDescriptionInappropriate = errors.New("job description inappropriate")
)
