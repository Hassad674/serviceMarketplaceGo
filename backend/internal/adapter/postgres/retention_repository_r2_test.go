package postgres_test

// Integration tests for the B.2 cold-tier sweep
// (StrategyArchiveToR2). Same shape as
// retention_repository_test.go: gated behind
// MARKETPLACE_TEST_DATABASE_URL via the shared testDB helper.
//
// Migration 149 (audit_logs_archive.r2_key) must be applied before
// the test exercises the upload phase — the tests detect a missing
// column and skip cleanly rather than failing.

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/adapter/postgres"
	"marketplace-backend/internal/domain/retention"
	"marketplace-backend/internal/port/service"
)

// fakeArchiveWriter records every call so the test can assert keys
// and rows. It is not safe to use beyond the test that owns it; the
// mutex is just there because the sweep happens to run on a goroutine
// in the broader scheduler tests.
type fakeArchiveWriter struct {
	mu     sync.Mutex
	calls  []fakeArchiveCall
	failOn map[string]error
}

type fakeArchiveCall struct {
	Key  string
	Rows []service.AuditArchiveRow
}

func (f *fakeArchiveWriter) WriteJSONL(_ context.Context, key string, rows []service.AuditArchiveRow) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err, ok := f.failOn[key]; ok && err != nil {
		return err
	}
	// Copy to defend against post-call mutation.
	cp := make([]service.AuditArchiveRow, len(rows))
	copy(cp, rows)
	f.calls = append(f.calls, fakeArchiveCall{Key: key, Rows: cp})
	return nil
}

func (f *fakeArchiveWriter) Calls() []fakeArchiveCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]fakeArchiveCall, len(f.calls))
	copy(out, f.calls)
	return out
}

// hasArchiveR2Column returns true when migration 149 has been applied
// to the test database. Tests early-skip when the column is missing
// so a developer running the suite against a pre-149 DB does not see
// a confusing failure.
func hasArchiveR2Column(t *testing.T, db *sql.DB) bool {
	t.Helper()
	var ok bool
	err := db.QueryRow(`
        SELECT EXISTS (
          SELECT 1 FROM information_schema.columns
           WHERE table_name = 'audit_logs_archive'
             AND column_name = 'r2_key'
        )`).Scan(&ok)
	require.NoError(t, err)
	return ok
}

// seedArchiveRows inserts n rows into audit_logs_archive with
// archived_at = supplied timestamp. Returns the inserted ids so the
// test cleanup can drop them.
func seedArchiveRows(t *testing.T, db *sql.DB, n int, archivedAt time.Time, action string) []uuid.UUID {
	t.Helper()
	ctx := context.Background()
	ids := make([]uuid.UUID, 0, n)
	for i := 0; i < n; i++ {
		id := uuid.New()
		_, err := db.ExecContext(ctx, `
            INSERT INTO audit_logs_archive
                (id, user_id, action, resource_type, resource_id, metadata, ip_address, created_at, archived_at)
            VALUES ($1, NULL, $2, 'b2_test', NULL, $3::jsonb, NULL, $4, $5)`,
			id, action, fmt.Sprintf(`{"i":%d}`, i),
			archivedAt.Add(-time.Hour), archivedAt)
		require.NoError(t, err, "seed row %d", i)
		ids = append(ids, id)
	}
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM audit_logs_archive WHERE id = ANY($1)`, pgUUIDArray(ids))
	})
	return ids
}

// TestRetentionRepository_R2_RequiresWriter asserts the repository
// cannot silently no-op when the archive_to_r2 strategy is invoked
// without a writer wired in.
func TestRetentionRepository_R2_RequiresWriter(t *testing.T) {
	db := retentionTestDB(t)
	repo := postgres.NewRetentionRepository(db) // no .WithAuditArchiveWriter

	if !hasArchiveR2Column(t, db) {
		t.Skip("migration 149 not applied — skip B.2 tests")
	}

	policy := retention.Policy{
		Name:      "audit_r2_test",
		Table:     "audit_logs_archive",
		AgeColumn: "archived_at",
		MaxAge:    24 * time.Hour,
		Strategy:  retention.StrategyArchiveToR2,
		BatchSize: 100,
	}
	n, err := repo.Sweep(context.Background(), policy)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "WithAuditArchiveWriter")
	assert.Equal(t, 0, n)
}

// TestRetentionRepository_R2_RejectsForOtherTable verifies the
// strategy is hard-scoped to audit_logs_archive even when the
// allowlist would otherwise admit a similarly-named identifier.
func TestRetentionRepository_R2_RejectsForOtherTable(t *testing.T) {
	db := retentionTestDB(t)
	if !hasArchiveR2Column(t, db) {
		t.Skip("migration 149 not applied")
	}
	repo := postgres.NewRetentionRepository(db).WithAuditArchiveWriter(&fakeArchiveWriter{})

	policy := retention.Policy{
		Name:      "evil",
		Table:     "audit_logs", // not the cold table
		AgeColumn: "created_at",
		MaxAge:    24 * time.Hour,
		Strategy:  retention.StrategyArchiveToR2,
		BatchSize: 10,
	}
	_, err := repo.Sweep(context.Background(), policy)
	require.Error(t, err)
}

// TestRetentionRepository_R2_TwoPhaseEndToEnd covers the headline
// contract: tick 1 uploads a batch + stamps r2_key, tick 2
// hard-deletes the same rows. Tick 3 is a no-op.
func TestRetentionRepository_R2_TwoPhaseEndToEnd(t *testing.T) {
	db := retentionTestDB(t)
	if !hasArchiveR2Column(t, db) {
		t.Skip("migration 149 not applied — skip B.2 tests")
	}

	writer := &fakeArchiveWriter{}
	repo := postgres.NewRetentionRepository(db).WithAuditArchiveWriter(writer)

	tag := fmt.Sprintf("b2-e2e-%s", uuid.New().String()[:8])
	old := seedArchiveRows(t, db, 7, time.Now().Add(-30*24*time.Hour), tag) // 30 days old
	young := seedArchiveRows(t, db, 3, time.Now(), tag+"-young")            // not eligible

	policy := retention.Policy{
		Name:      "audit_logs_archive_to_r2_test",
		Table:     "audit_logs_archive",
		AgeColumn: "archived_at",
		MaxAge:    7 * 24 * time.Hour, // 1 week cutoff for the test
		Strategy:  retention.StrategyArchiveToR2,
		BatchSize: 100,
	}

	ctx := context.Background()

	// Tick 1: upload phase. Writer receives the batch, rows now have r2_key.
	n, err := repo.Sweep(ctx, policy)
	require.NoError(t, err)
	assert.Equal(t, len(old), n, "expect all 7 old rows uploaded in one batch")

	calls := writer.Calls()
	require.Len(t, calls, 1)
	assert.NotEmpty(t, calls[0].Key)
	assert.Contains(t, calls[0].Key, "audit-cold/")
	assert.Contains(t, calls[0].Key, ".jsonl.gz")
	assert.Len(t, calls[0].Rows, len(old))

	// All old rows now have r2_key; young rows still NULL.
	var oldUploaded int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM audit_logs_archive WHERE id = ANY($1) AND r2_key IS NOT NULL`, pgUUIDArray(old)).Scan(&oldUploaded))
	assert.Equal(t, len(old), oldUploaded)

	var youngUploaded int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM audit_logs_archive WHERE id = ANY($1) AND r2_key IS NOT NULL`, pgUUIDArray(young)).Scan(&youngUploaded))
	assert.Equal(t, 0, youngUploaded, "young rows must remain in Postgres only")

	// Tick 2: delete phase. Rows that were uploaded are now gone.
	n, err = repo.Sweep(ctx, policy)
	require.NoError(t, err)
	assert.Equal(t, len(old), n, "expect the 7 uploaded rows hard-deleted")

	var oldLeft int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM audit_logs_archive WHERE id = ANY($1)`, pgUUIDArray(old)).Scan(&oldLeft))
	assert.Equal(t, 0, oldLeft)

	var youngLeft int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM audit_logs_archive WHERE id = ANY($1)`, pgUUIDArray(young)).Scan(&youngLeft))
	assert.Equal(t, len(young), youngLeft, "young rows untouched")

	// Tick 3: idempotent no-op.
	n, err = repo.Sweep(ctx, policy)
	require.NoError(t, err)
	assert.Equal(t, 0, n)

	// Writer was called exactly once across the three ticks.
	assert.Len(t, writer.Calls(), 1)
}

// TestRetentionRepository_R2_BatchingAcrossTicks asserts that
// uploading more rows than the batch size takes multiple ticks and
// each tick's key is unique (year/month bucket + UUID).
func TestRetentionRepository_R2_BatchingAcrossTicks(t *testing.T) {
	db := retentionTestDB(t)
	if !hasArchiveR2Column(t, db) {
		t.Skip("migration 149 not applied")
	}
	writer := &fakeArchiveWriter{}
	repo := postgres.NewRetentionRepository(db).WithAuditArchiveWriter(writer)

	tag := fmt.Sprintf("b2-batch-%s", uuid.New().String()[:8])
	seedArchiveRows(t, db, 12, time.Now().Add(-30*24*time.Hour), tag)

	policy := retention.Policy{
		Name:      "audit_logs_archive_to_r2_test",
		Table:     "audit_logs_archive",
		AgeColumn: "archived_at",
		MaxAge:    7 * 24 * time.Hour,
		Strategy:  retention.StrategyArchiveToR2,
		BatchSize: 5, // 12 rows / 5 = 3 upload ticks
	}

	ctx := context.Background()

	// Three upload ticks (5 + 5 + 2 = 12 rows uploaded).
	for i, want := range []int{5, 5, 2} {
		n, err := repo.Sweep(ctx, policy)
		require.NoError(t, err, "upload tick %d", i)
		assert.Equal(t, want, n, "upload tick %d count mismatch", i)
	}

	// Three keys, all distinct (UUID component).
	calls := writer.Calls()
	require.Len(t, calls, 3)
	keysSeen := map[string]bool{}
	for _, c := range calls {
		assert.False(t, keysSeen[c.Key], "duplicate key %q", c.Key)
		keysSeen[c.Key] = true
	}

	// Now delete-phase ticks drain the table.
	totalDeleted := 0
	for i := 0; i < 5 && totalDeleted < 12; i++ {
		n, err := repo.Sweep(ctx, policy)
		require.NoError(t, err)
		totalDeleted += n
	}
	assert.Equal(t, 12, totalDeleted)
}
