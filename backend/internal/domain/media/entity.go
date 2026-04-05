package media

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

// ModerationStatus represents the moderation lifecycle of a media item.
type ModerationStatus string

const (
	StatusPending  ModerationStatus = "pending"
	StatusApproved ModerationStatus = "approved"
	StatusFlagged  ModerationStatus = "flagged"
	StatusRejected ModerationStatus = "rejected"
)

func (s ModerationStatus) IsValid() bool {
	switch s {
	case StatusPending, StatusApproved, StatusFlagged, StatusRejected:
		return true
	}
	return false
}

// Context represents where the media is used.
type Context string

const (
	ContextProfilePhoto     Context = "profile_photo"
	ContextProfileVideo     Context = "profile_video"
	ContextMessageAttach    Context = "message_attachment"
	ContextReviewVideo      Context = "review_video"
	ContextJobVideo         Context = "job_video"
	ContextReferrerVideo    Context = "referrer_video"
	ContextIdentityDocument Context = "identity_document"
)

func (c Context) IsValid() bool {
	switch c {
	case ContextProfilePhoto, ContextProfileVideo, ContextMessageAttach,
		ContextReviewVideo, ContextJobVideo, ContextReferrerVideo,
		ContextIdentityDocument:
		return true
	}
	return false
}

// ModerationLabel represents a single label returned by content moderation.
type ModerationLabel struct {
	Name       string  `json:"name"`
	Confidence float64 `json:"confidence"`
	ParentName string  `json:"parent_name,omitempty"`
}

// Media represents an uploaded media item tracked for moderation.
type Media struct {
	ID                uuid.UUID
	UploaderID        uuid.UUID
	FileURL           string
	FileName          string
	FileType          string
	FileSize          int64
	Context           Context
	ContextID         *uuid.UUID
	ModerationStatus  ModerationStatus
	ModerationLabels  []ModerationLabel
	ModerationScore   float64
	RekognitionJobID  *string
	ReviewedAt        *time.Time
	ReviewedBy        *uuid.UUID
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// NewMediaInput groups parameters for creating a new Media record.
type NewMediaInput struct {
	UploaderID uuid.UUID
	FileURL    string
	FileName   string
	FileType   string
	FileSize   int64
	Context    Context
	ContextID  *uuid.UUID
}

// NewMedia creates a validated Media entity from the given input.
func NewMedia(in NewMediaInput) (*Media, error) {
	if in.UploaderID == uuid.Nil {
		return nil, ErrMissingUploader
	}
	if strings.TrimSpace(in.FileURL) == "" {
		return nil, ErrMissingFileURL
	}
	if strings.TrimSpace(in.FileName) == "" {
		return nil, ErrMissingFileName
	}
	if strings.TrimSpace(in.FileType) == "" {
		return nil, ErrMissingFileType
	}
	if !in.Context.IsValid() {
		return nil, ErrInvalidContext
	}

	now := time.Now()
	return &Media{
		ID:               uuid.New(),
		UploaderID:       in.UploaderID,
		FileURL:          in.FileURL,
		FileName:         in.FileName,
		FileType:         in.FileType,
		FileSize:         in.FileSize,
		Context:          in.Context,
		ContextID:        in.ContextID,
		ModerationStatus: StatusPending,
		ModerationLabels: nil,
		ModerationScore:  0,
		CreatedAt:        now,
		UpdatedAt:        now,
	}, nil
}

// Approve marks the media as approved by an admin.
func (m *Media) Approve(reviewerID uuid.UUID) {
	now := time.Now()
	m.ModerationStatus = StatusApproved
	m.ReviewedAt = &now
	m.ReviewedBy = &reviewerID
	m.UpdatedAt = now
}

// Reject marks the media as rejected by an admin.
func (m *Media) Reject(reviewerID uuid.UUID) {
	now := time.Now()
	m.ModerationStatus = StatusRejected
	m.ReviewedAt = &now
	m.ReviewedBy = &reviewerID
	m.UpdatedAt = now
}

// Flag marks the media as flagged by automated moderation.
func (m *Media) Flag(labels []ModerationLabel, score float64) {
	m.ModerationStatus = StatusFlagged
	m.ModerationLabels = labels
	m.ModerationScore = score
	m.UpdatedAt = time.Now()
}

// AutoApprove marks the media as approved by automated moderation.
func (m *Media) AutoApprove(score float64) {
	now := time.Now()
	m.ModerationStatus = StatusApproved
	m.ModerationScore = score
	m.ReviewedAt = &now
	m.UpdatedAt = now
}

// AutoReject marks the media as rejected by automated moderation.
// No reviewer is set because the rejection is system-driven.
func (m *Media) AutoReject(labels []ModerationLabel, score float64) {
	now := time.Now()
	m.ModerationStatus = StatusRejected
	m.ModerationLabels = labels
	m.ModerationScore = score
	m.ReviewedAt = &now
	m.UpdatedAt = now
}

// SetJobID records the async moderation job identifier.
func (m *Media) SetJobID(jobID string) {
	m.RekognitionJobID = &jobID
	m.UpdatedAt = time.Now()
}
