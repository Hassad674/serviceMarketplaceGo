package postgres_test

// Integration tests for BUG-NEW-07 — audit_logs RLS policy now has an
// explicit WITH CHECK (true). Without it, INSERTs from the non-
// superuser application role were rejected when app.current_user_id
// was not set in the transaction (background workers, system actor
// audit entries, login-failure logs before tenant context exists).
//
// Run:
//
//	MARKETPLACE_TEST_DATABASE_URL=postgres://postgres:postgres@localhost:5435/marketplace_go_bugs_high?sslmode=disable \
//	  go test ./internal/adapter/postgres/ -run TestRLS_AuditLogs_InsertWithoutTenantContext -count=1 -race

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/adapter/postgres"
)

// TestRLS_AuditLogs_InsertWithoutTenantContext_SucceedsAfterFix is the
// targeted regression for BUG-NEW-07. As the non-superuser application
// role with NO app.current_user_id set, INSERT into audit_logs MUST
// succeed — the fix added WITH CHECK (true) to the policy so writes
// are unconditional while reads remain filtered.
//
// Pre-fix: INSERT would fail because USING was used as the WITH CHECK
// fallback, evaluating user_id = NULL = NULL → reject.
func TestRLS_AuditLogs_InsertWithoutTenantContext_SucceedsAfterFix(t *testing.T) {
	db := testDB(t)
	ensureRLSTestRole(t, db)
	ctx := context.Background()

	// Create a user we can reference in the audit_log row. Bypassing
	// RLS works here because we're still the postgres superuser at
	// connection time.
	actorID := insertTestUser(t, db)

	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()

	// Drop superuser by SET ROLE to the non-bypassrls role. NOT setting
	// app.current_user_id — this is the system-actor / background-job
	// path the bug applied to.
	setRLSRole(t, ctx, tx)

	// Assert the WITH CHECK fix is in place: insert must succeed even
	// though current_setting('app.current_user_id', true) is NULL.
	auditID := uuid.New()
	_, err = tx.ExecContext(ctx, `
		INSERT INTO audit_logs (id, user_id, action) VALUES ($1, $2, 'system_action')
	`, auditID, actorID)
	require.NoError(t, err,
		"BUG-NEW-07: INSERT into audit_logs MUST succeed even when app.current_user_id is unset")
}

// TestRLS_AuditLogs_InsertWithoutTenantContext_RejectedBeforeFix is a
// negative-control documenting WHY the fix was needed. We simulate the
// PRE-fix behaviour by recreating a USING-only policy on a temporary
// table fixture — the simulation mirrors exactly what migration 125
// installed before migration 129 ran. Asserts the simulated INSERT
// would have been rejected, so the fix's purpose is grounded in
// reproduced behaviour, not an abstract claim.
func TestRLS_AuditLogs_InsertWithoutTenantContext_RejectedBeforeFix(t *testing.T) {
	db := testDB(t)
	ensureRLSTestRole(t, db)
	ctx := context.Background()

	// Create a temp table mirroring the audit_logs schema so we can
	// install a USING-only policy and verify the rejection without
	// touching the production table.
	tableName := "audit_logs_buggy_repro_" + uuid.NewString()[:8]
	_, err := db.ExecContext(ctx, `
		CREATE TEMP TABLE `+tableName+` (
			id UUID PRIMARY KEY,
			user_id UUID NOT NULL,
			action TEXT NOT NULL
		)
	`)
	require.NoError(t, err)

	// Enable RLS + FORCE (mirrors migration 125 lines for audit_logs).
	_, err = db.ExecContext(ctx, `ALTER TABLE `+tableName+` ENABLE ROW LEVEL SECURITY`)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `ALTER TABLE `+tableName+` FORCE ROW LEVEL SECURITY`)
	require.NoError(t, err)

	// Install the BUGGY USING-only policy. WITH CHECK omitted on
	// purpose — that's the bug being reproduced.
	policyName := "buggy_isolation_" + uuid.NewString()[:8]
	_, err = db.ExecContext(ctx, `CREATE POLICY `+policyName+` ON `+tableName+`
		USING (user_id = current_setting('app.current_user_id', true)::uuid)`)
	require.NoError(t, err)

	// Grant insert to the rls test role so the only thing rejecting
	// the INSERT is RLS, not grants.
	_, err = db.ExecContext(ctx, `GRANT SELECT, INSERT, UPDATE, DELETE ON TABLE `+tableName+` TO `+rlsTestRole)
	require.NoError(t, err)

	// Cleanup the temp table — temp tables are session-scoped but be
	// explicit so quick re-runs don't pile up policies.
	t.Cleanup(func() {
		ctx2, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_, _ = db.ExecContext(ctx2, `DROP TABLE IF EXISTS `+tableName)
	})

	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()

	setRLSRole(t, ctx, tx)

	// Insert without app.current_user_id set — the buggy policy MUST
	// reject this. Postgres surfaces RLS rejection as a "violates row-
	// level security policy" error.
	actorID := uuid.New()
	auditID := uuid.New()
	_, err = tx.ExecContext(ctx, `INSERT INTO `+tableName+` (id, user_id, action) VALUES ($1, $2, 'system_action')`,
		auditID, actorID)
	require.Error(t, err, "buggy USING-only policy MUST reject the INSERT — proves the fix is needed")
	assert.ErrorContains(t, err, "row-level security",
		"the rejection error MUST come from RLS (not from a typo / FK)")
}

// TestRLS_AuditLogs_InsertWithUserContext_StillWorks is the same-tenant
// happy path — when app.current_user_id IS set and matches the inserted
// user_id, the row is written. Confirms the fix didn't break the
// authenticated-user audit path.
func TestRLS_AuditLogs_InsertWithUserContext_StillWorks(t *testing.T) {
	db := testDB(t)
	ensureRLSTestRole(t, db)
	ctx := context.Background()

	actorID := insertTestUser(t, db)

	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()

	setRLSRole(t, ctx, tx)
	require.NoError(t, postgres.SetCurrentUser(ctx, tx, actorID))

	auditID := uuid.New()
	_, err = tx.ExecContext(ctx, `
		INSERT INTO audit_logs (id, user_id, action) VALUES ($1, $2, 'login_success')
	`, auditID, actorID)
	require.NoError(t, err, "authenticated audit insert MUST still work after the fix")

	// SELECT the row back — same user_id, same context → visible.
	var got uuid.UUID
	err = tx.QueryRowContext(ctx, `SELECT id FROM audit_logs WHERE id = $1`, auditID).Scan(&got)
	require.NoError(t, err)
	assert.Equal(t, auditID, got)
}

// TestRLS_AuditLogs_SelectStillFilteredByUser is the regression: the
// fix relaxed WITH CHECK but USING (the read-side filter) is unchanged.
// A user MUST still only see their own audit rows.
func TestRLS_AuditLogs_SelectStillFilteredByUser(t *testing.T) {
	db := testDB(t)
	ensureRLSTestRole(t, db)
	ctx := context.Background()

	userA := insertTestUser(t, db)
	userB := insertTestUser(t, db)

	// Insert as superuser (bypasses RLS) — one row per user.
	auditA := uuid.New()
	auditB := uuid.New()
	_, err := db.ExecContext(ctx, `INSERT INTO audit_logs (id, user_id, action) VALUES ($1, $2, 'login_success')`, auditA, userA)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `INSERT INTO audit_logs (id, user_id, action) VALUES ($1, $2, 'login_success')`, auditB, userB)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM audit_logs WHERE id IN ($1, $2)`, auditA, auditB)
	})

	// Read as userA — only auditA must be visible.
	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()

	setRLSRole(t, ctx, tx)
	require.NoError(t, postgres.SetCurrentUser(ctx, tx, userA))

	visibleA, visibleB := 0, 0
	err = tx.QueryRowContext(ctx, `SELECT count(*) FROM audit_logs WHERE id = $1`, auditA).Scan(&visibleA)
	require.NoError(t, err)
	err = tx.QueryRowContext(ctx, `SELECT count(*) FROM audit_logs WHERE id = $1`, auditB).Scan(&visibleB)
	require.NoError(t, err)

	assert.Equal(t, 1, visibleA, "userA must see their own audit row")
	assert.Equal(t, 0, visibleB, "userA MUST NOT see userB's audit row — read filter unchanged by the fix")
}
