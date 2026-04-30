package repository

// Segregated reader / writer / auth-store / KYC-store interfaces for
// the user feature. Carved out of UserRepository (15 methods).
//
// Four families:
//   - UserReader    — single + admin-list read paths and counts.
//     The existing UserBatchReader (above) is the bulk-fetch sibling
//     and stays as-is — it was already segregated.
//   - UserWriter    — life-cycle mutations (create, update, delete) and
//     existence probes used by registration + admin tooling.
//   - UserAuthStore — session_version and last_active fields written by
//     the auth flow (login/logout/permissions-change) and read by the
//     auth middleware to reject stale tokens.
//   - UserKYCStore  — the email-notification kill-switch is the only
//     KYC-adjacent setting left on the user row; it is named
//     UserKYCStore for symmetry with the org-side store but carries
//     just one method today. Reserved namespace for future KYC fields.
//
// Segregation rationale: the auth path shouldn't pull the admin list
// API; the admin path shouldn't pull session-version mutations; a feature
// that just wants to flip the email opt-in shouldn't pull in either.

import (
	"context"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/user"
)

// UserReader exposes read paths over the users table.
type UserReader interface {
	GetByID(ctx context.Context, id uuid.UUID) (*user.User, error)
	GetByEmail(ctx context.Context, email string) (*user.User, error)
	ExistsByEmail(ctx context.Context, email string) (bool, error)
	ListAdmin(ctx context.Context, filters AdminUserFilters) ([]*user.User, string, error)
	CountAdmin(ctx context.Context, filters AdminUserFilters) (int, error)
	CountByRole(ctx context.Context) (map[string]int, error)
	CountByStatus(ctx context.Context) (map[string]int, error)
	RecentSignups(ctx context.Context, limit int) ([]*user.User, error)
}

// UserWriter exposes life-cycle mutation paths.
type UserWriter interface {
	Create(ctx context.Context, u *user.User) error
	Update(ctx context.Context, u *user.User) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// UserAuthStore covers the auth-bookkeeping fields:
//   - session_version: incremented on permission changes / password
//     resets / role changes; the auth middleware compares it against
//     the JWT to reject stale tokens (immediate revocation).
//   - last_active_at: bumped on login + message-send so the search
//     ranker can boost recently-active profiles.
type UserAuthStore interface {
	BumpSessionVersion(ctx context.Context, userID uuid.UUID) (int, error)
	GetSessionVersion(ctx context.Context, userID uuid.UUID) (int, error)
	TouchLastActive(ctx context.Context, userID uuid.UUID) error
}

// UserKYCStore covers user-row KYC-adjacent fields. Today only the
// email-notifications kill-switch — preserved as a separate port so a
// feature that needs to flip the opt-in (preferences page, admin tool)
// can declare just this surface and leave the rest of the user API out
// of its dependency graph.
type UserKYCStore interface {
	UpdateEmailNotificationsEnabled(ctx context.Context, userID uuid.UUID, enabled bool) error
}

// Compile-time guarantee that the wide UserRepository contract is
// always equivalent to the union of its segregated children.
var _ UserRepository = (interface {
	UserReader
	UserWriter
	UserAuthStore
	UserKYCStore
})(nil)
