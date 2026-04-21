package repository

import (
	"context"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/subscription"
)

// SubscriptionRepository persists subscriptions and provides the few
// lookups the app layer needs. Keep the surface minimal — each added
// method becomes a contract every adapter AND every mock must satisfy.
//
// Update is preferred over Save/Upsert: the app layer always knows if
// it just created or just mutated a row. FindOpenByUser excludes
// canceled/unpaid rows — past subscriptions stay around for history but
// are irrelevant when asking "is this user subscribed right now".
type SubscriptionRepository interface {
	// Create inserts a new row. Fails with a duplicate-key error if the
	// partial unique index (user_id where status in open states) is
	// violated — meaning the user already has an open subscription.
	Create(ctx context.Context, s *subscription.Subscription) error

	// FindOpenByUser returns the single row for user_id whose status is
	// one of (incomplete, active, past_due). Returns subscription.ErrNotFound
	// when no such row exists; nil + error for real I/O failures.
	FindOpenByUser(ctx context.Context, userID uuid.UUID) (*subscription.Subscription, error)

	// FindByStripeID is the lookup path used by webhook handlers. The
	// stripe_subscription_id column is globally unique.
	FindByStripeID(ctx context.Context, stripeSubscriptionID string) (*subscription.Subscription, error)

	// Update persists every mutable column of the row. Callers must have
	// obtained the row through one of the Find methods first.
	Update(ctx context.Context, s *subscription.Subscription) error
}
