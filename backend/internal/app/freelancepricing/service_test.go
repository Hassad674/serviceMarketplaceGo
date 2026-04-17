package freelancepricing_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appfreelance "marketplace-backend/internal/app/freelancepricing"
	domainfreelance "marketplace-backend/internal/domain/freelancepricing"
)

type mockFreelancePricingRepo struct {
	upsert            func(ctx context.Context, p *domainfreelance.Pricing) error
	findByProfileID   func(ctx context.Context, profileID uuid.UUID) (*domainfreelance.Pricing, error)
	deleteByProfileID func(ctx context.Context, profileID uuid.UUID) error
}

func (m *mockFreelancePricingRepo) Upsert(ctx context.Context, p *domainfreelance.Pricing) error {
	return m.upsert(ctx, p)
}
func (m *mockFreelancePricingRepo) FindByProfileID(ctx context.Context, id uuid.UUID) (*domainfreelance.Pricing, error) {
	return m.findByProfileID(ctx, id)
}
func (m *mockFreelancePricingRepo) DeleteByProfileID(ctx context.Context, id uuid.UUID) error {
	return m.deleteByProfileID(ctx, id)
}

func intp(v int64) *int64 { return &v }

func TestService_Upsert_ValidatesBeforePersisting(t *testing.T) {
	persistCalled := false
	repo := &mockFreelancePricingRepo{
		upsert: func(ctx context.Context, p *domainfreelance.Pricing) error {
			persistCalled = true
			return nil
		},
	}
	svc := appfreelance.NewService(repo)

	// Invalid currency "pct" must NEVER reach the repository.
	_, err := svc.Upsert(context.Background(), appfreelance.UpsertInput{
		ProfileID: uuid.New(),
		Type:      domainfreelance.TypeDaily,
		MinAmount: 100,
		Currency:  "pct",
	})
	assert.ErrorIs(t, err, domainfreelance.ErrInvalidCurrencyForType)
	assert.False(t, persistCalled, "validation must short-circuit before persisting")
}

func TestService_Upsert_HappyPath(t *testing.T) {
	profileID := uuid.New()
	var persisted *domainfreelance.Pricing
	repo := &mockFreelancePricingRepo{
		upsert: func(ctx context.Context, p *domainfreelance.Pricing) error {
			persisted = p
			return nil
		},
	}
	svc := appfreelance.NewService(repo)

	// V1 pricing simplification: `daily` is the single allowed type
	// for the freelance persona. Persisted row must round-trip
	// through the service unchanged.
	got, err := svc.Upsert(context.Background(), appfreelance.UpsertInput{
		ProfileID:  profileID,
		Type:       domainfreelance.TypeDaily,
		MinAmount:  85000,
		Currency:   "EUR",
		Note:       "TJM ajustable",
		Negotiable: true,
	})
	require.NoError(t, err)
	assert.Equal(t, persisted, got)
	assert.Equal(t, profileID, got.ProfileID)
	assert.Equal(t, domainfreelance.TypeDaily, got.Type)
	assert.Equal(t, int64(85000), got.MinAmount)
}

// TestService_Upsert_RejectsDeprecatedTypes asserts the V1 whitelist:
// only `daily` is accepted on writes. Every other domain-valid type
// must fail fast with ErrPricingTypeNotAllowed and MUST NOT reach
// the repository layer.
func TestService_Upsert_RejectsDeprecatedTypes(t *testing.T) {
	cases := []struct {
		name  string
		input appfreelance.UpsertInput
	}{
		{
			name: "hourly",
			input: appfreelance.UpsertInput{
				Type:      domainfreelance.TypeHourly,
				MinAmount: 7500,
				Currency:  "EUR",
			},
		},
		{
			name: "project_from",
			input: appfreelance.UpsertInput{
				Type:      domainfreelance.TypeProjectFrom,
				MinAmount: 500000,
				Currency:  "EUR",
			},
		},
		{
			name: "project_range",
			input: appfreelance.UpsertInput{
				Type:      domainfreelance.TypeProjectRange,
				MinAmount: 500000,
				MaxAmount: intp(1500000),
				Currency:  "EUR",
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			persistCalled := false
			repo := &mockFreelancePricingRepo{
				upsert: func(ctx context.Context, p *domainfreelance.Pricing) error {
					persistCalled = true
					return nil
				},
			}
			svc := appfreelance.NewService(repo)

			tc.input.ProfileID = uuid.New()
			_, err := svc.Upsert(context.Background(), tc.input)
			assert.ErrorIs(t, err, domainfreelance.ErrPricingTypeNotAllowed)
			assert.False(t, persistCalled, "deprecated type must never reach the repository")
		})
	}
}

func TestService_Get_PassesThroughAndNotFound(t *testing.T) {
	profileID := uuid.New()
	stub := &domainfreelance.Pricing{ProfileID: profileID, Type: domainfreelance.TypeDaily, MinAmount: 42, Currency: "EUR"}

	repo := &mockFreelancePricingRepo{
		findByProfileID: func(ctx context.Context, id uuid.UUID) (*domainfreelance.Pricing, error) {
			if id == profileID {
				return stub, nil
			}
			return nil, domainfreelance.ErrPricingNotFound
		},
	}
	svc := appfreelance.NewService(repo)

	got, err := svc.Get(context.Background(), profileID)
	require.NoError(t, err)
	assert.Equal(t, stub, got)

	_, err = svc.Get(context.Background(), uuid.New())
	assert.ErrorIs(t, err, domainfreelance.ErrPricingNotFound)
}

func TestService_Delete_PropagatesErrors(t *testing.T) {
	boom := errors.New("delete failed")
	repo := &mockFreelancePricingRepo{
		deleteByProfileID: func(ctx context.Context, id uuid.UUID) error {
			return boom
		},
	}
	svc := appfreelance.NewService(repo)

	err := svc.Delete(context.Background(), uuid.New())
	assert.ErrorIs(t, err, boom)
}
