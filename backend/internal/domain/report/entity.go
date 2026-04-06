package report

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

// TargetType represents what kind of resource is being reported.
type TargetType string

const (
	TargetMessage     TargetType = "message"
	TargetUser        TargetType = "user"
	TargetJob         TargetType = "job"
	TargetApplication TargetType = "application"
	TargetReview      TargetType = "review"
)

// Reason represents why the report was filed.
type Reason string

const (
	ReasonHarassment             Reason = "harassment"
	ReasonFraud                  Reason = "fraud"
	ReasonFraudOrScam            Reason = "fraud_or_scam"
	ReasonSpam                   Reason = "spam"
	ReasonInappropriateContent   Reason = "inappropriate_content"
	ReasonFakeProfile            Reason = "fake_profile"
	ReasonUnprofessionalBehavior Reason = "unprofessional_behavior"
	ReasonMisleadingDescription  Reason = "misleading_description"
	ReasonOther                  Reason = "other"
)

// Status represents the lifecycle state of a report.
type Status string

const (
	StatusPending   Status = "pending"
	StatusReviewed  Status = "reviewed"
	StatusResolved  Status = "resolved"
	StatusDismissed Status = "dismissed"
)

// MaxDescriptionLength is the maximum allowed length for a report description.
const MaxDescriptionLength = 2000

var validMessageReasons = map[Reason]bool{
	ReasonHarassment:           true,
	ReasonFraud:                true,
	ReasonSpam:                 true,
	ReasonInappropriateContent: true,
	ReasonOther:                true,
}

var validUserReasons = map[Reason]bool{
	ReasonHarassment:             true,
	ReasonFraud:                  true,
	ReasonSpam:                   true,
	ReasonFakeProfile:            true,
	ReasonUnprofessionalBehavior: true,
	ReasonOther:                  true,
}

var validJobReasons = map[Reason]bool{
	ReasonFraudOrScam:           true,
	ReasonMisleadingDescription: true,
	ReasonInappropriateContent:  true,
	ReasonSpam:                  true,
	ReasonOther:                 true,
}

var validApplicationReasons = map[Reason]bool{
	ReasonFraudOrScam:          true,
	ReasonSpam:                 true,
	ReasonInappropriateContent: true,
	ReasonOther:                true,
}

// Report represents a user-submitted report against a message or user.
type Report struct {
	ID             uuid.UUID
	ReporterID     uuid.UUID
	TargetType     TargetType
	TargetID       uuid.UUID
	ConversationID uuid.UUID
	Reason         Reason
	Description    string
	Status         Status
	AdminNote      string
	ResolvedAt     *time.Time
	ResolvedBy     *uuid.UUID
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// NewReportInput groups parameters for creating a new Report.
type NewReportInput struct {
	ReporterID     uuid.UUID
	TargetType     TargetType
	TargetID       uuid.UUID
	ConversationID uuid.UUID
	Reason         Reason
	Description    string
}

// NewReport creates a validated Report from the given input.
func NewReport(in NewReportInput) (*Report, error) {
	if in.ReporterID == uuid.Nil {
		return nil, ErrMissingReporter
	}
	if in.TargetID == uuid.Nil {
		return nil, ErrMissingTarget
	}
	reasonsByTarget := map[TargetType]map[Reason]bool{
		TargetMessage:     validMessageReasons,
		TargetUser:        validUserReasons,
		TargetJob:         validJobReasons,
		TargetApplication: validApplicationReasons,
	}
	allowed, ok := reasonsByTarget[in.TargetType]
	if !ok {
		return nil, ErrInvalidTargetType
	}
	if in.TargetType == TargetUser && in.ReporterID == in.TargetID {
		return nil, ErrSelfReport
	}
	if !allowed[in.Reason] {
		return nil, ErrReasonNotAllowedForType
	}

	desc := strings.TrimSpace(in.Description)
	if len(desc) > MaxDescriptionLength {
		return nil, ErrDescriptionTooLong
	}

	now := time.Now()
	return &Report{
		ID:             uuid.New(),
		ReporterID:     in.ReporterID,
		TargetType:     in.TargetType,
		TargetID:       in.TargetID,
		ConversationID: in.ConversationID,
		Reason:         in.Reason,
		Description:    desc,
		Status:         StatusPending,
		CreatedAt:      now,
		UpdatedAt:      now,
	}, nil
}
