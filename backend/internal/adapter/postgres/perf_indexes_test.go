package postgres_test

// Tests for migration 152_perf_indexes_hot_queries.
//
// Asserts that every covering index added in slot 152 exists on the
// target table with the exact name and the expected partial-clause /
// column list. The intent is to fail the build the moment someone
// accidentally drops or renames one of these indexes — the planner
// would silently fall back to Seq Scan and the regression would
// only show up under production load.
//
// All checks introspect pg_indexes (catalog view) rather than
// pg_class so we can match the WHERE clause text on partial indexes.
// Pure SELECTs — no schema or data mutation. Skips when
// MARKETPLACE_TEST_DATABASE_URL is not set so unit-only runs stay
// green.

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// perfIndexTestDB mirrors sessionTestDB in session_repository_test.go.
// Pulled into its own helper to keep this file self-contained — the
// index assertions do not need any of the user/session bootstrap.
func perfIndexTestDB(t *testing.T) *sql.DB {
	t.Helper()

	dsn := os.Getenv("MARKETPLACE_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("MARKETPLACE_TEST_DATABASE_URL not set — skipping postgres integration test")
	}

	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err, "open test database")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, db.PingContext(ctx), "ping test database")

	t.Cleanup(func() { _ = db.Close() })
	return db
}

// fetchIndexDef returns the indexdef text from pg_indexes for the
// (schema, table, name) tuple. Returns empty string when the index
// is missing so the caller can produce a precise failure message.
func fetchIndexDef(t *testing.T, db *sql.DB, schema, table, name string) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var def string
	err := db.QueryRowContext(ctx, `
		SELECT indexdef
		FROM pg_indexes
		WHERE schemaname = $1 AND tablename = $2 AND indexname = $3
	`, schema, table, name).Scan(&def)
	if err == sql.ErrNoRows {
		return ""
	}
	require.NoError(t, err)
	return def
}

// TestPerfIndexes_152_UserSessionsActiveByUser asserts the partial
// index on (user_id, expires_at DESC) WHERE revoked_at IS NULL is
// present, matches the WHERE clause, and is keyed on the right
// columns.
func TestPerfIndexes_152_UserSessionsActiveByUser(t *testing.T) {
	db := perfIndexTestDB(t)

	def := fetchIndexDef(t, db, "public", "user_sessions", "idx_user_sessions_active_by_user")
	require.NotEmpty(t, def, "idx_user_sessions_active_by_user is missing — re-apply migration 152")

	// The exact normalised text Postgres returns from pg_indexes.
	assert.Contains(t, def, "(user_id, expires_at DESC)",
		"expected key columns (user_id, expires_at DESC) in %q", def)
	assert.Contains(t, def, "WHERE (revoked_at IS NULL)",
		"expected partial clause WHERE revoked_at IS NULL in %q", def)
}

// TestPerfIndexes_152_AuditLogsUserTimeSeries asserts the composite
// partial index on (user_id, created_at DESC, id DESC) WHERE user_id
// IS NOT NULL is present and shaped exactly as the cursor query
// expects.
func TestPerfIndexes_152_AuditLogsUserTimeSeries(t *testing.T) {
	db := perfIndexTestDB(t)

	def := fetchIndexDef(t, db, "public", "audit_logs", "idx_audit_logs_user_time_series")
	require.NotEmpty(t, def, "idx_audit_logs_user_time_series is missing — re-apply migration 152")

	assert.Contains(t, def, "(user_id, created_at DESC, id DESC)",
		"expected key columns (user_id, created_at DESC, id DESC) in %q", def)
	assert.Contains(t, def, "WHERE (user_id IS NOT NULL)",
		"expected partial clause WHERE user_id IS NOT NULL in %q", def)
}

// TestPerfIndexes_152_ProfileViewEventsOrgTimeCameFrom asserts the
// covering index on (organization_id, created_at DESC, came_from)
// is present. No partial clause for this one — the visibility query
// reads any persona / any non-null came_from.
func TestPerfIndexes_152_ProfileViewEventsOrgTimeCameFrom(t *testing.T) {
	db := perfIndexTestDB(t)

	def := fetchIndexDef(t, db, "public", "profile_view_events", "idx_pve_org_time_came_from")
	require.NotEmpty(t, def, "idx_pve_org_time_came_from is missing — re-apply migration 152")

	assert.Contains(t, def, "(organization_id, created_at DESC, came_from)",
		"expected key columns (organization_id, created_at DESC, came_from) in %q", def)
	// Sanity: no partial clause leaked in.
	assert.False(t, strings.Contains(def, " WHERE "),
		"idx_pve_org_time_came_from must NOT be a partial index — got %q", def)
}

// TestPerfIndexes_152_AllPresent is a single roll-up test for CI
// dashboards. Fails fast if ANY of the three indexes is missing.
func TestPerfIndexes_152_AllPresent(t *testing.T) {
	db := perfIndexTestDB(t)

	cases := []struct {
		table string
		name  string
	}{
		{"user_sessions", "idx_user_sessions_active_by_user"},
		{"audit_logs", "idx_audit_logs_user_time_series"},
		{"profile_view_events", "idx_pve_org_time_came_from"},
	}
	for _, c := range cases {
		def := fetchIndexDef(t, db, "public", c.table, c.name)
		assert.NotEmptyf(t, def, "missing index %s on %s — migration 152 not applied", c.name, c.table)
	}
}
