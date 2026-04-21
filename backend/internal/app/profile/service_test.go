package profileapp

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/domain/profile"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/port/service"
)

// --- mock ---

type mockProfileRepo struct {
	getByOrgIDFn          func(ctx context.Context, orgID uuid.UUID) (*profile.Profile, error)
	updateFn              func(ctx context.Context, p *profile.Profile) error
	createFn              func(ctx context.Context, p *profile.Profile) error
	searchPublicFn        func(ctx context.Context, orgTypeFilter string, referrerOnly bool, cursor string, limit int) ([]*profile.PublicProfile, string, error)
	updateLocationFn          func(ctx context.Context, orgID uuid.UUID, input repository.LocationInput) error
	updateLanguagesFn         func(ctx context.Context, orgID uuid.UUID, pro, conv []string) error
	updateAvailabilityFn      func(ctx context.Context, orgID uuid.UUID, direct *profile.AvailabilityStatus, referrer *profile.AvailabilityStatus) error
	updateClientDescriptionFn func(ctx context.Context, orgID uuid.UUID, clientDescription string) error
}

func (m *mockProfileRepo) Create(ctx context.Context, p *profile.Profile) error {
	if m.createFn != nil {
		return m.createFn(ctx, p)
	}
	return nil
}

func (m *mockProfileRepo) GetByOrganizationID(ctx context.Context, orgID uuid.UUID) (*profile.Profile, error) {
	if m.getByOrgIDFn != nil {
		return m.getByOrgIDFn(ctx, orgID)
	}
	return nil, fmt.Errorf("profile not found")
}

func (m *mockProfileRepo) Update(ctx context.Context, p *profile.Profile) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, p)
	}
	return nil
}

func (m *mockProfileRepo) SearchPublic(ctx context.Context, orgTypeFilter string, referrerOnly bool, cursor string, limit int) ([]*profile.PublicProfile, string, error) {
	if m.searchPublicFn != nil {
		return m.searchPublicFn(ctx, orgTypeFilter, referrerOnly, cursor, limit)
	}
	return nil, "", nil
}

func (m *mockProfileRepo) GetPublicProfilesByOrgIDs(_ context.Context, _ []uuid.UUID) ([]*profile.PublicProfile, error) {
	return []*profile.PublicProfile{}, nil
}

func (m *mockProfileRepo) OrgProfilesByUserIDs(_ context.Context, _ []uuid.UUID) (map[uuid.UUID]*profile.PublicProfile, error) {
	return map[uuid.UUID]*profile.PublicProfile{}, nil
}

func (m *mockProfileRepo) UpdateLocation(ctx context.Context, orgID uuid.UUID, input repository.LocationInput) error {
	if m.updateLocationFn != nil {
		return m.updateLocationFn(ctx, orgID, input)
	}
	return nil
}

func (m *mockProfileRepo) UpdateLanguages(ctx context.Context, orgID uuid.UUID, pro, conv []string) error {
	if m.updateLanguagesFn != nil {
		return m.updateLanguagesFn(ctx, orgID, pro, conv)
	}
	return nil
}

func (m *mockProfileRepo) UpdateAvailability(ctx context.Context, orgID uuid.UUID, direct *profile.AvailabilityStatus, referrer *profile.AvailabilityStatus) error {
	if m.updateAvailabilityFn != nil {
		return m.updateAvailabilityFn(ctx, orgID, direct, referrer)
	}
	return nil
}

func (m *mockProfileRepo) UpdateClientDescription(ctx context.Context, orgID uuid.UUID, clientDescription string) error {
	if m.updateClientDescriptionFn != nil {
		return m.updateClientDescriptionFn(ctx, orgID, clientDescription)
	}
	return nil
}

// --- mock Geocoder ---

type mockGeocoder struct {
	fn func(ctx context.Context, city, country string) (float64, float64, error)
}

func (m *mockGeocoder) Geocode(ctx context.Context, city, country string) (float64, float64, error) {
	if m.fn == nil {
		return 0, 0, service.ErrGeocodingFailed
	}
	return m.fn(ctx, city, country)
}

// --- helpers ---

func newTestProfileService(repo *mockProfileRepo) *Service {
	if repo == nil {
		repo = &mockProfileRepo{}
	}
	return NewService(repo)
}

func existingProfile(orgID uuid.UUID) *profile.Profile {
	p := profile.NewProfile(orgID)
	p.Title = "Go Developer"
	p.About = "I build backend systems"
	return p
}

// --- GetProfile tests ---

func TestProfileService_GetProfile_Success(t *testing.T) {
	orgID := uuid.New()
	expected := existingProfile(orgID)

	repo := &mockProfileRepo{
		getByOrgIDFn: func(_ context.Context, id uuid.UUID) (*profile.Profile, error) {
			if id == orgID {
				return expected, nil
			}
			return nil, fmt.Errorf("profile not found")
		},
	}

	svc := newTestProfileService(repo)

	result, err := svc.GetProfile(context.Background(), orgID)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, orgID, result.OrganizationID)
	assert.Equal(t, "Go Developer", result.Title)
	assert.Equal(t, "I build backend systems", result.About)
}

func TestProfileService_GetProfile_NotFound(t *testing.T) {
	repo := &mockProfileRepo{
		getByOrgIDFn: func(_ context.Context, _ uuid.UUID) (*profile.Profile, error) {
			return nil, fmt.Errorf("profile not found")
		},
	}

	svc := newTestProfileService(repo)

	result, err := svc.GetProfile(context.Background(), uuid.New())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "get profile")
	assert.Nil(t, result)
}

// --- UpdateProfile tests ---

func TestProfileService_UpdateProfile_Success(t *testing.T) {
	orgID := uuid.New()
	existing := existingProfile(orgID)
	var updatedProfile *profile.Profile

	repo := &mockProfileRepo{
		getByOrgIDFn: func(_ context.Context, id uuid.UUID) (*profile.Profile, error) {
			if id == orgID {
				return existing, nil
			}
			return nil, fmt.Errorf("profile not found")
		},
		updateFn: func(_ context.Context, p *profile.Profile) error {
			updatedProfile = p
			return nil
		},
	}

	svc := newTestProfileService(repo)

	input := UpdateProfileInput{
		Title: "Senior Go Developer",
		About: "Experienced backend engineer",
	}

	result, err := svc.UpdateProfile(context.Background(), orgID, input)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "Senior Go Developer", result.Title)
	assert.Equal(t, "Experienced backend engineer", result.About)
	assert.NotNil(t, updatedProfile, "profile should have been persisted")
}

func TestProfileService_UpdateProfile_PartialUpdate(t *testing.T) {
	orgID := uuid.New()
	existing := existingProfile(orgID)

	repo := &mockProfileRepo{
		getByOrgIDFn: func(_ context.Context, _ uuid.UUID) (*profile.Profile, error) {
			return existing, nil
		},
	}

	svc := newTestProfileService(repo)

	input := UpdateProfileInput{
		Title: "Updated Title",
	}

	result, err := svc.UpdateProfile(context.Background(), orgID, input)

	require.NoError(t, err)
	assert.Equal(t, "Updated Title", result.Title)
	assert.Equal(t, "I build backend systems", result.About, "about should remain unchanged")
}

func TestProfileService_UpdateProfile_EmptyInputKeepsExisting(t *testing.T) {
	orgID := uuid.New()
	existing := existingProfile(orgID)

	repo := &mockProfileRepo{
		getByOrgIDFn: func(_ context.Context, _ uuid.UUID) (*profile.Profile, error) {
			return existing, nil
		},
	}

	svc := newTestProfileService(repo)

	input := UpdateProfileInput{}

	result, err := svc.UpdateProfile(context.Background(), orgID, input)

	require.NoError(t, err)
	assert.Equal(t, "Go Developer", result.Title, "title should remain unchanged")
	assert.Equal(t, "I build backend systems", result.About, "about should remain unchanged")
}

func TestProfileService_UpdateProfile_ProfileNotFound(t *testing.T) {
	repo := &mockProfileRepo{
		getByOrgIDFn: func(_ context.Context, _ uuid.UUID) (*profile.Profile, error) {
			return nil, fmt.Errorf("profile not found")
		},
	}

	svc := newTestProfileService(repo)

	result, err := svc.UpdateProfile(context.Background(), uuid.New(), UpdateProfileInput{
		Title: "New Title",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "get profile")
	assert.Nil(t, result)
}

func TestProfileService_UpdateProfile_PersistenceFailure(t *testing.T) {
	orgID := uuid.New()
	existing := existingProfile(orgID)

	repo := &mockProfileRepo{
		getByOrgIDFn: func(_ context.Context, _ uuid.UUID) (*profile.Profile, error) {
			return existing, nil
		},
		updateFn: func(_ context.Context, _ *profile.Profile) error {
			return fmt.Errorf("database connection lost")
		},
	}

	svc := newTestProfileService(repo)

	result, err := svc.UpdateProfile(context.Background(), orgID, UpdateProfileInput{
		Title: "New Title",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "update profile")
	assert.Nil(t, result)
}

func TestProfileService_UpdateProfile_ReferrerFields(t *testing.T) {
	orgID := uuid.New()
	existing := existingProfile(orgID)

	repo := &mockProfileRepo{
		getByOrgIDFn: func(_ context.Context, _ uuid.UUID) (*profile.Profile, error) {
			return existing, nil
		},
	}

	svc := newTestProfileService(repo)

	input := UpdateProfileInput{
		ReferrerAbout:    "I connect talent with opportunity",
		ReferrerVideoURL: "https://example.com/referrer-video.mp4",
	}

	result, err := svc.UpdateProfile(context.Background(), orgID, input)

	require.NoError(t, err)
	assert.Equal(t, "I connect talent with opportunity", result.ReferrerAbout)
	assert.Equal(t, "https://example.com/referrer-video.mp4", result.ReferrerVideoURL)
}

// --- SearchPublic tests ---

func TestProfileService_SearchPublic_Success(t *testing.T) {
	expected := []*profile.PublicProfile{
		{
			OrganizationID: uuid.New(),
			Name:           "John Doe",
			OrgType:        "provider_personal",
			Title:          "Go Developer",
		},
		{
			OrganizationID: uuid.New(),
			Name:           "Jane Agency",
			OrgType:        "agency",
			Title:          "Full Stack Agency",
		},
	}

	repo := &mockProfileRepo{
		searchPublicFn: func(_ context.Context, _ string, _ bool, _ string, _ int) ([]*profile.PublicProfile, string, error) {
			return expected, "", nil
		},
	}

	svc := newTestProfileService(repo)

	results, nextCursor, err := svc.SearchPublic(context.Background(), "", false, "", 20)

	require.NoError(t, err)
	require.Len(t, results, 2)
	assert.Equal(t, "John Doe", results[0].Name)
	assert.Equal(t, "Jane Agency", results[1].Name)
	assert.Empty(t, nextCursor)
}

func TestProfileService_SearchPublic_WithOrgTypeFilter(t *testing.T) {
	var capturedOrgType string

	repo := &mockProfileRepo{
		searchPublicFn: func(_ context.Context, orgTypeFilter string, _ bool, _ string, _ int) ([]*profile.PublicProfile, string, error) {
			capturedOrgType = orgTypeFilter
			return []*profile.PublicProfile{}, "", nil
		},
	}

	svc := newTestProfileService(repo)

	_, _, err := svc.SearchPublic(context.Background(), "provider_personal", false, "", 20)

	require.NoError(t, err)
	assert.Equal(t, "provider_personal", capturedOrgType)
}

func TestProfileService_SearchPublic_ReferrerOnly(t *testing.T) {
	var capturedReferrerOnly bool

	repo := &mockProfileRepo{
		searchPublicFn: func(_ context.Context, _ string, referrerOnly bool, _ string, _ int) ([]*profile.PublicProfile, string, error) {
			capturedReferrerOnly = referrerOnly
			return []*profile.PublicProfile{}, "", nil
		},
	}

	svc := newTestProfileService(repo)

	_, _, err := svc.SearchPublic(context.Background(), "", true, "", 20)

	require.NoError(t, err)
	assert.True(t, capturedReferrerOnly)
}

func TestProfileService_SearchPublic_EmptyResult(t *testing.T) {
	repo := &mockProfileRepo{
		searchPublicFn: func(_ context.Context, _ string, _ bool, _ string, _ int) ([]*profile.PublicProfile, string, error) {
			return []*profile.PublicProfile{}, "", nil
		},
	}

	svc := newTestProfileService(repo)

	results, nextCursor, err := svc.SearchPublic(context.Background(), "", false, "", 20)

	require.NoError(t, err)
	assert.Empty(t, results)
	assert.Empty(t, nextCursor)
}

func TestProfileService_SearchPublic_RepositoryFailure(t *testing.T) {
	repo := &mockProfileRepo{
		searchPublicFn: func(_ context.Context, _ string, _ bool, _ string, _ int) ([]*profile.PublicProfile, string, error) {
			return nil, "", fmt.Errorf("database timeout")
		},
	}

	svc := newTestProfileService(repo)

	results, nextCursor, err := svc.SearchPublic(context.Background(), "", false, "", 20)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "search public profiles")
	assert.Nil(t, results)
	assert.Empty(t, nextCursor)
}

func TestProfileService_SearchPublic_LimitPassthrough(t *testing.T) {
	var capturedLimit int

	repo := &mockProfileRepo{
		searchPublicFn: func(_ context.Context, _ string, _ bool, _ string, limit int) ([]*profile.PublicProfile, string, error) {
			capturedLimit = limit
			return []*profile.PublicProfile{}, "", nil
		},
	}

	svc := newTestProfileService(repo)

	_, _, err := svc.SearchPublic(context.Background(), "", false, "", 50)

	require.NoError(t, err)
	assert.Equal(t, 50, capturedLimit)
}

// -----------------------------------------------------------------
// Tier 1 completion: UpdateLocation
// -----------------------------------------------------------------

func TestProfileService_UpdateLocation_NormalizesAndPersists(t *testing.T) {
	orgID := uuid.New()
	var captured repository.LocationInput
	repo := &mockProfileRepo{
		updateLocationFn: func(_ context.Context, _ uuid.UUID, in repository.LocationInput) error {
			captured = in
			return nil
		},
	}
	svc := NewService(repo)

	radius := 50
	err := svc.UpdateLocation(context.Background(), orgID, UpdateLocationInput{
		City:           "  Paris  ",
		CountryCode:    "  fr ",
		WorkMode:       []string{"remote", "remote", "nomad", "hybrid"},
		TravelRadiusKm: &radius,
	})

	require.NoError(t, err)
	assert.Equal(t, "Paris", captured.City, "city should be trimmed")
	assert.Equal(t, "FR", captured.CountryCode, "country code should be upper + trimmed")
	assert.Equal(t, []string{"remote", "hybrid"}, captured.WorkMode,
		"work modes should be deduped and filtered")
	require.NotNil(t, captured.TravelRadiusKm)
	assert.Equal(t, 50, *captured.TravelRadiusKm)
	// Without a geocoder, coordinates stay nil.
	assert.Nil(t, captured.Latitude)
	assert.Nil(t, captured.Longitude)
}

func TestProfileService_UpdateLocation_InvalidCountryCode(t *testing.T) {
	repo := &mockProfileRepo{
		updateLocationFn: func(_ context.Context, _ uuid.UUID, _ repository.LocationInput) error {
			t.Fatal("repository should not be called on validation failure")
			return nil
		},
	}
	svc := NewService(repo)

	err := svc.UpdateLocation(context.Background(), uuid.New(), UpdateLocationInput{
		City:        "Paris",
		CountryCode: "france",
	})
	assert.ErrorIs(t, err, profile.ErrInvalidCountryCode)
}

func TestProfileService_UpdateLocation_WithGeocoder_SuccessAttachesCoords(t *testing.T) {
	var captured repository.LocationInput
	repo := &mockProfileRepo{
		updateLocationFn: func(_ context.Context, _ uuid.UUID, in repository.LocationInput) error {
			captured = in
			return nil
		},
	}
	geo := &mockGeocoder{
		fn: func(_ context.Context, city, country string) (float64, float64, error) {
			assert.Equal(t, "Paris", city)
			assert.Equal(t, "FR", country)
			return 48.8566, 2.3522, nil
		},
	}
	svc := NewService(repo).WithGeocoder(geo)

	err := svc.UpdateLocation(context.Background(), uuid.New(), UpdateLocationInput{
		City:        "Paris",
		CountryCode: "FR",
	})
	require.NoError(t, err)
	require.NotNil(t, captured.Latitude)
	require.NotNil(t, captured.Longitude)
	assert.InDelta(t, 48.8566, *captured.Latitude, 0.0001)
	assert.InDelta(t, 2.3522, *captured.Longitude, 0.0001)
}

func TestProfileService_UpdateLocation_GeocoderFailure_StillSavesWithoutCoords(t *testing.T) {
	var captured repository.LocationInput
	repo := &mockProfileRepo{
		updateLocationFn: func(_ context.Context, _ uuid.UUID, in repository.LocationInput) error {
			captured = in
			return nil
		},
	}
	geo := &mockGeocoder{
		fn: func(_ context.Context, _, _ string) (float64, float64, error) {
			return 0, 0, service.ErrGeocodingFailed
		},
	}
	svc := NewService(repo).WithGeocoder(geo)

	err := svc.UpdateLocation(context.Background(), uuid.New(), UpdateLocationInput{
		City:        "Atlantis",
		CountryCode: "FR",
	})
	require.NoError(t, err, "geocoder failure must not fail the save")
	assert.Nil(t, captured.Latitude)
	assert.Nil(t, captured.Longitude)
	assert.Equal(t, "Atlantis", captured.City, "city still persisted")
}

func TestProfileService_UpdateLocation_ClientCoords_SkipsGeocoder(t *testing.T) {
	var captured repository.LocationInput
	var geocodeCalled bool
	repo := &mockProfileRepo{
		updateLocationFn: func(_ context.Context, _ uuid.UUID, in repository.LocationInput) error {
			captured = in
			return nil
		},
	}
	geo := &mockGeocoder{
		fn: func(_ context.Context, _, _ string) (float64, float64, error) {
			geocodeCalled = true
			return 99, 99, nil
		},
	}
	svc := NewService(repo).WithGeocoder(geo)

	clientLat := 45.758
	clientLng := 4.835
	err := svc.UpdateLocation(context.Background(), uuid.New(), UpdateLocationInput{
		City:        "Lyon",
		CountryCode: "FR",
		Latitude:    &clientLat,
		Longitude:   &clientLng,
	})
	require.NoError(t, err)
	assert.False(t, geocodeCalled, "client-supplied coords must short-circuit the geocoder")
	require.NotNil(t, captured.Latitude)
	require.NotNil(t, captured.Longitude)
	assert.InDelta(t, 45.758, *captured.Latitude, 0.0001)
	assert.InDelta(t, 4.835, *captured.Longitude, 0.0001)
}

func TestProfileService_UpdateLocation_PartialClientCoords_FallsBackToGeocoder(t *testing.T) {
	var captured repository.LocationInput
	repo := &mockProfileRepo{
		updateLocationFn: func(_ context.Context, _ uuid.UUID, in repository.LocationInput) error {
			captured = in
			return nil
		},
	}
	geo := &mockGeocoder{
		fn: func(_ context.Context, _, _ string) (float64, float64, error) {
			return 48.8566, 2.3522, nil
		},
	}
	svc := NewService(repo).WithGeocoder(geo)

	clientLat := 45.758
	err := svc.UpdateLocation(context.Background(), uuid.New(), UpdateLocationInput{
		City:        "Paris",
		CountryCode: "FR",
		Latitude:    &clientLat, // only one of the two → fall back to geocoder
	})
	require.NoError(t, err)
	require.NotNil(t, captured.Latitude)
	assert.InDelta(t, 48.8566, *captured.Latitude, 0.0001,
		"partial client coords must trigger the geocoder fallback")
}

func TestProfileService_UpdateLocation_EmptyCityCountry_SkipsGeocoder(t *testing.T) {
	var geocodeCalled bool
	repo := &mockProfileRepo{}
	geo := &mockGeocoder{
		fn: func(_ context.Context, _, _ string) (float64, float64, error) {
			geocodeCalled = true
			return 0, 0, nil
		},
	}
	svc := NewService(repo).WithGeocoder(geo)

	err := svc.UpdateLocation(context.Background(), uuid.New(), UpdateLocationInput{
		City:        "",
		CountryCode: "",
	})
	require.NoError(t, err)
	assert.False(t, geocodeCalled, "empty inputs should not trigger a geocode call")
}

func TestProfileService_UpdateLocation_PersistError(t *testing.T) {
	repo := &mockProfileRepo{
		updateLocationFn: func(_ context.Context, _ uuid.UUID, _ repository.LocationInput) error {
			return errors.New("db blew up")
		},
	}
	svc := NewService(repo)

	err := svc.UpdateLocation(context.Background(), uuid.New(), UpdateLocationInput{
		City:        "Paris",
		CountryCode: "FR",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "update location")
	assert.Contains(t, err.Error(), "persist")
}

// -----------------------------------------------------------------
// Tier 1 completion: UpdateLanguages
// -----------------------------------------------------------------

func TestProfileService_UpdateLanguages_NormalizesAndPersists(t *testing.T) {
	var capPro, capConv []string
	repo := &mockProfileRepo{
		updateLanguagesFn: func(_ context.Context, _ uuid.UUID, pro, conv []string) error {
			capPro = pro
			capConv = conv
			return nil
		},
	}
	svc := NewService(repo)

	err := svc.UpdateLanguages(context.Background(), uuid.New(),
		[]string{"fr", "fr", "FR", "en", "english"},
		[]string{"es", "it", "xx"},
	)
	require.NoError(t, err)
	assert.Equal(t, []string{"fr", "en"}, capPro)
	assert.Equal(t, []string{"es", "it", "xx"}, capConv,
		"xx passes the 2-letter shape check — validation is intentionally lenient")
}

func TestProfileService_UpdateLanguages_PersistError(t *testing.T) {
	repo := &mockProfileRepo{
		updateLanguagesFn: func(_ context.Context, _ uuid.UUID, _, _ []string) error {
			return errors.New("disk full")
		},
	}
	svc := NewService(repo)

	err := svc.UpdateLanguages(context.Background(), uuid.New(), []string{"fr"}, []string{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "update languages")
}

// -----------------------------------------------------------------
// Tier 1 completion: UpdateAvailability
// -----------------------------------------------------------------

func TestProfileService_UpdateAvailability_DirectOnly(t *testing.T) {
	var capDirect *profile.AvailabilityStatus
	var capRef *profile.AvailabilityStatus
	repo := &mockProfileRepo{
		updateAvailabilityFn: func(_ context.Context, _ uuid.UUID, d *profile.AvailabilityStatus, r *profile.AvailabilityStatus) error {
			capDirect = d
			capRef = r
			return nil
		},
	}
	svc := NewService(repo)

	direct := profile.AvailabilitySoon
	err := svc.UpdateAvailability(context.Background(), uuid.New(), &direct, nil)
	require.NoError(t, err)
	require.NotNil(t, capDirect)
	assert.Equal(t, profile.AvailabilitySoon, *capDirect)
	assert.Nil(t, capRef)
}

func TestProfileService_UpdateAvailability_WithReferrerSlot(t *testing.T) {
	var capDirect *profile.AvailabilityStatus
	var capRef *profile.AvailabilityStatus
	repo := &mockProfileRepo{
		updateAvailabilityFn: func(_ context.Context, _ uuid.UUID, d *profile.AvailabilityStatus, r *profile.AvailabilityStatus) error {
			capDirect = d
			capRef = r
			return nil
		},
	}
	svc := NewService(repo)

	refStatus := profile.AvailabilityNot
	err := svc.UpdateAvailability(context.Background(), uuid.New(), nil, &refStatus)
	require.NoError(t, err)
	assert.Nil(t, capDirect)
	require.NotNil(t, capRef)
	assert.Equal(t, profile.AvailabilityNot, *capRef)
}

func TestProfileService_UpdateAvailability_NoneProvided(t *testing.T) {
	repo := &mockProfileRepo{
		updateAvailabilityFn: func(_ context.Context, _ uuid.UUID, _ *profile.AvailabilityStatus, _ *profile.AvailabilityStatus) error {
			t.Fatal("repository should not be called when nothing is provided")
			return nil
		},
	}
	svc := NewService(repo)

	err := svc.UpdateAvailability(context.Background(), uuid.New(), nil, nil)
	assert.ErrorIs(t, err, profile.ErrInvalidAvailabilityStatus)
}

func TestProfileService_UpdateAvailability_InvalidDirect(t *testing.T) {
	repo := &mockProfileRepo{
		updateAvailabilityFn: func(_ context.Context, _ uuid.UUID, _ *profile.AvailabilityStatus, _ *profile.AvailabilityStatus) error {
			t.Fatal("repository should not be called on validation failure")
			return nil
		},
	}
	svc := NewService(repo)

	bad := profile.AvailabilityStatus("maybe")
	err := svc.UpdateAvailability(context.Background(), uuid.New(), &bad, nil)
	assert.ErrorIs(t, err, profile.ErrInvalidAvailabilityStatus)
}

func TestProfileService_UpdateAvailability_InvalidReferrer(t *testing.T) {
	repo := &mockProfileRepo{}
	svc := NewService(repo)

	direct := profile.AvailabilityNow
	bad := profile.AvailabilityStatus("sort of")
	err := svc.UpdateAvailability(context.Background(), uuid.New(), &direct, &bad)
	assert.ErrorIs(t, err, profile.ErrInvalidAvailabilityStatus)
}

func TestProfileService_UpdateAvailability_PersistError(t *testing.T) {
	repo := &mockProfileRepo{
		updateAvailabilityFn: func(_ context.Context, _ uuid.UUID, _ *profile.AvailabilityStatus, _ *profile.AvailabilityStatus) error {
			return errors.New("db gone")
		},
	}
	svc := NewService(repo)

	direct := profile.AvailabilityNow
	err := svc.UpdateAvailability(context.Background(), uuid.New(), &direct, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "update availability")
}

// Safety check: WithGeocoder(nil) must be a no-op — keeps existing
// call sites that pass nil from crashing.
func TestProfileService_WithGeocoder_NilIsNoOp(t *testing.T) {
	svc := NewService(&mockProfileRepo{}).WithGeocoder(nil)
	require.NotNil(t, svc)
	assert.Nil(t, svc.geocoder)
}
