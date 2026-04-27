package invoicing_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domain "marketplace-backend/internal/domain/invoicing"
)

// GetBillingProfileSnapshotForStripe is the read-only port the
// subscription app calls before creating an Embedded Checkout session.
// These tests cover the three branches:
//   - profile present + populated → snapshot mirrors every field
//   - profile present but empty → zero-value snapshot, IsEmpty() == true
//   - profile not found → zero-value snapshot, no error (caller skips)
//   - infrastructure error (other than ErrNotFound) → wrapped error

func TestGetBillingProfileSnapshotForStripe_FullProfile(t *testing.T) {
	svc, _, profiles, _, _, _, _ := newSvc(t)
	orgID := uuid.New()

	profiles.findByOrgFn = func(_ context.Context, _ uuid.UUID) (*domain.BillingProfile, error) {
		return &domain.BillingProfile{
			OrganizationID: orgID,
			LegalName:      "Acme Studio SARL",
			TradingName:    "Acme",
			AddressLine1:   "1 rue de la Paix",
			AddressLine2:   "Bât. B",
			PostalCode:     "75001",
			City:           "Paris",
			Country:        "FR",
			InvoicingEmail: "billing@acme.fr",
			VATNumber:      "FR12345678901",
		}, nil
	}

	snap, err := svc.GetBillingProfileSnapshotForStripe(context.Background(), orgID)

	require.NoError(t, err)
	assert.False(t, snap.IsEmpty())
	assert.Equal(t, "Acme Studio SARL", snap.LegalName)
	assert.Equal(t, "1 rue de la Paix", snap.AddressLine1)
	assert.Equal(t, "Bât. B", snap.AddressLine2)
	assert.Equal(t, "75001", snap.PostalCode)
	assert.Equal(t, "Paris", snap.City)
	assert.Equal(t, "FR", snap.Country)
	assert.Equal(t, "billing@acme.fr", snap.InvoicingEmail)
	assert.Equal(t, "FR12345678901", snap.VATNumber)
}

func TestGetBillingProfileSnapshotForStripe_NotFound(t *testing.T) {
	svc, _, profiles, _, _, _, _ := newSvc(t)
	orgID := uuid.New()

	profiles.findByOrgFn = func(_ context.Context, _ uuid.UUID) (*domain.BillingProfile, error) {
		return nil, domain.ErrNotFound
	}

	snap, err := svc.GetBillingProfileSnapshotForStripe(context.Background(), orgID)

	require.NoError(t, err, "missing profile must yield empty snapshot, no error — caller skips enrichment")
	assert.True(t, snap.IsEmpty())
}

func TestGetBillingProfileSnapshotForStripe_RepositoryError(t *testing.T) {
	svc, _, profiles, _, _, _, _ := newSvc(t)
	orgID := uuid.New()

	profiles.findByOrgFn = func(_ context.Context, _ uuid.UUID) (*domain.BillingProfile, error) {
		return nil, errors.New("db unavailable")
	}

	_, err := svc.GetBillingProfileSnapshotForStripe(context.Background(), orgID)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "get billing snapshot for stripe")
}

func TestGetBillingProfileSnapshotForStripe_RejectsNilOrgID(t *testing.T) {
	svc, _, _, _, _, _, _ := newSvc(t)

	_, err := svc.GetBillingProfileSnapshotForStripe(context.Background(), uuid.Nil)

	require.Error(t, err)
}
