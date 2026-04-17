package postgres_test

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/adapter/postgres"
	"marketplace-backend/internal/app/searchanalytics"
)

// search_analytics_repository_test.go is gated by
// MARKETPLACE_TEST_DATABASE_URL so the default `go test ./...` run
// does not require a Postgres. When the var is set the test opens a
// real connection, truncates the search_queries table, and exercises
// the adapter against the migration-112 schema.

func analyticsTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("MARKETPLACE_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set MARKETPLACE_TEST_DATABASE_URL to run search analytics integration tests")
	}
	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	require.NoError(t, db.Ping())
	// Clean slate per run.
	_, err = db.Exec("DELETE FROM search_queries")
	require.NoError(t, err)
	return db
}

func TestSearchAnalyticsRepository_InsertAndRecordClick(t *testing.T) {
	db := analyticsTestDB(t)
	defer db.Close()
	repo := postgres.NewSearchAnalyticsRepository(db)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)
	row := &searchanalytics.SearchRow{
		SearchID:     "test-search-1",
		UserID:       "",
		SessionID:    "sess-1",
		Query:        "react paris",
		FilterBy:     "persona:freelance",
		SortBy:       "rating_score:desc",
		Persona:      "freelance",
		ResultsCount: 10,
		LatencyMs:    42,
		CreatedAt:    now,
	}
	require.NoError(t, repo.InsertSearch(ctx, row))

	// Idempotent: second insert with same search_id is a no-op.
	row.ResultsCount = 999
	require.NoError(t, repo.InsertSearch(ctx, row))

	// Read back and assert original values survived the ON CONFLICT.
	var results int
	err := db.QueryRow("SELECT results_count FROM search_queries WHERE search_id = $1", row.SearchID).Scan(&results)
	require.NoError(t, err)
	assert.Equal(t, 10, results)

	// Click update succeeds.
	clickAt := now.Add(3 * time.Second)
	err = repo.RecordClick(ctx, row.SearchID, "11111111-1111-1111-1111-111111111111", 2, clickAt)
	require.NoError(t, err)

	var position int
	err = db.QueryRow("SELECT clicked_position FROM search_queries WHERE search_id = $1", row.SearchID).Scan(&position)
	require.NoError(t, err)
	assert.Equal(t, 2, position)

	// Click on unknown search_id returns ErrNotFound.
	err = repo.RecordClick(ctx, "does-not-exist", "11111111-1111-1111-1111-111111111111", 0, clickAt)
	assert.ErrorIs(t, err, searchanalytics.ErrNotFound)
}
