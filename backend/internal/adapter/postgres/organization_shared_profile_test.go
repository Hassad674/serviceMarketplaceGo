package postgres_test

// Integration tests for the shared-profile write methods on
// OrganizationRepository (migration 096 columns).

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
	"marketplace-backend/internal/port/repository"
)

// newTestSharedProfileOrg creates a provider_personal org for
// shared-profile tests. Separate from the other helpers so we do
// not drag in the freelance profile seeding.
func newTestSharedProfileOrg(t *testing.T) uuid.UUID {
	t.Helper()
	db := testDB(t)
	ownerID := insertTestUser(t, db)

	org, err := organization.NewOrganization(ownerID, organization.OrgTypeProviderPersonal, "Shared Profile Test")
	require.NoError(t, err)
	member, err := organization.NewMember(org.ID, ownerID, organization.RoleOwner, "")
	require.NoError(t, err)

	orgRepo := postgres.NewOrganizationRepository(db, job.WeeklyQuota)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, orgRepo.CreateWithOwnerMembership(ctx, org, member))

	return org.ID
}

func float64p(v float64) *float64 { return &v }
func intpInt(v int) *int           { return &v }

func TestOrganizationRepository_UpdateSharedLocation_PersistsBlock(t *testing.T) {
	db := testDB(t)
	orgID := newTestSharedProfileOrg(t)
	repo := postgres.NewOrganizationRepository(db, job.WeeklyQuota)

	input := repository.SharedProfileLocationInput{
		City:           "Marseille",
		CountryCode:    "FR",
		Latitude:       float64p(43.2965),
		Longitude:      float64p(5.3698),
		WorkMode:       []string{"on_site", "hybrid"},
		TravelRadiusKm: intpInt(100),
	}
	require.NoError(t, repo.UpdateSharedLocation(context.Background(), orgID, input))

	shared, err := repo.GetSharedProfile(context.Background(), orgID)
	require.NoError(t, err)
	assert.Equal(t, "Marseille", shared.City)
	assert.Equal(t, "FR", shared.CountryCode)
	require.NotNil(t, shared.Latitude)
	assert.InDelta(t, 43.2965, *shared.Latitude, 0.0001)
	require.NotNil(t, shared.Longitude)
	assert.InDelta(t, 5.3698, *shared.Longitude, 0.0001)
	assert.ElementsMatch(t, []string{"on_site", "hybrid"}, shared.WorkMode)
	require.NotNil(t, shared.TravelRadiusKm)
	assert.Equal(t, 100, *shared.TravelRadiusKm)
}

func TestOrganizationRepository_UpdateSharedLocation_NilClearsNullables(t *testing.T) {
	db := testDB(t)
	orgID := newTestSharedProfileOrg(t)
	repo := postgres.NewOrganizationRepository(db, job.WeeklyQuota)
	ctx := context.Background()

	// Seed with values first.
	require.NoError(t, repo.UpdateSharedLocation(ctx, orgID, repository.SharedProfileLocationInput{
		City:           "Paris",
		CountryCode:    "FR",
		Latitude:       float64p(48.8566),
		Longitude:      float64p(2.3522),
		WorkMode:       []string{"remote"},
		TravelRadiusKm: intpInt(50),
	}))

	// Clear lat/lng/radius/work_mode.
	require.NoError(t, repo.UpdateSharedLocation(ctx, orgID, repository.SharedProfileLocationInput{
		City:        "",
		CountryCode: "",
	}))

	shared, err := repo.GetSharedProfile(ctx, orgID)
	require.NoError(t, err)
	assert.Empty(t, shared.City)
	assert.Empty(t, shared.CountryCode)
	assert.Nil(t, shared.Latitude)
	assert.Nil(t, shared.Longitude)
	assert.Empty(t, shared.WorkMode)
	assert.Nil(t, shared.TravelRadiusKm)
}

func TestOrganizationRepository_UpdateSharedLanguages_ReplacesBoth(t *testing.T) {
	db := testDB(t)
	orgID := newTestSharedProfileOrg(t)
	repo := postgres.NewOrganizationRepository(db, job.WeeklyQuota)

	require.NoError(t, repo.UpdateSharedLanguages(context.Background(), orgID,
		[]string{"fr", "en"}, []string{"es"}))

	shared, err := repo.GetSharedProfile(context.Background(), orgID)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"fr", "en"}, shared.LanguagesProfessional)
	assert.ElementsMatch(t, []string{"es"}, shared.LanguagesConversational)
}

func TestOrganizationRepository_UpdateSharedPhotoURL_Persists(t *testing.T) {
	db := testDB(t)
	orgID := newTestSharedProfileOrg(t)
	repo := postgres.NewOrganizationRepository(db, job.WeeklyQuota)

	require.NoError(t, repo.UpdateSharedPhotoURL(context.Background(), orgID,
		"https://example.com/p.png"))

	shared, err := repo.GetSharedProfile(context.Background(), orgID)
	require.NoError(t, err)
	assert.Equal(t, "https://example.com/p.png", shared.PhotoURL)
}

func TestOrganizationRepository_UpdateSharedLocation_NotFound(t *testing.T) {
	db := testDB(t)
	repo := postgres.NewOrganizationRepository(db, job.WeeklyQuota)

	err := repo.UpdateSharedLocation(context.Background(), uuid.New(), repository.SharedProfileLocationInput{})
	assert.ErrorIs(t, err, organization.ErrOrgNotFound)
}

func TestOrganizationRepository_GetSharedProfile_NotFound(t *testing.T) {
	db := testDB(t)
	repo := postgres.NewOrganizationRepository(db, job.WeeklyQuota)

	_, err := repo.GetSharedProfile(context.Background(), uuid.New())
	assert.ErrorIs(t, err, organization.ErrOrgNotFound)
}
