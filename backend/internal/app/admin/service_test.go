package admin

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/domain/audit"
	"marketplace-backend/internal/domain/user"
)

// newTestService builds a Service with the minimum mocks needed for
// the SEC-05 / SEC-13 admin tests: user repo, audit repo, session
// service, and broadcaster. Other dependencies stay nil because the
// suspend/ban/unban paths don't touch them.
func newTestService() (
	*Service,
	*mockUserRepo,
	*mockAuditRepo,
	*mockSessionService,
	*mockBroadcaster,
) {
	users := &mockUserRepo{}
	audits := &mockAuditRepo{}
	sessions := &mockSessionService{}
	broadcaster := &mockBroadcaster{}

	svc := NewService(ServiceDeps{
		Users:       users,
		Audit:       audits,
		SessionSvc:  sessions,
		Broadcaster: broadcaster,
	})
	return svc, users, audits, sessions, broadcaster
}

// makeUser returns a fresh user with the given id, ready for the
// admin service tests. Centralised so each test isn't 8 lines of
// boilerplate.
func makeUser(id uuid.UUID) *user.User {
	return &user.User{
		ID:    id,
		Email: "victim@example.com",
		Role:  user.RoleProvider,
	}
}

// =====================================================================
// SEC-05: Suspend/Ban must bump session_version + delete sessions
// =====================================================================

func TestAdminService_SuspendUser_BumpsSessionVersionAndPurgesSessions(t *testing.T) {
	svc, users, audits, sessions, broadcaster := newTestService()
	adminID := uuid.New()
	uid := uuid.New()
	users.getByIDFn = func(_ context.Context, _ uuid.UUID) (*user.User, error) {
		return makeUser(uid), nil
	}

	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	err := svc.SuspendUser(context.Background(), adminID, uid, "harassment", &expiresAt)
	require.NoError(t, err)

	// SEC-05 invariants
	bumpCalls := users.snapshotBumpCalls()
	require.Len(t, bumpCalls, 1, "expected exactly one BumpSessionVersion")
	assert.Equal(t, uid, bumpCalls[0])

	deleteCalls := sessions.snapshotDeleteCalls()
	require.Len(t, deleteCalls, 1, "expected exactly one DeleteByUserID")
	assert.Equal(t, uid, deleteCalls[0])

	assert.Len(t, broadcaster.suspensionCalls, 1, "WS broadcast must fire")

	// SEC-13 + BUG-NEW-09 invariants
	entries := audits.snapshot()
	require.Len(t, entries, 1)
	assert.Equal(t, audit.ActionAdminUserSuspend, entries[0].Action)
	require.NotNil(t, entries[0].UserID)
	assert.Equal(t, adminID, *entries[0].UserID,
		"BUG-NEW-09: audit actor MUST be the admin, not the suspended user")
	require.NotNil(t, entries[0].ResourceID)
	assert.Equal(t, uid, *entries[0].ResourceID,
		"BUG-NEW-09: ResourceID MUST be the suspended user")
	assert.Equal(t, "harassment", entries[0].Metadata["reason"])
}

func TestAdminService_BanUser_BumpsSessionVersionAndPurgesSessions(t *testing.T) {
	svc, users, audits, sessions, _ := newTestService()
	adminID := uuid.New()
	uid := uuid.New()
	users.getByIDFn = func(_ context.Context, _ uuid.UUID) (*user.User, error) {
		return makeUser(uid), nil
	}

	err := svc.BanUser(context.Background(), adminID, uid, "fraud")
	require.NoError(t, err)

	bumpCalls := users.snapshotBumpCalls()
	require.Len(t, bumpCalls, 1, "Ban must bump session_version (SEC-05)")
	assert.Equal(t, uid, bumpCalls[0])

	deleteCalls := sessions.snapshotDeleteCalls()
	require.Len(t, deleteCalls, 1)
	assert.Equal(t, uid, deleteCalls[0])

	entries := audits.snapshot()
	require.Len(t, entries, 1)
	assert.Equal(t, audit.ActionAdminUserBan, entries[0].Action)
	require.NotNil(t, entries[0].UserID)
	assert.Equal(t, adminID, *entries[0].UserID,
		"BUG-NEW-09: audit actor MUST be the admin, not the banned user")
	require.NotNil(t, entries[0].ResourceID)
	assert.Equal(t, uid, *entries[0].ResourceID)
	assert.Equal(t, "fraud", entries[0].Metadata["reason"])
}

func TestAdminService_UnsuspendUser_EmitsAuditOnly(t *testing.T) {
	svc, users, audits, sessions, _ := newTestService()
	adminID := uuid.New()
	uid := uuid.New()
	users.getByIDFn = func(_ context.Context, _ uuid.UUID) (*user.User, error) {
		return makeUser(uid), nil
	}

	err := svc.UnsuspendUser(context.Background(), adminID, uid)
	require.NoError(t, err)

	// Unsuspend doesn't bump or purge — the user is being restored,
	// they need their session to keep working. Only the audit fires.
	assert.Empty(t, users.snapshotBumpCalls(),
		"unsuspend must NOT bump session_version")
	assert.Empty(t, sessions.snapshotDeleteCalls(),
		"unsuspend must NOT purge sessions")

	entries := audits.snapshot()
	require.Len(t, entries, 1)
	assert.Equal(t, audit.ActionAdminUserUnsuspend, entries[0].Action)
	require.NotNil(t, entries[0].UserID)
	assert.Equal(t, adminID, *entries[0].UserID,
		"BUG-NEW-09: actor=admin")
	require.NotNil(t, entries[0].ResourceID)
	assert.Equal(t, uid, *entries[0].ResourceID,
		"BUG-NEW-09: resource=target user")
}

func TestAdminService_UnbanUser_EmitsAuditOnly(t *testing.T) {
	svc, users, audits, sessions, _ := newTestService()
	adminID := uuid.New()
	uid := uuid.New()
	users.getByIDFn = func(_ context.Context, _ uuid.UUID) (*user.User, error) {
		return makeUser(uid), nil
	}

	err := svc.UnbanUser(context.Background(), adminID, uid)
	require.NoError(t, err)

	// Same as unsuspend: don't disturb existing sessions on restore.
	assert.Empty(t, users.snapshotBumpCalls())
	assert.Empty(t, sessions.snapshotDeleteCalls())

	entries := audits.snapshot()
	require.Len(t, entries, 1)
	assert.Equal(t, audit.ActionAdminUserUnban, entries[0].Action)
	require.NotNil(t, entries[0].UserID)
	assert.Equal(t, adminID, *entries[0].UserID, "BUG-NEW-09: actor=admin")
	require.NotNil(t, entries[0].ResourceID)
	assert.Equal(t, uid, *entries[0].ResourceID, "BUG-NEW-09: resource=target")
}

// TestAdminService_SuspendUser_GetByIDFailureDoesNotEmitAudit guards
// against an audit row being written for an action that didn't actually
// happen. If the user lookup fails, the suspend exits early before any
// state change — the audit row would be misleading.
func TestAdminService_SuspendUser_GetByIDFailureDoesNotEmitAudit(t *testing.T) {
	svc, users, audits, _, _ := newTestService()
	users.getByIDFn = func(_ context.Context, _ uuid.UUID) (*user.User, error) {
		return nil, user.ErrUserNotFound
	}

	err := svc.SuspendUser(context.Background(), uuid.New(), uuid.New(), "any", nil)
	require.Error(t, err)
	assert.Empty(t, audits.snapshot(),
		"failed lookups must not produce audit rows")
}

// TestAdminService_SuspendUser_NilExpiresAtSafe ensures the audit row
// builder copes with an open-ended suspension (expires_at == nil).
func TestAdminService_SuspendUser_NilExpiresAtSafe(t *testing.T) {
	svc, users, audits, _, _ := newTestService()
	uid := uuid.New()
	users.getByIDFn = func(_ context.Context, _ uuid.UUID) (*user.User, error) {
		return makeUser(uid), nil
	}

	err := svc.SuspendUser(context.Background(), uuid.New(), uid, "indefinite", nil)
	require.NoError(t, err)

	entries := audits.snapshot()
	require.Len(t, entries, 1)
	assert.Equal(t, "", entries[0].Metadata["expires_at"],
		"nil expiry must serialise to empty string")
}

// TestAdminService_SuspendBumpFailureBestEffort verifies the SEC-05
// best-effort rule: a BumpSessionVersion failure logs but doesn't
// abort the suspension. The user is still saved as suspended, and the
// audit row still fires, even if Redis (sessions) is also degraded.
func TestAdminService_SuspendBumpFailureBestEffort(t *testing.T) {
	svc, users, audits, _, _ := newTestService()
	uid := uuid.New()
	users.getByIDFn = func(_ context.Context, _ uuid.UUID) (*user.User, error) {
		return makeUser(uid), nil
	}
	users.bumpErr = assertErr("postgres restarting")

	err := svc.SuspendUser(context.Background(), uuid.New(), uid, "x", nil)
	assert.NoError(t, err, "suspend must succeed even when bump fails")
	assert.Len(t, audits.snapshot(), 1)
}

// assertErr is a tiny helper for the test-only error returned by
// mockUserRepo.bumpErr; spelled out instead of using fmt.Errorf so the
// linter doesn't complain about an unused import in the mocks file.
type assertErr string

func (e assertErr) Error() string { return string(e) }
