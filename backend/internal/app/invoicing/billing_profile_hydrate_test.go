package invoicing_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domain "marketplace-backend/internal/domain/invoicing"
	portservice "marketplace-backend/internal/port/service"
)

// HydrateFromPaymentBillingDetails is the post-confirmation hook that
// merges Stripe billing_details captured inline on the Payment Element
// into the org's billing profile. The contract:
//
//   - empty input fields NEVER overwrite existing data,
//   - the user-typed value on the standalone billing-profile page wins
//     over Stripe's freshly-typed value,
//   - missing profile rows are seeded with a sensible stub,
//   - country codes are normalised to upper-case ISO-3166 alpha-2,
//   - empty payloads short-circuit with no DB write,
//   - infra errors (other than NotFound) propagate so the webhook
//     handler can decide what to do.

func TestHydrateFromPaymentBillingDetails_FullData_NewProfile(t *testing.T) {
	svc, _, profiles, _, _, _, _ := newSvc(t)
	orgID := uuid.New()

	profiles.findByOrgFn = func(_ context.Context, _ uuid.UUID) (*domain.BillingProfile, error) {
		return nil, domain.ErrNotFound
	}
	var saved *domain.BillingProfile
	profiles.upsertFn = func(_ context.Context, p *domain.BillingProfile) error {
		saved = p
		return nil
	}

	bd := portservice.PaymentBillingDetails{
		Name:         "Acme Studio SARL",
		Email:        "billing@acme.fr",
		AddressLine1: "1 rue de la Paix",
		AddressLine2: "Bât. B",
		City:         "Paris",
		PostalCode:   "75001",
		Country:      "fr", // lowercase from Stripe
	}

	err := svc.HydrateFromPaymentBillingDetails(context.Background(), orgID, bd)

	require.NoError(t, err)
	require.NotNil(t, saved)
	assert.Equal(t, orgID, saved.OrganizationID)
	assert.Equal(t, "Acme Studio SARL", saved.LegalName)
	assert.Equal(t, "1 rue de la Paix", saved.AddressLine1)
	assert.Equal(t, "Bât. B", saved.AddressLine2)
	assert.Equal(t, "Paris", saved.City)
	assert.Equal(t, "75001", saved.PostalCode)
	assert.Equal(t, "FR", saved.Country, "country must be normalised to upper-case")
	assert.Equal(t, "billing@acme.fr", saved.InvoicingEmail)
	assert.Equal(t, domain.ProfileBusiness, saved.ProfileType, "stub default")
}

func TestHydrateFromPaymentBillingDetails_PartialData_PreservesExistingFields(t *testing.T) {
	svc, _, profiles, _, _, _, _ := newSvc(t)
	orgID := uuid.New()

	// Existing profile has the legal name + SIRET filled by the user via
	// the standalone settings page. Stripe only collected an address.
	// The hydration must keep the user-typed values intact.
	profiles.findByOrgFn = func(_ context.Context, _ uuid.UUID) (*domain.BillingProfile, error) {
		return &domain.BillingProfile{
			OrganizationID: orgID,
			ProfileType:    domain.ProfileBusiness,
			LegalName:      "User-Typed Legal Name",
			TaxID:          "12345678901234",
			Country:        "FR",
		}, nil
	}
	var saved *domain.BillingProfile
	profiles.upsertFn = func(_ context.Context, p *domain.BillingProfile) error {
		saved = p
		return nil
	}

	bd := portservice.PaymentBillingDetails{
		Name:         "Stripe-Provided Name", // existing wins
		AddressLine1: "10 avenue du Test",
		City:         "Lyon",
		PostalCode:   "69001",
	}

	err := svc.HydrateFromPaymentBillingDetails(context.Background(), orgID, bd)

	require.NoError(t, err)
	require.NotNil(t, saved)
	assert.Equal(t, "User-Typed Legal Name", saved.LegalName, "user-typed value must win over Stripe")
	assert.Equal(t, "12345678901234", saved.TaxID, "SIRET must be preserved")
	assert.Equal(t, "10 avenue du Test", saved.AddressLine1, "empty field gets filled")
	assert.Equal(t, "Lyon", saved.City)
	assert.Equal(t, "69001", saved.PostalCode)
	assert.Equal(t, "FR", saved.Country, "existing country preserved")
}

func TestHydrateFromPaymentBillingDetails_EmptyInput_NoOp(t *testing.T) {
	svc, _, profiles, _, _, _, _ := newSvc(t)
	orgID := uuid.New()

	upsertCalls := 0
	profiles.upsertFn = func(_ context.Context, _ *domain.BillingProfile) error {
		upsertCalls++
		return nil
	}

	err := svc.HydrateFromPaymentBillingDetails(
		context.Background(),
		orgID,
		portservice.PaymentBillingDetails{}, // every field empty
	)

	require.NoError(t, err)
	assert.Equal(t, 0, upsertCalls, "empty payload must NOT trigger a DB write")
}

func TestHydrateFromPaymentBillingDetails_RepositoryError_Propagates(t *testing.T) {
	svc, _, profiles, _, _, _, _ := newSvc(t)
	orgID := uuid.New()

	profiles.findByOrgFn = func(_ context.Context, _ uuid.UUID) (*domain.BillingProfile, error) {
		return nil, errors.New("db unavailable")
	}

	err := svc.HydrateFromPaymentBillingDetails(
		context.Background(),
		orgID,
		portservice.PaymentBillingDetails{Name: "Test"},
	)

	require.Error(t, err, "infra errors (other than NotFound) must propagate")
}

func TestHydrateFromPaymentBillingDetails_NilOrgID_Fails(t *testing.T) {
	svc, _, _, _, _, _, _ := newSvc(t)

	err := svc.HydrateFromPaymentBillingDetails(
		context.Background(),
		uuid.Nil,
		portservice.PaymentBillingDetails{Name: "Test"},
	)

	require.Error(t, err)
}

func TestHydrateFromPaymentBillingDetails_HasAny_Edges(t *testing.T) {
	tests := []struct {
		name string
		bd   portservice.PaymentBillingDetails
		want bool
	}{
		{"all empty", portservice.PaymentBillingDetails{}, false},
		{"name only", portservice.PaymentBillingDetails{Name: "X"}, true},
		{"address1 only", portservice.PaymentBillingDetails{AddressLine1: "X"}, true},
		{"city only", portservice.PaymentBillingDetails{City: "X"}, true},
		{"postal only", portservice.PaymentBillingDetails{PostalCode: "X"}, true},
		{"country only", portservice.PaymentBillingDetails{Country: "X"}, true},
		{"email only — does NOT count for the gate", portservice.PaymentBillingDetails{Email: "x@y.fr"}, false},
		{"phone only — does NOT count", portservice.PaymentBillingDetails{Phone: "+33"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.bd.HasAny())
		})
	}
}
