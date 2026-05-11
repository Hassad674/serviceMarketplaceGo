package handler

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	app "marketplace-backend/internal/app/automateddecision"
	"marketplace-backend/internal/domain/automateddecision"
	"marketplace-backend/internal/handler/middleware"
)

// TestAppealHandler_FileAppeal_EmptyReason_Returns400 asserts the
// reason-required rule is enforced at the handler-service boundary.
// Regression: an earlier version let the empty string through and
// the domain panicked when computing the byte length.
func TestAppealHandler_FileAppeal_EmptyReason_Returns400(t *testing.T) {
	h, repo, email := newAppealHandlerForTest()
	uid := uuid.New()
	ctx := context.WithValue(context.Background(), middleware.ContextKeyUserID, uid)

	body := mustJSON(t, map[string]any{
		"decision_type": "moderation",
		"reference_id":  "mod-x",
		"reason":        "",
	})
	r := httptest.NewRequest(http.MethodPost, "/api/v1/me/automated-decision-appeals",
		bytes.NewReader(body)).WithContext(ctx)
	r.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	h.FileAppeal(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 on empty reason, got %d (%s)", w.Code, w.Body.String())
	}
	if repo.created != nil {
		t.Errorf("repo MUST NOT be called on empty reason")
	}
	if email.calls != 0 {
		t.Errorf("email MUST NOT fire on validation failure")
	}
}

// TestAppealHandler_FileAppeal_ReferenceIDRequired_Returns400 asserts
// the reference-id rule is enforced.
func TestAppealHandler_FileAppeal_ReferenceIDRequired_Returns400(t *testing.T) {
	h, repo, _ := newAppealHandlerForTest()
	uid := uuid.New()
	ctx := context.WithValue(context.Background(), middleware.ContextKeyUserID, uid)

	body := mustJSON(t, map[string]any{
		"decision_type": "moderation",
		"reference_id":  "",
		"reason":        "Mon contenu est légitime.",
	})
	r := httptest.NewRequest(http.MethodPost, "/api/v1/me/automated-decision-appeals",
		bytes.NewReader(body)).WithContext(ctx)
	r.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	h.FileAppeal(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 on empty reference_id, got %d (%s)", w.Code, w.Body.String())
	}
	if repo.created != nil {
		t.Errorf("repo MUST NOT be called on empty reference_id")
	}
}

// TestAppealHandler_FileAppeal_ReasonTooLong_Returns400 asserts the
// 5000-byte reason cap (RGPD art. 22 reasonable text limit). A 5001-
// byte body must be rejected at the domain layer.
func TestAppealHandler_FileAppeal_ReasonTooLong_Returns400(t *testing.T) {
	h, repo, _ := newAppealHandlerForTest()
	uid := uuid.New()
	ctx := context.WithValue(context.Background(), middleware.ContextKeyUserID, uid)

	// 5001 ASCII bytes of "a"
	reason := strings.Repeat("a", 5001)
	body := mustJSON(t, map[string]any{
		"decision_type": "moderation",
		"reference_id":  "mod-1",
		"reason":        reason,
	})
	r := httptest.NewRequest(http.MethodPost, "/api/v1/me/automated-decision-appeals",
		bytes.NewReader(body)).WithContext(ctx)
	r.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	h.FileAppeal(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 on 5001-byte reason, got %d (%s)", w.Code, w.Body.String())
	}
	if repo.created != nil {
		t.Errorf("repo MUST NOT be called when reason exceeds cap")
	}
}

// TestAppealHandler_FileAppeal_Reason5000_Accepted asserts the cap is
// inclusive — exactly 5000 bytes must be allowed.
func TestAppealHandler_FileAppeal_Reason5000_Accepted(t *testing.T) {
	h, repo, email := newAppealHandlerForTest()
	uid := uuid.New()
	ctx := context.WithValue(context.Background(), middleware.ContextKeyUserID, uid)

	reason := strings.Repeat("a", 5000)
	body := mustJSON(t, map[string]any{
		"decision_type": "moderation",
		"reference_id":  "mod-1",
		"reason":        reason,
	})
	r := httptest.NewRequest(http.MethodPost, "/api/v1/me/automated-decision-appeals",
		bytes.NewReader(body)).WithContext(ctx)
	r.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	h.FileAppeal(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 on exactly 5000-byte reason, got %d (%s)", w.Code, w.Body.String())
	}
	if repo.created == nil {
		t.Error("expected row persisted at boundary")
	}
	if email.calls != 1 {
		t.Errorf("expected email to fire once on success, got %d", email.calls)
	}
}

// TestAppealHandler_FileAppeal_RepoErrorReturns500 asserts a generic
// persistence error becomes a 500 (not leaked as the underlying SQL).
func TestAppealHandler_FileAppeal_RepoErrorReturns500(t *testing.T) {
	repo := &stubAppealRepo{err: errors.New("connection refused")}
	email := &stubAppealEmail{}
	svc := app.NewService(app.ServiceDeps{
		Repo:       repo,
		Email:      email,
		AdminEmail: "rgpd@marketplace.test",
	})
	h := NewAutomatedDecisionAppealHandler(svc)

	uid := uuid.New()
	ctx := context.WithValue(context.Background(), middleware.ContextKeyUserID, uid)
	body := mustJSON(t, map[string]any{
		"decision_type": "moderation",
		"reference_id":  "ref",
		"reason":        "Reason ok",
	})
	r := httptest.NewRequest(http.MethodPost, "/api/v1/me/automated-decision-appeals",
		bytes.NewReader(body)).WithContext(ctx)
	r.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	h.FileAppeal(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d (%s)", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "internal_error") {
		t.Errorf("expected internal_error code, got body=%s", w.Body.String())
	}
	// Body must NOT leak the underlying SQL error.
	if strings.Contains(w.Body.String(), "connection refused") {
		t.Errorf("response must not leak internal error details: %s", w.Body.String())
	}
}

// TestAppealHandler_FileAppeal_RejectsInvalidJSON asserts a malformed
// body returns invalid_body, not 500.
func TestAppealHandler_FileAppeal_RejectsInvalidJSON(t *testing.T) {
	h, _, _ := newAppealHandlerForTest()
	uid := uuid.New()
	ctx := context.WithValue(context.Background(), middleware.ContextKeyUserID, uid)

	r := httptest.NewRequest(http.MethodPost, "/api/v1/me/automated-decision-appeals",
		bytes.NewReader([]byte("{not-json"))).WithContext(ctx)
	r.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	h.FileAppeal(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "invalid_body") {
		t.Errorf("expected invalid_body code, got body=%s", w.Body.String())
	}
}

// TestAppealHandler_FileAppeal_RejectsExtraFields asserts the
// DisallowUnknownFields contract — a body with an unexpected key is
// 400, not silently accepted.
func TestAppealHandler_FileAppeal_RejectsExtraFields(t *testing.T) {
	h, _, _ := newAppealHandlerForTest()
	uid := uuid.New()
	ctx := context.WithValue(context.Background(), middleware.ContextKeyUserID, uid)

	body := mustJSON(t, map[string]any{
		"decision_type":  "moderation",
		"reference_id":   "ref",
		"reason":         "ok",
		"injection_attempt": "would-be-bypass",
	})
	r := httptest.NewRequest(http.MethodPost, "/api/v1/me/automated-decision-appeals",
		bytes.NewReader(body)).WithContext(ctx)
	r.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	h.FileAppeal(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d (%s)", w.Code, w.Body.String())
	}
}

// TestAppealHandler_FileAppeal_AllValidDecisionTypes covers every
// canonical decision_type enum value, asserting each rounds-trips to a
// persisted row. Acts as a contract pin so a future addition / removal
// of an enum value must update this list.
func TestAppealHandler_FileAppeal_AllValidDecisionTypes(t *testing.T) {
	canonical := []automateddecision.DecisionType{
		automateddecision.DecisionMod,
		automateddecision.DecisionRanking,
		automateddecision.DecisionPayment,
	}
	for _, dt := range canonical {
		t.Run(string(dt), func(t *testing.T) {
			h, repo, _ := newAppealHandlerForTest()
			uid := uuid.New()
			ctx := context.WithValue(context.Background(), middleware.ContextKeyUserID, uid)
			body := mustJSON(t, map[string]any{
				"decision_type": string(dt),
				"reference_id":  "ref-" + string(dt),
				"reason":        "Reason for " + string(dt),
			})
			r := httptest.NewRequest(http.MethodPost, "/api/v1/me/automated-decision-appeals",
				bytes.NewReader(body)).WithContext(ctx)
			r.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			h.FileAppeal(w, r)

			if w.Code != http.StatusCreated {
				t.Fatalf("expected 201 for %q, got %d (%s)", dt, w.Code, w.Body.String())
			}
			if repo.created == nil || repo.created.DecisionType != dt {
				t.Errorf("expected row with DecisionType=%q persisted, got %+v", dt, repo.created)
			}
		})
	}
}
