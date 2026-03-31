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
	getByUserIDFn   func(ctx context.Context, userID uuid.UUID) (*profile.Profile, error)
	updateFn        func(ctx context.Context, p *profile.Profile) error
	createFn        func(ctx context.Context, p *profile.Profile) error
	searchPublicFn  func(ctx context.Context, roleFilter string, referrerOnly bool, limit int) ([]*profile.PublicProfile, error)
}

func (m *mockProfileRepo) Create(ctx context.Context, p *profile.Profile) error {
	if m.createFn != nil {
		return m.createFn(ctx, p)
	}
	return nil
}

func (m *mockProfileRepo) GetByUserID(ctx context.Context, userID uuid.UUID) (*profile.Profile, error) {
	if m.getByUserIDFn != nil {
		return m.getByUserIDFn(ctx, userID)
	}
	return nil, fmt.Errorf("profile not found")
}

func (m *mockProfileRepo) Update(ctx context.Context, p *profile.Profile) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, p)
	}
	return nil
}

func (m *mockProfileRepo) SearchPublic(ctx context.Context, roleFilter string, referrerOnly bool, limit int) ([]*profile.PublicProfile, error) {
	if m.searchPublicFn != nil {
		return m.searchPublicFn(ctx, roleFilter, referrerOnly, limit)
	}
	return nil, nil
}

func (m *mockProfileRepo) GetPublicProfilesByUserIDs(_ context.Context, _ []uuid.UUID) ([]*profile.PublicProfile, error) {
	return []*profile.PublicProfile{}, nil
}

// --- helpers ---

func newTestProfileService(repo *mockProfileRepo) *Service {
	if repo == nil {
		repo = &mockProfileRepo{}
	}
	return NewService(repo)
}

func existingProfile(userID uuid.UUID) *profile.Profile {
	p := profile.NewProfile(userID)
	p.Title = "Go Developer"
	p.About = "I build backend systems"
	return p
}

// --- GetProfile tests ---

func TestProfileService_GetProfile_Success(t *testing.T) {
	userID := uuid.New()
	expected := existingProfile(userID)

	repo := &mockProfileRepo{
		getByUserIDFn: func(_ context.Context, id uuid.UUID) (*profile.Profile, error) {
			if id == userID {
				return expected, nil
			}
			return nil, fmt.Errorf("profile not found")
		},
	}

	svc := newTestProfileService(repo)

	result, err := svc.GetProfile(context.Background(), userID)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, userID, result.UserID)
	assert.Equal(t, "Go Developer", result.Title)
	assert.Equal(t, "I build backend systems", result.About)
}

func TestProfileService_GetProfile_NotFound(t *testing.T) {
	repo := &mockProfileRepo{
		getByUserIDFn: func(_ context.Context, _ uuid.UUID) (*profile.Profile, error) {
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
	userID := uuid.New()
	existing := existingProfile(userID)
	var updatedProfile *profile.Profile

	repo := &mockProfileRepo{
		getByUserIDFn: func(_ context.Context, id uuid.UUID) (*profile.Profile, error) {
			if id == userID {
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

	result, err := svc.UpdateProfile(context.Background(), userID, input)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "Senior Go Developer", result.Title)
	assert.Equal(t, "Experienced backend engineer", result.About)
	assert.NotNil(t, updatedProfile, "profile should have been persisted")
}

func TestProfileService_UpdateProfile_PartialUpdate(t *testing.T) {
	userID := uuid.New()
	existing := existingProfile(userID)

	repo := &mockProfileRepo{
		getByUserIDFn: func(_ context.Context, _ uuid.UUID) (*profile.Profile, error) {
			return existing, nil
		},
	}

	svc := newTestProfileService(repo)

	// Only update title, leave about unchanged
	input := UpdateProfileInput{
		Title: "Updated Title",
	}

	result, err := svc.UpdateProfile(context.Background(), userID, input)

	require.NoError(t, err)
	assert.Equal(t, "Updated Title", result.Title)
	assert.Equal(t, "I build backend systems", result.About, "about should remain unchanged")
}

func TestProfileService_UpdateProfile_EmptyInputKeepsExisting(t *testing.T) {
	userID := uuid.New()
	existing := existingProfile(userID)

	repo := &mockProfileRepo{
		getByUserIDFn: func(_ context.Context, _ uuid.UUID) (*profile.Profile, error) {
			return existing, nil
		},
	}

	svc := newTestProfileService(repo)

	// Empty input should leave everything as-is
	input := UpdateProfileInput{}

	result, err := svc.UpdateProfile(context.Background(), userID, input)

	require.NoError(t, err)
	assert.Equal(t, "Go Developer", result.Title, "title should remain unchanged")
	assert.Equal(t, "I build backend systems", result.About, "about should remain unchanged")
}

func TestProfileService_UpdateProfile_ProfileNotFound(t *testing.T) {
	repo := &mockProfileRepo{
		getByUserIDFn: func(_ context.Context, _ uuid.UUID) (*profile.Profile, error) {
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
	userID := uuid.New()
	existing := existingProfile(userID)

	repo := &mockProfileRepo{
		getByUserIDFn: func(_ context.Context, _ uuid.UUID) (*profile.Profile, error) {
			return existing, nil
		},
		updateFn: func(_ context.Context, _ *profile.Profile) error {
			return fmt.Errorf("database connection lost")
		},
	}

	svc := newTestProfileService(repo)

	result, err := svc.UpdateProfile(context.Background(), userID, UpdateProfileInput{
		Title: "New Title",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "update profile")
	assert.Nil(t, result)
}

func TestProfileService_UpdateProfile_ReferrerFields(t *testing.T) {
	userID := uuid.New()
	existing := existingProfile(userID)

	repo := &mockProfileRepo{
		getByUserIDFn: func(_ context.Context, _ uuid.UUID) (*profile.Profile, error) {
			return existing, nil
		},
	}

	svc := newTestProfileService(repo)

	input := UpdateProfileInput{
		ReferrerAbout:    "I connect talent with opportunity",
		ReferrerVideoURL: "https://example.com/referrer-video.mp4",
	}

	result, err := svc.UpdateProfile(context.Background(), userID, input)

	require.NoError(t, err)
	assert.Equal(t, "I connect talent with opportunity", result.ReferrerAbout)
	assert.Equal(t, "https://example.com/referrer-video.mp4", result.ReferrerVideoURL)
}

// --- SearchPublic tests ---

func TestProfileService_SearchPublic_Success(t *testing.T) {
	expected := []*profile.PublicProfile{
		{
			UserID:      uuid.New(),
			DisplayName: "John Doe",
			Role:        "provider",
			Title:       "Go Developer",
		},
		{
			UserID:      uuid.New(),
			DisplayName: "Jane Agency",
			Role:        "agency",
			Title:       "Full Stack Agency",
		},
	}

	repo := &mockProfileRepo{
		searchPublicFn: func(_ context.Context, roleFilter string, referrerOnly bool, limit int) ([]*profile.PublicProfile, error) {
			return expected, nil
		},
	}

	svc := newTestProfileService(repo)

	results, err := svc.SearchPublic(context.Background(), "", false, 20)

	require.NoError(t, err)
	require.Len(t, results, 2)
	assert.Equal(t, "John Doe", results[0].DisplayName)
	assert.Equal(t, "Jane Agency", results[1].DisplayName)
}

func TestProfileService_SearchPublic_WithRoleFilter(t *testing.T) {
	var capturedRole string

	repo := &mockProfileRepo{
		searchPublicFn: func(_ context.Context, roleFilter string, _ bool, _ int) ([]*profile.PublicProfile, error) {
			capturedRole = roleFilter
			return []*profile.PublicProfile{}, nil
		},
	}

	svc := newTestProfileService(repo)

	_, err := svc.SearchPublic(context.Background(), "provider", false, 20)

	require.NoError(t, err)
	assert.Equal(t, "provider", capturedRole)
}

func TestProfileService_SearchPublic_ReferrerOnly(t *testing.T) {
	var capturedReferrerOnly bool

	repo := &mockProfileRepo{
		searchPublicFn: func(_ context.Context, _ string, referrerOnly bool, _ int) ([]*profile.PublicProfile, error) {
			capturedReferrerOnly = referrerOnly
			return []*profile.PublicProfile{}, nil
		},
	}

	svc := newTestProfileService(repo)

	_, err := svc.SearchPublic(context.Background(), "", true, 20)

	require.NoError(t, err)
	assert.True(t, capturedReferrerOnly)
}

func TestProfileService_SearchPublic_EmptyResult(t *testing.T) {
	repo := &mockProfileRepo{
		searchPublicFn: func(_ context.Context, _ string, _ bool, _ int) ([]*profile.PublicProfile, error) {
			return []*profile.PublicProfile{}, nil
		},
	}

	svc := newTestProfileService(repo)

	results, err := svc.SearchPublic(context.Background(), "", false, 20)

	require.NoError(t, err)
	assert.Empty(t, results)
}

func TestProfileService_SearchPublic_RepositoryFailure(t *testing.T) {
	repo := &mockProfileRepo{
		searchPublicFn: func(_ context.Context, _ string, _ bool, _ int) ([]*profile.PublicProfile, error) {
			return nil, fmt.Errorf("database timeout")
		},
	}

	svc := newTestProfileService(repo)

	results, err := svc.SearchPublic(context.Background(), "", false, 20)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "search public profiles")
	assert.Nil(t, results)
}

func TestProfileService_SearchPublic_LimitPassthrough(t *testing.T) {
	var capturedLimit int

	repo := &mockProfileRepo{
		searchPublicFn: func(_ context.Context, _ string, _ bool, limit int) ([]*profile.PublicProfile, error) {
			capturedLimit = limit
			return []*profile.PublicProfile{}, nil
		},
	}

	svc := newTestProfileService(repo)

	_, err := svc.SearchPublic(context.Background(), "", false, 50)

	require.NoError(t, err)
	assert.Equal(t, 50, capturedLimit)
}
