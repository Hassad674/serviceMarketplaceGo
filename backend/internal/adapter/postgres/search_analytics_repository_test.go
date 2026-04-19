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

// TestSearchAnalyticsRepository_Stats exercises Totals + TopQueries +
// ZeroResultQueries against a real Postgres. Uses a deterministic
// seed so the aggregation numbers can be asserted exactly.
func TestSearchAnalyticsRepository_Stats(t *testing.T) {
	db := analyticsTestDB(t)
	defer db.Close()
	repo := postgres.NewSearchAnalyticsRepository(db)
	ctx := context.Background()

	base := time.Now().UTC().Add(-1 * time.Hour).Truncate(time.Second)
	rows := []*searchanalytics.SearchRow{
		{SearchID: "s1", Query: "react", Persona: "freelance", ResultsCount: 10, LatencyMs: 40, CreatedAt: base},
		{SearchID: "s2", Query: "React", Persona: "freelance", ResultsCount: 12, LatencyMs: 60, CreatedAt: base.Add(1 * time.Minute)},
		{SearchID: "s3", Query: "golang", Persona: "freelance", ResultsCount: 0, LatencyMs: 30, CreatedAt: base.Add(2 * time.Minute)},
		{SearchID: "s4", Query: "golang", Persona: "agency", ResultsCount: 0, LatencyMs: 90, CreatedAt: base.Add(3 * time.Minute)},
		{SearchID: "s5", Query: "python", Persona: "freelance", ResultsCount: 0, LatencyMs: 200, CreatedAt: base.Add(4 * time.Minute)},
	}
	for _, row := range rows {
		require.NoError(t, repo.InsertSearch(ctx, row))
	}

	// Click on s1 so TopQueries reports a non-zero CTR for "react".
	require.NoError(t, repo.RecordClick(ctx, "s1", "11111111-1111-1111-1111-111111111111", 0, base.Add(5*time.Minute)))

	window := searchanalytics.StatsFilter{From: base.Add(-time.Hour), To: base.Add(time.Hour)}

	totals, err := repo.Totals(ctx, window)
	require.NoError(t, err)
	assert.Equal(t, 5, totals.TotalSearches)
	assert.Equal(t, 3, totals.ZeroResults) // s3, s4, s5
	assert.InDelta(t, 0.6, totals.ZeroResultRate, 1e-9)
	assert.InDelta(t, 84.0, totals.AvgLatencyMs, 1.0) // (40+60+30+90+200)/5 = 84
	assert.Greater(t, totals.P95LatencyMs, 100.0)

	// With persona filter, only freelance rows count (s1,s2,s3,s5).
	freelance, err := repo.Totals(ctx, searchanalytics.StatsFilter{
		From: window.From, To: window.To, Persona: "freelance",
	})
	require.NoError(t, err)
	assert.Equal(t, 4, freelance.TotalSearches)

	top, err := repo.TopQueries(ctx, window, 10)
	require.NoError(t, err)
	require.NotEmpty(t, top)
	// "golang" has count=2 (s3+s4), "react" has count=2 (s1+s2 case-folded),
	// "python" has count=1. Tie broken by query ASC -> golang,react,python.
	assert.Equal(t, "golang", top[0].Query)
	assert.Equal(t, 2, top[0].Count)
	// "react" with one click out of two → CTR 0.5.
	for _, row := range top {
		if row.Query == "react" {
			assert.InDelta(t, 0.5, row.CTR, 1e-9)
			assert.InDelta(t, 11.0, row.AvgResults, 1e-9)
		}
	}

	zero, err := repo.ZeroResultQueries(ctx, window, 10)
	require.NoError(t, err)
	require.NotEmpty(t, zero)
	// Only golang (count=2) + python (count=1) should appear; react has no zero rows.
	for _, row := range zero {
		if row.Query == "react" {
			t.Errorf("react must not appear in zero-result list, got %+v", row)
		}
	}
	assert.Equal(t, "golang", zero[0].Query)
	assert.Equal(t, 2, zero[0].Count)
}

// TestSearchAnalyticsRepository_AttachResultFeatures exercises the
// LTR feature-vector persistence added in migration 113. Requires the
// 113 schema to be present (gated on MARKETPLACE_TEST_DATABASE_URL).
func TestSearchAnalyticsRepository_AttachResultFeatures(t *testing.T) {
	db := analyticsTestDB(t)
	defer db.Close()
	repo := postgres.NewSearchAnalyticsRepository(db)
	ctx := context.Background()

	base := time.Now().UTC().Truncate(time.Second)
	row := &searchanalytics.SearchRow{
		SearchID:     "ltr-test-1",
		SessionID:    "sess-1",
		Query:        "react paris",
		Persona:      "freelance",
		ResultsCount: 2,
		LatencyMs:    32,
		CreatedAt:    base,
	}
	require.NoError(t, repo.InsertSearch(ctx, row))

	// Craft a canonical payload identical to what the service-layer
	// encoder produces.
	results := []searchanalytics.RankedResult{
		{DocID: "doc-1", RankPosition: 1, FinalScore: 87.3,
			Features: map[string]float64{"text_match": 0.82, "rating": 0.69}},
		{DocID: "doc-2", RankPosition: 2, FinalScore: 85.0,
			Features: map[string]float64{"text_match": 0.78, "rating": 0.90}},
	}
	payload, sha, err := searchanalytics.EncodeResultPayload(results)
	require.NoError(t, err)

	require.NoError(t, repo.AttachResultFeatures(ctx, "ltr-test-1", payload, sha))

	// Read back and verify the payload parses as a valid JSON array.
	var gotPayload, gotSHA string
	err = db.QueryRow(
		"SELECT result_features_json::text, result_vector_sha FROM search_queries WHERE search_id = $1",
		"ltr-test-1",
	).Scan(&gotPayload, &gotSHA)
	require.NoError(t, err)
	assert.Equal(t, sha, gotSHA)
	assert.Contains(t, gotPayload, `"doc_id": "doc-1"`)
	assert.Contains(t, gotPayload, `"rank_position": 1`)

	// Re-attach same payload → idempotent (no error, no rows touched by SHA guard).
	require.NoError(t, repo.AttachResultFeatures(ctx, "ltr-test-1", payload, sha))

	// Attach a DIFFERENT payload (new SHA) → overwrite.
	resultsV2 := append([]searchanalytics.RankedResult(nil), results...)
	resultsV2[0].FinalScore = 99.0
	payload2, sha2, err := searchanalytics.EncodeResultPayload(resultsV2)
	require.NoError(t, err)
	assert.NotEqual(t, sha, sha2)
	require.NoError(t, repo.AttachResultFeatures(ctx, "ltr-test-1", payload2, sha2))

	err = db.QueryRow(
		"SELECT result_vector_sha FROM search_queries WHERE search_id = $1",
		"ltr-test-1",
	).Scan(&gotSHA)
	require.NoError(t, err)
	assert.Equal(t, sha2, gotSHA, "latest attach wins when the SHA differs")

	// Unknown search_id → ErrNotFound.
	err = repo.AttachResultFeatures(ctx, "does-not-exist", payload2, sha2)
	assert.ErrorIs(t, err, searchanalytics.LTRErrNotFound)

	// Empty search_id → synchronous validation error.
	err = repo.AttachResultFeatures(ctx, "", payload2, sha2)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty search_id")
}
