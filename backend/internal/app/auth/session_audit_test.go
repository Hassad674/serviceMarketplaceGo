package auth

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/domain/session"
	"marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/port/service"
)

// mockUserSessionRepo is an in-memory UserSessionRepository used by
// the B.4 tests. Tracks Create/Revoke/RevokeAllForUser invocations
// so the assertions can verify "the row was created" / "the parent
// row was revoked" without a real database.
type mockUserSessionRepo struct {
	mu             sync.Mutex
	created        []*session.Session
	revokedJTIs    []string
	revokedUserIDs []uuid.UUID
	createErr      error
	revokeErr      error
}

func (m *mockUserSessionRepo) Create(_ context.Context, s *session.Session) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := *s
	m.created = append(m.created, &cp)
	return nil
}

func (m *mockUserSessionRepo) FindByJTI(_ context.Context, jti string) (*session.Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, s := range m.created {
		if s.JTI == jti {
			cp := *s
			return &cp, nil
		}
	}
	return nil, session.ErrNotFound
}

func (m *mockUserSessionRepo) Touch(_ context.Context, _ string) error {
	return nil
}

func (m *mockUserSessionRepo) Revoke(_ context.Context, jti string) error {
	if m.revokeErr != nil {
		return m.revokeErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.revokedJTIs = append(m.revokedJTIs, jti)
	return nil
}

func (m *mockUserSessionRepo) RevokeAllForUser(_ context.Context, userID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.revokedUserIDs = append(m.revokedUserIDs, userID)
	return nil
}

func (m *mockUserSessionRepo) ListActiveByUser(_ context.Context, _ uuid.UUID) ([]*session.Session, error) {
	return nil, nil
}

// UpdateGeoCity / FindByID / RevokeByID / RevokeAllForUserExceptJTI
// are no-op stubs satisfying the SEC-SESSIONS additions to the port.
// The B.4 tests do not assert on these — the SEC-SESSIONS test
// surface lives in sessions_handler_test.go and exercises the
// postgres adapter directly.
func (m *mockUserSessionRepo) UpdateGeoCity(_ context.Context, _ string, _ string, _ string) error {
	return nil
}
func (m *mockUserSessionRepo) FindByID(_ context.Context, id uuid.UUID) (*session.Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, s := range m.created {
		if s.ID == id {
			cp := *s
			return &cp, nil
		}
	}
	return nil, session.ErrNotFound
}
func (m *mockUserSessionRepo) RevokeByID(_ context.Context, _ uuid.UUID) error  { return nil }
func (m *mockUserSessionRepo) RevokeAllForUserExceptJTI(_ context.Context, _ uuid.UUID, _ string) error {
	return nil
}

func (m *mockUserSessionRepo) snapshot() (created []*session.Session, revokedJTIs []string, revokedUsers []uuid.UUID) {
	m.mu.Lock()
	defer m.mu.Unlock()
	c := make([]*session.Session, len(m.created))
	copy(c, m.created)
	r := make([]string, len(m.revokedJTIs))
	copy(r, m.revokedJTIs)
	u := make([]uuid.UUID, len(m.revokedUserIDs))
	copy(u, m.revokedUserIDs)
	return c, r, u
}

// validFingerprint is the canonical (UA hash, anonymized IP) pair the
// tests pass through every login/refresh.
func validFingerprint() SessionFingerprint {
	return SessionFingerprint{
		UserAgentHash: "deadbeefcafef00d",
		IPAnonymized:  "203.0.113.0/24",
	}
}

// newAuthServiceForSessionTest assembles the auth service with the
// minimum wiring needed for the B.4 hooks: user repo, hasher, token
// service, refresh blacklist, and the user_sessions repo under test.
//
// The token service is a mockTokenService that issues "refresh_token_<jti>"
// strings and validates them by parsing back the JTI portion — that
// keeps the test in lock-step with the production assumption that
// every fresh refresh token carries a JTI.
func newAuthServiceForSessionTest(
	t *testing.T,
	users *user.User,
	sessionRepo repository.UserSessionRepository,
) (*Service, *mockTokenService) {
	t.Helper()

	userRepo := &mockUserRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*user.User, error) {
			return users, nil
		},
		getByEmailFn: func(_ context.Context, _ string) (*user.User, error) {
			return users, nil
		},
		existsByEmailFn: func(_ context.Context, _ string) (bool, error) {
			return false, nil
		},
	}

	tokens := &mockTokenService{
		generateRefreshFn: func(uid uuid.UUID) (string, error) {
			// Fresh JTI per call so the audit row's JTI is unique.
			jti := uuid.NewString()
			return "rt:" + jti + ":" + uid.String(), nil
		},
		validateRefreshFn: func(token string) (*service.TokenClaims, error) {
			// Decode "rt:<jti>:<userID>".
			if len(token) < 4 || token[:3] != "rt:" {
				return nil, user.ErrUnauthorized
			}
			rest := token[3:]
			// jti is the next 36 chars (UUID), then ":", then userID.
			if len(rest) < 37 || rest[36] != ':' {
				return nil, user.ErrUnauthorized
			}
			jti := rest[:36]
			userIDStr := rest[37:]
			uid, err := uuid.Parse(userIDStr)
			if err != nil {
				return nil, user.ErrUnauthorized
			}
			return &service.TokenClaims{
				UserID:    uid,
				JTI:       jti,
				ExpiresAt: time.Now().Add(24 * time.Hour),
			}, nil
		},
	}

	svc := NewServiceWithDeps(ServiceDeps{
		Users:        userRepo,
		Resets:       &mockPasswordResetRepo{},
		Hasher:       &mockHasher{},
		Tokens:       tokens,
		Email:        &mockEmailService{},
		UserSessions: sessionRepo,
	})
	return svc, tokens
}

func newSessionTestUser(t *testing.T) *user.User {
	t.Helper()
	u, err := user.NewUser(
		"alice@example.com",
		"hashed_StrongPass1!",
		"Alice",
		"Smith",
		"Alice S.",
		user.RoleProvider,
	)
	require.NoError(t, err)
	return u
}

// --- Login → row created ---

func TestSessionAudit_Login_RowCreated(t *testing.T) {
	u := newSessionTestUser(t)
	repo := &mockUserSessionRepo{}
	svc, _ := newAuthServiceForSessionTest(t, u, repo)

	out, err := svc.Login(context.Background(), LoginInput{
		Email:       "alice@example.com",
		Password:    "StrongPass1!",
		Fingerprint: validFingerprint(),
	})
	require.NoError(t, err)
	require.NotNil(t, out)

	created, _, _ := repo.snapshot()
	require.Len(t, created, 1)
	row := created[0]
	assert.Equal(t, u.ID, row.UserID)
	assert.Equal(t, session.LoginMethodPassword, row.LoginMethod)
	assert.Empty(t, row.ParentJTI, "first login has no parent")
	assert.Equal(t, "203.0.113.0/24", row.IPAnonymized)
	assert.Equal(t, "deadbeefcafef00d", row.UserAgentHash)
	assert.NotEmpty(t, row.JTI)
	assert.True(t, row.Active(time.Now()))
}

func TestSessionAudit_Login_NoFingerprint_SkipsRow(t *testing.T) {
	u := newSessionTestUser(t)
	repo := &mockUserSessionRepo{}
	svc, _ := newAuthServiceForSessionTest(t, u, repo)

	_, err := svc.Login(context.Background(), LoginInput{
		Email:    "alice@example.com",
		Password: "StrongPass1!",
		// no fingerprint
	})
	require.NoError(t, err)

	created, _, _ := repo.snapshot()
	assert.Empty(t, created, "missing fingerprint must skip the row write")
}

// --- Register (password) → row created ---

func TestSessionAudit_Register_RowCreated(t *testing.T) {
	repo := &mockUserSessionRepo{}
	users := &mockUserRepo{
		existsByEmailFn: func(_ context.Context, _ string) (bool, error) { return false, nil },
	}
	tokens := &mockTokenService{
		generateRefreshFn: func(uid uuid.UUID) (string, error) {
			return "rt:" + uuid.NewString() + ":" + uid.String(), nil
		},
		validateRefreshFn: func(token string) (*service.TokenClaims, error) {
			rest := token[3:]
			jti := rest[:36]
			uid, _ := uuid.Parse(rest[37:])
			return &service.TokenClaims{
				UserID:    uid,
				JTI:       jti,
				ExpiresAt: time.Now().Add(24 * time.Hour),
			}, nil
		},
	}
	svc := NewServiceWithDeps(ServiceDeps{
		Users:        users,
		Resets:       &mockPasswordResetRepo{},
		Hasher:       &mockHasher{},
		Tokens:       tokens,
		Email:        &mockEmailService{},
		UserSessions: repo,
	})

	out, err := svc.Register(context.Background(), RegisterInput{
		Email:       "bob@example.com",
		Password:    "StrongPass1!",
		FirstName:   "Bob",
		LastName:    "Bond",
		DisplayName: "Bob B.",
		Role:        user.RoleProvider,
		Fingerprint: validFingerprint(),
	})
	require.NoError(t, err)
	require.NotNil(t, out)

	created, _, _ := repo.snapshot()
	require.Len(t, created, 1)
	assert.Equal(t, session.LoginMethodPassword, created[0].LoginMethod)
	assert.Empty(t, created[0].ParentJTI)
}

// --- Refresh → old revoked + new created with parent_jti ---

func TestSessionAudit_Refresh_RotatesChain(t *testing.T) {
	u := newSessionTestUser(t)
	repo := &mockUserSessionRepo{}
	svc, _ := newAuthServiceForSessionTest(t, u, repo)

	// Seed: a successful login creates the parent session.
	_, err := svc.Login(context.Background(), LoginInput{
		Email:       "alice@example.com",
		Password:    "StrongPass1!",
		Fingerprint: validFingerprint(),
	})
	require.NoError(t, err)
	createdAfterLogin, _, _ := repo.snapshot()
	require.Len(t, createdAfterLogin, 1)
	parentJTI := createdAfterLogin[0].JTI
	parentRefresh := "rt:" + parentJTI + ":" + u.ID.String()

	// Refresh with a fresh fingerprint.
	out, err := svc.RefreshTokenWithFingerprint(context.Background(), parentRefresh, SessionFingerprint{
		UserAgentHash: "0123456789abcdef",
		IPAnonymized:  "198.51.100.0/24",
	})
	require.NoError(t, err)
	require.NotNil(t, out)

	created, revokedJTIs, _ := repo.snapshot()
	require.Len(t, created, 2, "rotation must add a second row")
	require.Len(t, revokedJTIs, 1)
	assert.Equal(t, parentJTI, revokedJTIs[0], "parent row must be revoked")

	child := created[1]
	assert.Equal(t, parentJTI, child.ParentJTI, "child must reference parent_jti")
	assert.Equal(t, session.LoginMethodRefresh, child.LoginMethod)
	assert.Equal(t, "198.51.100.0/24", child.IPAnonymized)
	assert.Equal(t, "0123456789abcdef", child.UserAgentHash)
	assert.NotEqual(t, parentJTI, child.JTI, "child has its own JTI")
}

// --- Logout (RevokeRefreshToken) → revoked_at set ---

func TestSessionAudit_Logout_RevokesRow(t *testing.T) {
	u := newSessionTestUser(t)
	repo := &mockUserSessionRepo{}
	svc, _ := newAuthServiceForSessionTest(t, u, repo)

	_, err := svc.Login(context.Background(), LoginInput{
		Email:       "alice@example.com",
		Password:    "StrongPass1!",
		Fingerprint: validFingerprint(),
	})
	require.NoError(t, err)
	created, _, _ := repo.snapshot()
	require.Len(t, created, 1)
	jti := created[0].JTI

	svc.RevokeRefreshToken(context.Background(), "rt:"+jti+":"+u.ID.String())

	_, revokedJTIs, _ := repo.snapshot()
	require.Len(t, revokedJTIs, 1)
	assert.Equal(t, jti, revokedJTIs[0])
}

// --- Reuse detection → all sessions for the user revoked ---

func TestSessionAudit_RefreshReplay_RevokesAllForUser(t *testing.T) {
	u := newSessionTestUser(t)
	repo := &mockUserSessionRepo{}
	svc, tokens := newAuthServiceForSessionTest(t, u, repo)

	// Wire a real-ish blacklist that already marks the JTI as
	// blacklisted so the reuse-detection branch fires on the first
	// refresh attempt.
	bl := newMockRefreshBlacklist()
	jti := uuid.NewString()
	require.NoError(t, bl.Add(context.Background(), jti, time.Hour))
	svc.refreshBlacklist = bl

	// Stub the token service to validate a token referencing that JTI.
	tokens.validateRefreshFn = func(_ string) (*service.TokenClaims, error) {
		return &service.TokenClaims{
			UserID:    u.ID,
			JTI:       jti,
			ExpiresAt: time.Now().Add(time.Hour),
		}, nil
	}

	_, err := svc.RefreshTokenWithFingerprint(context.Background(), "rt:replayed", validFingerprint())
	assert.ErrorIs(t, err, user.ErrUnauthorized)

	_, _, revokedUsers := repo.snapshot()
	require.Len(t, revokedUsers, 1, "reuse detection must mass-revoke for the user")
	assert.Equal(t, u.ID, revokedUsers[0])
}
