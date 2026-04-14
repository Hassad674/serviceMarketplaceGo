package postgres_test

// Integration tests for the Tier 1 completion methods on
// ProfileRepository (migration 083). Verifies the focused update
// methods (UpdateLocation / UpdateLanguages / UpdateAvailability)
// write only their own columns and that the read path hydrates the
// domain struct correctly.
//
// Gated behind MARKETPLACE_TEST_DATABASE_URL — auto-skip when unset.

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
	"marketplace-backend/internal/port/repository"
)

// newTestOrgForProfile creates a fresh org and auto-creates its
// profile via the repository's ensureProfile fallback, returning
// the org id. Shared helper for every test in this file.
func newTestOrgForProfile(t *testing.T) uuid.UUID {
	t.Helper()
	db := testDB(t)
	ownerID := insertTestUser(t, db)

	org, err := organization.NewOrganization(ownerID, organization.OrgTypeAgency, "Profile Tier1 Org")
	require.NoError(t, err)
	member, err := organization.NewMember(org.ID, ownerID, organization.RoleOwner, "")
	require.NoError(t, err)

	orgRepo := postgres.NewOrganizationRepository(db, job.WeeklyQuota)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, orgRepo.CreateWithOwnerMembership(ctx, org, member))

	// Touch the profile repo to force auto-creation via
	// GetByOrganizationID's ensureProfile path.
	profileRepo := postgres.NewProfileRepository(db)
	_, err = profileRepo.GetByOrganizationID(ctx, org.ID)
	require.NoError(t, err)

	return org.ID
}

func TestProfileRepository_Tier1_DefaultsOnAutoCreate(t *testing.T) {
	db := testDB(t)
	repo := postgres.NewProfileRepository(db)
	orgID := newTestOrgForProfile(t)

	p, err := repo.GetByOrganizationID(context.Background(), orgID)
	require.NoError(t, err)
	require.NotNil(t, p)

	assert.Empty(t, p.City)
	assert.Empty(t, p.CountryCode)
	assert.Nil(t, p.Latitude)
	assert.Nil(t, p.Longitude)
	assert.Len(t, p.WorkMode, 0)
	assert.Nil(t, p.TravelRadiusKm)
	assert.Len(t, p.LanguagesProfessional, 0)
	assert.Len(t, p.LanguagesConversational, 0)
	assert.Equal(t, profile.AvailabilityNow, p.AvailabilityStatus,
		"db default should be 'available_now' for new profiles")
	assert.Nil(t, p.ReferrerAvailabilityStatus)
}

func TestProfileRepository_UpdateLocation_RoundTrip(t *testing.T) {
	db := testDB(t)
	repo := postgres.NewProfileRepository(db)
	orgID := newTestOrgForProfile(t)

	lat := 48.8566
	lng := 2.3522
	radius := 50
	err := repo.UpdateLocation(context.Background(), orgID, repository.LocationInput{
		City:           "Paris",
		CountryCode:    "FR",
		Latitude:       &lat,
		Longitude:      &lng,
		WorkMode:       []string{"remote", "hybrid"},
		TravelRadiusKm: &radius,
	})
	require.NoError(t, err)

	p, err := repo.GetByOrganizationID(context.Background(), orgID)
	require.NoError(t, err)
	assert.Equal(t, "Paris", p.City)
	assert.Equal(t, "FR", p.CountryCode)
	require.NotNil(t, p.Latitude)
	assert.InDelta(t, 48.8566, *p.Latitude, 0.0001)
	require.NotNil(t, p.Longitude)
	assert.InDelta(t, 2.3522, *p.Longitude, 0.0001)
	assert.ElementsMatch(t, []string{"remote", "hybrid"}, p.WorkMode)
	require.NotNil(t, p.TravelRadiusKm)
	assert.Equal(t, 50, *p.TravelRadiusKm)
}

func TestProfileRepository_UpdateLocation_NilCoordsClearColumns(t *testing.T) {
	db := testDB(t)
	repo := postgres.NewProfileRepository(db)
	orgID := newTestOrgForProfile(t)

	ctx := context.Background()

	lat := 1.0
	lng := 2.0
	require.NoError(t, repo.UpdateLocation(ctx, orgID, repository.LocationInput{
		City: "A", CountryCode: "FR", Latitude: &lat, Longitude: &lng,
	}))
	// Now clear by passing nil pointers.
	require.NoError(t, repo.UpdateLocation(ctx, orgID, repository.LocationInput{
		City: "", CountryCode: "",
	}))

	p, err := repo.GetByOrganizationID(ctx, orgID)
	require.NoError(t, err)
	assert.Nil(t, p.Latitude)
	assert.Nil(t, p.Longitude)
	assert.Empty(t, p.City)
	assert.Empty(t, p.CountryCode)
}

func TestProfileRepository_UpdateLocation_LeavesTitleUntouched(t *testing.T) {
	db := testDB(t)
	repo := postgres.NewProfileRepository(db)
	orgID := newTestOrgForProfile(t)

	ctx := context.Background()

	// Seed the classic title via Update.
	p, err := repo.GetByOrganizationID(ctx, orgID)
	require.NoError(t, err)
	p.Title = "Backend Agency"
	require.NoError(t, repo.Update(ctx, p))

	// Update the location block.
	require.NoError(t, repo.UpdateLocation(ctx, orgID, repository.LocationInput{
		City: "Lyon", CountryCode: "FR",
	}))

	// Title must be untouched.
	after, err := repo.GetByOrganizationID(ctx, orgID)
	require.NoError(t, err)
	assert.Equal(t, "Backend Agency", after.Title)
	assert.Equal(t, "Lyon", after.City)
}

func TestProfileRepository_UpdateLanguages_RoundTrip(t *testing.T) {
	db := testDB(t)
	repo := postgres.NewProfileRepository(db)
	orgID := newTestOrgForProfile(t)

	ctx := context.Background()

	require.NoError(t, repo.UpdateLanguages(ctx, orgID,
		[]string{"fr", "en"},
		[]string{"es", "it"},
	))

	p, err := repo.GetByOrganizationID(ctx, orgID)
	require.NoError(t, err)
	assert.Equal(t, []string{"fr", "en"}, p.LanguagesProfessional)
	assert.Equal(t, []string{"es", "it"}, p.LanguagesConversational)
}

func TestProfileRepository_UpdateLanguages_ReplacesArrays(t *testing.T) {
	db := testDB(t)
	repo := postgres.NewProfileRepository(db)
	orgID := newTestOrgForProfile(t)

	ctx := context.Background()

	require.NoError(t, repo.UpdateLanguages(ctx, orgID, []string{"fr", "en", "es"}, []string{"de"}))
	require.NoError(t, repo.UpdateLanguages(ctx, orgID, []string{"fr"}, []string{}))

	p, err := repo.GetByOrganizationID(ctx, orgID)
	require.NoError(t, err)
	assert.Equal(t, []string{"fr"}, p.LanguagesProfessional)
	assert.Len(t, p.LanguagesConversational, 0)
}

func TestProfileRepository_UpdateAvailability_DirectOnly(t *testing.T) {
	db := testDB(t)
	repo := postgres.NewProfileRepository(db)
	orgID := newTestOrgForProfile(t)

	require.NoError(t, repo.UpdateAvailability(context.Background(), orgID, profile.AvailabilitySoon, nil))

	p, err := repo.GetByOrganizationID(context.Background(), orgID)
	require.NoError(t, err)
	assert.Equal(t, profile.AvailabilitySoon, p.AvailabilityStatus)
	assert.Nil(t, p.ReferrerAvailabilityStatus)
}

func TestProfileRepository_UpdateAvailability_WithReferrerSlot(t *testing.T) {
	db := testDB(t)
	repo := postgres.NewProfileRepository(db)
	orgID := newTestOrgForProfile(t)

	ref := profile.AvailabilityNot
	require.NoError(t, repo.UpdateAvailability(context.Background(), orgID, profile.AvailabilityNow, &ref))

	p, err := repo.GetByOrganizationID(context.Background(), orgID)
	require.NoError(t, err)
	require.NotNil(t, p.ReferrerAvailabilityStatus)
	assert.Equal(t, profile.AvailabilityNot, *p.ReferrerAvailabilityStatus)
}

func TestProfileRepository_UpdateAvailability_ClearReferrer(t *testing.T) {
	db := testDB(t)
	repo := postgres.NewProfileRepository(db)
	orgID := newTestOrgForProfile(t)

	ctx := context.Background()
	ref := profile.AvailabilitySoon
	require.NoError(t, repo.UpdateAvailability(ctx, orgID, profile.AvailabilityNow, &ref))
	require.NoError(t, repo.UpdateAvailability(ctx, orgID, profile.AvailabilityNow, nil))

	p, err := repo.GetByOrganizationID(ctx, orgID)
	require.NoError(t, err)
	assert.Nil(t, p.ReferrerAvailabilityStatus)
}
