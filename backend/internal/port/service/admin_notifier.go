package service

import (
	"context"

	"github.com/google/uuid"
)

// AdminNotifierService manages per-admin notification counters backed by Redis.
// Counters are category-scoped and keyed per admin user so that reading one
// admin's queue does not affect another.
type AdminNotifierService interface {
	// IncrementAll increments a category counter for ALL admin users.
	IncrementAll(ctx context.Context, category string) error
	// GetAll returns counters for a specific admin user.
	GetAll(ctx context.Context, adminID uuid.UUID) (map[string]int64, error)
	// Reset resets a category counter for a specific admin user.
	Reset(ctx context.Context, adminID uuid.UUID, category string) error
}

// Admin notification category constants.
const (
	AdminNotifReports         = "reports"
	AdminNotifMediaRejected   = "media_rejected"
	AdminNotifUsersSuspended  = "users_suspended"
	AdminNotifMessagesHidden  = "messages_hidden"
	AdminNotifMediaFlagged    = "media_flagged"
	AdminNotifMessagesFlagged = "messages_flagged"
	AdminNotifReviewsFlagged  = "reviews_flagged"
)

// AdminNotifCategories lists all notification categories for iteration.
var AdminNotifCategories = []string{
	AdminNotifReports,
	AdminNotifMediaRejected,
	AdminNotifUsersSuspended,
	AdminNotifMessagesHidden,
	AdminNotifMediaFlagged,
	AdminNotifMessagesFlagged,
	AdminNotifReviewsFlagged,
}
