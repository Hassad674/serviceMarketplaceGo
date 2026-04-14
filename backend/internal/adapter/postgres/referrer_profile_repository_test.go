package postgres_test

// Integration tests for ReferrerProfileRepository (migration 098).

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/adapter/postgres"
	"marketplace-backend/internal/domain/job"
	"marketplace-backend/internal/domain/organization"
	"marketplace-backend/internal/domain/profile"
)

// newTestReferrerOrg creates a provider_personal org but does NOT
// pre-seed the referrer row — that way GetOrCreate can exercise
// its lazy-create path.
func newTestReferrerOrg(t *testing.T) uuid.UUID {
	t.Helper()
	db := testDB(t)
	ownerID := insertTestUser(t, db)

	org, err := organization.NewOrganization(ownerID, organization.OrgTypeProviderPersonal, "Referrer Test Org")
	require.NoError(t, err)
	member, err := organization.NewMember(org.ID, ownerID, organization.RoleOwner, "")
	require.NoError(t, err)

	orgRepo := postgres.NewOrganizationRepository(db, job.WeeklyQuota)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, orgRepo.CreateWithOwnerMembership(ctx, org, member))

	return org.ID
}

func TestReferrerProfileRepository_GetOrCreateByOrgID_LazilyCreates(t *testing.T) {
	db := testDB(t)
	orgID := newTestReferrerOrg(t)
	repo := postgres.NewReferrerProfileRepository(db)
	ctx := context.Background()

	// First call — no row exists yet.
	view1, err := repo.GetOrCreateByOrgID(ctx, orgID)
	require.NoError(t, err)
	require.NotNil(t, view1)
	assert.Equal(t, orgID, view1.Profile.OrganizationID)
	assert.Equal(t, profile.AvailabilityNow, view1.Profile.AvailabilityStatus)

	// Second call — same row, same ID (idempotent lazy create).
	view2, err := repo.GetOrCreateByOrgID(ctx, orgID)
	require.NoError(t, err)
	assert.Equal(t, view1.Profile.ID, view2.Profile.ID,
		"second GetOrCreate must not re-create a fresh row")
}

func TestReferrerProfileRepository_GetOrCreate_IncludesSharedBlock(t *testing.T) {
	db := testDB(t)
	orgID := newTestReferrerOrg(t)

	_, err := db.Exec(`
		UPDATE organizations
		   SET photo_url              = 'https://example.com/r.png',
		       city                   = 'Lyon',
		       country_code           = 'FR',
		       work_mode              = ARRAY['remote'],
		       languages_professional = ARRAY['fr']
		 WHERE id = $1`, orgID)
	require.NoError(t, err)

	repo := postgres.NewReferrerProfileRepository(db)
	view, err := repo.GetOrCreateByOrgID(context.Background(), orgID)
	require.NoError(t, err)
	assert.Equal(t, "https://example.com/r.png", view.Shared.PhotoURL)
	assert.Equal(t, "Lyon", view.Shared.City)
	assert.ElementsMatch(t, []string{"remote"}, view.Shared.WorkMode)
}

func TestReferrerProfileRepository_UpdateCore_PersistsTriplet(t *testing.T) {
	db := testDB(t)
	orgID := newTestReferrerOrg(t)
	repo := postgres.NewReferrerProfileRepository(db)
	ctx := context.Background()

	_, err := repo.GetOrCreateByOrgID(ctx, orgID) // lazy-create
	require.NoError(t, err)

	require.NoError(t, repo.UpdateCore(ctx, orgID,
		"Top Apporteur",
		"Finds deals",
		"https://example.com/v.mp4",
	))

	view, err := repo.GetOrCreateByOrgID(ctx, orgID)
	require.NoError(t, err)
	assert.Equal(t, "Top Apporteur", view.Profile.Title)
	assert.Equal(t, "Finds deals", view.Profile.About)
	assert.Equal(t, "https://example.com/v.mp4", view.Profile.VideoURL)
}

func TestReferrerProfileRepository_UpdateAvailability_PersistsValue(t *testing.T) {
	db := testDB(t)
	orgID := newTestReferrerOrg(t)
	repo := postgres.NewReferrerProfileRepository(db)
	ctx := context.Background()

	_, err := repo.GetOrCreateByOrgID(ctx, orgID) // lazy-create
	require.NoError(t, err)

	require.NoError(t, repo.UpdateAvailability(ctx, orgID, profile.AvailabilitySoon))

	view, err := repo.GetOrCreateByOrgID(ctx, orgID)
	require.NoError(t, err)
	assert.Equal(t, profile.AvailabilitySoon, view.Profile.AvailabilityStatus)
}

func TestReferrerProfileRepository_UpdateExpertiseDomains_ReplacesList(t *testing.T) {
	db := testDB(t)
	orgID := newTestReferrerOrg(t)
	repo := postgres.NewReferrerProfileRepository(db)
	ctx := context.Background()

	_, err := repo.GetOrCreateByOrgID(ctx, orgID)
	require.NoError(t, err)

	require.NoError(t, repo.UpdateExpertiseDomains(ctx, orgID, []string{"marketing", "sales"}))

	view, err := repo.GetOrCreateByOrgID(ctx, orgID)
	require.NoError(t, err)
	assert.Equal(t, []string{"marketing", "sales"}, view.Profile.ExpertiseDomains)
}
