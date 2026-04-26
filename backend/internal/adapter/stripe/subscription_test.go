package stripe_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	stripeadapter "marketplace-backend/internal/adapter/stripe"
	portservice "marketplace-backend/internal/port/service"
)

// EnrichCustomerWithBillingProfile is the only branch testable without
// mocking the Stripe SDK's global backend — the early returns (empty
// customer id, empty snapshot) short-circuit before any HTTP call.
// The actual Stripe round-trip is exercised by integration smoke tests
// in Phase 4. Keeping these light tests guards the contract anyway.

func TestEnrichCustomerWithBillingProfile_RejectsEmptyCustomerID(t *testing.T) {
	svc := stripeadapter.NewSubscriptionService("sk_test_dummy")

	err := svc.EnrichCustomerWithBillingProfile(context.Background(), "", portservice.BillingProfileStripeSnapshot{
		LegalName: "Acme",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "customer id is required")
}

func TestEnrichCustomerWithBillingProfile_NoOpOnEmptySnapshot(t *testing.T) {
	// Critical guarantee: an empty snapshot must NOT produce a Stripe
	// Customer.Update call (which would silently clear pre-existing
	// fields with empty strings). The function returns nil so the
	// caller continues.
	svc := stripeadapter.NewSubscriptionService("sk_test_dummy")

	err := svc.EnrichCustomerWithBillingProfile(context.Background(), "cus_test_123", portservice.BillingProfileStripeSnapshot{})

	require.NoError(t, err, "empty snapshot must short-circuit without hitting Stripe")
}

// IsEmpty is the short-circuit predicate used by Subscribe and by the
// adapter itself. Make sure it agrees with what the adapter checks.

func TestBillingProfileStripeSnapshot_IsEmpty(t *testing.T) {
	tests := []struct {
		name string
		snap portservice.BillingProfileStripeSnapshot
		want bool
	}{
		{
			name: "zero value",
			snap: portservice.BillingProfileStripeSnapshot{},
			want: true,
		},
		{
			name: "with legal name only",
			snap: portservice.BillingProfileStripeSnapshot{LegalName: "Acme"},
			want: false,
		},
		{
			name: "with country only",
			snap: portservice.BillingProfileStripeSnapshot{Country: "FR"},
			want: false,
		},
		{
			name: "vat-only is treated as empty (we won't push metadata-only)",
			snap: portservice.BillingProfileStripeSnapshot{VATNumber: "FR12345678901"},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.snap.IsEmpty())
		})
	}
}
