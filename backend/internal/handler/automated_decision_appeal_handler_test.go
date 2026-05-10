package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	app "marketplace-backend/internal/app/automateddecision"
	"marketplace-backend/internal/domain/automateddecision"
	"marketplace-backend/internal/handler/middleware"
	"marketplace-backend/internal/port/service"
)

// ---- inline mocks ----

type stubAppealRepo struct {
	created *automateddecision.Appeal
	err     error
}

func (s *stubAppealRepo) Create(_ context.Context, appeal *automateddecision.Appeal) error {
	if s.err != nil {
		return s.err
	}
	s.created = appeal
	return nil
}

// stubAppealEmail satisfies service.EmailService — only SendNotification
// is exercised; the others are no-op so the mock matches the wide port
// without dragging additional collaborators into the test.
type stubAppealEmail struct{ calls int }

var _ service.EmailService = (*stubAppealEmail)(nil)

func (s *stubAppealEmail) SendPasswordReset(context.Context, string, string) error { return nil }
func (s *stubAppealEmail) SendNotification(_ context.Context, _, _, _ string) error {
	s.calls++
	return nil
}
func (s *stubAppealEmail) SendTeamInvitation(context.Context, service.TeamInvitationEmailInput) error {
	return nil
}
func (s *stubAppealEmail) SendRolePermissionsChanged(
	context.Context,
	service.RolePermissionsChangedEmailInput,
) error {
	return nil
}

func newAppealHandlerForTest() (*AutomatedDecisionAppealHandler, *stubAppealRepo, *stubAppealEmail) {
	repo := &stubAppealRepo{}
	email := &stubAppealEmail{}
	svc := app.NewService(app.ServiceDeps{
		Repo:       repo,
		Email:      email,
		AdminEmail: "rgpd@marketplace.test",
	})
	return NewAutomatedDecisionAppealHandler(svc), repo, email
}

// ---- tests ----

func TestAppealHandler_FileAppeal_Unauthenticated_Returns401(t *testing.T) {
	h, repo, email := newAppealHandlerForTest()
	body := mustJSON(t, map[string]any{
		"decision_type": "moderation",
		"reference_id":  "mod-123",
		"reason":        "Mon contenu est conforme.",
	})
	r := httptest.NewRequest(http.MethodPost, "/api/v1/me/automated-decision-appeals",
		bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	h.FileAppeal(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d (%s)", w.Code, w.Body.String())
	}
	if repo.created != nil {
		t.Errorf("repo MUST NOT be called when caller is unauthenticated")
	}
	if email.calls != 0 {
		t.Errorf("email MUST NOT fire when caller is unauthenticated")
	}
}

func TestAppealHandler_FileAppeal_ValidPayload_Returns201_AndPersistsRow(t *testing.T) {
	h, repo, email := newAppealHandlerForTest()
	uid := uuid.New()
	ctx := context.WithValue(context.Background(), middleware.ContextKeyUserID, uid)

	body := mustJSON(t, map[string]any{
		"decision_type": "ranking",
		"reference_id":  "trace-42",
		"reason":        "Mon profil ne ressort plus dans la recherche depuis 2 semaines.",
	})
	r := httptest.NewRequest(http.MethodPost, "/api/v1/me/automated-decision-appeals",
		bytes.NewReader(body)).WithContext(ctx)
	r.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	h.FileAppeal(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d (%s)", w.Code, w.Body.String())
	}
	if repo.created == nil {
		t.Fatal("expected a row persisted")
	}
	if repo.created.UserID != uid {
		t.Errorf("expected user_id %s, got %s", uid, repo.created.UserID)
	}
	if repo.created.DecisionType != automateddecision.DecisionRanking {
		t.Errorf("expected ranking, got %s", repo.created.DecisionType)
	}
	if repo.created.Status != automateddecision.StatusPending {
		t.Errorf("expected pending status, got %s", repo.created.Status)
	}
	if email.calls != 1 {
		t.Errorf("expected admin email to fire once, got %d calls", email.calls)
	}
	// Response body shape — must echo id + created_at.
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON response: %v", err)
	}
	if id, _ := resp["id"].(string); id == "" {
		t.Errorf("response is missing id; body=%s", w.Body.String())
	}
	if status, _ := resp["status"].(string); status != "pending" {
		t.Errorf("response status mismatch; body=%s", w.Body.String())
	}
}

func TestAppealHandler_FileAppeal_InvalidDecisionType_Returns400(t *testing.T) {
	h, repo, email := newAppealHandlerForTest()
	uid := uuid.New()
	ctx := context.WithValue(context.Background(), middleware.ContextKeyUserID, uid)

	body := mustJSON(t, map[string]any{
		"decision_type": "search-ranking", // not the canonical "ranking"
		"reference_id":  "ref",
		"reason":        "reason",
	})
	r := httptest.NewRequest(http.MethodPost, "/api/v1/me/automated-decision-appeals",
		bytes.NewReader(body)).WithContext(ctx)
	r.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	h.FileAppeal(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d (%s)", w.Code, w.Body.String())
	}
	if repo.created != nil {
		t.Errorf("repo MUST NOT be called on invalid decision_type")
	}
	if email.calls != 0 {
		t.Errorf("email MUST NOT fire on invalid input")
	}
}

func TestAppealHandler_FileAppeal_NilService_Returns503(t *testing.T) {
	h := NewAutomatedDecisionAppealHandler(nil)
	body := mustJSON(t, map[string]any{
		"decision_type": "moderation",
		"reference_id":  "ref",
		"reason":        "reason",
	})
	r := httptest.NewRequest(http.MethodPost, "/api/v1/me/automated-decision-appeals",
		bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	h.FileAppeal(w, r)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d (%s)", w.Code, w.Body.String())
	}
}
