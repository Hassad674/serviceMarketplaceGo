package freelanceprofile_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appfreelance "marketplace-backend/internal/app/freelanceprofile"
	domainfreelance "marketplace-backend/internal/domain/freelanceprofile"
	"marketplace-backend/internal/domain/profile"
	"marketplace-backend/internal/port/repository"
)

// mockFreelanceProfileRepo is a hand-rolled mock for the tests in
// this file. Every method is a function field so a single test can
// swap behaviours without constructing a new mock type each time.
type mockFreelanceProfileRepo struct {
	getByOrgID             func(ctx context.Context, orgID uuid.UUID) (*repository.FreelanceProfileView, error)
	getOrCreateByOrgID     func(ctx context.Context, orgID uuid.UUID) (*repository.FreelanceProfileView, error)
	updateCore             func(ctx context.Context, orgID uuid.UUID, title, about, videoURL string) error
	updateAvailability     func(ctx context.Context, orgID uuid.UUID, status profile.AvailabilityStatus) error
	updateExpertiseDomains func(ctx context.Context, orgID uuid.UUID, domains []string) error
}

func (m *mockFreelanceProfileRepo) GetByOrgID(ctx context.Context, orgID uuid.UUID) (*repository.FreelanceProfileView, error) {
	return m.getByOrgID(ctx, orgID)
}
func (m *mockFreelanceProfileRepo) GetOrCreateByOrgID(ctx context.Context, orgID uuid.UUID) (*repository.FreelanceProfileView, error) {
	if m.getOrCreateByOrgID != nil {
		return m.getOrCreateByOrgID(ctx, orgID)
	}
	// Fallback to the strict read so tests that only wire getByOrgID
	// keep working — the service's owner path now calls
	// GetOrCreateByOrgID internally.
	return m.getByOrgID(ctx, orgID)
}
func (m *mockFreelanceProfileRepo) UpdateCore(ctx context.Context, orgID uuid.UUID, title, about, videoURL string) error {
	return m.updateCore(ctx, orgID, title, about, videoURL)
}
func (m *mockFreelanceProfileRepo) UpdateAvailability(ctx context.Context, orgID uuid.UUID, status profile.AvailabilityStatus) error {
	return m.updateAvailability(ctx, orgID, status)
}
func (m *mockFreelanceProfileRepo) UpdateExpertiseDomains(ctx context.Context, orgID uuid.UUID, domains []string) error {
	return m.updateExpertiseDomains(ctx, orgID, domains)
}

// newStubView returns a minimal FreelanceProfileView suitable for
// tests that do not care about the payload shape, only whether
// something non-nil was returned.
func newStubView(orgID uuid.UUID) *repository.FreelanceProfileView {
	return &repository.FreelanceProfileView{
		Profile: &domainfreelance.Profile{
			ID:                 uuid.New(),
			OrganizationID:     orgID,
			AvailabilityStatus: profile.AvailabilityNow,
			ExpertiseDomains:   []string{},
		},
	}
}

func TestService_GetByOrgID_PassesThroughRepoResult(t *testing.T) {
	orgID := uuid.New()
	stub := newStubView(orgID)
	repo := &mockFreelanceProfileRepo{
		getByOrgID: func(ctx context.Context, id uuid.UUID) (*repository.FreelanceProfileView, error) {
			assert.Equal(t, orgID, id)
			return stub, nil
		},
	}
	svc := appfreelance.NewService(repo)

	got, err := svc.GetByOrgID(context.Background(), orgID)
	require.NoError(t, err)
	assert.Equal(t, stub, got)
}

func TestService_GetByOrgID_WrapsRepoError(t *testing.T) {
	repo := &mockFreelanceProfileRepo{
		getByOrgID: func(ctx context.Context, id uuid.UUID) (*repository.FreelanceProfileView, error) {
			return nil, domainfreelance.ErrProfileNotFound
		},
	}
	svc := appfreelance.NewService(repo)

	_, err := svc.GetByOrgID(context.Background(), uuid.New())
	assert.ErrorIs(t, err, domainfreelance.ErrProfileNotFound)
}

func TestService_UpdateCore_TrimsAndRefetches(t *testing.T) {
	orgID := uuid.New()
	var gotTitle, gotAbout, gotVideo string
	repo := &mockFreelanceProfileRepo{
		updateCore: func(ctx context.Context, id uuid.UUID, title, about, videoURL string) error {
			gotTitle = title
			gotAbout = about
			gotVideo = videoURL
			return nil
		},
		getByOrgID: func(ctx context.Context, id uuid.UUID) (*repository.FreelanceProfileView, error) {
			return newStubView(id), nil
		},
	}
	svc := appfreelance.NewService(repo)

	_, err := svc.UpdateCore(context.Background(), orgID, appfreelance.UpdateCoreInput{
		Title:    "  Senior Go Engineer  ",
		About:    "\nBuilds marketplaces.\n",
		VideoURL: " https://example.com/v.mp4 ",
	})
	require.NoError(t, err)
	assert.Equal(t, "Senior Go Engineer", gotTitle)
	assert.Equal(t, "Builds marketplaces.", gotAbout)
	assert.Equal(t, "https://example.com/v.mp4", gotVideo)
}

func TestService_UpdateCore_PropagatesRepoError(t *testing.T) {
	boom := errors.New("database exploded")
	repo := &mockFreelanceProfileRepo{
		updateCore: func(ctx context.Context, id uuid.UUID, _, _, _ string) error {
			return boom
		},
	}
	svc := appfreelance.NewService(repo)

	_, err := svc.UpdateCore(context.Background(), uuid.New(), appfreelance.UpdateCoreInput{})
	assert.ErrorIs(t, err, boom)
}

func TestService_UpdateAvailability_ValidatesInput(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantErr bool
	}{
		{"valid now", "available_now", false},
		{"valid soon", "available_soon", false},
		{"valid not", "not_available", false},
		{"empty", "", true},
		{"unknown", "maybe", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &mockFreelanceProfileRepo{
				updateAvailability: func(ctx context.Context, id uuid.UUID, status profile.AvailabilityStatus) error {
					return nil
				},
				getByOrgID: func(ctx context.Context, id uuid.UUID) (*repository.FreelanceProfileView, error) {
					return newStubView(id), nil
				},
			}
			svc := appfreelance.NewService(repo)
			_, err := svc.UpdateAvailability(context.Background(), uuid.New(), tc.raw)
			if tc.wantErr {
				assert.ErrorIs(t, err, profile.ErrInvalidAvailabilityStatus)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestService_UpdateExpertise_NormalizesInput(t *testing.T) {
	var captured []string
	repo := &mockFreelanceProfileRepo{
		updateExpertiseDomains: func(ctx context.Context, id uuid.UUID, domains []string) error {
			captured = domains
			return nil
		},
		getByOrgID: func(ctx context.Context, id uuid.UUID) (*repository.FreelanceProfileView, error) {
			return newStubView(id), nil
		},
	}
	svc := appfreelance.NewService(repo)

	_, err := svc.UpdateExpertise(context.Background(), uuid.New(),
		[]string{"  development ", "design_ui_ux", "", "development"},
	)
	require.NoError(t, err)
	assert.Equal(t, []string{"development", "design_ui_ux"}, captured,
		"trimmed + deduped, preserving first-occurrence order")
}

func TestService_UpdateExpertise_NilInputYieldsEmptySlice(t *testing.T) {
	var captured []string
	captured = []string{"sentinel"}
	repo := &mockFreelanceProfileRepo{
		updateExpertiseDomains: func(ctx context.Context, id uuid.UUID, domains []string) error {
			captured = domains
			return nil
		},
		getByOrgID: func(ctx context.Context, id uuid.UUID) (*repository.FreelanceProfileView, error) {
			return newStubView(id), nil
		},
	}
	svc := appfreelance.NewService(repo)

	_, err := svc.UpdateExpertise(context.Background(), uuid.New(), nil)
	require.NoError(t, err)
	assert.NotNil(t, captured)
	assert.Empty(t, captured)
}
