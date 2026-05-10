package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	consentapp "marketplace-backend/internal/app/consent"
	"marketplace-backend/internal/domain/consent"
	"marketplace-backend/internal/handler/middleware"
)

type stubConsentRepo struct {
	created *consent.Entry
	err     error
}

func (s *stubConsentRepo) Create(_ context.Context, entry *consent.Entry) error {
	s.created = entry
	return s.err
}

func newConsentHandlerForTest(t *testing.T) (*ConsentHandler, *stubConsentRepo) {
	t.Helper()
	repo := &stubConsentRepo{}
	svc := consentapp.NewService(repo)
	return NewConsentHandler(svc), repo
}

func TestConsentHandler_Log_Success(t *testing.T) {
	h, repo := newConsentHandlerForTest(t)

	body := mustJSON(t, map[string]any{
		"action":     "accept_all",
		"categories": []string{"analytics", "functional"},
		"session_id": "sess-1",
	})
	r := httptest.NewRequest(http.MethodPost, "/api/v1/consent/log", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("User-Agent", "Mozilla/5.0 (test)")
	r.RemoteAddr = "203.0.113.42:1234"

	w := httptest.NewRecorder()
	h.Log(w, r)

	if w.Code != http.StatusNoContent {
		t.Fatalf("status %d body=%s", w.Code, w.Body.String())
	}
	if repo.created == nil {
		t.Fatal("expected entry persisted")
	}
	if repo.created.IPAnonymized != "203.0.x.x" {
		t.Errorf("expected truncated IP, got %q", repo.created.IPAnonymized)
	}
	if len(repo.created.UserAgentHash) != 64 {
		t.Errorf("expected 64-hex sha256 UA hash")
	}
	if repo.created.UserID != nil {
		t.Errorf("anonymous request should not stamp user_id")
	}
}

func TestConsentHandler_Log_RejectsInvalidAction(t *testing.T) {
	h, repo := newConsentHandlerForTest(t)

	body := mustJSON(t, map[string]any{
		"action":     "ignore_me", // not in enum
		"categories": []string{"analytics"},
	})
	r := httptest.NewRequest(http.MethodPost, "/api/v1/consent/log", bytes.NewReader(body))
	r.RemoteAddr = "1.2.3.4:5"

	w := httptest.NewRecorder()
	h.Log(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d (%s)", w.Code, w.Body.String())
	}
	if repo.created != nil {
		t.Errorf("repo MUST NOT be called on invalid input")
	}
}

func TestConsentHandler_Log_HonoursXForwardedFor(t *testing.T) {
	h, repo := newConsentHandlerForTest(t)

	body := mustJSON(t, map[string]any{
		"action":     "refuse_all",
		"categories": []string{"functional"},
	})
	r := httptest.NewRequest(http.MethodPost, "/api/v1/consent/log", bytes.NewReader(body))
	r.Header.Set("X-Forwarded-For", "198.51.100.7, 10.0.0.1")
	r.Header.Set("User-Agent", "Mozilla/5.0 (proxy)")
	r.RemoteAddr = "127.0.0.1:0"

	w := httptest.NewRecorder()
	h.Log(w, r)

	if w.Code != http.StatusNoContent {
		t.Fatalf("status %d body=%s", w.Code, w.Body.String())
	}
	if repo.created.IPAnonymized != "198.51.x.x" {
		t.Errorf("expected XFF IP truncated, got %q", repo.created.IPAnonymized)
	}
}

func TestConsentHandler_Log_NilService_Returns503(t *testing.T) {
	h := NewConsentHandler(nil)
	body := mustJSON(t, map[string]any{"action": "accept_all", "categories": []string{"analytics"}})
	r := httptest.NewRequest(http.MethodPost, "/api/v1/consent/log", bytes.NewReader(body))
	r.RemoteAddr = "1.2.3.4:5"

	w := httptest.NewRecorder()
	h.Log(w, r)
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}
}

func TestConsentHandler_Log_AuthenticatedUserStampsUserID(t *testing.T) {
	// We mimic the middleware contract by injecting a userID via the
	// same context key the middleware uses (validated by GetUserID).
	// The handler must pick that up and stamp it on the persisted row.
	repo := &stubConsentRepo{}
	svc := consentapp.NewService(repo)
	h := NewConsentHandler(svc)

	uid := uuid.New()
	ctx := context.WithValue(context.Background(), middleware.ContextKeyUserID, uid)
	body := mustJSON(t, map[string]any{"action": "accept_all", "categories": []string{"analytics"}})
	r := httptest.NewRequest(http.MethodPost, "/api/v1/consent/log", bytes.NewReader(body)).WithContext(ctx)
	r.Header.Set("User-Agent", "Mozilla/5.0 (auth-test)")
	r.RemoteAddr = "1.2.3.4:5"

	w := httptest.NewRecorder()
	h.Log(w, r)
	if w.Code != http.StatusNoContent {
		t.Fatalf("status %d", w.Code)
	}
	if repo.created.UserID == nil || *repo.created.UserID != uid {
		t.Errorf("expected user_id %v, got %v", uid, repo.created.UserID)
	}
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
