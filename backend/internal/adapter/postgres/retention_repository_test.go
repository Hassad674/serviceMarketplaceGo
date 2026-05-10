package postgres_test

// Integration tests for the retention sweep adapter. Gated behind
// MARKETPLACE_TEST_DATABASE_URL via the shared testDB helper — they
// auto-skip when the env var is unset so `go test ./...` stays green
// on a developer machine without a live Postgres.
//
// Each test seeds a small, isolated set of rows under a unique tag
// so concurrent test runs against the same database do not interfere
// (rows are filtered on a UUID-typed marker column where possible,
// or on a recent created_at window otherwise).

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/adapter/postgres"
	"marketplace-backend/internal/domain/retention"
)

// retentionTestDB is a thin wrapper around the global testDB helper —
// kept distinct so failures here are easy to grep apart from other
// integration suites that share the helper.
func retentionTestDB(t *testing.T) *sql.DB {
	t.Helper()
	if os.Getenv("MARKETPLACE_TEST_DATABASE_URL") == "" {
		t.Skip("MARKETPLACE_TEST_DATABASE_URL not set — skipping retention integration test")
	}
	return testDB(t)
}

// seedSearchQueries inserts n rows with the supplied created_at and
// returns the list of inserted ids so the test can clean up. We use
// search_queries because it is NOT RLS-protected (no ALTER TABLE
// ENABLE RLS in migration 125), nor does it carry FKs that would
// pull in an entire user/org graph just to make one row.
func seedSearchQueries(t *testing.T, db *sql.DB, n int, createdAt time.Time, tag string) []uuid.UUID {
	t.Helper()
	ctx := context.Background()
	ids := make([]uuid.UUID, 0, n)
	userID := uuid.New() // a placeholder — search_queries.user_id is nullable + ON DELETE SET NULL
	for i := 0; i < n; i++ {
		id := uuid.New()
		_, err := db.ExecContext(ctx, `
            INSERT INTO search_queries
                (id, user_id, session_id, query, persona, created_at)
            VALUES ($1, $2, $3, $4, 'all', $5)`,
			id, nil, fmt.Sprintf("%s-%s", tag, id), tag, createdAt)
		require.NoError(t, err, "seed search_queries[%d]", i)
		ids = append(ids, id)
		_ = userID
	}
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM search_queries WHERE id = ANY($1)`, pgUUIDArray(ids))
	})
	return ids
}

// pgUUIDArray formats a slice of UUIDs as the literal Postgres
// expects for `= ANY($1)` cleanup. database/sql + lib/pq has no
// generic pq.Array(uuid.UUID slice) so we marshal manually. The
// shape must be `{uuid1,uuid2,…}`.
func pgUUIDArray(ids []uuid.UUID) string {
	if len(ids) == 0 {
		return "{}"
	}
	out := "{"
	for i, id := range ids {
		if i > 0 {
			out += ","
		}
		out += id.String()
	}
	out += "}"
	return out
}

// TestRetentionRepository_DeleteSweep covers the headline contract:
// a delete policy removes exactly the eligible rows and is
// idempotent — a second sweep returns zero.
func TestRetentionRepository_DeleteSweep(t *testing.T) {
	db := retentionTestDB(t)
	repo := postgres.NewRetentionRepository(db)

	tag := fmt.Sprintf("ret-del-%s", uuid.New().String()[:8])
	old := seedSearchQueries(t, db, 12, time.Now().Add(-400*24*time.Hour), tag) // 400d old, way past 12mo
	young := seedSearchQueries(t, db, 8, time.Now(), tag+"-young")

	// Use a delete policy (test scaffold) to verify the generic delete
	// path. The production search_queries policy is anonymize, but
	// the underlying DELETE branch is what we are exercising here.
	policy := retention.Policy{
		Name:      "test_search_delete",
		Table:     "search_queries",
		AgeColumn: "created_at",
		MaxAge:    365 * 24 * time.Hour,
		Strategy:  retention.StrategyDelete,
		BatchSize: 5,
	}

	// First sweep: 5 rows deleted (batch size).
	n, err := repo.Sweep(context.Background(), policy)
	require.NoError(t, err)
	assert.Equal(t, 5, n)

	// Second sweep: 5 more.
	n, err = repo.Sweep(context.Background(), policy)
	require.NoError(t, err)
	assert.Equal(t, 5, n)

	// Third: only 2 left (12 total - 10 deleted).
	n, err = repo.Sweep(context.Background(), policy)
	require.NoError(t, err)
	assert.Equal(t, 2, n)

	// Fourth: idempotent zero.
	n, err = repo.Sweep(context.Background(), policy)
	require.NoError(t, err)
	assert.Equal(t, 0, n)

	// Young rows untouched.
	var youngLeft int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM search_queries WHERE id = ANY($1)`, pgUUIDArray(young)).Scan(&youngLeft))
	assert.Equal(t, 8, youngLeft)

	_ = old // ids returned for the cleanup func; rows already deleted by sweep.
}

// TestRetentionRepository_AnonymizeSweep covers the production
// search_queries policy: rows older than the cutoff have user_id and
// session_id zeroed; the row itself stays for analytics.
func TestRetentionRepository_AnonymizeSweep(t *testing.T) {
	db := retentionTestDB(t)
	repo := postgres.NewRetentionRepository(db)

	ctx := context.Background()
	tag := fmt.Sprintf("ret-anon-%s", uuid.New().String()[:8])

	// Need real user_id values to verify they get NULLed. Insert a
	// minimal user row first (cleanup nukes the user at the end).
	userID := uuid.New()
	_, err := db.ExecContext(ctx, `
        INSERT INTO users (id, email, hashed_password, first_name, last_name, display_name, role, account_type)
        VALUES ($1, $2, 'h', 'A', 'B', 'AB', 'enterprise', 'marketplace_owner')`,
		userID, fmt.Sprintf("%s@retention.test", tag))
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM users WHERE id = $1`, userID) })

	// Seed 4 old rows with user_id + session_id set, plus 2 young.
	oldIDs := make([]uuid.UUID, 0, 4)
	for i := 0; i < 4; i++ {
		id := uuid.New()
		_, err := db.ExecContext(ctx, `
            INSERT INTO search_queries (id, user_id, session_id, query, persona, created_at)
            VALUES ($1, $2, $3, $4, 'all', $5)`,
			id, userID, fmt.Sprintf("sess-%d", i), tag, time.Now().Add(-400*24*time.Hour))
		require.NoError(t, err)
		oldIDs = append(oldIDs, id)
	}
	youngID := uuid.New()
	_, err = db.ExecContext(ctx, `
        INSERT INTO search_queries (id, user_id, session_id, query, persona, created_at)
        VALUES ($1, $2, 'young-sess', $3, 'all', NOW())`,
		youngID, userID, tag+"-young")
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM search_queries WHERE id = ANY($1)`, pgUUIDArray(append(oldIDs, youngID)))
	})

	policy := retention.Policy{
		Name:             "test_search_anon",
		Table:            "search_queries",
		AgeColumn:        "created_at",
		MaxAge:           365 * 24 * time.Hour,
		Strategy:         retention.StrategyAnonymize,
		AnonymizeColumns: []string{"user_id", "session_id"},
		BatchSize:        100,
	}

	n, err := repo.Sweep(ctx, policy)
	require.NoError(t, err)
	assert.Equal(t, 4, n, "all 4 old rows anonymized")

	// Old rows: user_id + session_id NULL, query preserved.
	for _, id := range oldIDs {
		var u, s sql.NullString
		var q string
		err := db.QueryRow(`SELECT user_id::text, session_id, query FROM search_queries WHERE id = $1`, id).Scan(&u, &s, &q)
		require.NoError(t, err)
		assert.False(t, u.Valid, "user_id should be NULL for %s", id)
		assert.False(t, s.Valid, "session_id should be NULL for %s", id)
		assert.Equal(t, tag, q, "query text preserved")
	}

	// Young row untouched.
	var u sql.NullString
	require.NoError(t, db.QueryRow(`SELECT user_id::text FROM search_queries WHERE id = $1`, youngID).Scan(&u))
	assert.True(t, u.Valid, "young row's user_id should NOT be NULL")

	// Idempotent — second sweep does nothing because the IS NOT NULL
	// guard skips already-anonymized rows.
	n, err = repo.Sweep(ctx, policy)
	require.NoError(t, err)
	assert.Equal(t, 0, n)
}

// TestRetentionRepository_RejectsUnallowlistedTable verifies the
// security guardrail: a Policy referencing a table outside the
// adapter's allowlist must error before any SQL is built.
func TestRetentionRepository_RejectsUnallowlistedTable(t *testing.T) {
	db := retentionTestDB(t)
	repo := postgres.NewRetentionRepository(db)

	policy := retention.Policy{
		Name:      "evil",
		Table:     "users", // not in the allowlist
		AgeColumn: "created_at",
		MaxAge:    24 * time.Hour,
		Strategy:  retention.StrategyDelete,
	}
	n, err := repo.Sweep(context.Background(), policy)
	require.Error(t, err)
	assert.Equal(t, 0, n)
}

func TestRetentionRepository_RejectsUnallowlistedAgeColumn(t *testing.T) {
	db := retentionTestDB(t)
	repo := postgres.NewRetentionRepository(db)
	policy := retention.Policy{
		Name:      "evil-col",
		Table:     "messages",
		AgeColumn: "deleted_at", // not in allowlist for messages
		MaxAge:    24 * time.Hour,
		Strategy:  retention.StrategyDelete,
	}
	_, err := repo.Sweep(context.Background(), policy)
	require.Error(t, err)
}

// TestRetentionRepository_AuditLogsArchive covers the archive
// strategy end-to-end: rows older than the cutoff are MOVED into
// audit_logs_archive (insert + delete in one tx), and the live table
// no longer carries them.
func TestRetentionRepository_AuditLogsArchive(t *testing.T) {
	db := retentionTestDB(t)
	repo := postgres.NewRetentionRepository(db)
	ctx := context.Background()

	// Skip if the archive table is missing — happens when migration
	// 142 has not been applied to the shared DB yet. The retention
	// integration suite is not the right place to require it.
	var ok bool
	require.NoError(t, db.QueryRow(`SELECT to_regclass('audit_logs_archive') IS NOT NULL`).Scan(&ok))
	if !ok {
		t.Skip("audit_logs_archive table not present — apply migration 142 to enable this test")
	}

	tag := fmt.Sprintf("ret-arch-%s", uuid.New().String()[:8])
	oldIDs := make([]uuid.UUID, 0, 5)
	for i := 0; i < 5; i++ {
		id := uuid.New()
		_, err := db.ExecContext(ctx, `
            INSERT INTO audit_logs (id, user_id, action, resource_type, metadata, created_at)
            VALUES ($1, NULL, $2, 'retention_test', $3::jsonb, $4)`,
			id, tag, fmt.Sprintf(`{"tag":"%s"}`, tag), time.Now().Add(-30*30*24*time.Hour)) // 30 months old
		require.NoError(t, err)
		oldIDs = append(oldIDs, id)
	}
	youngID := uuid.New()
	_, err := db.ExecContext(ctx, `
        INSERT INTO audit_logs (id, user_id, action, resource_type, metadata, created_at)
        VALUES ($1, NULL, $2, 'retention_test', '{}'::jsonb, NOW())`,
		youngID, tag)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM audit_logs WHERE id = ANY($1)`, pgUUIDArray(append(oldIDs, youngID)))
		_, _ = db.Exec(`DELETE FROM audit_logs_archive WHERE id = ANY($1)`, pgUUIDArray(oldIDs))
	})

	policy := retention.Policy{
		Name:         "test_audit_archive",
		Table:        "audit_logs",
		AgeColumn:    "created_at",
		MaxAge:       24 * 30 * 24 * time.Hour, // 24 months
		Strategy:     retention.StrategyArchive,
		ArchiveTable: "audit_logs_archive",
		BatchSize:    100,
	}

	n, err := repo.Sweep(ctx, policy)
	require.NoError(t, err)
	assert.Equal(t, 5, n)

	// All 5 old rows now in archive.
	var archived int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM audit_logs_archive WHERE id = ANY($1)`, pgUUIDArray(oldIDs)).Scan(&archived))
	assert.Equal(t, 5, archived)

	// All 5 old rows gone from live table.
	var liveOld int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM audit_logs WHERE id = ANY($1)`, pgUUIDArray(oldIDs)).Scan(&liveOld))
	assert.Equal(t, 0, liveOld)

	// Young row still in live.
	var liveYoung int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM audit_logs WHERE id = $1`, youngID).Scan(&liveYoung))
	assert.Equal(t, 1, liveYoung)

	// Idempotent.
	n, err = repo.Sweep(ctx, policy)
	require.NoError(t, err)
	assert.Equal(t, 0, n)
}
