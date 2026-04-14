package postgres_test

// Integration tests for FreelancePricingRepository (migration 099).

import (
	"context"
	"testing"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/adapter/postgres"
	"marketplace-backend/internal/domain/freelancepricing"
)

func intp(v int64) *int64 { return &v }

func TestFreelancePricingRepository_UpsertFindDeleteRoundTrip(t *testing.T) {
	db := testDB(t)
	_, profileID := newTestFreelanceOrg(t)
	repo := postgres.NewFreelancePricingRepository(db)
	ctx := context.Background()

	// First write — daily scalar.
	p1, err := freelancepricing.NewPricing(freelancepricing.NewPricingInput{
		ProfileID:  profileID,
		Type:       freelancepricing.TypeDaily,
		MinAmount:  60000,
		Currency:   "EUR",
		Note:       "TJM standard",
		Negotiable: true,
	})
	require.NoError(t, err)
	require.NoError(t, repo.Upsert(ctx, p1))

	got, err := repo.FindByProfileID(ctx, profileID)
	require.NoError(t, err)
	assert.Equal(t, freelancepricing.TypeDaily, got.Type)
	assert.Equal(t, int64(60000), got.MinAmount)
	assert.Nil(t, got.MaxAmount)
	assert.Equal(t, "EUR", got.Currency)
	assert.Equal(t, "TJM standard", got.Note)
	assert.True(t, got.Negotiable)

	// Upsert — project_range replaces the daily row.
	p2, err := freelancepricing.NewPricing(freelancepricing.NewPricingInput{
		ProfileID: profileID,
		Type:      freelancepricing.TypeProjectRange,
		MinAmount: 500000,
		MaxAmount: intp(1500000),
		Currency:  "EUR",
	})
	require.NoError(t, err)
	require.NoError(t, repo.Upsert(ctx, p2))

	got, err = repo.FindByProfileID(ctx, profileID)
	require.NoError(t, err)
	assert.Equal(t, freelancepricing.TypeProjectRange, got.Type)
	assert.Equal(t, int64(500000), got.MinAmount)
	require.NotNil(t, got.MaxAmount)
	assert.Equal(t, int64(1500000), *got.MaxAmount)

	// Delete — subsequent FindByProfileID must return ErrPricingNotFound.
	require.NoError(t, repo.DeleteByProfileID(ctx, profileID))
	_, err = repo.FindByProfileID(ctx, profileID)
	assert.ErrorIs(t, err, freelancepricing.ErrPricingNotFound)

	// Delete is idempotent.
	require.NoError(t, repo.DeleteByProfileID(ctx, profileID))
}

func TestFreelancePricingRepository_FindByProfileID_NotFound(t *testing.T) {
	db := testDB(t)
	repo := postgres.NewFreelancePricingRepository(db)

	_, err := repo.FindByProfileID(context.Background(), uuid.New())
	assert.ErrorIs(t, err, freelancepricing.ErrPricingNotFound)
}
