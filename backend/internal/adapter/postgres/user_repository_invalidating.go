package postgres

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/port/service"
)

// InvalidatingUserRepository wraps a repository.UserRepository and
// invokes a service.SessionVersionInvalidator after every successful
// BumpSessionVersion. Every other method is forwarded verbatim to
// the inner repository via embedding, so the wrapper stays minimal
// even as the UserRepository contract grows.
//
// QW-HARDENING: this closes the first leak left by QW1/QW2. The
// CachedSessionVersionChecker already exposed Invalidate, but the 6+
// production call sites that bump users.session_version
// (admin_overrides, transfer_service, role_overrides_service,
// membership_service, admin/service, auth/service_account,
// auth/service_more) did NOT call it — so a revoked session
// remained valid for up to 30 seconds (the cache TTL).
//
// Wrapping the repository instead of injecting an invalidator into
// each service has three benefits:
//
//   - Zero call-site churn. Every existing
//     `users.BumpSessionVersion(...)` keeps working unchanged.
//   - Single audit point. If a future service forgets to invalidate,
//     the cache is still consistent.
//   - The invalidation is GUARANTEED to follow a successful bump
//     (transactional intent), because the only path that reaches the
//     invalidator is "inner returned (newVersion, nil err)".
//
// Failure semantics — invalidation is best-effort. The bump itself
// is durable (committed in Postgres) and the cache will heal on its
// own within the 30s TTL even if the Redis DEL fails. We log a WARN
// rather than fail the bump, because failing a successful logout-all
// / role-change for a cache eviction would be a worse outcome than a
// brief stale window.
type InvalidatingUserRepository struct {
	repository.UserRepository
	invalidator service.SessionVersionInvalidator
}

// NewInvalidatingUserRepository wires the decorator. Pass the
// postgres-backed UserRepository as `inner` and the cache adapter
// (CachedSessionVersionChecker) as `invalidator`. Both arguments
// MUST be non-nil — constructing the wrapper with a nil invalidator
// would silently regress the fix.
func NewInvalidatingUserRepository(
	inner repository.UserRepository,
	invalidator service.SessionVersionInvalidator,
) *InvalidatingUserRepository {
	if inner == nil {
		panic("postgres.NewInvalidatingUserRepository: inner repository is nil")
	}
	if invalidator == nil {
		panic("postgres.NewInvalidatingUserRepository: invalidator is nil")
	}
	return &InvalidatingUserRepository{
		UserRepository: inner,
		invalidator:    invalidator,
	}
}

// BumpSessionVersion delegates to the inner repository and, on
// success, fires the cache invalidator so the next authenticated
// request observes the new session_version immediately instead of
// waiting for the 30s TTL.
func (r *InvalidatingUserRepository) BumpSessionVersion(
	ctx context.Context,
	userID uuid.UUID,
) (int, error) {
	newVersion, err := r.UserRepository.BumpSessionVersion(ctx, userID)
	if err != nil {
		// Inner failed — do NOT invalidate. The bump did not commit,
		// so the cache (whether stale or fresh) is still consistent
		// with the database. Evicting now would force a useless
		// extra round-trip on the next request.
		return newVersion, err
	}
	if iErr := r.invalidator.Invalidate(ctx, userID); iErr != nil {
		// The bump itself succeeded — never propagate the eviction
		// failure. The cache will heal within the TTL. We do not
		// retry because the same Redis pressure that caused the DEL
		// to fail would likely fail again on retry.
		slog.Warn("session version cache invalidation failed after bump",
			"user_id", userID, "new_version", newVersion, "error", iErr)
	}
	return newVersion, nil
}

// Compile-time assertion: the wrapper still satisfies the full
// UserRepository contract via embedding.
var _ repository.UserRepository = (*InvalidatingUserRepository)(nil)
