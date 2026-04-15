package postgres_test

// Integration tests for FreelanceProfileRepository (migration 097).
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
	"marketplace-backend/internal/domain/freelanceprofile"
	"marketplace-backend/internal/domain/job"
	"marketplace-backend/internal/domain/organization"
	"marketplace-backend/internal/domain/profile"
)

// newTestFreelanceOrg creates a provider_personal org AND inserts a
// blank freelance_profiles row for it — matching what the split
// migration 101 would do in production.
func newTestFreelanceOrg(t *testing.T) (uuid.UUID, uuid.UUID) {
	t.Helper()
	db := testDB(t)
	ownerID := insertTestUser(t, db)

	org, err := organization.NewOrganization(ownerID, organization.OrgTypeProviderPersonal, "Split Test")
	require.NoError(t, err)
	member, err := organization.NewMember(org.ID, ownerID, organization.RoleOwner, "")
	require.NoError(t, err)

	orgRepo := postgres.NewOrganizationRepository(db, job.WeeklyQuota)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, orgRepo.CreateWithOwnerMembership(ctx, org, member))

	p := freelanceprofile.New(org.ID)
	_, err = db.Exec(`
		INSERT INTO freelance_profiles (
			id, organization_id, title, about, video_url,
			availability_status, expertise_domains, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		p.ID, p.OrganizationID, p.Title, p.About, p.VideoURL,
		string(p.AvailabilityStatus), "{}",
		p.CreatedAt, p.UpdatedAt,
	)
	require.NoError(t, err, "seed freelance profile row")

	return org.ID, p.ID
}

func TestFreelanceProfileRepository_GetByOrgID_ReturnsJoinedShared(t *testing.T) {
	db := testDB(t)
	orgID, _ := newTestFreelanceOrg(t)

	// Plant shared-profile values on the org row so the JOIN has
	// something to read back.
	_, err := db.Exec(`
		UPDATE organizations
		   SET photo_url              = 'https://example.com/p.png',
		       city                   = 'Paris',
		       country_code           = 'FR',
		       latitude               = 48.8566,
		       longitude              = 2.3522,
		       work_mode              = ARRAY['remote','hybrid'],
		       travel_radius_km       = 50,
		       languages_professional = ARRAY['fr','en'],
		       languages_conversational = ARRAY['es']
		 WHERE id = $1`, orgID)
	require.NoError(t, err)

	repo := postgres.NewFreelanceProfileRepository(db)
	view, err := repo.GetByOrgID(context.Background(), orgID)
	require.NoError(t, err)
	require.NotNil(t, view.Profile)

	assert.Equal(t, orgID, view.Profile.OrganizationID)
	assert.Equal(t, profile.AvailabilityNow, view.Profile.AvailabilityStatus)
	assert.Equal(t, []string{}, view.Profile.ExpertiseDomains)

	assert.Equal(t, "https://example.com/p.png", view.Shared.PhotoURL)
	assert.Equal(t, "Paris", view.Shared.City)
	assert.Equal(t, "FR", view.Shared.CountryCode)
	require.NotNil(t, view.Shared.Latitude)
	assert.InDelta(t, 48.8566, *view.Shared.Latitude, 0.0001)
	require.NotNil(t, view.Shared.Longitude)
	assert.InDelta(t, 2.3522, *view.Shared.Longitude, 0.0001)
	assert.ElementsMatch(t, []string{"remote", "hybrid"}, view.Shared.WorkMode)
	require.NotNil(t, view.Shared.TravelRadiusKm)
	assert.Equal(t, 50, *view.Shared.TravelRadiusKm)
	assert.ElementsMatch(t, []string{"fr", "en"}, view.Shared.LanguagesProfessional)
	assert.ElementsMatch(t, []string{"es"}, view.Shared.LanguagesConversational)
}

func TestFreelanceProfileRepository_GetByOrgID_NotFound(t *testing.T) {
	db := testDB(t)
	repo := postgres.NewFreelanceProfileRepository(db)

	_, err := repo.GetByOrgID(context.Background(), uuid.New())
	assert.ErrorIs(t, err, freelanceprofile.ErrProfileNotFound)
}

func TestFreelanceProfileRepository_UpdateCore_PersistsTriplet(t *testing.T) {
	db := testDB(t)
	orgID, _ := newTestFreelanceOrg(t)
	repo := postgres.NewFreelanceProfileRepository(db)
	ctx := context.Background()

	require.NoError(t, repo.UpdateCore(ctx, orgID,
		"Senior Go Engineer",
		"Builds marketplaces.",
		"https://example.com/v.mp4",
	))

	view, err := repo.GetByOrgID(ctx, orgID)
	require.NoError(t, err)
	assert.Equal(t, "Senior Go Engineer", view.Profile.Title)
	assert.Equal(t, "Builds marketplaces.", view.Profile.About)
	assert.Equal(t, "https://example.com/v.mp4", view.Profile.VideoURL)
}

func TestFreelanceProfileRepository_UpdateAvailability_PersistsValue(t *testing.T) {
	db := testDB(t)
	orgID, _ := newTestFreelanceOrg(t)
	repo := postgres.NewFreelanceProfileRepository(db)

	require.NoError(t, repo.UpdateAvailability(context.Background(), orgID, profile.AvailabilityNot))

	view, err := repo.GetByOrgID(context.Background(), orgID)
	require.NoError(t, err)
	assert.Equal(t, profile.AvailabilityNot, view.Profile.AvailabilityStatus)
}

func TestFreelanceProfileRepository_UpdateExpertiseDomains_ReplacesList(t *testing.T) {
	db := testDB(t)
	orgID, _ := newTestFreelanceOrg(t)
	repo := postgres.NewFreelanceProfileRepository(db)
	ctx := context.Background()

	require.NoError(t, repo.UpdateExpertiseDomains(ctx, orgID, []string{"development", "design_ui_ux"}))

	view, err := repo.GetByOrgID(ctx, orgID)
	require.NoError(t, err)
	assert.Equal(t, []string{"development", "design_ui_ux"}, view.Profile.ExpertiseDomains)

	// Replace with a different list.
	require.NoError(t, repo.UpdateExpertiseDomains(ctx, orgID, []string{"marketing"}))
	view, err = repo.GetByOrgID(ctx, orgID)
	require.NoError(t, err)
	assert.Equal(t, []string{"marketing"}, view.Profile.ExpertiseDomains)

	// Nil clears.
	require.NoError(t, repo.UpdateExpertiseDomains(ctx, orgID, nil))
	view, err = repo.GetByOrgID(ctx, orgID)
	require.NoError(t, err)
	assert.Equal(t, []string{}, view.Profile.ExpertiseDomains)
}

func TestFreelanceProfileRepository_UpdateCore_NotFound(t *testing.T) {
	db := testDB(t)
	repo := postgres.NewFreelanceProfileRepository(db)

	err := repo.UpdateCore(context.Background(), uuid.New(), "x", "y", "z")
	assert.ErrorIs(t, err, freelanceprofile.ErrProfileNotFound)
}

func TestFreelanceProfileRepository_UpdateVideo_PersistsAndClearsInIsolation(t *testing.T) {
	db := testDB(t)
	orgID, _ := newTestFreelanceOrg(t)
	repo := postgres.NewFreelanceProfileRepository(db)
	ctx := context.Background()

	// Seed title/about so we can assert UpdateVideo never touches them.
	require.NoError(t, repo.UpdateCore(ctx, orgID, "Senior Go Engineer", "Builds marketplaces.", ""))

	require.NoError(t, repo.UpdateVideo(ctx, orgID, "https://example.com/intro.mp4"))
	view, err := repo.GetByOrgID(ctx, orgID)
	require.NoError(t, err)
	assert.Equal(t, "https://example.com/intro.mp4", view.Profile.VideoURL)
	assert.Equal(t, "Senior Go Engineer", view.Profile.Title)
	assert.Equal(t, "Builds marketplaces.", view.Profile.About)

	// Empty string clears the column (used by the DELETE path).
	require.NoError(t, repo.UpdateVideo(ctx, orgID, ""))
	view, err = repo.GetByOrgID(ctx, orgID)
	require.NoError(t, err)
	assert.Equal(t, "", view.Profile.VideoURL)
	assert.Equal(t, "Senior Go Engineer", view.Profile.Title)
}

func TestFreelanceProfileRepository_UpdateVideo_NotFound(t *testing.T) {
	db := testDB(t)
	repo := postgres.NewFreelanceProfileRepository(db)

	err := repo.UpdateVideo(context.Background(), uuid.New(), "https://example.com/v.mp4")
	assert.ErrorIs(t, err, freelanceprofile.ErrProfileNotFound)
}

func TestFreelanceProfileRepository_GetVideoURL_ReturnsCurrentValueOrNotFound(t *testing.T) {
	db := testDB(t)
	orgID, _ := newTestFreelanceOrg(t)
	repo := postgres.NewFreelanceProfileRepository(db)
	ctx := context.Background()

	// Default seed leaves video_url empty.
	got, err := repo.GetVideoURL(ctx, orgID)
	require.NoError(t, err)
	assert.Equal(t, "", got)

	require.NoError(t, repo.UpdateVideo(ctx, orgID, "https://example.com/v.mp4"))
	got, err = repo.GetVideoURL(ctx, orgID)
	require.NoError(t, err)
	assert.Equal(t, "https://example.com/v.mp4", got)

	_, err = repo.GetVideoURL(ctx, uuid.New())
	assert.ErrorIs(t, err, freelanceprofile.ErrProfileNotFound)
}
