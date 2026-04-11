package repository

import (
	"context"

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

// UserBatchReader is a focused, additive interface that exposes the
// batch user-fetch capability without bloating the main UserRepository
// contract. Consumers that only need to join a secondary dataset
// against the users table (team member listings, review aggregations,
// dispute participants) depend on THIS interface — never on the larger
// UserRepository — so adding new bulk methods here does not force
// every UserRepository mock in the codebase to implement them.
//
// The concrete postgres adapter satisfies both UserRepository and
// UserBatchReader because it provides the union of their methods.
type UserBatchReader interface {
	// GetByIDs batch-fetches users by their ids in a single query.
	// Returns the slice in no particular order — callers must map by
	// id if they need a specific ordering. Missing ids are silently
	// dropped from the result (not an error) because the primary use
	// case is joining a secondary dataset and partial matches are
	// expected.
	GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*user.User, error)
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

	// Session version (migration 056, wired in Phase 3).
	// Incremented whenever the user's effective permissions change.
	// The auth middleware compares the JWT's session_version against
	// the current value and rejects mismatches with 401 — this is how
	// "immediate revocation" takes effect for role changes, removals,
	// suspensions, and password resets.
	BumpSessionVersion(ctx context.Context, userID uuid.UUID) (int, error)
	GetSessionVersion(ctx context.Context, userID uuid.UUID) (int, error)
}
