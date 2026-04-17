package referrerpricing_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appreferrer "marketplace-backend/internal/app/referrerpricing"
	domainreferrer "marketplace-backend/internal/domain/referrerpricing"
)

type mockReferrerPricingRepo struct {
	upsert            func(ctx context.Context, p *domainreferrer.Pricing) error
	findByProfileID   func(ctx context.Context, profileID uuid.UUID) (*domainreferrer.Pricing, error)
	deleteByProfileID func(ctx context.Context, profileID uuid.UUID) error
}

func (m *mockReferrerPricingRepo) Upsert(ctx context.Context, p *domainreferrer.Pricing) error {
	return m.upsert(ctx, p)
}
func (m *mockReferrerPricingRepo) FindByProfileID(ctx context.Context, id uuid.UUID) (*domainreferrer.Pricing, error) {
	return m.findByProfileID(ctx, id)
}
func (m *mockReferrerPricingRepo) DeleteByProfileID(ctx context.Context, id uuid.UUID) error {
	return m.deleteByProfileID(ctx, id)
}

func intp(v int64) *int64 { return &v }

func TestService_Upsert_ValidationRejectsBeforePersist(t *testing.T) {
	persistCalled := false
	repo := &mockReferrerPricingRepo{
		upsert: func(ctx context.Context, p *domainreferrer.Pricing) error {
			persistCalled = true
			return nil
		},
	}
	svc := appreferrer.NewService(repo)

	// commission_pct needs currency "pct" — EUR must fail domain
	// validation before reaching the repository.
	_, err := svc.Upsert(context.Background(), appreferrer.UpsertInput{
		ProfileID: uuid.New(),
		Type:      domainreferrer.TypeCommissionPct,
		MinAmount: 500,
		MaxAmount: intp(2000),
		Currency:  "EUR",
	})
	assert.ErrorIs(t, err, domainreferrer.ErrInvalidCurrencyForType)
	assert.False(t, persistCalled)
}

func TestService_Upsert_HappyPath(t *testing.T) {
	profileID := uuid.New()
	var persisted *domainreferrer.Pricing
	repo := &mockReferrerPricingRepo{
		upsert: func(ctx context.Context, p *domainreferrer.Pricing) error {
			persisted = p
			return nil
		},
	}
	svc := appreferrer.NewService(repo)

	// V1 pricing simplification: `commission_pct` is the single
	// allowed type for the referrer persona. Happy path writes it
	// with the required "pct" currency and basis-point amounts.
	got, err := svc.Upsert(context.Background(), appreferrer.UpsertInput{
		ProfileID:  profileID,
		Type:       domainreferrer.TypeCommissionPct,
		MinAmount:  500,
		MaxAmount:  intp(1500),
		Currency:   "pct",
		Negotiable: true,
	})
	require.NoError(t, err)
	assert.Equal(t, persisted, got)
	assert.Equal(t, domainreferrer.TypeCommissionPct, got.Type)
}

// TestService_Upsert_RejectsDeprecatedTypes asserts the V1 whitelist:
// only `commission_pct` is accepted on writes. The legacy
// `commission_flat` must fail fast without touching the repository.
func TestService_Upsert_RejectsDeprecatedTypes(t *testing.T) {
	deprecated := []domainreferrer.PricingType{
		domainreferrer.TypeCommissionFlat,
	}
	for _, tp := range deprecated {
		t.Run(string(tp), func(t *testing.T) {
			persistCalled := false
			repo := &mockReferrerPricingRepo{
				upsert: func(ctx context.Context, p *domainreferrer.Pricing) error {
					persistCalled = true
					return nil
				},
			}
			svc := appreferrer.NewService(repo)

			_, err := svc.Upsert(context.Background(), appreferrer.UpsertInput{
				ProfileID: uuid.New(),
				Type:      tp,
				MinAmount: 50000,
				Currency:  "EUR",
			})
			assert.ErrorIs(t, err, domainreferrer.ErrPricingTypeNotAllowed)
			assert.False(t, persistCalled, "deprecated type must never reach the repository")
		})
	}
}

func TestService_Get_PassesThroughAndNotFound(t *testing.T) {
	profileID := uuid.New()
	stub := &domainreferrer.Pricing{ProfileID: profileID, Type: domainreferrer.TypeCommissionFlat, MinAmount: 42, Currency: "EUR"}

	repo := &mockReferrerPricingRepo{
		findByProfileID: func(ctx context.Context, id uuid.UUID) (*domainreferrer.Pricing, error) {
			if id == profileID {
				return stub, nil
			}
			return nil, domainreferrer.ErrPricingNotFound
		},
	}
	svc := appreferrer.NewService(repo)

	got, err := svc.Get(context.Background(), profileID)
	require.NoError(t, err)
	assert.Equal(t, stub, got)

	_, err = svc.Get(context.Background(), uuid.New())
	assert.ErrorIs(t, err, domainreferrer.ErrPricingNotFound)
}

func TestService_Delete_PropagatesErrors(t *testing.T) {
	boom := errors.New("delete failed")
	repo := &mockReferrerPricingRepo{
		deleteByProfileID: func(ctx context.Context, id uuid.UUID) error {
			return boom
		},
	}
	svc := appreferrer.NewService(repo)

	err := svc.Delete(context.Background(), uuid.New())
	assert.ErrorIs(t, err, boom)
}
