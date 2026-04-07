package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// ModerationSource identifies the origin of a moderation item.
type ModerationSource string

const (
	ModerationSourceHumanReport ModerationSource = "human_report"
	ModerationSourceAutoMedia   ModerationSource = "auto_media"
	ModerationSourceAutoText    ModerationSource = "auto_text"
)

// ModerationItem is a unified view over reports, flagged messages, flagged reviews, and flagged media.
type ModerationItem struct {
	ID              uuid.UUID
	Source          ModerationSource
	ContentType     string  // "report", "message", "review", "media"
	ContentID       uuid.UUID
	ContentPreview  string
	Status          string  // "pending", "resolved", "dismissed", "approved", "hidden"
	ModerationScore float64
	Reason          string  // report reason or moderation label(s)
	UserInvolvedID  uuid.UUID
	UserInvolvedName string
	UserInvolvedRole string
	ConversationID  *uuid.UUID
	CreatedAt       time.Time
}

// ModerationFilters groups query parameters for the unified moderation listing.
type ModerationFilters struct {
	Source string // "human_report", "auto_media", "auto_text" or empty for all
	Type   string // "report", "message", "review", "media" or empty for all
	Status string // "pending", "resolved", "dismissed" etc. or empty for all
	Sort   string // "newest" (default), "oldest", "score"
	Page   int
	Limit  int
}

// AdminModerationRepository defines read operations for the unified moderation queue.
type AdminModerationRepository interface {
	List(ctx context.Context, filters ModerationFilters) ([]ModerationItem, error)
	Count(ctx context.Context, filters ModerationFilters) (int, error)
	PendingCount(ctx context.Context) (int, error)
}
