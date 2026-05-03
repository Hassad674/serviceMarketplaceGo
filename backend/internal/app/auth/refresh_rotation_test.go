package auth

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/domain/audit"
	"marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/port/service"
)

// newRotationService builds an auth service wired with the
// refresh-blacklist + audit dependencies needed to exercise SEC-06.
// Returns the service, the mock blacklist, the mock audit repo, and
// the existing user fixture used by every test in this file. Each
// helper test builds its own token mock to control the claims the
// rotation flow sees.
func newRotationService(t *testing.T) (
	*Service,
	*mockRefreshBlacklist,
	*mockAuditRepo,
	*user.User,
	*mockTokenService,
) {
	t.Helper()
	existing := &user.User{
		ID:    uuid.New(),
		Email: "rotate@example.com",
		Role:  user.RoleProvider,
	}
	users := &mockUserRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*user.User, error) {
			if id == existing.ID {
				return existing, nil
			}
			return nil, user.ErrUserNotFound
		},
	}

	blacklist := newMockRefreshBlacklist()
	audits := newMockAuditRepo()
	tokens := &mockTokenService{}

	svc := NewServiceWithDeps(ServiceDeps{
		Users:            users,
		Resets:           &mockPasswordResetRepo{},
		Hasher:           &mockHasher{},
		Tokens:           tokens,
		Email:            &mockEmailService{},
		RefreshBlacklist: blacklist,
		Audits:           audits,
		FrontendURL:      "https://example.com",
	})
	return svc, blacklist, audits, existing, tokens
}

func TestAuthService_RefreshToken_RotationBlacklistsOldJTI(t *testing.T) {
	// SEC-06: a successful refresh must blacklist the JTI of the token
	// that was just exchanged so a replay returns 401.
	svc, blacklist, audits, u, tokens := newRotationService(t)

	expiry := time.Now().Add(time.Hour)
	tokens.validateRefreshFn = func(_ string) (*service.TokenClaims, error) {
		return &service.TokenClaims{
			UserID:    u.ID,
			JTI:       "jti-original",
			ExpiresAt: expiry,
		}, nil
	}

	out, err := svc.RefreshToken(context.Background(), "old_refresh_token")
	require.NoError(t, err)
	require.NotNil(t, out)

	has, err := blacklist.Has(context.Background(), "jti-original")
	require.NoError(t, err)
	assert.True(t, has, "old JTI must be blacklisted after a successful rotation")
	// SEC-13 emits a token_refresh audit on every successful rotation,
	// so the snapshot is non-empty. Assert that NO token_reuse_detected
	// row was written — that is the SEC-06-specific invariant under test.
	for _, e := range audits.Snapshot() {
		assert.NotEqual(t, audit.ActionTokenReuseDetected, e.Action,
			"successful rotation must NOT emit token_reuse_detected")
	}
}

func TestAuthService_RefreshToken_ReplayedJTIReturnsUnauthorized(t *testing.T) {
	// SEC-06: replaying a refresh token whose JTI is already on the
	// blacklist returns 401 + emits a token_reuse_detected audit row
	// so the SOC has a breadcrumb to investigate possible token theft.
	svc, blacklist, audits, u, tokens := newRotationService(t)

	// Pre-blacklist the JTI to simulate "this token was already
	// rotated by a legitimate prior call".
	require.NoError(t, blacklist.Add(context.Background(), "jti-stolen", time.Hour))

	tokens.validateRefreshFn = func(_ string) (*service.TokenClaims, error) {
		return &service.TokenClaims{
			UserID:    u.ID,
			JTI:       "jti-stolen",
			ExpiresAt: time.Now().Add(time.Hour),
		}, nil
	}

	out, err := svc.RefreshToken(context.Background(), "replayed_token")
	assert.ErrorIs(t, err, user.ErrUnauthorized)
	assert.Nil(t, out)

	// Find the token_reuse_detected row. SEC-13 may also emit other
	// audit events on this code path (none expected today, but be
	// resilient to future additions like e.g. an authentication failure
	// counter). Filter the snapshot to the action we care about.
	var reuse *audit.Entry
	for _, e := range audits.Snapshot() {
		if e.Action == audit.ActionTokenReuseDetected {
			reuse = e
			break
		}
	}
	require.NotNil(t, reuse, "replayed refresh must emit a token_reuse_detected row")
	assert.Equal(t, audit.ResourceTypeUser, reuse.ResourceType)
	require.NotNil(t, reuse.UserID)
	assert.Equal(t, u.ID, *reuse.UserID)
	assert.Equal(t, "jti-stolen", reuse.Metadata["jti"])
}

// TestRefresh_ReplayRevokesEntireFamily covers F.5 S2: per RFC OAuth
// 2.1 §4.13.2, a detected refresh-token replay must revoke the
// ENTIRE token family — not just the replayed token. We achieve that
// by bumping users.session_version, which makes every existing
// access token fail the middleware version check on its next request.
//
// The hard contract under test: after a replay is detected,
// BumpSessionVersion is called for the offending user_id so the
// attacker's parallel access tokens stop working immediately.
func TestAuthService_RefreshToken_ReplayRevokesEntireFamily(t *testing.T) {
	svc, blacklist, _, u, tokens := newRotationService(t)
	users := svc.users.(*mockUserRepo)

	// Pre-blacklist the JTI to simulate a replay attempt.
	require.NoError(t, blacklist.Add(context.Background(), "jti-family-replay", time.Hour))

	tokens.validateRefreshFn = func(_ string) (*service.TokenClaims, error) {
		return &service.TokenClaims{
			UserID:    u.ID,
			JTI:       "jti-family-replay",
			ExpiresAt: time.Now().Add(time.Hour),
		}, nil
	}

	// Sanity: no bumps yet.
	require.Empty(t, users.snapshotBumpCalls())

	out, err := svc.RefreshToken(context.Background(), "replayed_token")
	assert.ErrorIs(t, err, user.ErrUnauthorized)
	assert.Nil(t, out)

	// HARD CONTRACT: BumpSessionVersion was called for the user whose
	// token was replayed. This is what invalidates every other access
	// token already issued for that user — RFC 6749 §10.4 / OAuth 2.1
	// §4.13.2 family revocation.
	bumps := users.snapshotBumpCalls()
	require.Len(t, bumps, 1, "F.5 S2: replay must trigger exactly one BumpSessionVersion call")
	assert.Equal(t, u.ID, bumps[0],
		"F.5 S2: bump must target the user whose token family was compromised")
}

func TestAuthService_RefreshToken_BlacklistReadFailureFailsOpen(t *testing.T) {
	// SEC-06: a Redis blip on the blacklist read must NOT lock every
	// user out — we trust the SessionVersion check on the next
	// mutation to catch a real compromise. The handler should still
	// rotate the pair successfully.
	svc, blacklist, audits, u, tokens := newRotationService(t)
	blacklist.hasErr = assertedRedisError{}

	tokens.validateRefreshFn = func(_ string) (*service.TokenClaims, error) {
		return &service.TokenClaims{
			UserID:    u.ID,
			JTI:       "jti-redis-blip",
			ExpiresAt: time.Now().Add(time.Hour),
		}, nil
	}

	out, err := svc.RefreshToken(context.Background(), "valid_token")
	require.NoError(t, err, "blacklist read failure must NOT fail the request")
	require.NotNil(t, out)
	// The blacklist write also fails (same hasErr drives addErr too if
	// we set it — here addErr is unset so the write succeeds and the
	// rotation completes). The audit log must NOT contain a
	// token_reuse_detected entry — the legitimate token_refresh audit
	// emission added by SEC-13 is expected and ignored here.
	for _, e := range audits.Snapshot() {
		assert.NotEqual(t, audit.ActionTokenReuseDetected, e.Action,
			"blacklist read failure must not be misinterpreted as reuse")
	}
}

func TestAuthService_RefreshToken_NoBlacklistWiredKeepsLegacyBehavior(t *testing.T) {
	// Backwards compat: when the blacklist is not wired (older tests,
	// minimal deployments) the rotation flow must work exactly as
	// before — issue a fresh pair and do nothing else.
	u := &user.User{ID: uuid.New(), Email: "x@example.com", Role: user.RoleAgency}
	users := &mockUserRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*user.User, error) {
			if id == u.ID {
				return u, nil
			}
			return nil, user.ErrUserNotFound
		},
	}
	tokens := &mockTokenService{
		validateRefreshFn: func(_ string) (*service.TokenClaims, error) {
			return &service.TokenClaims{
				UserID:    u.ID,
				JTI:       "any-jti",
				ExpiresAt: time.Now().Add(time.Hour),
			}, nil
		},
	}

	svc := NewService(users, &mockPasswordResetRepo{}, &mockHasher{}, tokens, &mockEmailService{}, "https://example.com")

	out, err := svc.RefreshToken(context.Background(), "any_token")
	require.NoError(t, err)
	require.NotNil(t, out)
}

func TestAuthService_RefreshToken_ExpiredTTLNotBlacklisted(t *testing.T) {
	// Edge case: the refresh token's claimed ExpiresAt has already
	// passed. The blacklist Add becomes a no-op (negative TTL is
	// rejected by the adapter) — the entry would be useless anyway.
	svc, blacklist, _, u, tokens := newRotationService(t)

	tokens.validateRefreshFn = func(_ string) (*service.TokenClaims, error) {
		return &service.TokenClaims{
			UserID:    u.ID,
			JTI:       "jti-already-expired",
			ExpiresAt: time.Now().Add(-1 * time.Hour),
		}, nil
	}

	out, err := svc.RefreshToken(context.Background(), "stale_token")
	require.NoError(t, err)
	require.NotNil(t, out)
	has, _ := blacklist.Has(context.Background(), "jti-already-expired")
	assert.False(t, has, "negative TTL must not produce a blacklist entry")
}

func TestAuthService_RefreshToken_NoJTIIsRotatedWithoutBlacklisting(t *testing.T) {
	// Pre-Phase-1 tokens may not carry a JTI claim. We must still
	// rotate them (the user is owed a working session), but we cannot
	// blacklist them — the blacklist key would be empty.
	svc, blacklist, _, u, tokens := newRotationService(t)

	tokens.validateRefreshFn = func(_ string) (*service.TokenClaims, error) {
		return &service.TokenClaims{
			UserID:    u.ID,
			JTI:       "",
			ExpiresAt: time.Now().Add(time.Hour),
		}, nil
	}

	out, err := svc.RefreshToken(context.Background(), "legacy_token")
	require.NoError(t, err)
	require.NotNil(t, out)

	// Empty JTI must never write a blacklist entry — that would
	// poison "jti-blank" for every other legacy token.
	assert.Equal(t, 0, blacklist.Count(),
		"empty jti must NOT be blacklisted")
}

func TestAuthService_RevokeRefreshToken_BlacklistsTheJTI(t *testing.T) {
	// Logout flow: posting the refresh token blacklists it
	// immediately so any subsequent /auth/refresh fails 401.
	svc, blacklist, _, u, tokens := newRotationService(t)

	tokens.validateRefreshFn = func(_ string) (*service.TokenClaims, error) {
		return &service.TokenClaims{
			UserID:    u.ID,
			JTI:       "jti-logout",
			ExpiresAt: time.Now().Add(time.Hour),
		}, nil
	}

	svc.RevokeRefreshToken(context.Background(), "valid_refresh_token")

	has, err := blacklist.Has(context.Background(), "jti-logout")
	require.NoError(t, err)
	assert.True(t, has, "logout must blacklist the refresh JTI")
}

func TestAuthService_RevokeRefreshToken_InvalidTokenIsNoop(t *testing.T) {
	// An attacker hitting /logout with garbage must never crash the
	// handler — we just silently no-op.
	svc, blacklist, _, _, tokens := newRotationService(t)
	tokens.validateRefreshFn = func(_ string) (*service.TokenClaims, error) {
		return nil, assertedTokenError{}
	}

	svc.RevokeRefreshToken(context.Background(), "garbage")
	assert.Equal(t, 0, blacklist.Count())
}

func TestAuthService_RevokeRefreshToken_EmptyTokenIsNoop(t *testing.T) {
	// Web mode logout has no refresh token in the body — RevokeRefreshToken
	// must silently no-op.
	svc, blacklist, _, _, _ := newRotationService(t)
	svc.RevokeRefreshToken(context.Background(), "")
	assert.Equal(t, 0, blacklist.Count())
}

func TestAuthService_RevokeRefreshToken_NoBlacklistWiredIsNoop(t *testing.T) {
	// Without the blacklist wired the logout still succeeds — there
	// is nothing to do at the JTI layer, the session was already
	// cleared at the handler.
	tokens := &mockTokenService{}
	svc := NewService(&mockUserRepo{}, &mockPasswordResetRepo{}, &mockHasher{}, tokens, &mockEmailService{}, "")

	// Should not panic regardless of the input.
	svc.RevokeRefreshToken(context.Background(), "anything")
}

// --- assertion helpers ---

// assertedRedisError is a sentinel returned by the mock blacklist when
// we want to simulate a Redis transient failure. Carrying a distinct
// type keeps the assert.ErrorIs comparisons (if any test adds one)
// stable without colliding with errors.New.
type assertedRedisError struct{}

func (assertedRedisError) Error() string { return "simulated redis failure" }

// assertedTokenError is the token-validation counterpart, used to
// drive the RevokeRefreshToken invalid-token path.
type assertedTokenError struct{}

func (assertedTokenError) Error() string { return "simulated token failure" }
