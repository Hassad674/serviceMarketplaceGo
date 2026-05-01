package handler

// Handler-level integration tests for BUG-NEW-09. The pre-fix handler
// passed the SUSPENDED user's id as the audit actor. After the fix:
// actor = admin (from JWT context), resource = target (URL path).

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	adminapp "marketplace-backend/internal/app/admin"
	"marketplace-backend/internal/domain/audit"
	"marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/handler/middleware"
	"marketplace-backend/internal/port/repository"
)

// auditAuctionUserRepo lets us return a non-nil user from GetByID and
// no-op every other call. Other UserRepository methods are auto-
// satisfied by the embedded interface (panic if called — tests will
// catch any unexpected path).
type auditAuctionUserRepo struct {
	repository.UserRepository
	user *user.User
}

func (r *auditAuctionUserRepo) GetByID(_ context.Context, _ uuid.UUID) (*user.User, error) {
	cp := *r.user
	return &cp, nil
}
func (r *auditAuctionUserRepo) Update(_ context.Context, _ *user.User) error { return nil }
func (r *auditAuctionUserRepo) BumpSessionVersion(_ context.Context, _ uuid.UUID) (int, error) {
	return 1, nil
}

// recordingAuditRepo captures every audit row written so the test can
// assert UserID (actor) and ResourceID (target) are correct. Read-side
// methods are not used by SuspendUser / BanUser.
type recordingAuditRepo struct {
	repository.AuditRepository
	mu      sync.Mutex
	entries []*audit.Entry
}

func (r *recordingAuditRepo) Log(_ context.Context, e *audit.Entry) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.entries = append(r.entries, e)
	return nil
}

// newAdminHandlerForAuditTest wires a real adminapp.Service against the
// minimum mocks needed for SuspendUser / BanUser to run end-to-end.
// Returns the audit recorder so the test can read what was logged.
func newAdminHandlerForAuditTest(t *testing.T, target *user.User) (*AdminHandler, *recordingAuditRepo) {
	t.Helper()
	users := &auditAuctionUserRepo{user: target}
	audits := &recordingAuditRepo{}
	svc := adminapp.NewService(adminapp.ServiceDeps{
		Users: users,
		Audit: audits,
	})
	return NewAdminHandler(svc), audits
}

// adminAuthRequest wires the admin's user id into the JWT context the
// way middleware.Auth would in production. Chi route param `id` is
// the target.
func adminAuthRequest(t *testing.T, method, target string, adminID, targetID uuid.UUID, body []byte) *http.Request {
	t.Helper()
	var reqBody *bytes.Reader
	if body != nil {
		reqBody = bytes.NewReader(body)
	} else {
		reqBody = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, target, reqBody)
	ctx := req.Context()
	ctx = context.WithValue(ctx, middleware.ContextKeyUserID, adminID)
	ctx = context.WithValue(ctx, middleware.ContextKeyIsAdmin, true)

	// Inject {id} chi route param so parseAdminUserID works.
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", targetID.String())
	ctx = context.WithValue(ctx, chi.RouteCtxKey, rctx)

	return req.WithContext(ctx)
}

// TestSuspendUserHandler_AuditActorIsAdmin_ResourceIsTarget — pin the
// BUG-NEW-09 fix on the suspend path. Pre-fix audit row had user_id =
// suspended-user-id; post-fix it is the admin.
func TestSuspendUserHandler_AuditActorIsAdmin_ResourceIsTarget(t *testing.T) {
	adminID := uuid.New()
	targetID := uuid.New()

	target := &user.User{ID: targetID, Email: "victim@example.com", Role: user.RoleProvider}
	h, audits := newAdminHandlerForAuditTest(t, target)

	body, err := json.Marshal(map[string]any{"reason": "harassment"})
	require.NoError(t, err)

	req := adminAuthRequest(t, http.MethodPost, "/api/v1/admin/users/"+targetID.String()+"/suspend",
		adminID, targetID, body)
	rec := httptest.NewRecorder()

	h.SuspendUser(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, "suspend must succeed")

	require.Len(t, audits.entries, 1, "exactly one audit row")
	entry := audits.entries[0]
	assert.Equal(t, audit.ActionAdminUserSuspend, entry.Action)
	require.NotNil(t, entry.UserID, "audit actor must be set")
	assert.Equal(t, adminID, *entry.UserID,
		"BUG-NEW-09: audit actor MUST be the admin user_id, not the suspended user")
	require.NotNil(t, entry.ResourceID)
	assert.Equal(t, targetID, *entry.ResourceID,
		"BUG-NEW-09: ResourceID MUST be the suspended user")
	assert.Equal(t, "harassment", entry.Metadata["reason"])
}

// TestBanUserHandler_AuditActorIsAdmin_ResourceIsTarget — same coverage
// for the ban path. Twin of the suspend test above.
func TestBanUserHandler_AuditActorIsAdmin_ResourceIsTarget(t *testing.T) {
	adminID := uuid.New()
	targetID := uuid.New()

	target := &user.User{ID: targetID, Email: "fraudster@example.com", Role: user.RoleProvider}
	h, audits := newAdminHandlerForAuditTest(t, target)

	body, err := json.Marshal(map[string]any{"reason": "fraud"})
	require.NoError(t, err)

	req := adminAuthRequest(t, http.MethodPost, "/api/v1/admin/users/"+targetID.String()+"/ban",
		adminID, targetID, body)
	rec := httptest.NewRecorder()

	h.BanUser(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, "ban must succeed")

	require.Len(t, audits.entries, 1)
	entry := audits.entries[0]
	assert.Equal(t, audit.ActionAdminUserBan, entry.Action)
	require.NotNil(t, entry.UserID)
	assert.Equal(t, adminID, *entry.UserID,
		"BUG-NEW-09: audit actor MUST be the admin, not the banned user")
	require.NotNil(t, entry.ResourceID)
	assert.Equal(t, targetID, *entry.ResourceID,
		"BUG-NEW-09: ResourceID MUST be the banned user")
	assert.Equal(t, "fraud", entry.Metadata["reason"])
}

// TestSuspendUserHandler_NoAdminContext_ActorIsNil regression — when
// the auth middleware is absent (system-script invocation, mis-wired
// route), the audit row records actor=nil ("system") rather than
// silently logging the suspended user's id as actor (which is what
// the pre-fix code did and is the worst possible accountability
// failure).
func TestSuspendUserHandler_NoAdminContext_ActorIsNil(t *testing.T) {
	targetID := uuid.New()

	target := &user.User{ID: targetID, Email: "victim@example.com", Role: user.RoleProvider}
	h, audits := newAdminHandlerForAuditTest(t, target)

	body, err := json.Marshal(map[string]any{"reason": "test"})
	require.NoError(t, err)

	// Build request WITHOUT setting middleware.ContextKeyUserID — the
	// handler's middleware.GetUserID returns uuid.Nil and the service
	// records actor=nil rather than misattributing the action.
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users/"+targetID.String()+"/suspend",
		bytes.NewReader(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", targetID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rec := httptest.NewRecorder()
	h.SuspendUser(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	require.Len(t, audits.entries, 1)
	entry := audits.entries[0]
	assert.Nil(t, entry.UserID,
		"BUG-NEW-09: missing admin context → actor=nil (system), NOT misattribution to the target")
	require.NotNil(t, entry.ResourceID)
	assert.Equal(t, targetID, *entry.ResourceID,
		"target user is still recorded as the resource")
}
