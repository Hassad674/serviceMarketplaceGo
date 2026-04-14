package postgres_test

// Integration tests for ReferrerPricingRepository (migration 100).

import (
	"context"
	"testing"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/adapter/postgres"
	"marketplace-backend/internal/domain/referrerpricing"
)

// newTestReferrerProfileID creates a provider_personal org, lazily
// creates its referrer profile via the repository (exercising the
// lazy-create path), and returns the freshly-allocated profile id
// so pricing tests can bind to a real FK target.
func newTestReferrerProfileID(t *testing.T) uuid.UUID {
	t.Helper()
	db := testDB(t)
	orgID := newTestReferrerOrg(t)
	repo := postgres.NewReferrerProfileRepository(db)
	view, err := repo.GetOrCreateByOrgID(context.Background(), orgID)
	require.NoError(t, err)
	return view.Profile.ID
}

func TestReferrerPricingRepository_UpsertFindDeleteRoundTrip(t *testing.T) {
	db := testDB(t)
	profileID := newTestReferrerProfileID(t)
	repo := postgres.NewReferrerPricingRepository(db)
	ctx := context.Background()

	// First write — commission_pct range.
	p1, err := referrerpricing.NewPricing(referrerpricing.NewPricingInput{
		ProfileID: profileID,
		Type:      referrerpricing.TypeCommissionPct,
		MinAmount: 500,  // 5%
		MaxAmount: intp(2000), // 20%
		Currency:  "pct",
		Note:      "apporteur",
	})
	require.NoError(t, err)
	require.NoError(t, repo.Upsert(ctx, p1))

	got, err := repo.FindByProfileID(ctx, profileID)
	require.NoError(t, err)
	assert.Equal(t, referrerpricing.TypeCommissionPct, got.Type)
	assert.Equal(t, int64(500), got.MinAmount)
	require.NotNil(t, got.MaxAmount)
	assert.Equal(t, int64(2000), *got.MaxAmount)
	assert.Equal(t, "pct", got.Currency)
	assert.Equal(t, "apporteur", got.Note)

	// Upsert — commission_flat replaces the pct row.
	p2, err := referrerpricing.NewPricing(referrerpricing.NewPricingInput{
		ProfileID:  profileID,
		Type:       referrerpricing.TypeCommissionFlat,
		MinAmount:  50000,
		Currency:   "EUR",
		Negotiable: true,
	})
	require.NoError(t, err)
	require.NoError(t, repo.Upsert(ctx, p2))

	got, err = repo.FindByProfileID(ctx, profileID)
	require.NoError(t, err)
	assert.Equal(t, referrerpricing.TypeCommissionFlat, got.Type)
	assert.Equal(t, int64(50000), got.MinAmount)
	assert.Nil(t, got.MaxAmount)
	assert.Equal(t, "EUR", got.Currency)
	assert.True(t, got.Negotiable)

	// Delete — idempotent.
	require.NoError(t, repo.DeleteByProfileID(ctx, profileID))
	_, err = repo.FindByProfileID(ctx, profileID)
	assert.ErrorIs(t, err, referrerpricing.ErrPricingNotFound)
	require.NoError(t, repo.DeleteByProfileID(ctx, profileID))
}

func TestReferrerPricingRepository_FindByProfileID_NotFound(t *testing.T) {
	db := testDB(t)
	repo := postgres.NewReferrerPricingRepository(db)

	_, err := repo.FindByProfileID(context.Background(), uuid.New())
	assert.ErrorIs(t, err, referrerpricing.ErrPricingNotFound)
}
