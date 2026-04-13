package postgres_test

// Integration tests for the PostgreSQL-backed SkillCatalogRepository
// (adapter implementation of port/repository.SkillCatalogRepository,
// migration 081 schema).
//
// These tests talk to a real PostgreSQL and are gated behind the
// MARKETPLACE_TEST_DATABASE_URL environment variable. When the variable
// is absent (the common case on CI and fresh checkouts) the whole
// suite skips — no test ever fails because Docker is not running. To
// run it against the local dev stack, point it at a disposable DB:
//
//	MARKETPLACE_TEST_DATABASE_URL=postgres://postgres:postgres@localhost:5434/marketplace_go_test?sslmode=disable \
//	  go test ./internal/adapter/postgres/ -run TestSkillCatalog -count=1 -v
//
// The suite creates rows with randomized skill_text values and cleans
// them up in t.Cleanup so reruns stay isolated. It never touches any
// existing rows on a shared database.

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/adapter/postgres"
	domainskill "marketplace-backend/internal/domain/skill"
)

// testSkillTag returns a short, test-unique suffix used to tag every
// skill_text inserted by a single test run so cleanups and queries
// can target them without colliding with curated seed data that may
// already exist in the database.
func testSkillTag(t *testing.T) string {
	t.Helper()
	return "sktest-" + uuid.New().String()[:8]
}

// cleanupSkillsByTag registers a t.Cleanup that removes every
// skills_catalog row whose skill_text contains the given tag, and
// every profile_skills row referencing those. The function is
// idempotent and safe to call multiple times in one test.
func cleanupSkillsByTag(t *testing.T, tag string) {
	t.Helper()
	db := testDB(t)
	t.Cleanup(func() {
		_, _ = db.Exec(
			`DELETE FROM profile_skills WHERE skill_text LIKE '%' || $1 || '%'`,
			tag,
		)
		_, _ = db.Exec(
			`DELETE FROM skills_catalog WHERE skill_text LIKE '%' || $1 || '%'`,
			tag,
		)
	})
}

// upsertSkill is a convenience wrapper that builds + upserts a
// curated or user-created catalog entry under the provided tag, so
// tests can fabricate fixtures in one line.
func upsertSkill(t *testing.T, repo *postgres.SkillCatalogRepository, rawText string, keys []string, curated bool) *domainskill.CatalogEntry {
	t.Helper()
	entry, err := domainskill.NewCatalogEntry(rawText, rawText, keys, curated)
	require.NoError(t, err, "build catalog entry for %q", rawText)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, repo.Upsert(ctx, entry), "upsert %q", rawText)
	return entry
}

// bumpUsage increments a skill's usage counter N times. Useful to
// fabricate deterministic ordering by usage_count in ListCurated tests.
func bumpUsage(t *testing.T, repo *postgres.SkillCatalogRepository, skillText string, n int) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for i := 0; i < n; i++ {
		require.NoError(t, repo.IncrementUsageCount(ctx, skillText), "increment usage for %q", skillText)
	}
}

func TestSkillCatalogRepository(t *testing.T) {
	db := testDB(t)
	repo := postgres.NewSkillCatalogRepository(db)

	t.Run("Upsert_InsertsNew", func(t *testing.T) {
		tag := testSkillTag(t)
		cleanupSkillsByTag(t, tag)

		upsertSkill(t, repo, "react "+tag, []string{"development"}, true)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		found, err := repo.FindByText(ctx, "react "+tag)
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, "react "+tag, found.SkillText)
		assert.Equal(t, "react "+tag, found.DisplayText)
		assert.Equal(t, []string{"development"}, found.ExpertiseKeys)
		assert.True(t, found.IsCurated)
		assert.Equal(t, 0, found.UsageCount)
	})

	t.Run("Upsert_UpdatesExisting", func(t *testing.T) {
		tag := testSkillTag(t)
		cleanupSkillsByTag(t, tag)

		// First insert — curated with one expertise key.
		upsertSkill(t, repo, "golang "+tag, []string{"development"}, true)
		// Bump usage so we can verify it is PRESERVED across the update.
		bumpUsage(t, repo, "golang "+tag, 4)

		// Re-upsert with different display/keys/curated flag.
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		updated, err := domainskill.NewCatalogEntry(
			"golang "+tag,
			"Go (new display)",
			[]string{"development", "devops"},
			false,
		)
		require.NoError(t, err)
		require.NoError(t, repo.Upsert(ctx, updated))

		found, err := repo.FindByText(ctx, "golang "+tag)
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, "Go (new display)", found.DisplayText)
		assert.ElementsMatch(t, []string{"development", "devops"}, found.ExpertiseKeys)
		assert.False(t, found.IsCurated)
		// usage_count is preserved on update — this is the critical contract.
		assert.Equal(t, 4, found.UsageCount, "Upsert must preserve usage_count")
	})

	t.Run("FindByText_NotFound", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		found, err := repo.FindByText(ctx, "does-not-exist-"+uuid.New().String())
		require.NoError(t, err, "FindByText must NOT return a sentinel error on miss")
		assert.Nil(t, found)
	})

	t.Run("ListCuratedByExpertise_FiltersCurated", func(t *testing.T) {
		tag := testSkillTag(t)
		cleanupSkillsByTag(t, tag)
		key := "skills-e2e-key-" + tag

		upsertSkill(t, repo, "alpha "+tag, []string{key}, true)
		upsertSkill(t, repo, "beta "+tag, []string{key}, true)
		upsertSkill(t, repo, "gamma "+tag, []string{key}, false) // not curated

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		list, err := repo.ListCuratedByExpertise(ctx, key, 50)
		require.NoError(t, err)
		assert.Len(t, list, 2, "only curated entries should be returned")

		texts := collectSkillTexts(list)
		assert.Contains(t, texts, "alpha "+tag)
		assert.Contains(t, texts, "beta "+tag)
		assert.NotContains(t, texts, "gamma "+tag)
	})

	t.Run("ListCuratedByExpertise_OrderedByUsage", func(t *testing.T) {
		tag := testSkillTag(t)
		cleanupSkillsByTag(t, tag)
		key := "skills-order-key-" + tag

		upsertSkill(t, repo, "low "+tag, []string{key}, true)
		upsertSkill(t, repo, "mid "+tag, []string{key}, true)
		upsertSkill(t, repo, "high "+tag, []string{key}, true)

		bumpUsage(t, repo, "low "+tag, 1)
		bumpUsage(t, repo, "mid "+tag, 5)
		bumpUsage(t, repo, "high "+tag, 10)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		list, err := repo.ListCuratedByExpertise(ctx, key, 50)
		require.NoError(t, err)
		require.Len(t, list, 3)

		assert.Equal(t, "high "+tag, list[0].SkillText)
		assert.Equal(t, "mid "+tag, list[1].SkillText)
		assert.Equal(t, "low "+tag, list[2].SkillText)
	})

	t.Run("ListCuratedByExpertise_LimitRespected", func(t *testing.T) {
		tag := testSkillTag(t)
		cleanupSkillsByTag(t, tag)
		key := "skills-limit-key-" + tag

		for i := 0; i < 10; i++ {
			upsertSkill(t, repo, fmt.Sprintf("item%02d %s", i, tag), []string{key}, true)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		list, err := repo.ListCuratedByExpertise(ctx, key, 5)
		require.NoError(t, err)
		assert.Len(t, list, 5, "limit must be respected")
	})

	t.Run("ListCuratedByExpertise_UnknownKeyReturnsEmpty", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		list, err := repo.ListCuratedByExpertise(ctx, "unknown-key-"+uuid.New().String(), 50)
		require.NoError(t, err)
		assert.NotNil(t, list, "empty result must still be a non-nil slice")
		assert.Len(t, list, 0)
	})

	t.Run("CountCuratedByExpertise", func(t *testing.T) {
		tag := testSkillTag(t)
		cleanupSkillsByTag(t, tag)
		key := "skills-count-key-" + tag

		upsertSkill(t, repo, "one "+tag, []string{key}, true)
		upsertSkill(t, repo, "two "+tag, []string{key}, true)
		upsertSkill(t, repo, "three "+tag, []string{key}, true)
		// A non-curated row that must NOT be counted.
		upsertSkill(t, repo, "four "+tag, []string{key}, false)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		count, err := repo.CountCuratedByExpertise(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, 3, count)
	})

	t.Run("SearchAutocomplete_PrefixMatch", func(t *testing.T) {
		tag := testSkillTag(t)
		cleanupSkillsByTag(t, tag)

		// skill_text is lowercased by NormalizeSkillText via NewCatalogEntry
		// so the stored value becomes e.g. "react sktest-xxxx".
		upsertSkill(t, repo, "react "+tag, []string{"development"}, true)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		list, err := repo.SearchAutocomplete(ctx, "react "+tag[:4], 20)
		require.NoError(t, err)

		texts := collectSkillTexts(list)
		assert.Contains(t, texts, "react "+tag, "prefix match must surface the row")
	})

	t.Run("SearchAutocomplete_CuratedBeforeUserCreated", func(t *testing.T) {
		tag := testSkillTag(t)
		cleanupSkillsByTag(t, tag)

		// Distinct prefixes to keep the trigram branch out of it.
		upsertSkill(t, repo, "react "+tag, []string{"development"}, true)
		upsertSkill(t, repo, "react-native "+tag, []string{"mobile"}, false)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		list, err := repo.SearchAutocomplete(ctx, "react", 20)
		require.NoError(t, err)

		// Find indices of our two tagged rows and verify the curated one
		// comes before the user-created one.
		curatedIdx, userIdx := -1, -1
		for i, e := range list {
			switch e.SkillText {
			case "react " + tag:
				curatedIdx = i
			case "react-native " + tag:
				userIdx = i
			}
		}
		require.NotEqual(t, -1, curatedIdx, "curated row must be in results")
		require.NotEqual(t, -1, userIdx, "user-created row must be in results")
		assert.Less(t, curatedIdx, userIdx, "curated must rank before user-created")
	})

	t.Run("IncrementUsageCount", func(t *testing.T) {
		tag := testSkillTag(t)
		cleanupSkillsByTag(t, tag)

		upsertSkill(t, repo, "incr "+tag, []string{"development"}, true)
		bumpUsage(t, repo, "incr "+tag, 3)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		found, err := repo.FindByText(ctx, "incr "+tag)
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, 3, found.UsageCount)
	})

	t.Run("DecrementUsageCount_NeverNegative", func(t *testing.T) {
		tag := testSkillTag(t)
		cleanupSkillsByTag(t, tag)

		upsertSkill(t, repo, "decr "+tag, []string{"development"}, true)
		bumpUsage(t, repo, "decr "+tag, 1)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		// Decrement twice — expected result is 0, not -1.
		require.NoError(t, repo.DecrementUsageCount(ctx, "decr "+tag))
		require.NoError(t, repo.DecrementUsageCount(ctx, "decr "+tag))

		found, err := repo.FindByText(ctx, "decr "+tag)
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, 0, found.UsageCount, "counter must be clamped at zero")
	})
}

// collectSkillTexts pulls the skill_text out of a slice of catalog
// entries so assertions can use assert.Contains / assert.Len on a
// flat []string instead of a slice of structs.
func collectSkillTexts(entries []*domainskill.CatalogEntry) []string {
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		out = append(out, e.SkillText)
	}
	return out
}
