package middleware

import (
	"context"
	"errors"
	"log/slog"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/user"
)

// UserState captures the per-request "live" view of a user account.
// The auth middleware queries it on every authenticated request so a
// promotion / demotion / suspension propagates without forcing the
// user to log out and back in.
//
// Why a struct instead of returning the full user.User: the wire
// surface stays minimal (3 booleans + a status enum), which keeps
// hand-rolled mocks tractable and avoids exposing repository internals
// to the middleware package. New "auth-relevant" fields are added
// here when needed; everything else stays in the domain entity.
type UserState struct {
	IsAdmin bool
	Status  user.UserStatus
}

// IsActive reports whether the user is allowed to act on the API. A
// `banned` user is rejected outright. `suspended` is currently treated
// as still active for read paths (existing behaviour); future work
// can branch on `Status` directly when read/write asymmetry is needed.
func (s UserState) IsActive() bool {
	return s.Status != user.StatusBanned
}

// UserStateChecker is the minimal interface the auth middleware uses
// to read the live admin / status flags on every authenticated
// request. Same-layer collaboration with the user repository — not
// declared in port/ for the same reason as SessionVersionChecker.
//
// The implementation in production is Redis-cached (30s TTL) so the
// hot path absorbs login bursts without hitting Postgres for each
// authenticated call. A non-cached adapter (e.g. tests or
// development) is also a valid implementation.
//
// Error semantics:
//   - ErrUserNotFound: the user row was deleted between login and the
//     current request. Middleware treats this as a hard session
//     invalidation (401, the same way we handle a deleted user during
//     session_version checks).
//   - Any other error: transient (DB outage, Redis blip). The caller
//     decides how to handle it (fail-closed in production via 503,
//     fall-through to the snapshot in dev so a contributor's broken
//     local DB doesn't lock everyone out).
type UserStateChecker interface {
	GetUserState(ctx context.Context, userID uuid.UUID) (UserState, error)
}

// userStateOutcome encodes the auth middleware's decision on an
// is_admin / status lookup. The shape mirrors sessionVersionOutcome so
// the two checks can share a single fail-open / fail-closed branch in
// the caller.
type userStateOutcome int

const (
	// userStateOK — checker succeeded; the returned UserState is
	// authoritative and overrides the snapshot.
	userStateOK userStateOutcome = iota
	// userStateUserGone — the user row has been deleted. The
	// session/JWT is bound to a zombie account; force the client to
	// re-authenticate.
	userStateUserGone
	// userStateBanned — user.Status == banned. The middleware
	// short-circuits with 403 regardless of the requested resource.
	userStateBanned
	// userStateLookupFailed — transient DB/Redis failure; the
	// caller maps this to 503 in production and to "trust snapshot"
	// in dev/test, mirroring the session-version policy.
	userStateLookupFailed
)

// resolveUserState consults the live checker and returns the
// outcome the caller should branch on. A nil checker is interpreted
// as "no live override available" — the snapshot is trusted as-is
// (used by tests and legacy deployments).
//
// On userStateOK the returned UserState is non-zero and ready to be
// stamped onto the request context. On every other outcome, the
// returned struct is the zero value and must not be used.
func resolveUserState(
	ctx context.Context,
	checker UserStateChecker,
	userID uuid.UUID,
	snapshot UserState,
) (UserState, userStateOutcome) {
	if checker == nil {
		// No checker wired (test path) — trust the snapshot. We still
		// short-circuit a banned snapshot so unit tests can exercise
		// the ban path without a checker.
		if !snapshot.IsActive() {
			return UserState{}, userStateBanned
		}
		return snapshot, userStateOK
	}

	live, err := checker.GetUserState(ctx, userID)
	if err != nil {
		if errors.Is(err, user.ErrUserNotFound) {
			return UserState{}, userStateUserGone
		}
		slog.Error("auth: user state lookup failed",
			"user_id", userID, "error", err)
		return UserState{}, userStateLookupFailed
	}
	if !live.IsActive() {
		return UserState{}, userStateBanned
	}
	return live, userStateOK
}
