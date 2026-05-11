package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/domain/session"
	"marketplace-backend/internal/handler"
	"marketplace-backend/internal/handler/middleware"
)

// fakeSessionsRepo is a tiny in-memory UserSessionRepository used by
// the sessions handler tests. It only implements the methods the
// handler invokes; the rest of the port interface returns zero values
// (the handler never calls them in this test surface).
type fakeSessionsRepo struct {
	mu           sync.Mutex
	rows         []*session.Session
	revoked      []uuid.UUID
	revokedExcpt []string
	notFound     bool
	failList     bool
}

func (r *fakeSessionsRepo) Create(_ context.Context, _ *session.Session) error    { return nil }
func (r *fakeSessionsRepo) FindByJTI(_ context.Context, _ string) (*session.Session, error) {
	return nil, session.ErrNotFound
}
func (r *fakeSessionsRepo) Touch(_ context.Context, _ string) error    { return nil }
func (r *fakeSessionsRepo) Revoke(_ context.Context, _ string) error    { return nil }
func (r *fakeSessionsRepo) RevokeAllForUser(_ context.Context, _ uuid.UUID) error { return nil }
func (r *fakeSessionsRepo) UpdateGeoCity(_ context.Context, _ string, _ string, _ string) error {
	return nil
}
func (r *fakeSessionsRepo) ListActiveByUser(_ context.Context, userID uuid.UUID) ([]*session.Session, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.failList {
		return nil, session.ErrUserIDRequired
	}
	out := make([]*session.Session, 0, len(r.rows))
	for _, s := range r.rows {
		if s.UserID == userID {
			cp := *s
			out = append(out, &cp)
		}
	}
	return out, nil
}
func (r *fakeSessionsRepo) FindByID(_ context.Context, id uuid.UUID) (*session.Session, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.notFound {
		return nil, session.ErrNotFound
	}
	for _, s := range r.rows {
		if s.ID == id {
			cp := *s
			return &cp, nil
		}
	}
	return nil, session.ErrNotFound
}
func (r *fakeSessionsRepo) RevokeByID(_ context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.revoked = append(r.revoked, id)
	return nil
}
func (r *fakeSessionsRepo) RevokeAllForUserExceptJTI(_ context.Context, _ uuid.UUID, exceptJTI string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.revokedExcpt = append(r.revokedExcpt, exceptJTI)
	return nil
}

func sessReqWithUser(method, target string, userID uuid.UUID) *http.Request {
	req := httptest.NewRequest(method, target, nil)
	ctx := context.WithValue(req.Context(), middleware.ContextKeyUserID, userID)
	return req.WithContext(ctx)
}

func mkSession(userID uuid.UUID, deviceLabel, browser, os, city string) *session.Session {
	return &session.Session{
		ID:           uuid.New(),
		UserID:       userID,
		JTI:          uuid.NewString(),
		LoginMethod:  session.LoginMethodPassword,
		CreatedAt:    time.Now().UTC().Add(-time.Hour),
		LastUsedAt:   time.Now().UTC(),
		ExpiresAt:    time.Now().UTC().Add(24 * time.Hour),
		DeviceLabel:  deviceLabel,
		Browser:      browser,
		OS:           os,
		City:         city,
		CountryCode:  "FR",
		IPAnonymized: "203.0.113.0/24",
	}
}

func TestSessionsHandler_List_HappyPath(t *testing.T) {
	user := uuid.New()
	repo := &fakeSessionsRepo{
		rows: []*session.Session{
			mkSession(user, "Ordinateur de bureau (Chrome)", "Chrome", "Windows", "Paris"),
			mkSession(user, "iPhone (Safari)", "Safari", "iOS", "Lyon"),
			// Foreign row that must NEVER leak.
			mkSession(uuid.New(), "iPad (Safari)", "Safari", "iOS", "Marseille"),
		},
	}
	h := handler.NewSessionsHandler(repo, "session_id")
	w := httptest.NewRecorder()
	h.List(w, sessReqWithUser(http.MethodGet, "/api/v1/me/sessions", user))

	require.Equal(t, http.StatusOK, w.Code)
	var body struct {
		Data []struct {
			ID          string `json:"id"`
			DeviceLabel string `json:"device_label"`
			Browser     string `json:"browser"`
			OS          string `json:"os"`
			City        string `json:"city"`
			CountryCode string `json:"country_code"`
			IsCurrent   bool   `json:"is_current"`
		} `json:"data"`
	}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	require.Len(t, body.Data, 2, "must not leak the foreign row")
	require.Equal(t, "Ordinateur de bureau (Chrome)", body.Data[0].DeviceLabel)
	require.Equal(t, "Paris", body.Data[0].City)
	require.Equal(t, "FR", body.Data[0].CountryCode)
}

func TestSessionsHandler_List_Unauthorized_NoUserInContext(t *testing.T) {
	repo := &fakeSessionsRepo{}
	h := handler.NewSessionsHandler(repo, "session_id")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/me/sessions", nil)
	h.List(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestSessionsHandler_List_NilRepoReturns503(t *testing.T) {
	h := handler.NewSessionsHandler(nil, "session_id")
	w := httptest.NewRecorder()
	h.List(w, sessReqWithUser(http.MethodGet, "/api/v1/me/sessions", uuid.New()))
	require.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestSessionsHandler_List_FallbackDeviceLabel(t *testing.T) {
	// A pre-migration-150 row has empty device_label — the handler
	// must fall back to "Appareil inconnu" so the wire shape is
	// always populated.
	user := uuid.New()
	row := mkSession(user, "", "", "", "")
	repo := &fakeSessionsRepo{rows: []*session.Session{row}}
	h := handler.NewSessionsHandler(repo, "session_id")
	w := httptest.NewRecorder()
	h.List(w, sessReqWithUser(http.MethodGet, "/api/v1/me/sessions", user))
	require.Equal(t, http.StatusOK, w.Code)
	var body struct {
		Data []struct {
			DeviceLabel string `json:"device_label"`
		} `json:"data"`
	}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	require.Equal(t, handler.UnknownDeviceLabel, body.Data[0].DeviceLabel)
}

func TestSessionsHandler_Revoke_OwnerOK(t *testing.T) {
	user := uuid.New()
	row := mkSession(user, "Ordinateur de bureau (Chrome)", "Chrome", "Windows", "Paris")
	repo := &fakeSessionsRepo{rows: []*session.Session{row}}
	h := handler.NewSessionsHandler(repo, "session_id")

	// Build a chi router so the {id} URL param resolves through the
	// production routing code path.
	r := chi.NewRouter()
	r.Delete("/me/sessions/{id}", h.Revoke)
	rec := httptest.NewRecorder()
	req := sessReqWithUser(http.MethodDelete, "/me/sessions/"+row.ID.String(), user)
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNoContent, rec.Code)
	require.Len(t, repo.revoked, 1)
	require.Equal(t, row.ID, repo.revoked[0])
}

func TestSessionsHandler_Revoke_ForeignSession403(t *testing.T) {
	owner := uuid.New()
	row := mkSession(owner, "iPad (Safari)", "Safari", "iOS", "Lyon")
	repo := &fakeSessionsRepo{rows: []*session.Session{row}}
	h := handler.NewSessionsHandler(repo, "session_id")

	r := chi.NewRouter()
	r.Delete("/me/sessions/{id}", h.Revoke)
	rec := httptest.NewRecorder()
	// Attacker = a different user.
	attacker := uuid.New()
	req := sessReqWithUser(http.MethodDelete, "/me/sessions/"+row.ID.String(), attacker)
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusForbidden, rec.Code)
	require.Empty(t, repo.revoked, "must NOT revoke a foreign session")
}

func TestSessionsHandler_Revoke_InvalidUUID400(t *testing.T) {
	repo := &fakeSessionsRepo{}
	h := handler.NewSessionsHandler(repo, "session_id")
	r := chi.NewRouter()
	r.Delete("/me/sessions/{id}", h.Revoke)
	rec := httptest.NewRecorder()
	req := sessReqWithUser(http.MethodDelete, "/me/sessions/not-a-uuid", uuid.New())
	r.ServeHTTP(rec, req)
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestSessionsHandler_Revoke_NotFound404(t *testing.T) {
	repo := &fakeSessionsRepo{notFound: true}
	h := handler.NewSessionsHandler(repo, "session_id")
	r := chi.NewRouter()
	r.Delete("/me/sessions/{id}", h.Revoke)
	rec := httptest.NewRecorder()
	req := sessReqWithUser(http.MethodDelete, "/me/sessions/"+uuid.NewString(), uuid.New())
	r.ServeHTTP(rec, req)
	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestSessionsHandler_RevokeAllExceptCurrent_OK(t *testing.T) {
	user := uuid.New()
	repo := &fakeSessionsRepo{}
	h := handler.NewSessionsHandler(repo, "session_id")
	r := chi.NewRouter()
	r.Post("/me/sessions/revoke-others", h.RevokeAllExceptCurrent)
	rec := httptest.NewRecorder()
	req := sessReqWithUser(http.MethodPost, "/me/sessions/revoke-others", user)
	r.ServeHTTP(rec, req)
	require.Equal(t, http.StatusNoContent, rec.Code)
	require.Len(t, repo.revokedExcpt, 1)
	// Cookie not present in the test → exceptJTI falls back to ''.
	require.Equal(t, "", repo.revokedExcpt[0])
}
