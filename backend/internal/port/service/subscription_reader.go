package service

import (
	"context"

	"github.com/google/uuid"
)

// SubscriptionReader is the TINY interface the billing / payment layers
// consume to know whether a user is currently Premium. It exposes ONE
// method so a feature as simple as fee calculation never has to pull in
// the whole subscription domain and so the subscription feature remains
// fully removable from the app — delete the feature, leave the interface
// unimplemented (nil), and billing falls back to the pleins-tarifs path.
//
// The implementation lives in internal/app/subscription/service.go; the
// adapter is the Redis-backed subscription cache wired in main.go.
type SubscriptionReader interface {
	// IsActive reports whether the user is entitled to the fee waiver
	// right now. Implementations MUST be cheap (sub-millisecond hit on a
	// hot path — every milestone release calls this) and MUST fail-open
	// conservatively: on transient errors return (false, err) so the
	// caller applies the normal fee rather than accidentally granting a
	// free milestone to a cache miss.
	IsActive(ctx context.Context, userID uuid.UUID) (bool, error)
}
