package profileapp

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/domain/profile"
)

// --- mock ---

type mockProfileRepo struct {
	getByOrgIDFn   func(ctx context.Context, orgID uuid.UUID) (*profile.Profile, error)
	updateFn       func(ctx context.Context, p *profile.Profile) error
	createFn       func(ctx context.Context, p *profile.Profile) error
	searchPublicFn func(ctx context.Context, orgTypeFilter string, referrerOnly bool, cursor string, limit int) ([]*profile.PublicProfile, string, error)
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
