package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"marketplace-backend/internal/domain/user"
)

type AdminUserFilters struct {
	Role     string
	Status   string
	Search   string
	Cursor   string
	Limit    int
	Page     int
	Reported bool
}

type UserRepository interface {
	Create(ctx context.Context, u *user.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*user.User, error)
	GetByEmail(ctx context.Context, email string) (*user.User, error)
	Update(ctx context.Context, u *user.User) error
	Delete(ctx context.Context, id uuid.UUID) error
	ExistsByEmail(ctx context.Context, email string) (bool, error)
	ListAdmin(ctx context.Context, filters AdminUserFilters) ([]*user.User, string, error)
	CountAdmin(ctx context.Context, filters AdminUserFilters) (int, error)
	CountByRole(ctx context.Context) (map[string]int, error)
	CountByStatus(ctx context.Context) (map[string]int, error)
	RecentSignups(ctx context.Context, limit int) ([]*user.User, error)

	// Stripe account operations (migration 040).
	GetStripeAccount(ctx context.Context, userID uuid.UUID) (accountID, country string, err error)
	FindUserIDByStripeAccount(ctx context.Context, accountID string) (uuid.UUID, error)
	SetStripeAccount(ctx context.Context, userID uuid.UUID, accountID, country string) error
	ClearStripeAccount(ctx context.Context, userID uuid.UUID) error
	GetStripeLastState(ctx context.Context, userID uuid.UUID) ([]byte, error)
	SaveStripeLastState(ctx context.Context, userID uuid.UUID, state []byte) error

	// KYC enforcement (migration 044).
	SetKYCFirstEarning(ctx context.Context, userID uuid.UUID, at time.Time) error
	GetKYCPendingUsers(ctx context.Context) ([]*user.User, error)
	SaveKYCNotificationState(ctx context.Context, userID uuid.UUID, state map[string]time.Time) error

	// Session version (migration 056, wired in Phase 3).
	// Incremented whenever the user's effective permissions change.
	// The auth middleware compares the JWT's session_version against
	// the current value and rejects mismatches with 401 — this is how
	// "immediate revocation" takes effect for role changes, removals,
	// suspensions, and password resets.
	BumpSessionVersion(ctx context.Context, userID uuid.UUID) (int, error)
	GetSessionVersion(ctx context.Context, userID uuid.UUID) (int, error)
}
