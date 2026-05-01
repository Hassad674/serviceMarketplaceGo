package postgres_test

// Tests for the RLS tenant-context wrap on AuditRepository.
//
// BUG-NEW-04 path 2/8 — audit_logs. The audit_logs table is RLS-
// protected by migration 125 with the policy
//
//   USING (user_id = current_setting('app.current_user_id', true)::uuid)
//
// Migration 129 already added WITH CHECK (true) so INSERTs are
// unconditional even without context (BUG-NEW-07). This commit adds
// the symmetric wrap on the read paths AND on Log so:
//
//  1. The Log path uses RunInTxWithTenant with entry.UserID — keeps
//     parity with the rest of the RLS migration and locks in
//     defense-in-depth in case the WITH CHECK (true) is ever
//     tightened in a future migration.
//  2. ListByUser fires inside RunInTxWithTenant(uuid.Nil, userID, ...)
//     so the rows actually return when the application role is
//     NOSUPERUSER NOBYPASSRLS.
//
// Note: ListByResource is NOT currently called in production code
// (only test mocks reference it) — but we still wrap it on the
// tenant-aware path because the contract advertises read access
// and an admin tool tomorrow could surface those rows. The wrap
// uses uuid.Nil (caller has no specific user context — they want
// to see ALL actors who touched the resource) so it returns the
// empty set under non-superuser, which is a safe failure mode.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/adapter/postgres"
	"marketplace-backend/internal/domain/audit"
)

// ---------------------------------------------------------------------------
// Unit tests (no DB)
// ---------------------------------------------------------------------------

func TestAuditRepository_WithTxRunner_ReturnsSameRepo(t *testing.T) {
	repo := postgres.NewAuditRepository(nil)
	runner := postgres.NewTxRunner(nil)
	got := repo.WithTxRunner(runner)
	assert.Same(t, repo, got)
}

func TestAuditRepository_WithTxRunner_NilRunner_NoPanic(t *testing.T) {
	repo := postgres.NewAuditRepository(nil)
	got := repo.WithTxRunner(nil)
	assert.NotNil(t, got)
}

// TestAuditRepository_Log_NilEntry returns an explicit error rather
// than panicking — the service layer logs and continues without
// breaking the business flow.
func TestAuditRepository_Log_NilEntry(t *testing.T) {
	repo := postgres.NewAuditRepository(nil)
	err := repo.Log(context.Background(), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nil entry")
}

// ---------------------------------------------------------------------------
// Integration tests — gated on MARKETPLACE_TEST_DATABASE_URL
// ---------------------------------------------------------------------------

// TestAuditRepository_Log_UnderRLS_Succeeds asserts Log goes through
// even under the non-superuser role + non-RLS-bypass user. WITH CHECK
// (true) from migration 129 already covers this for unset context, but
// the wrap provides the same guarantee under a tighter future policy.
func TestAuditRepository_Log_UnderRLS_Succeeds(t *testing.T) {
	db := testDB(t)
	ensureRLSTestRole(t, db)
	ctx := context.Background()

	actorID := insertTestUser(t, db)
	repo := postgres.NewAuditRepository(db).WithTxRunner(postgres.NewTxRunner(db))

	entry := &audit.Entry{
		ID:        uuid.New(),
		UserID:    &actorID,
		Action:    "login_success",
		CreatedAt: time.Now(),
		Metadata:  map[string]any{"src": "test"},
	}
	require.NoError(t, repo.Log(ctx, entry), "Log under tenant context must persist the row")

	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM audit_logs WHERE id = $1`, entry.ID) })
}

// TestAuditRepository_Log_NilUserID_Succeeds covers the system-actor
// path: an audit entry with no user (background worker, login failure
// before tenant context). entry.UserID == nil → repo passes uuid.Nil
// to the runner → SetTenantContext skips the user setter → WITH CHECK
// (true) lets the row through.
func TestAuditRepository_Log_NilUserID_Succeeds(t *testing.T) {
	db := testDB(t)
	ensureRLSTestRole(t, db)
	ctx := context.Background()

	repo := postgres.NewAuditRepository(db).WithTxRunner(postgres.NewTxRunner(db))

	entry := &audit.Entry{
		ID:        uuid.New(),
		UserID:    nil, // system actor
		Action:    "system_cleanup",
		CreatedAt: time.Now(),
		Metadata:  map[string]any{"job": "scheduler"},
	}
	require.NoError(t, repo.Log(ctx, entry),
		"Log with nil UserID (system actor) must persist via WITH CHECK (true)")

	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM audit_logs WHERE id = $1`, entry.ID) })
}

// TestAuditRepository_ListByUser_UnderRLS asserts ListByUser sees the
// rows when wrapped in tenant context. Without the wrap the rls test
// role would see zero rows because the policy filters by user_id.
func TestAuditRepository_ListByUser_UnderRLS(t *testing.T) {
	db := testDB(t)
	ensureRLSTestRole(t, db)
	ctx := context.Background()

	actorID := insertTestUser(t, db)
	repo := postgres.NewAuditRepository(db).WithTxRunner(postgres.NewTxRunner(db))

	for i := 0; i < 3; i++ {
		entry := &audit.Entry{
			ID:        uuid.New(),
			UserID:    &actorID,
			Action:    audit.Action("test.action"),
			CreatedAt: time.Now().Add(-time.Duration(i) * time.Minute),
			Metadata:  map[string]any{"i": i},
		}
		require.NoError(t, repo.Log(ctx, entry))
		t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM audit_logs WHERE id = $1`, entry.ID) })
	}

	entries, _, err := repo.ListByUser(ctx, actorID, "", 10)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(entries), 3,
		"ListByUser under tenant context must return the actor's rows")
}

// TestAuditRepository_Legacy_NoTxRunner_StillWorks confirms backwards
// compat: building the repo with only a *sql.DB keeps the legacy non-
// transaction path so existing unit tests keep passing.
func TestAuditRepository_Legacy_NoTxRunner_StillWorks(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	actorID := insertTestUser(t, db)
	repo := postgres.NewAuditRepository(db) // no txRunner — legacy path

	entry := &audit.Entry{
		ID:        uuid.New(),
		UserID:    &actorID,
		Action:    "legacy_test",
		CreatedAt: time.Now(),
		Metadata:  map[string]any{"src": "legacy"},
	}
	require.NoError(t, repo.Log(ctx, entry),
		"legacy path must keep working for unit tests with only *sql.DB")
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM audit_logs WHERE id = $1`, entry.ID) })
}
