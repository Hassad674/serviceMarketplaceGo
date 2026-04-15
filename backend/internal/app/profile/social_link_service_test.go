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
	listFn   func(ctx context.Context, orgID uuid.UUID, persona profile.SocialLinkPersona) ([]*profile.SocialLink, error)
	upsertFn func(ctx context.Context, link *profile.SocialLink) error
	deleteFn func(ctx context.Context, orgID uuid.UUID, persona profile.SocialLinkPersona, platform string) error
}

func (m *mockSocialLinkRepo) ListByOrganizationPersona(
	ctx context.Context,
	orgID uuid.UUID,
	persona profile.SocialLinkPersona,
) ([]*profile.SocialLink, error) {
	return m.listFn(ctx, orgID, persona)
}

func (m *mockSocialLinkRepo) Upsert(ctx context.Context, link *profile.SocialLink) error {
	return m.upsertFn(ctx, link)
}

func (m *mockSocialLinkRepo) Delete(
	ctx context.Context,
	orgID uuid.UUID,
	persona profile.SocialLinkPersona,
	platform string,
) error {
	return m.deleteFn(ctx, orgID, persona, platform)
}

func newSvc(t *testing.T, persona profile.SocialLinkPersona, repo *mockSocialLinkRepo) *SocialLinkService {
	t.Helper()
	svc, err := NewSocialLinkService(repo, persona)
	require.NoError(t, err)
	return svc
}

func TestNewSocialLinkService_InvalidPersona(t *testing.T) {
	repo := &mockSocialLinkRepo{}
	svc, err := NewSocialLinkService(repo, profile.SocialLinkPersona("admin"))
	assert.Nil(t, svc)
	assert.ErrorIs(t, err, profile.ErrInvalidPersona)
}

func TestSocialLinkService_Persona_IsScoped(t *testing.T) {
	personas := []profile.SocialLinkPersona{
		profile.PersonaFreelance,
		profile.PersonaReferrer,
		profile.PersonaAgency,
	}

	for _, persona := range personas {
		t.Run(string(persona), func(t *testing.T) {
			var captured *profile.SocialLink
			repo := &mockSocialLinkRepo{
				upsertFn: func(_ context.Context, link *profile.SocialLink) error {
					captured = link
					return nil
				},
			}
			svc := newSvc(t, persona, repo)

			err := svc.Upsert(context.Background(), uuid.New(), UpsertInput{
				Platform: "LinkedIn",
				URL:      "https://linkedin.com/in/user",
			})

			require.NoError(t, err)
			require.NotNil(t, captured)
			assert.Equal(t, persona, captured.Persona)
			assert.Equal(t, "linkedin", captured.Platform)
			assert.Equal(t, "https://linkedin.com/in/user", captured.URL)
		})
	}
}

func TestSocialLinkService_Upsert_InvalidPlatform(t *testing.T) {
	repo := &mockSocialLinkRepo{}
	svc := newSvc(t, profile.PersonaFreelance, repo)

	err := svc.Upsert(context.Background(), uuid.New(), UpsertInput{
		Platform: "tiktok",
		URL:      "https://tiktok.com/@user",
	})

	assert.ErrorIs(t, err, profile.ErrInvalidPlatform)
}

func TestSocialLinkService_Upsert_InvalidURL(t *testing.T) {
	repo := &mockSocialLinkRepo{}
	svc := newSvc(t, profile.PersonaReferrer, repo)

	err := svc.Upsert(context.Background(), uuid.New(), UpsertInput{
		Platform: "github",
		URL:      "not-a-url",
	})

	assert.ErrorIs(t, err, profile.ErrInvalidURL)
}

func TestSocialLinkService_Delete_ScopedByPersona(t *testing.T) {
	var capturedPersona profile.SocialLinkPersona
	var capturedPlatform string
	repo := &mockSocialLinkRepo{
		deleteFn: func(_ context.Context, _ uuid.UUID, persona profile.SocialLinkPersona, platform string) error {
			capturedPersona = persona
			capturedPlatform = platform
			return nil
		},
	}
	svc := newSvc(t, profile.PersonaReferrer, repo)

	err := svc.Delete(context.Background(), uuid.New(), "GitHub")

	require.NoError(t, err)
	assert.Equal(t, profile.PersonaReferrer, capturedPersona)
	assert.Equal(t, "github", capturedPlatform)
}

func TestSocialLinkService_Delete_InvalidPlatform(t *testing.T) {
	repo := &mockSocialLinkRepo{}
	svc := newSvc(t, profile.PersonaFreelance, repo)

	err := svc.Delete(context.Background(), uuid.New(), "snapchat")

	assert.ErrorIs(t, err, profile.ErrInvalidPlatform)
}

func TestSocialLinkService_ListByOrganization_ScopedByPersona(t *testing.T) {
	orgID := uuid.New()
	expected := []*profile.SocialLink{
		{OrganizationID: orgID, Persona: profile.PersonaFreelance, Platform: "github", URL: "https://github.com/user"},
		{OrganizationID: orgID, Persona: profile.PersonaFreelance, Platform: "linkedin", URL: "https://linkedin.com/in/user"},
	}
	var capturedPersona profile.SocialLinkPersona
	repo := &mockSocialLinkRepo{
		listFn: func(_ context.Context, _ uuid.UUID, persona profile.SocialLinkPersona) ([]*profile.SocialLink, error) {
			capturedPersona = persona
			return expected, nil
		},
	}
	svc := newSvc(t, profile.PersonaFreelance, repo)

	links, err := svc.ListByOrganization(context.Background(), orgID)

	require.NoError(t, err)
	assert.Len(t, links, 2)
	assert.Equal(t, profile.PersonaFreelance, capturedPersona)
	assert.Equal(t, "github", links[0].Platform)
}
