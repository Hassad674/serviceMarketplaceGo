package profileapp

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/domain/profile"
)

// mockSocialLinkRepo is a manual mock for repository.SocialLinkRepository.
type mockSocialLinkRepo struct {
	listByUserFn func(ctx context.Context, userID uuid.UUID) ([]*profile.SocialLink, error)
	upsertFn     func(ctx context.Context, link *profile.SocialLink) error
	deleteFn     func(ctx context.Context, userID uuid.UUID, platform string) error
}

func (m *mockSocialLinkRepo) ListByUser(ctx context.Context, userID uuid.UUID) ([]*profile.SocialLink, error) {
	return m.listByUserFn(ctx, userID)
}

func (m *mockSocialLinkRepo) Upsert(ctx context.Context, link *profile.SocialLink) error {
	return m.upsertFn(ctx, link)
}

func (m *mockSocialLinkRepo) Delete(ctx context.Context, userID uuid.UUID, platform string) error {
	return m.deleteFn(ctx, userID, platform)
}

func TestSocialLinkService_Upsert_ValidPlatform(t *testing.T) {
	var captured *profile.SocialLink
	repo := &mockSocialLinkRepo{
		upsertFn: func(_ context.Context, link *profile.SocialLink) error {
			captured = link
			return nil
		},
	}
	svc := NewSocialLinkService(repo)

	err := svc.Upsert(context.Background(), uuid.New(), UpsertInput{
		Platform: "LinkedIn",
		URL:      "https://linkedin.com/in/user",
	})

	require.NoError(t, err)
	assert.Equal(t, "linkedin", captured.Platform)
	assert.Equal(t, "https://linkedin.com/in/user", captured.URL)
}

func TestSocialLinkService_Upsert_InvalidPlatform(t *testing.T) {
	repo := &mockSocialLinkRepo{}
	svc := NewSocialLinkService(repo)

	err := svc.Upsert(context.Background(), uuid.New(), UpsertInput{
		Platform: "tiktok",
		URL:      "https://tiktok.com/@user",
	})

	assert.ErrorIs(t, err, profile.ErrInvalidPlatform)
}

func TestSocialLinkService_Upsert_InvalidURL(t *testing.T) {
	repo := &mockSocialLinkRepo{}
	svc := NewSocialLinkService(repo)

	err := svc.Upsert(context.Background(), uuid.New(), UpsertInput{
		Platform: "github",
		URL:      "not-a-url",
	})

	assert.ErrorIs(t, err, profile.ErrInvalidURL)
}

func TestSocialLinkService_Delete_ValidPlatform(t *testing.T) {
	var deletedPlatform string
	repo := &mockSocialLinkRepo{
		deleteFn: func(_ context.Context, _ uuid.UUID, platform string) error {
			deletedPlatform = platform
			return nil
		},
	}
	svc := NewSocialLinkService(repo)

	err := svc.Delete(context.Background(), uuid.New(), "GitHub")

	require.NoError(t, err)
	assert.Equal(t, "github", deletedPlatform)
}

func TestSocialLinkService_Delete_InvalidPlatform(t *testing.T) {
	repo := &mockSocialLinkRepo{}
	svc := NewSocialLinkService(repo)

	err := svc.Delete(context.Background(), uuid.New(), "snapchat")

	assert.ErrorIs(t, err, profile.ErrInvalidPlatform)
}

func TestSocialLinkService_ListByUser(t *testing.T) {
	userID := uuid.New()
	expected := []*profile.SocialLink{
		{UserID: userID, Platform: "github", URL: "https://github.com/user"},
		{UserID: userID, Platform: "linkedin", URL: "https://linkedin.com/in/user"},
	}
	repo := &mockSocialLinkRepo{
		listByUserFn: func(_ context.Context, _ uuid.UUID) ([]*profile.SocialLink, error) {
			return expected, nil
		},
	}
	svc := NewSocialLinkService(repo)

	links, err := svc.ListByUser(context.Background(), userID)

	require.NoError(t, err)
	assert.Len(t, links, 2)
	assert.Equal(t, "github", links[0].Platform)
}
