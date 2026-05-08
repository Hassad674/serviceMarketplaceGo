package handler_test

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	securityapp "marketplace-backend/internal/app/security"
	"marketplace-backend/internal/domain/audit"
	"marketplace-backend/internal/handler"
	"marketplace-backend/internal/handler/middleware"
)

// secFakeAuditRepo backs the security handler tests. It only needs to
// honour ListByUser; Log/ListByResource are unused but required to
// satisfy the AuditRepository interface.
type secFakeAuditRepo struct {
	rowsByUser map[uuid.UUID][]*audit.Entry
	nextCursor string
	calls      int
	err        error
}

func (r *secFakeAuditRepo) Log(_ context.Context, _ *audit.Entry) error { return nil }
func (r *secFakeAuditRepo) ListByResource(_ context.Context, _ audit.ResourceType, _ uuid.UUID, _ string, _ int) ([]*audit.Entry, string, error) {
	return nil, "", nil
}
func (r *secFakeAuditRepo) ListByUser(_ context.Context, userID uuid.UUID, _ string, _ int) ([]*audit.Entry, string, error) {
	r.calls++
	if r.err != nil {
		return nil, "", r.err
	}
	return r.rowsByUser[userID], r.nextCursor, nil
}

func newSecHandler(repo *secFakeAuditRepo) *handler.SecurityHandler {
	svc := securityapp.NewService(repo)
	return handler.NewSecurityHandler(svc)
}

func mkAuthEntry(t *testing.T, action audit.Action, userID uuid.UUID, ts time.Time, ua, ip string) *audit.Entry {
	t.Helper()
	uidCopy := userID
	entry := &audit.Entry{
		ID:        uuid.New(),
		UserID:    &uidCopy,
		Action:    action,
		Metadata:  map[string]any{},
		CreatedAt: ts,
	}
	if ua != "" {
		entry.Metadata["user_agent"] = ua
	}
	if ip != "" {
		parsed := net.ParseIP(ip)
		if parsed != nil {
			entry.IPAddress = &parsed
		}
	}
	return entry
}

func reqWithUser(method, target string, userID uuid.UUID) *http.Request {
	req := httptest.NewRequest(method, target, nil)
	ctx := context.WithValue(req.Context(), middleware.ContextKeyUserID, userID)
	return req.WithContext(ctx)
}

func TestSecurityHandler_ListActivity_HappyPath(t *testing.T) {
	user := uuid.New()
	now := time.Now().UTC()
	repo := &secFakeAuditRepo{
		rowsByUser: map[uuid.UUID][]*audit.Entry{
			user: {
				mkAuthEntry(t, audit.ActionLoginSuccess, user, now,
					"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) Chrome/120.0.0.0",
					"203.0.113.4"),
				mkAuthEntry(t, audit.ActionLogout, user, now.Add(-time.Hour),
					"Mozilla/5.0 (iPhone; CPU iPhone OS 16_5 like Mac OS X) Version/16.5 Mobile/15E148 Safari/604.1",
					"198.51.100.7"),
			},
		},
	}
	h := newSecHandler(repo)
	w := httptest.NewRecorder()
	h.ListActivity(w, reqWithUser(http.MethodGet, "/api/v1/me/security/activity", user))

	require.Equal(t, http.StatusOK, w.Code)
	var body struct {
		Data []struct {
			ID               string `json:"id"`
			Action           string `json:"action"`
			IPAddress        string `json:"ip_address"`
			UserAgentSummary string `json:"user_agent_summary"`
			AccessKind       string `json:"access_kind"`
			CreatedAt        string `json:"created_at"`
		} `json:"data"`
		NextCursor string `json:"next_cursor"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Len(t, body.Data, 2)
	assert.Equal(t, "auth.login_success", body.Data[0].Action)
	assert.Equal(t, "203.0.113.4", body.Data[0].IPAddress)
	assert.Equal(t, "desktop", body.Data[0].AccessKind)
	assert.Contains(t, body.Data[0].UserAgentSummary, "Chrome")

	assert.Equal(t, "auth.logout", body.Data[1].Action)
	assert.Equal(t, "mobile", body.Data[1].AccessKind)
	assert.Contains(t, body.Data[1].UserAgentSummary, "Safari")
}

func TestSecurityHandler_ListActivity_OnlyOwnEvents(t *testing.T) {
	me := uuid.New()
	other := uuid.New()
	now := time.Now().UTC()
	repo := &secFakeAuditRepo{
		rowsByUser: map[uuid.UUID][]*audit.Entry{
			me:    {mkAuthEntry(t, audit.ActionLoginSuccess, me, now, "", "")},
			other: {mkAuthEntry(t, audit.ActionLoginSuccess, other, now, "", "")},
		},
	}
	h := newSecHandler(repo)
	w := httptest.NewRecorder()
	h.ListActivity(w, reqWithUser(http.MethodGet, "/api/v1/me/security/activity", me))
	require.Equal(t, http.StatusOK, w.Code)

	var body struct {
		Data []map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Len(t, body.Data, 1)
}

func TestSecurityHandler_ListActivity_FiltersNonAuthActions(t *testing.T) {
	user := uuid.New()
	now := time.Now().UTC()
	repo := &secFakeAuditRepo{
		rowsByUser: map[uuid.UUID][]*audit.Entry{
			user: {
				mkAuthEntry(t, audit.ActionReceiptView, user, now, "", ""),
				mkAuthEntry(t, audit.ActionLoginSuccess, user, now.Add(-time.Minute), "", ""),
				mkAuthEntry(t, audit.ActionMemberRoleChanged, user, now.Add(-2*time.Minute), "", ""),
				mkAuthEntry(t, audit.ActionPasswordResetRequest, user, now.Add(-3*time.Minute), "", ""),
			},
		},
	}
	h := newSecHandler(repo)
	w := httptest.NewRecorder()
	h.ListActivity(w, reqWithUser(http.MethodGet, "/api/v1/me/security/activity", user))
	require.Equal(t, http.StatusOK, w.Code)

	var body struct {
		Data []struct {
			Action string `json:"action"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Len(t, body.Data, 2)
	assert.Equal(t, "auth.login_success", body.Data[0].Action)
	assert.Equal(t, "auth.password_reset_request", body.Data[1].Action)
}

func TestSecurityHandler_ListActivity_PassesCursor(t *testing.T) {
	user := uuid.New()
	repo := &secFakeAuditRepo{
		rowsByUser: map[uuid.UUID][]*audit.Entry{user: nil},
		nextCursor: "next-cursor-value",
	}
	h := newSecHandler(repo)
	w := httptest.NewRecorder()
	h.ListActivity(w, reqWithUser(http.MethodGet, "/api/v1/me/security/activity?cursor=abc&limit=5", user))
	require.Equal(t, http.StatusOK, w.Code)
	var body struct {
		Data       []map[string]any `json:"data"`
		NextCursor string           `json:"next_cursor"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "next-cursor-value", body.NextCursor)
	assert.Empty(t, body.Data)
	assert.Equal(t, 1, repo.calls)
}

func TestSecurityHandler_ListActivity_Unauthorized(t *testing.T) {
	repo := &secFakeAuditRepo{rowsByUser: map[uuid.UUID][]*audit.Entry{}}
	h := newSecHandler(repo)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/me/security/activity", nil)
	w := httptest.NewRecorder()
	h.ListActivity(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestSecurityHandler_ListActivity_EmptyData(t *testing.T) {
	user := uuid.New()
	repo := &secFakeAuditRepo{rowsByUser: map[uuid.UUID][]*audit.Entry{}}
	h := newSecHandler(repo)
	w := httptest.NewRecorder()
	h.ListActivity(w, reqWithUser(http.MethodGet, "/api/v1/me/security/activity", user))
	require.Equal(t, http.StatusOK, w.Code)
	// Body must contain `"data": []` (not `null`) so the FE can iterate.
	assert.Contains(t, w.Body.String(), `"data":[]`)
}

func TestSecurityHandler_ListActivity_NilService_503(t *testing.T) {
	h := handler.NewSecurityHandler(nil)
	w := httptest.NewRecorder()
	h.ListActivity(w, reqWithUser(http.MethodGet, "/api/v1/me/security/activity", uuid.New()))
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}
