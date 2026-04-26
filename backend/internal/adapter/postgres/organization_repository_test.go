package postgres_test

// Integration tests for OrganizationRepository surface methods that
// don't already have coverage elsewhere. Gated behind
// MARKETPLACE_TEST_DATABASE_URL via the testDB helper in
// job_credit_repository_test.go — auto-skip when unset.
//
// Run against the local feature DB:
//
//	MARKETPLACE_TEST_DATABASE_URL=postgres://postgres:postgres@localhost:5435/marketplace_go_feat_invoicing?sslmode=disable \
//	  go test ./internal/adapter/postgres/ -run TestOrganizationRepository -count=1

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestOrganizationRepository_ListWithStripeAccount_ReturnsOnlyOnboardedOrgs
// seeds 3 onboarded orgs + 2 not-onboarded and asserts the method
// returns exactly the 3 onboarded ids.
func TestOrganizationRepository_ListWithStripeAccount_ReturnsOnlyOnboardedOrgs(t *testing.T) {
	db := testDB(t)
	repo := newOrgRepo(db)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Capture the baseline so we can isolate the diff after seeding —
	// the shared DB might already carry orgs from other tests.
	baseline, err := repo.ListWithStripeAccount(ctx)
	require.NoError(t, err)
	baselineSet := map[uuid.UUID]bool{}
	for _, id := range baseline {
		baselineSet[id] = true
	}

	// Seed 3 onboarded + 2 not-onboarded.
	var (
		onboarded    []uuid.UUID
		notOnboarded []uuid.UUID
	)
	for i := 0; i < 3; i++ {
		ownerID := insertTestUser(t, db)
		orgID := createOrg(t, repo, ownerID)
		require.NoError(t, repo.SetStripeAccount(ctx, orgID, "acct_test_"+orgID.String()[:8], "FR"))
		onboarded = append(onboarded, orgID)
	}
	for i := 0; i < 2; i++ {
		ownerID := insertTestUser(t, db)
		orgID := createOrg(t, repo, ownerID)
		notOnboarded = append(notOnboarded, orgID)
	}

	got, err := repo.ListWithStripeAccount(ctx)
	require.NoError(t, err)

	gotSet := map[uuid.UUID]bool{}
	for _, id := range got {
		gotSet[id] = true
	}

	for _, id := range onboarded {
		assert.True(t, gotSet[id], "onboarded org %s must be returned", id)
	}
	for _, id := range notOnboarded {
		assert.False(t, gotSet[id], "non-onboarded org %s must NOT be returned", id)
	}

	// Diff size relative to baseline must be exactly 3.
	diff := 0
	for id := range gotSet {
		if !baselineSet[id] {
			diff++
		}
	}
	assert.Equal(t, 3, diff, "exactly 3 new onboarded orgs surfaced by the diff vs baseline")
}

// TestOrganizationRepository_ListWithStripeAccount_EmptySafe verifies
// the method returns a non-nil empty slice (callers iterate safely)
// and never errors when called on a fresh DB.
func TestOrganizationRepository_ListWithStripeAccount_EmptySafe(t *testing.T) {
	db := testDB(t)
	repo := newOrgRepo(db)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	got, err := repo.ListWithStripeAccount(ctx)
	require.NoError(t, err)
	assert.NotNil(t, got, "must be a non-nil slice for safe iteration")
}
