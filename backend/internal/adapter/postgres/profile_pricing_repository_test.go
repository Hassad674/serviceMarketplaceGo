package postgres_test

// Integration tests for the PostgreSQL-backed
// ProfilePricingRepository (migration 083 schema). Gated behind
// MARKETPLACE_TEST_DATABASE_URL — auto-skip when unset.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/adapter/postgres"
	"marketplace-backend/internal/domain/job"
	"marketplace-backend/internal/domain/organization"
	domainpricing "marketplace-backend/internal/domain/profilepricing"
)

// newTestOrgForPricing creates a fresh agency org and returns its
// id. Separate from newTestOrgForSkills so each test suite stays
// runnable in isolation even if the other file moves.
func newTestOrgForPricing(t *testing.T) uuid.UUID {
	t.Helper()
	db := testDB(t)
	ownerID := insertTestUser(t, db)

	org, err := organization.NewOrganization(ownerID, organization.OrgTypeAgency, "Pricing Test Org")
	require.NoError(t, err)
	member, err := organization.NewMember(org.ID, ownerID, organization.RoleOwner, "")
	require.NoError(t, err)

	orgRepo := postgres.NewOrganizationRepository(db, job.WeeklyQuota)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, orgRepo.CreateWithOwnerMembership(ctx, org, member))

	return org.ID
}

func TestProfilePricingRepository_UpsertFindRoundTrip(t *testing.T) {
	db := testDB(t)
	repo := postgres.NewProfilePricingRepository(db)
	orgID := newTestOrgForPricing(t)

	ctx := context.Background()

	// direct daily
	p1, err := domainpricing.NewPricing(
		orgID, domainpricing.KindDirect, domainpricing.TypeDaily,
		60000, nil, "EUR", "TJM standard",
	)
	require.NoError(t, err)
	require.NoError(t, repo.Upsert(ctx, p1))

	// referral commission_pct
	max := int64(1500)
	p2, err := domainpricing.NewPricing(
		orgID, domainpricing.KindReferral, domainpricing.TypeCommissionPct,
		500, &max, "pct", "apporteur",
	)
	require.NoError(t, err)
	require.NoError(t, repo.Upsert(ctx, p2))

	got, err := repo.FindByOrgID(ctx, orgID)
	require.NoError(t, err)
	require.Len(t, got, 2)
	// Ordered by pricing_kind ASC — direct comes first.
	assert.Equal(t, domainpricing.KindDirect, got[0].Kind)
	assert.Equal(t, domainpricing.TypeDaily, got[0].Type)
	assert.Equal(t, int64(60000), got[0].MinAmount)
	assert.Nil(t, got[0].MaxAmount)
	assert.Equal(t, "EUR", got[0].Currency)
	assert.Equal(t, "TJM standard", got[0].Note)
	assert.False(t, got[0].CreatedAt.IsZero())

	assert.Equal(t, domainpricing.KindReferral, got[1].Kind)
	assert.Equal(t, domainpricing.TypeCommissionPct, got[1].Type)
	assert.Equal(t, int64(500), got[1].MinAmount)
	require.NotNil(t, got[1].MaxAmount)
	assert.Equal(t, int64(1500), *got[1].MaxAmount)
	assert.Equal(t, "pct", got[1].Currency)
}

func TestProfilePricingRepository_UpsertUpdatesExistingRow(t *testing.T) {
	db := testDB(t)
	repo := postgres.NewProfilePricingRepository(db)
	orgID := newTestOrgForPricing(t)

	ctx := context.Background()

	p1, err := domainpricing.NewPricing(orgID, domainpricing.KindDirect, domainpricing.TypeDaily, 50000, nil, "EUR", "v1")
	require.NoError(t, err)
	require.NoError(t, repo.Upsert(ctx, p1))

	// Second upsert of the same kind replaces the previous row.
	p2, err := domainpricing.NewPricing(orgID, domainpricing.KindDirect, domainpricing.TypeHourly, 8000, nil, "USD", "v2")
	require.NoError(t, err)
	require.NoError(t, repo.Upsert(ctx, p2))

	got, err := repo.FindByOrgID(ctx, orgID)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, domainpricing.TypeHourly, got[0].Type)
	assert.Equal(t, int64(8000), got[0].MinAmount)
	assert.Equal(t, "USD", got[0].Currency)
	assert.Equal(t, "v2", got[0].Note)
}

func TestProfilePricingRepository_FindByOrgID_EmptyReturnsEmptySlice(t *testing.T) {
	db := testDB(t)
	repo := postgres.NewProfilePricingRepository(db)
	orgID := newTestOrgForPricing(t)

	got, err := repo.FindByOrgID(context.Background(), orgID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Len(t, got, 0)
}

func TestProfilePricingRepository_ListByOrgIDs_SeedsEveryInput(t *testing.T) {
	db := testDB(t)
	repo := postgres.NewProfilePricingRepository(db)
	orgA := newTestOrgForPricing(t)
	orgB := newTestOrgForPricing(t)

	// Only orgA has pricing.
	p, err := domainpricing.NewPricing(orgA, domainpricing.KindDirect, domainpricing.TypeDaily, 50000, nil, "EUR", "")
	require.NoError(t, err)
	require.NoError(t, repo.Upsert(context.Background(), p))

	got, err := repo.ListByOrgIDs(context.Background(), []uuid.UUID{orgA, orgB})
	require.NoError(t, err)
	require.Len(t, got, 2)
	require.Len(t, got[orgA], 1)
	require.NotNil(t, got[orgB], "every input id must be present, even with zero pricing")
	assert.Len(t, got[orgB], 0)
}

func TestProfilePricingRepository_ListByOrgIDs_EmptyInput(t *testing.T) {
	db := testDB(t)
	repo := postgres.NewProfilePricingRepository(db)

	got, err := repo.ListByOrgIDs(context.Background(), nil)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Len(t, got, 0)
}

func TestProfilePricingRepository_DeleteByKind_Idempotent(t *testing.T) {
	db := testDB(t)
	repo := postgres.NewProfilePricingRepository(db)
	orgID := newTestOrgForPricing(t)

	ctx := context.Background()
	p, err := domainpricing.NewPricing(orgID, domainpricing.KindDirect, domainpricing.TypeDaily, 50000, nil, "EUR", "")
	require.NoError(t, err)
	require.NoError(t, repo.Upsert(ctx, p))

	// First delete removes the row.
	require.NoError(t, repo.DeleteByKind(ctx, orgID, domainpricing.KindDirect))

	// Second delete is a no-op, not an error.
	require.NoError(t, repo.DeleteByKind(ctx, orgID, domainpricing.KindDirect))

	got, err := repo.FindByOrgID(ctx, orgID)
	require.NoError(t, err)
	assert.Len(t, got, 0)
}

func TestProfilePricingRepository_DeleteByKind_PreservesOtherKind(t *testing.T) {
	db := testDB(t)
	repo := postgres.NewProfilePricingRepository(db)
	orgID := newTestOrgForPricing(t)

	ctx := context.Background()

	pDirect, err := domainpricing.NewPricing(orgID, domainpricing.KindDirect, domainpricing.TypeDaily, 50000, nil, "EUR", "")
	require.NoError(t, err)
	require.NoError(t, repo.Upsert(ctx, pDirect))

	max := int64(1000)
	pReferral, err := domainpricing.NewPricing(orgID, domainpricing.KindReferral, domainpricing.TypeCommissionPct, 500, &max, "pct", "")
	require.NoError(t, err)
	require.NoError(t, repo.Upsert(ctx, pReferral))

	require.NoError(t, repo.DeleteByKind(ctx, orgID, domainpricing.KindDirect))

	got, err := repo.FindByOrgID(ctx, orgID)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, domainpricing.KindReferral, got[0].Kind)
}
