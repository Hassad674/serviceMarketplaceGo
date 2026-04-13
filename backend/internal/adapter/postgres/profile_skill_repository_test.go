package postgres_test

// Integration tests for the PostgreSQL-backed ProfileSkillRepository
// (adapter implementation of port/repository.ProfileSkillRepository,
// migration 081 schema).
//
// Gated behind MARKETPLACE_TEST_DATABASE_URL, same pattern as the
// other postgres integration tests in this package.

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
	"marketplace-backend/internal/domain/organization"
	"marketplace-backend/internal/domain/job"
)

// newTestOrgForSkills creates a test user, an agency organization
// owned by that user, and returns the org id. Both rows are cleaned
// up via t.Cleanup on the user. The helper is intentionally separate
// from createOrg in job_credit_repository_test.go so the skills tests
// stay runnable even if that file moves — the two suites are free to
// evolve independently.
func newTestOrgForSkills(t *testing.T) uuid.UUID {
	t.Helper()
	db := testDB(t)
	ownerID := insertTestUser(t, db)

	org, err := organization.NewOrganization(ownerID, organization.OrgTypeAgency, "Skills Test Org")
	require.NoError(t, err)
	member, err := organization.NewMember(org.ID, ownerID, organization.RoleOwner, "")
	require.NoError(t, err)

	orgRepo := postgres.NewOrganizationRepository(db, job.WeeklyQuota)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, orgRepo.CreateWithOwnerMembership(ctx, org, member))

	return org.ID
}

// seedCatalogSkill upserts a catalog entry with the given normalized
// text so profile_skills can reference it via the FK. Returns the
// normalized skill_text so callers can feed it straight into
// ProfileSkill.SkillText.
func seedCatalogSkill(t *testing.T, rawText string) string {
	t.Helper()
	db := testDB(t)
	catalogRepo := postgres.NewSkillCatalogRepository(db)

	entry, err := domainskill.NewCatalogEntry(rawText, rawText, []string{"development"}, true)
	require.NoError(t, err, "build catalog entry for %q", rawText)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, catalogRepo.Upsert(ctx, entry))
	return entry.SkillText
}

// buildProfileSkills returns a slice of *ProfileSkill with contiguous
// 0-indexed positions, one per provided skill_text. Order of input
// is preserved: idx i becomes Position i.
func buildProfileSkills(orgID uuid.UUID, skillTexts []string) []*domainskill.ProfileSkill {
	out := make([]*domainskill.ProfileSkill, len(skillTexts))
	for i, text := range skillTexts {
		out[i] = &domainskill.ProfileSkill{
			OrganizationID: orgID,
			SkillText:      text,
			Position:       i,
		}
	}
	return out
}

func TestProfileSkillRepository(t *testing.T) {
	db := testDB(t)
	repo := postgres.NewProfileSkillRepository(db)

	t.Run("ListByOrgID_Empty", func(t *testing.T) {
		orgID := newTestOrgForSkills(t)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		skills, err := repo.ListByOrgID(ctx, orgID)
		require.NoError(t, err)
		assert.NotNil(t, skills, "empty list must still be a non-nil slice")
		assert.Len(t, skills, 0)
	})

	t.Run("ListByOrgID_Ordered", func(t *testing.T) {
		orgID := newTestOrgForSkills(t)
		tag := testSkillTag(t)
		cleanupSkillsByTag(t, tag)

		// Seed three distinct catalog entries (FK-safe).
		s0 := seedCatalogSkill(t, "zero "+tag)
		s1 := seedCatalogSkill(t, "one "+tag)
		s2 := seedCatalogSkill(t, "two "+tag)

		// Build with out-of-order positions to verify ORDER BY position ASC.
		skills := []*domainskill.ProfileSkill{
			{OrganizationID: orgID, SkillText: s2, Position: 2},
			{OrganizationID: orgID, SkillText: s0, Position: 0},
			{OrganizationID: orgID, SkillText: s1, Position: 1},
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		require.NoError(t, repo.ReplaceForOrg(ctx, orgID, skills))

		out, err := repo.ListByOrgID(ctx, orgID)
		require.NoError(t, err)
		require.Len(t, out, 3)
		assert.Equal(t, s0, out[0].SkillText, "position 0 first")
		assert.Equal(t, s1, out[1].SkillText, "position 1 second")
		assert.Equal(t, s2, out[2].SkillText, "position 2 third")
	})

	t.Run("ReplaceForOrg_InsertsAll", func(t *testing.T) {
		orgID := newTestOrgForSkills(t)
		tag := testSkillTag(t)
		cleanupSkillsByTag(t, tag)

		texts := []string{
			seedCatalogSkill(t, "a "+tag),
			seedCatalogSkill(t, "b "+tag),
			seedCatalogSkill(t, "c "+tag),
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		require.NoError(t, repo.ReplaceForOrg(ctx, orgID, buildProfileSkills(orgID, texts)))

		out, err := repo.ListByOrgID(ctx, orgID)
		require.NoError(t, err)
		assert.Len(t, out, 3)
	})

	t.Run("ReplaceForOrg_ClearsExisting", func(t *testing.T) {
		orgID := newTestOrgForSkills(t)
		tag := testSkillTag(t)
		cleanupSkillsByTag(t, tag)

		first := []string{
			seedCatalogSkill(t, "first1 "+tag),
			seedCatalogSkill(t, "first2 "+tag),
			seedCatalogSkill(t, "first3 "+tag),
		}
		second := []string{
			seedCatalogSkill(t, "second "+tag),
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		require.NoError(t, repo.ReplaceForOrg(ctx, orgID, buildProfileSkills(orgID, first)))
		require.NoError(t, repo.ReplaceForOrg(ctx, orgID, buildProfileSkills(orgID, second)))

		out, err := repo.ListByOrgID(ctx, orgID)
		require.NoError(t, err)
		require.Len(t, out, 1)
		assert.Equal(t, second[0], out[0].SkillText)
	})

	t.Run("ReplaceForOrg_EmptyClearsAll", func(t *testing.T) {
		orgID := newTestOrgForSkills(t)
		tag := testSkillTag(t)
		cleanupSkillsByTag(t, tag)

		seeded := []string{
			seedCatalogSkill(t, "x "+tag),
			seedCatalogSkill(t, "y "+tag),
			seedCatalogSkill(t, "z "+tag),
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		require.NoError(t, repo.ReplaceForOrg(ctx, orgID, buildProfileSkills(orgID, seeded)))
		require.NoError(t, repo.ReplaceForOrg(ctx, orgID, nil))

		out, err := repo.ListByOrgID(ctx, orgID)
		require.NoError(t, err)
		assert.Len(t, out, 0)
	})

	// TODO: atomicity of ReplaceForOrg (DELETE + INSERT within one tx)
	// cannot be asserted end-to-end without a custom injection hook on
	// the *sql.Tx. The pattern is identical to ExpertiseRepository.Replace
	// which already has exhaustive unit coverage; the code path is shared
	// enough in structure that a dedicated hook is deferred for now.

	t.Run("CountByOrg", func(t *testing.T) {
		orgID := newTestOrgForSkills(t)
		tag := testSkillTag(t)
		cleanupSkillsByTag(t, tag)

		texts := make([]string, 5)
		for i := range texts {
			texts[i] = seedCatalogSkill(t, fmt.Sprintf("cnt%d %s", i, tag))
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		require.NoError(t, repo.ReplaceForOrg(ctx, orgID, buildProfileSkills(orgID, texts)))

		count, err := repo.CountByOrg(ctx, orgID)
		require.NoError(t, err)
		assert.Equal(t, 5, count)
	})

	t.Run("DeleteAllByOrg", func(t *testing.T) {
		orgID := newTestOrgForSkills(t)
		tag := testSkillTag(t)
		cleanupSkillsByTag(t, tag)

		texts := []string{
			seedCatalogSkill(t, "d1 "+tag),
			seedCatalogSkill(t, "d2 "+tag),
			seedCatalogSkill(t, "d3 "+tag),
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		require.NoError(t, repo.ReplaceForOrg(ctx, orgID, buildProfileSkills(orgID, texts)))
		require.NoError(t, repo.DeleteAllByOrg(ctx, orgID))

		count, err := repo.CountByOrg(ctx, orgID)
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})

	t.Run("ForeignKeyConstraint_SkillMustExist", func(t *testing.T) {
		orgID := newTestOrgForSkills(t)

		// A skill_text that has NOT been inserted into skills_catalog.
		missing := "ghost-skill-" + uuid.New().String()
		skills := []*domainskill.ProfileSkill{
			{OrganizationID: orgID, SkillText: missing, Position: 0},
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err := repo.ReplaceForOrg(ctx, orgID, skills)
		require.Error(t, err, "FK violation must surface as an error")
	})
}
