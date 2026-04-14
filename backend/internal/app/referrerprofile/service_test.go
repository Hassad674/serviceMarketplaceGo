package referrerprofile_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appreferrer "marketplace-backend/internal/app/referrerprofile"
	"marketplace-backend/internal/domain/profile"
	domainreferrer "marketplace-backend/internal/domain/referrerprofile"
	"marketplace-backend/internal/port/repository"
)

type mockReferrerProfileRepo struct {
	getOrCreateByOrgID     func(ctx context.Context, orgID uuid.UUID) (*repository.ReferrerProfileView, error)
	updateCore             func(ctx context.Context, orgID uuid.UUID, title, about, videoURL string) error
	updateAvailability     func(ctx context.Context, orgID uuid.UUID, status profile.AvailabilityStatus) error
	updateExpertiseDomains func(ctx context.Context, orgID uuid.UUID, domains []string) error
}

func (m *mockReferrerProfileRepo) GetOrCreateByOrgID(ctx context.Context, orgID uuid.UUID) (*repository.ReferrerProfileView, error) {
	return m.getOrCreateByOrgID(ctx, orgID)
}
func (m *mockReferrerProfileRepo) UpdateCore(ctx context.Context, orgID uuid.UUID, title, about, videoURL string) error {
	return m.updateCore(ctx, orgID, title, about, videoURL)
}
func (m *mockReferrerProfileRepo) UpdateAvailability(ctx context.Context, orgID uuid.UUID, status profile.AvailabilityStatus) error {
	return m.updateAvailability(ctx, orgID, status)
}
func (m *mockReferrerProfileRepo) UpdateExpertiseDomains(ctx context.Context, orgID uuid.UUID, domains []string) error {
	return m.updateExpertiseDomains(ctx, orgID, domains)
}

func newStubView(orgID uuid.UUID) *repository.ReferrerProfileView {
	return &repository.ReferrerProfileView{
		Profile: &domainreferrer.Profile{
			ID:                 uuid.New(),
			OrganizationID:     orgID,
			AvailabilityStatus: profile.AvailabilityNow,
			ExpertiseDomains:   []string{},
		},
	}
}

func TestService_GetByOrgID_DelegatesToGetOrCreate(t *testing.T) {
	orgID := uuid.New()
	stub := newStubView(orgID)
	repo := &mockReferrerProfileRepo{
		getOrCreateByOrgID: func(ctx context.Context, id uuid.UUID) (*repository.ReferrerProfileView, error) {
			assert.Equal(t, orgID, id)
			return stub, nil
		},
	}
	svc := appreferrer.NewService(repo)

	got, err := svc.GetByOrgID(context.Background(), orgID)
	require.NoError(t, err)
	assert.Equal(t, stub, got)
}

func TestService_GetByOrgID_WrapsRepoError(t *testing.T) {
	boom := errors.New("db down")
	repo := &mockReferrerProfileRepo{
		getOrCreateByOrgID: func(ctx context.Context, id uuid.UUID) (*repository.ReferrerProfileView, error) {
			return nil, boom
		},
	}
	svc := appreferrer.NewService(repo)

	_, err := svc.GetByOrgID(context.Background(), uuid.New())
	assert.ErrorIs(t, err, boom)
}

func TestService_UpdateCore_EnsuresRowThenWritesThenRefetches(t *testing.T) {
	orgID := uuid.New()
	calls := []string{}
	repo := &mockReferrerProfileRepo{
		getOrCreateByOrgID: func(ctx context.Context, id uuid.UUID) (*repository.ReferrerProfileView, error) {
			calls = append(calls, "get")
			return newStubView(id), nil
		},
		updateCore: func(ctx context.Context, id uuid.UUID, title, about, videoURL string) error {
			calls = append(calls, "update")
			assert.Equal(t, "Top Apporteur", title)
			assert.Equal(t, "Finds deals", about)
			assert.Equal(t, "https://example.com/v.mp4", videoURL)
			return nil
		},
	}
	svc := appreferrer.NewService(repo)

	_, err := svc.UpdateCore(context.Background(), orgID, appreferrer.UpdateCoreInput{
		Title:    "  Top Apporteur  ",
		About:    "Finds deals",
		VideoURL: "https://example.com/v.mp4",
	})
	require.NoError(t, err)
	// Expected sequence: ensure (get) -> update -> refetch (get).
	assert.Equal(t, []string{"get", "update", "get"}, calls)
}

func TestService_UpdateAvailability_ValidatesInput(t *testing.T) {
	repo := &mockReferrerProfileRepo{
		getOrCreateByOrgID: func(ctx context.Context, id uuid.UUID) (*repository.ReferrerProfileView, error) {
			return newStubView(id), nil
		},
		updateAvailability: func(ctx context.Context, id uuid.UUID, status profile.AvailabilityStatus) error {
			return nil
		},
	}
	svc := appreferrer.NewService(repo)

	_, err := svc.UpdateAvailability(context.Background(), uuid.New(), "available_now")
	assert.NoError(t, err)

	_, err = svc.UpdateAvailability(context.Background(), uuid.New(), "nonsense")
	assert.ErrorIs(t, err, profile.ErrInvalidAvailabilityStatus)
}

func TestService_UpdateExpertise_Normalizes(t *testing.T) {
	var captured []string
	repo := &mockReferrerProfileRepo{
		getOrCreateByOrgID: func(ctx context.Context, id uuid.UUID) (*repository.ReferrerProfileView, error) {
			return newStubView(id), nil
		},
		updateExpertiseDomains: func(ctx context.Context, id uuid.UUID, domains []string) error {
			captured = domains
			return nil
		},
	}
	svc := appreferrer.NewService(repo)

	_, err := svc.UpdateExpertise(context.Background(), uuid.New(),
		[]string{"marketing", "  sales ", "marketing", ""})
	require.NoError(t, err)
	assert.Equal(t, []string{"marketing", "sales"}, captured)
}
