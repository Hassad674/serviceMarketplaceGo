package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/port/service"
)

// mockUserStateChecker records the (is_admin, status) responses the
// production checker would surface, isolated from any DB / Redis.
type mockUserStateChecker struct {
	getFn func(ctx context.Context, userID uuid.UUID) (UserState, error)
}

func (m *mockUserStateChecker) GetUserState(ctx context.Context, userID uuid.UUID) (UserState, error) {
	return m.getFn(ctx, userID)
}

// --- TestAuth_LiveIsAdminOverridesSnapshot ---
//
// THE bug pin. The session/JWT carry IsAdmin=false (snapshot at login)
// but the live state checker says is_admin=true (operator promoted
// the user via SQL). The middleware MUST stamp ContextKeyIsAdmin=true
// on the request context — otherwise the RequireAdmin middleware
// downstream returns 403 even though the DB row says the user is an
// admin.

func TestAuth_LiveIsAdminOverridesSnapshot_CookiePath(t *testing.T) {
	userID := uuid.New()
	sessionSvc := &mockSessionService{
		getFn: func(_ context.Context, _ string) (*service.Session, error) {
			return &service.Session{
				ID:             "sess-1",
				UserID:         userID,
				Role:           "agency",
				IsAdmin:        false, // SNAPSHOT — pre-promotion
				SessionVersion: 1,
			}, nil
		},
	}
	versions := &mockSessionVersionChecker{
		getFn: func(_ context.Context, _ uuid.UUID) (int, error) { return 1, nil },
	}
	state := &mockUserStateChecker{
		getFn: func(_ context.Context, _ uuid.UUID) (UserState, error) {
			return UserState{IsAdmin: true, Status: user.StatusActive}, nil // LIVE — post-promotion
		},
	}

	var stampedIsAdmin bool
	next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		stampedIsAdmin = GetIsAdmin(r.Context())
	})

	handler := AuthFromDeps(AuthDeps{
		TokenService:    &mockTokenService{validateAccessFn: func(string) (*service.TokenClaims, error) { return nil, errors.New("unused") }},
		SessionService:  sessionSvc,
		SessionVersions: versions,
		UserState:       state,
	})(next)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "abc"})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, "request must succeed")
	assert.True(t, stampedIsAdmin,
		"live is_admin=true MUST override session.IsAdmin=false snapshot")
}

func TestAuth_LiveIsAdminOverridesSnapshot_BearerPath(t *testing.T) {
	userID := uuid.New()
	tokenSvc := &mockTokenService{
		validateAccessFn: func(_ string) (*service.TokenClaims, error) {
			return &service.TokenClaims{
				UserID:         userID,
				Role:           "agency",
				IsAdmin:        false, // SNAPSHOT — pre-promotion
				SessionVersion: 1,
			}, nil
		},
	}
	sessionSvc := &mockSessionService{
		getFn: func(_ context.Context, _ string) (*service.Session, error) {
			return nil, errors.New("no session")
		},
	}
	versions := &mockSessionVersionChecker{
		getFn: func(_ context.Context, _ uuid.UUID) (int, error) { return 1, nil },
	}
	state := &mockUserStateChecker{
		getFn: func(_ context.Context, _ uuid.UUID) (UserState, error) {
			return UserState{IsAdmin: true, Status: user.StatusActive}, nil // LIVE
		},
	}

	var stampedIsAdmin bool
	next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		stampedIsAdmin = GetIsAdmin(r.Context())
	})

	handler := AuthFromDeps(AuthDeps{
		TokenService:    tokenSvc,
		SessionService:  sessionSvc,
		SessionVersions: versions,
		UserState:       state,
	})(next)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer t1")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.True(t, stampedIsAdmin,
		"live is_admin=true MUST override claims.IsAdmin=false snapshot")
}

// TestAuth_LiveDemotionOverridesSnapshot — symmetric path. The
// session/JWT say IsAdmin=true but the live state says is_admin=false.
// A demoted admin must lose access immediately, without waiting for
// the next refresh.
func TestAuth_LiveDemotionOverridesSnapshot(t *testing.T) {
	userID := uuid.New()
	sessionSvc := &mockSessionService{
		getFn: func(_ context.Context, _ string) (*service.Session, error) {
			return &service.Session{
				ID:             "sess-1",
				UserID:         userID,
				Role:           "agency",
				IsAdmin:        true, // SNAPSHOT — was admin
				SessionVersion: 1,
			}, nil
		},
	}
	versions := &mockSessionVersionChecker{
		getFn: func(_ context.Context, _ uuid.UUID) (int, error) { return 1, nil },
	}
	state := &mockUserStateChecker{
		getFn: func(_ context.Context, _ uuid.UUID) (UserState, error) {
			return UserState{IsAdmin: false, Status: user.StatusActive}, nil
		},
	}

	var stampedIsAdmin bool
	next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		stampedIsAdmin = GetIsAdmin(r.Context())
	})

	handler := AuthFromDeps(AuthDeps{
		TokenService:    &mockTokenService{validateAccessFn: func(string) (*service.TokenClaims, error) { return nil, errors.New("unused") }},
		SessionService:  sessionSvc,
		SessionVersions: versions,
		UserState:       state,
	})(next)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "abc"})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.False(t, stampedIsAdmin,
		"live is_admin=false MUST override session.IsAdmin=true snapshot — demotion is immediate")
}

// TestAuth_LiveBannedShortCircuits — a banned user must be 403'd by
// the auth middleware regardless of the snapshot. Tested on both
// transport paths.
func TestAuth_LiveBannedShortCircuits_Cookie(t *testing.T) {
	userID := uuid.New()
	sessionSvc := &mockSessionService{
		getFn: func(_ context.Context, _ string) (*service.Session, error) {
			return &service.Session{ID: "s", UserID: userID, Role: "agency", SessionVersion: 1}, nil
		},
	}
	versions := &mockSessionVersionChecker{getFn: func(_ context.Context, _ uuid.UUID) (int, error) { return 1, nil }}
	state := &mockUserStateChecker{
		getFn: func(_ context.Context, _ uuid.UUID) (UserState, error) {
			return UserState{IsAdmin: false, Status: user.StatusBanned}, nil
		},
	}

	nextCalled := false
	next := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) { nextCalled = true })

	handler := AuthFromDeps(AuthDeps{
		TokenService:    &mockTokenService{validateAccessFn: func(string) (*service.TokenClaims, error) { return nil, errors.New("unused") }},
		SessionService:  sessionSvc,
		SessionVersions: versions,
		UserState:       state,
	})(next)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "abc"})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code,
		"banned user must be 403'd before the protected handler runs")
	assert.Contains(t, rec.Body.String(), "account_banned")
	assert.False(t, nextCalled, "banned user must NOT reach the protected handler")
}

// TestAuth_LiveUserGoneRejects — user row was deleted between login
// and the request. Must be 401, same as the session_version path.
func TestAuth_LiveUserGoneRejects(t *testing.T) {
	userID := uuid.New()
	sessionSvc := &mockSessionService{
		getFn: func(_ context.Context, _ string) (*service.Session, error) {
			return &service.Session{ID: "s", UserID: userID, Role: "agency", SessionVersion: 1}, nil
		},
	}
	versions := &mockSessionVersionChecker{getFn: func(_ context.Context, _ uuid.UUID) (int, error) { return 1, nil }}
	state := &mockUserStateChecker{
		getFn: func(_ context.Context, _ uuid.UUID) (UserState, error) {
			return UserState{}, user.ErrUserNotFound
		},
	}

	next := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {})
	handler := AuthFromDeps(AuthDeps{
		TokenService:    &mockTokenService{validateAccessFn: func(string) (*service.TokenClaims, error) { return nil, errors.New("unused") }},
		SessionService:  sessionSvc,
		SessionVersions: versions,
		UserState:       state,
	})(next)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "abc"})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "session_invalid")
}

// TestAuth_UserStateLookupFailsClosedInProd — DB/Redis blip. The
// production wiring (FailClosedInProd=true) MUST return 503 instead
// of trusting the snapshot, so an attacker cannot bypass the live
// admin check by triggering the upstream incident.
func TestAuth_UserStateLookupFailsClosedInProd(t *testing.T) {
	userID := uuid.New()
	sessionSvc := &mockSessionService{
		getFn: func(_ context.Context, _ string) (*service.Session, error) {
			return &service.Session{ID: "s", UserID: userID, Role: "agency", SessionVersion: 1}, nil
		},
	}
	versions := &mockSessionVersionChecker{getFn: func(_ context.Context, _ uuid.UUID) (int, error) { return 1, nil }}
	state := &mockUserStateChecker{
		getFn: func(_ context.Context, _ uuid.UUID) (UserState, error) {
			return UserState{}, errors.New("postgres: connection refused")
		},
	}

	nextCalled := false
	next := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) { nextCalled = true })

	handler := AuthFromDeps(AuthDeps{
		TokenService:     &mockTokenService{validateAccessFn: func(string) (*service.TokenClaims, error) { return nil, errors.New("unused") }},
		SessionService:   sessionSvc,
		SessionVersions:  versions,
		UserState:        state,
		FailClosedInProd: true,
	})(next)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "abc"})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
	assert.Contains(t, rec.Body.String(), "auth_unavailable")
	assert.False(t, nextCalled, "fail-closed must short-circuit before the handler")
}

// TestAuth_UserStateLookupFailsOpenInDev — same blip, dev wiring.
// Trust the snapshot so a contributor's broken local DB does not
// lock them out.
func TestAuth_UserStateLookupFailsOpenInDev(t *testing.T) {
	userID := uuid.New()
	sessionSvc := &mockSessionService{
		getFn: func(_ context.Context, _ string) (*service.Session, error) {
			return &service.Session{ID: "s", UserID: userID, Role: "agency", IsAdmin: true, SessionVersion: 1}, nil
		},
	}
	versions := &mockSessionVersionChecker{getFn: func(_ context.Context, _ uuid.UUID) (int, error) { return 1, nil }}
	state := &mockUserStateChecker{
		getFn: func(_ context.Context, _ uuid.UUID) (UserState, error) {
			return UserState{}, errors.New("postgres: connection refused")
		},
	}

	var stampedIsAdmin bool
	nextCalled := false
	next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		nextCalled = true
		stampedIsAdmin = GetIsAdmin(r.Context())
	})

	handler := AuthFromDeps(AuthDeps{
		TokenService:     &mockTokenService{validateAccessFn: func(string) (*service.TokenClaims, error) { return nil, errors.New("unused") }},
		SessionService:   sessionSvc,
		SessionVersions:  versions,
		UserState:        state,
		FailClosedInProd: false,
	})(next)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "abc"})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.True(t, nextCalled, "dev mode must keep snapshot-trust fall-through")
	assert.True(t, stampedIsAdmin, "snapshot is_admin=true must be preserved on lookup failure")
}

// TestAuth_NilUserStateChecker — backwards compatibility. Existing
// test wiring that doesn't pass a UserStateChecker must keep working
// (snapshot is trusted as-is, but a banned snapshot still
// short-circuits).
func TestAuth_NilUserStateChecker_TrustsSnapshot(t *testing.T) {
	userID := uuid.New()
	sessionSvc := &mockSessionService{
		getFn: func(_ context.Context, _ string) (*service.Session, error) {
			return &service.Session{ID: "s", UserID: userID, Role: "agency", IsAdmin: false, SessionVersion: 1}, nil
		},
	}
	versions := &mockSessionVersionChecker{getFn: func(_ context.Context, _ uuid.UUID) (int, error) { return 1, nil }}

	var stampedIsAdmin bool
	next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		stampedIsAdmin = GetIsAdmin(r.Context())
	})

	handler := AuthFromDeps(AuthDeps{
		TokenService:    &mockTokenService{validateAccessFn: func(string) (*service.TokenClaims, error) { return nil, errors.New("unused") }},
		SessionService:  sessionSvc,
		SessionVersions: versions,
		// UserState: nil — legacy wiring
	})(next)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "abc"})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.False(t, stampedIsAdmin, "snapshot is_admin=false must be preserved when no checker")
}

// --- helpers ---

func TestUserState_IsActive(t *testing.T) {
	tests := []struct {
		name   string
		status user.UserStatus
		want   bool
	}{
		{"active", user.StatusActive, true},
		{"suspended counts as active for now", user.StatusSuspended, true},
		{"banned is inactive", user.StatusBanned, false},
		{"empty defaults to active (zero value)", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := UserState{Status: tt.status}
			assert.Equal(t, tt.want, s.IsActive())
		})
	}
}
