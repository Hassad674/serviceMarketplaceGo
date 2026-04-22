package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"marketplace-backend/internal/domain/organization"
	"marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/port/service"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- mock org overrides resolver ---

type mockOrgOverridesResolver struct {
	getFn func(ctx context.Context, orgID uuid.UUID) (organization.RoleOverrides, error)
}

func (m *mockOrgOverridesResolver) GetRoleOverrides(ctx context.Context, orgID uuid.UUID) (organization.RoleOverrides, error) {
	return m.getFn(ctx, orgID)
}

// --- mock session version checker ---

type mockSessionVersionChecker struct {
	getFn func(ctx context.Context, userID uuid.UUID) (int, error)
}

func (m *mockSessionVersionChecker) GetSessionVersion(ctx context.Context, userID uuid.UUID) (int, error) {
	return m.getFn(ctx, userID)
}

// --- mock types (local to this test file) ---

type mockTokenService struct {
	validateAccessFn func(token string) (*service.TokenClaims, error)
}

func (m *mockTokenService) GenerateAccessToken(_ service.AccessTokenInput) (string, error) {
	return "", nil
}

func (m *mockTokenService) GenerateRefreshToken(_ uuid.UUID) (string, error) {
	return "", nil
}

func (m *mockTokenService) ValidateAccessToken(token string) (*service.TokenClaims, error) {
	return m.validateAccessFn(token)
}

func (m *mockTokenService) ValidateRefreshToken(_ string) (*service.TokenClaims, error) {
	return nil, nil
}

type mockSessionService struct {
	getFn func(ctx context.Context, sessionID string) (*service.Session, error)
}

func (m *mockSessionService) Create(_ context.Context, _ service.CreateSessionInput) (*service.Session, error) {
	return nil, nil
}

func (m *mockSessionService) Get(ctx context.Context, sessionID string) (*service.Session, error) {
	return m.getFn(ctx, sessionID)
}

func (m *mockSessionService) Delete(_ context.Context, _ string) error { return nil }

func (m *mockSessionService) DeleteByUserID(_ context.Context, _ uuid.UUID) error { return nil }

func (m *mockSessionService) CreateWSToken(_ context.Context, _ uuid.UUID) (string, error) {
	return "", nil
}

func (m *mockSessionService) ValidateWSToken(_ context.Context, _ string) (uuid.UUID, error) {
	return uuid.UUID{}, nil
}

// --- helpers ---

func newAuthRecorder() (*bool, uuid.UUID, string, http.HandlerFunc) {
	called := false
	var gotUID uuid.UUID
	var gotRole string
	handler := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		called = true
		gotUID, _ = GetUserID(r.Context())
		gotRole = GetRole(r.Context())
	})
	return &called, gotUID, gotRole, handler
}

// --- tests ---

func TestAuth_ValidSessionCookie(t *testing.T) {
	userID := uuid.New()
	sessionSvc := &mockSessionService{
		getFn: func(_ context.Context, _ string) (*service.Session, error) {
			return &service.Session{ID: "sess-1", UserID: userID, Role: "agency"}, nil
		},
	}
	tokenSvc := &mockTokenService{
		validateAccessFn: func(_ string) (*service.TokenClaims, error) {
			return nil, errors.New("should not be called")
		},
	}

	var ctxUID uuid.UUID
	var ctxRole string
	nextCalled := false
	next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		nextCalled = true
		ctxUID, _ = GetUserID(r.Context())
		ctxRole = GetRole(r.Context())
	})

	handler := Auth(tokenSvc, sessionSvc, nil, nil)(next)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/profile", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "sess-1"})
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	require.True(t, nextCalled, "next handler must be called")
	assert.Equal(t, userID, ctxUID)
	assert.Equal(t, "agency", ctxRole)
}

func TestAuth_ExpiredSessionFallsBackToBearer(t *testing.T) {
	userID := uuid.New()
	sessionSvc := &mockSessionService{
		getFn: func(_ context.Context, _ string) (*service.Session, error) {
			return nil, errors.New("session expired")
		},
	}
	tokenSvc := &mockTokenService{
		validateAccessFn: func(token string) (*service.TokenClaims, error) {
			if token == "valid-access-token" {
				return &service.TokenClaims{
					UserID:    userID,
					Role:      "enterprise",
					ExpiresAt: time.Now().Add(15 * time.Minute),
				}, nil
			}
			return nil, errors.New("invalid token")
		},
	}

	var ctxUID uuid.UUID
	var ctxRole string
	nextCalled := false
	next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		nextCalled = true
		ctxUID, _ = GetUserID(r.Context())
		ctxRole = GetRole(r.Context())
	})

	handler := Auth(tokenSvc, sessionSvc, nil, nil)(next)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/profile", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "dead-sess"})
	req.Header.Set("Authorization", "Bearer valid-access-token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	require.True(t, nextCalled, "next handler must be called via bearer fallback")
	assert.Equal(t, userID, ctxUID)
	assert.Equal(t, "enterprise", ctxRole)
}

func TestAuth_ValidBearerToken(t *testing.T) {
	userID := uuid.New()
	tokenSvc := &mockTokenService{
		validateAccessFn: func(token string) (*service.TokenClaims, error) {
			if token == "good-token" {
				return &service.TokenClaims{
					UserID: userID,
					Role:   "provider",
				}, nil
			}
			return nil, errors.New("bad")
		},
	}
	sessionSvc := &mockSessionService{
		getFn: func(_ context.Context, _ string) (*service.Session, error) {
			return nil, errors.New("no session")
		},
	}

	var ctxRole string
	nextCalled := false
	next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		nextCalled = true
		ctxRole = GetRole(r.Context())
	})

	handler := Auth(tokenSvc, sessionSvc, nil, nil)(next)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer good-token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	require.True(t, nextCalled)
	assert.Equal(t, "provider", ctxRole)
}

func TestAuth_InvalidBearerToken(t *testing.T) {
	tokenSvc := &mockTokenService{
		validateAccessFn: func(_ string) (*service.TokenClaims, error) {
			return nil, errors.New("invalid signature")
		},
	}
	sessionSvc := &mockSessionService{
		getFn: func(_ context.Context, _ string) (*service.Session, error) {
			return nil, errors.New("no session")
		},
	}

	nextCalled := false
	next := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		nextCalled = true
	})

	handler := Auth(tokenSvc, sessionSvc, nil, nil)(next)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer bad-token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.False(t, nextCalled, "next must NOT be called with invalid token")
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAuth_NoAuthAtAll(t *testing.T) {
	tokenSvc := &mockTokenService{
		validateAccessFn: func(_ string) (*service.TokenClaims, error) {
			return nil, errors.New("no token")
		},
	}
	sessionSvc := &mockSessionService{
		getFn: func(_ context.Context, _ string) (*service.Session, error) {
			return nil, errors.New("no session")
		},
	}

	nextCalled := false
	next := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		nextCalled = true
	})

	handler := Auth(tokenSvc, sessionSvc, nil, nil)(next)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.False(t, nextCalled)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAuth_SessionTakesPriorityOverBearer(t *testing.T) {
	sessionUID := uuid.New()
	bearerUID := uuid.New()

	sessionSvc := &mockSessionService{
		getFn: func(_ context.Context, _ string) (*service.Session, error) {
			return &service.Session{
				ID: "sess-priority", UserID: sessionUID, Role: "agency",
			}, nil
		},
	}
	tokenSvc := &mockTokenService{
		validateAccessFn: func(_ string) (*service.TokenClaims, error) {
			return &service.TokenClaims{
				UserID: bearerUID, Role: "enterprise",
			}, nil
		},
	}

	var ctxUID uuid.UUID
	var ctxRole string
	next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		ctxUID, _ = GetUserID(r.Context())
		ctxRole = GetRole(r.Context())
	})

	handler := Auth(tokenSvc, sessionSvc, nil, nil)(next)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "sess-priority"})
	req.Header.Set("Authorization", "Bearer some-token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, sessionUID, ctxUID, "session user must take priority")
	assert.Equal(t, "agency", ctxRole, "session role must take priority")
}

// R16: when the session version checker reports that the backing user
// row no longer exists (e.g. an operator who left their org was hard-
// deleted), the middleware must reject the request with 401
// session_invalid so the client knows to clear its state and log out.
// This prevents the "zombie logged-in-but-deleted" state where a
// frontend keeps polling /auth/me and getting 404.
func TestAuth_BearerToken_UserDeletedReturns401SessionInvalid(t *testing.T) {
	userID := uuid.New()
	tokenSvc := &mockTokenService{
		validateAccessFn: func(_ string) (*service.TokenClaims, error) {
			return &service.TokenClaims{
				UserID:         userID,
				Role:           "provider",
				SessionVersion: 1,
				ExpiresAt:      time.Now().Add(15 * time.Minute),
			}, nil
		},
	}
	sessionSvc := &mockSessionService{
		getFn: func(_ context.Context, _ string) (*service.Session, error) {
			return nil, errors.New("no session")
		},
	}
	checker := &mockSessionVersionChecker{
		getFn: func(_ context.Context, _ uuid.UUID) (int, error) {
			return 0, user.ErrUserNotFound
		},
	}

	nextCalled := false
	next := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		nextCalled = true
	})

	handler := Auth(tokenSvc, sessionSvc, checker, nil)(next)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.False(t, nextCalled, "next must NOT be called when user is deleted")
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	var body map[string]string
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "session_invalid", body["error"],
		"deleted user must produce 'session_invalid' (not 'session_revoked') so the client logs out")
}

// Mirror of the bearer test above for the session cookie path (web clients).
func TestAuth_SessionCookie_UserDeletedReturns401SessionInvalid(t *testing.T) {
	userID := uuid.New()
	sessionSvc := &mockSessionService{
		getFn: func(_ context.Context, _ string) (*service.Session, error) {
			return &service.Session{
				ID:             "sess-zombie",
				UserID:         userID,
				Role:           "provider",
				SessionVersion: 1,
			}, nil
		},
	}
	tokenSvc := &mockTokenService{
		validateAccessFn: func(_ string) (*service.TokenClaims, error) {
			return nil, errors.New("should not be called")
		},
	}
	checker := &mockSessionVersionChecker{
		getFn: func(_ context.Context, _ uuid.UUID) (int, error) {
			return 0, user.ErrUserNotFound
		},
	}

	nextCalled := false
	next := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		nextCalled = true
	})

	handler := Auth(tokenSvc, sessionSvc, checker, nil)(next)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "sess-zombie"})
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.False(t, nextCalled, "next must NOT be called when user is deleted")
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	var body map[string]string
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "session_invalid", body["error"])
}

// Transient errors from the session version checker must still fall
// through (fail-open), otherwise a DB blip would lock out every user.
// Only user.ErrUserNotFound is treated as a hard revocation signal.
func TestAuth_BearerToken_TransientCheckerErrorFailsOpen(t *testing.T) {
	userID := uuid.New()
	tokenSvc := &mockTokenService{
		validateAccessFn: func(_ string) (*service.TokenClaims, error) {
			return &service.TokenClaims{
				UserID:         userID,
				Role:           "provider",
				SessionVersion: 1,
			}, nil
		},
	}
	sessionSvc := &mockSessionService{
		getFn: func(_ context.Context, _ string) (*service.Session, error) {
			return nil, errors.New("no session")
		},
	}
	checker := &mockSessionVersionChecker{
		getFn: func(_ context.Context, _ uuid.UUID) (int, error) {
			return 0, errors.New("redis connection refused")
		},
	}

	nextCalled := false
	next := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		nextCalled = true
	})

	handler := Auth(tokenSvc, sessionSvc, checker, nil)(next)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.True(t, nextCalled, "transient checker error must fail open")
}

// An explicit session_version bump (carried != current) must still
// produce the classic 401 session_revoked response.
func TestAuth_BearerToken_VersionMismatchReturnsSessionRevoked(t *testing.T) {
	userID := uuid.New()
	tokenSvc := &mockTokenService{
		validateAccessFn: func(_ string) (*service.TokenClaims, error) {
			return &service.TokenClaims{
				UserID:         userID,
				Role:           "provider",
				SessionVersion: 1,
			}, nil
		},
	}
	sessionSvc := &mockSessionService{
		getFn: func(_ context.Context, _ string) (*service.Session, error) {
			return nil, errors.New("no session")
		},
	}
	checker := &mockSessionVersionChecker{
		getFn: func(_ context.Context, _ uuid.UUID) (int, error) {
			return 2, nil // DB version 2, token carries 1 → revoked
		},
	}

	nextCalled := false
	next := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		nextCalled = true
	})

	handler := Auth(tokenSvc, sessionSvc, checker, nil)(next)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.False(t, nextCalled)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	var body map[string]string
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "session_revoked", body["error"])
}

// --- live perms resolution tests (R17-fix) ---

// TestAuth_LivePermsOverrideStaleSessionSnapshot asserts that the
// middleware ignores a stale `perms` snapshot baked into the session
// and instead injects the freshly-computed set from
// EffectivePermissionsFor(role, overrides). This is the regression
// guard for the "org_client_profile.edit missing from pre-R18 sessions"
// bug: the catalog grew but long-lived sessions never saw the new key.
func TestAuth_LivePermsOverrideStaleSessionSnapshot(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	tokenSvc := &mockTokenService{validateAccessFn: func(_ string) (*service.TokenClaims, error) { return nil, errors.New("not used") }}
	sessionSvc := &mockSessionService{getFn: func(_ context.Context, _ string) (*service.Session, error) {
		return &service.Session{
			UserID:         userID,
			Role:           "agency",
			OrganizationID: &orgID,
			OrgRole:        "owner",
			// Snapshot that is missing the new permission the test
			// will ask for. Mimics a session written before the
			// catalog grew.
			Permissions: []string{"jobs.view"},
		}, nil
	}}
	resolver := &mockOrgOverridesResolver{getFn: func(_ context.Context, _ uuid.UUID) (organization.RoleOverrides, error) {
		return nil, nil // no overrides — defaults apply
	}}

	var seenPerms []string
	next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		perms, _ := GetPermissions(r.Context())
		seenPerms = perms
	})

	handler := Auth(tokenSvc, sessionSvc, nil, resolver)(next)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "abc"})
	handler.ServeHTTP(httptest.NewRecorder(), req)

	// Owner defaults include PermOrgClientProfileEdit even though the
	// stale session snapshot did not — live resolution must supersede.
	assert.Contains(t, seenPerms, string(organization.PermOrgClientProfileEdit))
	// And the old snapshot-only entry must not leak through.
	assert.Greater(t, len(seenPerms), 1, "live resolution should produce the full owner set, not just the stale snapshot")
}

// TestAuth_LivePermsFallbackToSnapshotOnResolverError covers the
// fail-open branch: when the overrides resolver errors (e.g. Postgres
// blip), we keep the session's snapshot rather than locking the user
// out. Operators get an alert from the slog.Warn, but traffic stays up.
func TestAuth_LivePermsFallbackToSnapshotOnResolverError(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	snapshot := []string{"jobs.view", "proposals.view"}
	tokenSvc := &mockTokenService{validateAccessFn: func(_ string) (*service.TokenClaims, error) { return nil, errors.New("not used") }}
	sessionSvc := &mockSessionService{getFn: func(_ context.Context, _ string) (*service.Session, error) {
		return &service.Session{
			UserID:         userID,
			Role:           "agency",
			OrganizationID: &orgID,
			OrgRole:        "owner",
			Permissions:    snapshot,
		}, nil
	}}
	resolver := &mockOrgOverridesResolver{getFn: func(_ context.Context, _ uuid.UUID) (organization.RoleOverrides, error) {
		return nil, errors.New("db blip")
	}}

	var seenPerms []string
	next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		perms, _ := GetPermissions(r.Context())
		seenPerms = perms
	})

	handler := Auth(tokenSvc, sessionSvc, nil, resolver)(next)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "abc"})
	handler.ServeHTTP(httptest.NewRecorder(), req)

	assert.Equal(t, snapshot, seenPerms, "resolver error should fall open to the session snapshot")
}

// TestAuth_NoResolverKeepsSessionSnapshot ensures legacy deployments
// and tests that wire nil as the resolver keep the old behaviour
// (trust the session snapshot) without any silent denial.
func TestAuth_NoResolverKeepsSessionSnapshot(t *testing.T) {
	orgID := uuid.New()
	snapshot := []string{"jobs.view"}
	tokenSvc := &mockTokenService{validateAccessFn: func(_ string) (*service.TokenClaims, error) { return nil, errors.New("not used") }}
	sessionSvc := &mockSessionService{getFn: func(_ context.Context, _ string) (*service.Session, error) {
		return &service.Session{
			UserID:         uuid.New(),
			Role:           "agency",
			OrganizationID: &orgID,
			OrgRole:        "owner",
			Permissions:    snapshot,
		}, nil
	}}

	var seenPerms []string
	next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		perms, _ := GetPermissions(r.Context())
		seenPerms = perms
	})

	handler := Auth(tokenSvc, sessionSvc, nil, nil)(next)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "abc"})
	handler.ServeHTTP(httptest.NewRecorder(), req)

	assert.Equal(t, snapshot, seenPerms)
}
