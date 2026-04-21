package profileapp

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/domain/organization"
	"marketplace-backend/internal/domain/profile"
)

// newClientTestService wires a ClientProfileService with minimal
// in-memory mocks. Individual tests override the repo behaviour via
// the function fields. Keeping the helper tiny so every test reads
// top-to-bottom as a narrative of what it verifies.
func newClientTestService(profileRepo *mockProfileRepo, orgRepo *mockExpertiseOrgRepo) *ClientProfileService {
	if profileRepo == nil {
		profileRepo = &mockProfileRepo{}
	}
	if orgRepo == nil {
		orgRepo = &mockExpertiseOrgRepo{}
	}
	return NewClientProfileService(profileRepo, orgRepo)
}

func stringPtr(s string) *string { return &s }

func TestClientProfileService_UpdateClientProfile(t *testing.T) {
	orgID := uuid.New()

	t.Run("happy path — updates description only", func(t *testing.T) {
		var wroteDesc string
		profiles := &mockProfileRepo{
			updateClientDescriptionFn: func(_ context.Context, _ uuid.UUID, desc string) error {
				wroteDesc = desc
				return nil
			},
			getByOrgIDFn: func(_ context.Context, _ uuid.UUID) (*profile.Profile, error) {
				p := profile.NewProfile(orgID)
				p.ClientDescription = "We are a serious client"
				return p, nil
			},
		}
		orgs := &mockExpertiseOrgRepo{
			findByIDFn: func(_ context.Context, id uuid.UUID) (*organization.Organization, error) {
				return &organization.Organization{ID: id, Type: organization.OrgTypeEnterprise, Name: "Acme"}, nil
			},
		}
		svc := newClientTestService(profiles, orgs)

		out, err := svc.UpdateClientProfile(context.Background(), orgID, UpdateClientProfileInput{
			ClientDescription: stringPtr("We are a serious client"),
		})
		require.NoError(t, err)
		require.NotNil(t, out)
		assert.Equal(t, "We are a serious client", wroteDesc)
		assert.Equal(t, "We are a serious client", out.ClientDescription)
	})

	t.Run("happy path — updates company name only", func(t *testing.T) {
		var savedName string
		profiles := &mockProfileRepo{
			getByOrgIDFn: func(_ context.Context, _ uuid.UUID) (*profile.Profile, error) {
				return profile.NewProfile(orgID), nil
			},
		}
		orgs := &mockExpertiseOrgRepo{
			findByIDFn: func(_ context.Context, id uuid.UUID) (*organization.Organization, error) {
				return &organization.Organization{ID: id, Type: organization.OrgTypeAgency, Name: "Old Name"}, nil
			},
			updateFn: func(_ context.Context, org *organization.Organization) error {
				savedName = org.Name
				return nil
			},
		}
		svc := newClientTestService(profiles, orgs)

		_, err := svc.UpdateClientProfile(context.Background(), orgID, UpdateClientProfileInput{
			CompanyName: stringPtr("  New Corp  "),
		})
		require.NoError(t, err)
		assert.Equal(t, "New Corp", savedName, "company name must be trimmed before persist")
	})

	t.Run("provider_personal is forbidden", func(t *testing.T) {
		orgs := &mockExpertiseOrgRepo{
			findByIDFn: func(_ context.Context, id uuid.UUID) (*organization.Organization, error) {
				return &organization.Organization{ID: id, Type: organization.OrgTypeProviderPersonal}, nil
			},
		}
		svc := newClientTestService(nil, orgs)

		_, err := svc.UpdateClientProfile(context.Background(), orgID, UpdateClientProfileInput{
			ClientDescription: stringPtr("does not matter"),
		})
		assert.ErrorIs(t, err, profile.ErrForbiddenOrgType)
	})

	t.Run("client description over max length is rejected", func(t *testing.T) {
		orgs := &mockExpertiseOrgRepo{
			findByIDFn: func(_ context.Context, id uuid.UUID) (*organization.Organization, error) {
				return &organization.Organization{ID: id, Type: organization.OrgTypeAgency}, nil
			},
		}
		svc := newClientTestService(nil, orgs)

		tooLong := strings.Repeat("x", profile.MaxClientDescriptionLength+1)
		_, err := svc.UpdateClientProfile(context.Background(), orgID, UpdateClientProfileInput{
			ClientDescription: &tooLong,
		})
		assert.ErrorIs(t, err, profile.ErrClientDescriptionTooLong)
	})

	t.Run("empty company name is rejected", func(t *testing.T) {
		orgs := &mockExpertiseOrgRepo{
			findByIDFn: func(_ context.Context, id uuid.UUID) (*organization.Organization, error) {
				return &organization.Organization{ID: id, Type: organization.OrgTypeAgency}, nil
			},
		}
		svc := newClientTestService(nil, orgs)

		_, err := svc.UpdateClientProfile(context.Background(), orgID, UpdateClientProfileInput{
			CompanyName: stringPtr("   "),
		})
		assert.ErrorIs(t, err, organization.ErrNameRequired)
	})

	t.Run("missing org surfaces the resolve error", func(t *testing.T) {
		orgs := &mockExpertiseOrgRepo{
			findByIDFn: func(_ context.Context, _ uuid.UUID) (*organization.Organization, error) {
				return nil, organization.ErrOrgNotFound
			},
		}
		svc := newClientTestService(nil, orgs)

		_, err := svc.UpdateClientProfile(context.Background(), orgID, UpdateClientProfileInput{
			ClientDescription: stringPtr("x"),
		})
		assert.ErrorIs(t, err, organization.ErrOrgNotFound)
	})

	t.Run("description write failure bubbles up", func(t *testing.T) {
		wantErr := errors.New("boom")
		profiles := &mockProfileRepo{
			updateClientDescriptionFn: func(_ context.Context, _ uuid.UUID, _ string) error {
				return wantErr
			},
		}
		orgs := &mockExpertiseOrgRepo{
			findByIDFn: func(_ context.Context, id uuid.UUID) (*organization.Organization, error) {
				return &organization.Organization{ID: id, Type: organization.OrgTypeEnterprise}, nil
			},
		}
		svc := newClientTestService(profiles, orgs)

		_, err := svc.UpdateClientProfile(context.Background(), orgID, UpdateClientProfileInput{
			ClientDescription: stringPtr("x"),
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, wantErr)
	})
}

func TestIsClientProfileEnabled(t *testing.T) {
	tests := []struct {
		name    string
		orgType organization.OrgType
		want    bool
	}{
		{"agency", organization.OrgTypeAgency, true},
		{"enterprise", organization.OrgTypeEnterprise, true},
		{"provider_personal", organization.OrgTypeProviderPersonal, false},
		{"unknown", organization.OrgType("ghost"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isClientProfileEnabled(tt.orgType))
		})
	}
}
