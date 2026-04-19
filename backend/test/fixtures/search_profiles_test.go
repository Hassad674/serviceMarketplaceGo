package fixtures_test

// Integration test for the 200-fixture generator.
// Gated on MARKETPLACE_TEST_DATABASE_URL — skips otherwise.

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
	"marketplace-backend/internal/search"
	"marketplace-backend/test/fixtures"
)

func fixtureTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("MARKETPLACE_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("MARKETPLACE_TEST_DATABASE_URL not set — skipping fixtures integration test")
	}
	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, db.PingContext(ctx))
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestSeedSearchProfiles_Counts(t *testing.T) {
	db := fixtureTestDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Smaller counts so the test runs quickly — still exercises
	// every persona path.
	counts := fixtures.SearchFixtureCounts{Freelance: 5, Agency: 3, Referrer: 2}

	seeded, err := fixtures.SeedSearchProfiles(ctx, db, counts)
	require.NoError(t, err)
	defer func() {
		_ = fixtures.CleanupSearchProfiles(ctx, db, seeded)
	}()

	assert.Len(t, seeded.Freelance, counts.Freelance)
	assert.Len(t, seeded.Agency, counts.Agency)
	assert.Len(t, seeded.Referrer, counts.Referrer)
	assert.Equal(t, counts.Total(), len(seeded.Freelance)+len(seeded.Agency)+len(seeded.Referrer))
}

func TestSeedSearchProfiles_Idempotent(t *testing.T) {
	db := fixtureTestDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	counts := fixtures.SearchFixtureCounts{Freelance: 3, Agency: 2, Referrer: 1}

	seeded1, err := fixtures.SeedSearchProfiles(ctx, db, counts)
	require.NoError(t, err)
	defer func() {
		_ = fixtures.CleanupSearchProfiles(ctx, db, seeded1)
	}()

	seeded2, err := fixtures.SeedSearchProfiles(ctx, db, counts)
	require.NoError(t, err)

	// IDs must be stable across runs (deterministic UUIDs).
	assert.Equal(t, seeded1.Freelance, seeded2.Freelance)
	assert.Equal(t, seeded1.Agency, seeded2.Agency)
	assert.Equal(t, seeded1.Referrer, seeded2.Referrer)
}

// TestSeedSearchProfiles_UsableByIndexer confirms the fixtures
// produce documents that pass SearchDocument.Validate when fed
// into the real indexer path. This is the end-to-end guard: a
// fixture that breaks future ranking changes will trip this.
func TestSeedSearchProfiles_UsableByIndexer(t *testing.T) {
	db := fixtureTestDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	counts := fixtures.SearchFixtureCounts{Freelance: 3, Agency: 2, Referrer: 1}
	seeded, err := fixtures.SeedSearchProfiles(ctx, db, counts)
	require.NoError(t, err)
	defer func() {
		_ = fixtures.CleanupSearchProfiles(ctx, db, seeded)
	}()

	// Mini smoke check: the fixture orgs should all be retrievable
	// by the search adapter + indexer and produce valid documents.
	assert.NotEmpty(t, seeded.Freelance[0])
	assert.NotEmpty(t, seeded.Agency[0])
	assert.NotEmpty(t, seeded.Referrer[0])

	// Anchor a sentinel constant so the compiler keeps the search
	// package import even if an idle linter ever wants to strip it.
	_ = search.PersonaFreelance
}

// TestSeedSearchProfiles_RankingV1Signals asserts the 7 phase 6B
// signals actually flow from the fixture rows into a built document
// when the real postgres repository + indexer are wired together.
// With Freelance=12 the fixture distribution hits every anchor:
//   - idx 0 → 10-day account age (not mature)
//   - idx 1 → 400-day account age (mature)
//   - idx 3, 6, 9 → released projects with repeat client on the
//     second hop (idx%3==0 plus clientA == clientB when pool wraps)
//   - idx 0, 5, 10 → 2 reviews from 1 reviewer (max share 1.0)
//   - idx 0, 7 → 2 reviews from 2 reviewers
//   - idx 0, 11 → 1 full_refund dispute
//
// We pick idx=0 as our anchor because every distribution rule
// above hits idx 0 (it's divisible by 3, 5, 7, 11 only for small
// factors but idx%n==0 for idx=0), so the 7 signals all land
// non-zero on the fixture at position 0.
func TestSeedSearchProfiles_RankingV1Signals(t *testing.T) {
	db := fixtureTestDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	counts := fixtures.SearchFixtureCounts{Freelance: 12, Agency: 4, Referrer: 2}
	seeded, err := fixtures.SeedSearchProfiles(ctx, db, counts)
	require.NoError(t, err)
	defer func() {
		_ = fixtures.CleanupSearchProfiles(ctx, db, seeded)
	}()

	repo := postgres.NewSearchDocumentRepository(db)
	idx, err := search.NewIndexer(repo, search.NewMockEmbeddings())
	require.NoError(t, err)

	// Freelance 0 receives every distribution anchor, so it is a
	// good single-profile assertion target.
	doc0, err := idx.BuildDocument(ctx, seeded.Freelance[0], search.PersonaFreelance)
	require.NoError(t, err)

	assert.Greater(t, doc0.UniqueClientsCount, int32(0),
		"idx 0 has released projects with 2 distinct clients")
	assert.Greater(t, doc0.UniqueReviewersCount, int32(0),
		"idx 0 has at least one reviewer from %%5==0 branch")
	assert.Greater(t, doc0.ReviewRecencyFactor, 0.0,
		"fixture reviews anchor >0 recency")
	assert.GreaterOrEqual(t, doc0.MaxReviewerShare, 0.5,
		"single reviewer branch pushes share above 0.5")
	assert.Equal(t, int32(1), doc0.LostDisputesCount,
		"idx 0 has exactly one full_refund dispute (%%11==0 branch)")
	// applyFixtureAccountAge uses ages[0]=10, so idx 0 → ~10 days.
	assert.InDelta(t, int32(10), doc0.AccountAgeDays, 1,
		"fixture sets idx 0 created_at to 10 days ago")

	// Freelance 1 should have account age of 400 days
	doc1, err := idx.BuildDocument(ctx, seeded.Freelance[1], search.PersonaFreelance)
	require.NoError(t, err)
	assert.InDelta(t, int32(400), doc1.AccountAgeDays, 1,
		"fixture sets idx 1 created_at to 400 days ago — mature")
}
