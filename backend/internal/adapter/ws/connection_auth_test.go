package ws

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"marketplace-backend/internal/port/service"
)

// fakeTokenSvc satisfies service.TokenService. It is intentionally
// permissive — every JWT validates to a fixed UUID — so the test can
// prove that NONE of those JWTs grant WS access through the URL
// query string after SEC-15.
type fakeTokenSvc struct{}

func (fakeTokenSvc) GenerateAccessToken(_ service.AccessTokenInput) (string, error) {
	return "fake.access.token", nil
}
func (fakeTokenSvc) GenerateRefreshToken(_ uuid.UUID) (string, error) {
	return "fake.refresh.token", nil
}
func (fakeTokenSvc) ValidateAccessToken(_ string) (*service.TokenClaims, error) {
	return &service.TokenClaims{UserID: uuid.New(), ExpiresAt: time.Now().Add(time.Hour)}, nil
}
func (fakeTokenSvc) ValidateRefreshToken(_ string) (*service.TokenClaims, error) {
	return &service.TokenClaims{UserID: uuid.New(), ExpiresAt: time.Now().Add(time.Hour)}, nil
}

// fakeSessionSvc satisfies the subset of service.SessionService that
// authenticateWS calls. We control the answers per test by populating
// the maps before invoking.
type fakeSessionSvc struct {
	sessions map[string]uuid.UUID
	wsTokens map[string]uuid.UUID
}

func newFakeSessionSvc() *fakeSessionSvc {
	return &fakeSessionSvc{
		sessions: map[string]uuid.UUID{},
		wsTokens: map[string]uuid.UUID{},
	}
}

func (s *fakeSessionSvc) Create(_ context.Context, _ service.CreateSessionInput) (*service.Session, error) {
	return &service.Session{ID: "fake-session"}, nil
}

func (s *fakeSessionSvc) Get(_ context.Context, sessionID string) (*service.Session, error) {
	uid, ok := s.sessions[sessionID]
	if !ok {
		return nil, errors.New("session not found")
	}
	return &service.Session{ID: sessionID, UserID: uid}, nil
}

func (s *fakeSessionSvc) Delete(_ context.Context, _ string) error                { return nil }
func (s *fakeSessionSvc) Touch(_ context.Context, _ string) error                 { return nil }
func (s *fakeSessionSvc) DeleteByUserID(_ context.Context, _ uuid.UUID) error     { return nil }
func (s *fakeSessionSvc) CreateWSToken(_ context.Context, uid uuid.UUID) (string, error) {
	s.wsTokens["new-ws-token"] = uid
	return "new-ws-token", nil
}
func (s *fakeSessionSvc) ValidateWSToken(_ context.Context, token string) (uuid.UUID, error) {
	uid, ok := s.wsTokens[token]
	if !ok {
		return uuid.Nil, errors.New("invalid ws token")
	}
	return uid, nil
}

func TestAuthenticateWS_JWTInURLIsRejected(t *testing.T) {
	// SEC-15: the legacy "Strategy 3 — JWT-in-URL" path has been
	// removed. A request that ONLY carries a JWT in the query string
	// must be rejected. Even a JWT that would validate against the
	// token service must NOT be accepted via this transport.
	r := httptest.NewRequest(http.MethodGet, "/api/v1/ws?token=any.valid.jwt", nil)
	uid, err := authenticateWS(r, fakeTokenSvc{}, newFakeSessionSvc())
	assert.ErrorIs(t, err, errUnauthorizedWS)
	assert.Equal(t, uuid.UUID{}, uid)
}

func TestAuthenticateWS_AcceptsWSTokenQueryParam(t *testing.T) {
	// SEC-15: the only query-param credential allowed is the
	// short-lived single-use ws_token issued by /auth/ws-token.
	want := uuid.New()
	sess := newFakeSessionSvc()
	sess.wsTokens["ticket-1234"] = want

	r := httptest.NewRequest(http.MethodGet, "/api/v1/ws?ws_token=ticket-1234", nil)
	uid, err := authenticateWS(r, fakeTokenSvc{}, sess)
	assert.NoError(t, err)
	assert.Equal(t, want, uid)
}

func TestAuthenticateWS_AcceptsSessionCookie(t *testing.T) {
	want := uuid.New()
	sess := newFakeSessionSvc()
	sess.sessions["valid-session"] = want

	r := httptest.NewRequest(http.MethodGet, "/api/v1/ws", nil)
	r.AddCookie(&http.Cookie{Name: "session_id", Value: "valid-session"})

	uid, err := authenticateWS(r, fakeTokenSvc{}, sess)
	assert.NoError(t, err)
	assert.Equal(t, want, uid)
}

func TestAuthenticateWS_NoCredentialsRejected(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/api/v1/ws", nil)
	uid, err := authenticateWS(r, fakeTokenSvc{}, newFakeSessionSvc())
	assert.ErrorIs(t, err, errUnauthorizedWS)
	assert.Equal(t, uuid.UUID{}, uid)
}

func TestAuthenticateWS_TokenAndWSTokenBothPresent_PrefersWSToken(t *testing.T) {
	// Defence-in-depth: even when the legacy "token" param is sent
	// alongside a valid ws_token (a buggy client that hasn't migrated
	// yet), the JWT path must NOT be exercised. ws_token wins.
	want := uuid.New()
	sess := newFakeSessionSvc()
	sess.wsTokens["good-ticket"] = want

	r := httptest.NewRequest(http.MethodGet,
		"/api/v1/ws?token=any.valid.jwt&ws_token=good-ticket", nil)
	uid, err := authenticateWS(r, fakeTokenSvc{}, sess)
	assert.NoError(t, err)
	assert.Equal(t, want, uid, "ws_token must win when both credentials are present")
}
