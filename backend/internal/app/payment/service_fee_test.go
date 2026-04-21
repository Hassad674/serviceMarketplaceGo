package payment

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/domain/billing"
	domain "marketplace-backend/internal/domain/payment"
	domainuser "marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/port/service"
)

// feeStubUserRepo is a narrow stub that only answers GetByID.
// It embeds repository.UserRepository for the interface contract and panics
// on any unimplemented method so an unexpected call surfaces immediately.
type feeStubUserRepo struct {
	repository.UserRepository
	user *domainuser.User
}

func (s *feeStubUserRepo) GetByID(_ context.Context, _ uuid.UUID) (*domainuser.User, error) {
	if s.user == nil {
		return nil, errors.New("user not found")
	}
	return s.user, nil
}

// feeStubRecords captures the last persisted record for assertions and
// simulates the "first payment for this milestone" path by answering
// sql.ErrNoRows on GetByMilestoneID.
type feeStubRecords struct {
	repository.PaymentRecordRepository
	persisted *domain.PaymentRecord
}

func (r *feeStubRecords) GetByMilestoneID(_ context.Context, _ uuid.UUID) (*domain.PaymentRecord, error) {
	return nil, sql.ErrNoRows
}

func (r *feeStubRecords) Create(_ context.Context, rec *domain.PaymentRecord) error {
	r.persisted = rec
	return nil
}

// feeStubStripe records the PaymentIntent input so tests can assert the
// client total (proposal amount + Stripe fee, never including platform fee).
type feeStubStripe struct {
	service.StripeService
	intentCalls []service.CreatePaymentIntentInput
}

func (s *feeStubStripe) CreatePaymentIntent(_ context.Context, in service.CreatePaymentIntentInput) (*service.PaymentIntentResult, error) {
	s.intentCalls = append(s.intentCalls, in)
	return &service.PaymentIntentResult{
		PaymentIntentID: "pi_test_" + in.ProposalID,
		ClientSecret:    "cs_test_" + in.ProposalID,
	}, nil
}

// feeStubSubscriptionReader is a tiny SubscriptionReader used to exercise
// the Premium-waiver path. Nil instances mean "no feature wired", which
// matches the pre-Premium behaviour; explicit instances return whatever
// `active` was set to.
type feeStubSubscriptionReader struct {
	active bool
	err    error
}

func (s *feeStubSubscriptionReader) IsActive(_ context.Context, _ uuid.UUID) (bool, error) {
	return s.active, s.err
}

func newFeeTestService(role domainuser.Role) (*Service, *feeStubRecords, *feeStubStripe) {
	records := &feeStubRecords{}
	stripeStub := &feeStubStripe{}
	svc := NewService(ServiceDeps{
		Records: records,
		Users: &feeStubUserRepo{user: &domainuser.User{
			ID:   uuid.New(),
			Role: role,
		}},
		Stripe: stripeStub,
	})
	return svc, records, stripeStub
}

// newFeeTestServiceWithSubscription is like newFeeTestService but also
// wires a SubscriptionReader stub. Used by the Premium waiver cases.
func newFeeTestServiceWithSubscription(role domainuser.Role, subActive bool) (*Service, *feeStubRecords) {
	records := &feeStubRecords{}
	svc := NewService(ServiceDeps{
		Records: records,
		Users: &feeStubUserRepo{user: &domainuser.User{
			ID:   uuid.New(),
			Role: role,
		}},
		Stripe: &feeStubStripe{},
	})
	svc.SetSubscriptionReader(&feeStubSubscriptionReader{active: subActive})
	return svc, records
}

// TestCreatePaymentIntent_UsesBillingSchedule exercises the Phase A surgery
// directly: NewPaymentRecord receives a flat fee from the billing schedule
// instead of computing a 5% commission. Every (role, amount) combination
// listed in the fee schedule must produce the exact fee from the grid and
// the corresponding provider payout.
func TestCreatePaymentIntent_UsesBillingSchedule(t *testing.T) {
	tests := []struct {
		name           string
		role           domainuser.Role
		amountCents    int64
		wantFeeCents   int64
		wantNetCents   int64
		wantTierIndex  int
		wantBillingRole billing.Role
	}{
		// Freelance grid — 9 / 15 / 25 € with seuils 200 / 1000 €
		{"freelance tier 1 — 150€", domainuser.RoleProvider, 15000, 900, 14100, 0, billing.RoleFreelance},
		{"freelance tier 2 — 500€", domainuser.RoleProvider, 50000, 1500, 48500, 1, billing.RoleFreelance},
		{"freelance tier 3 — 2000€", domainuser.RoleProvider, 200000, 2500, 197500, 2, billing.RoleFreelance},
		{"freelance boundary — 200€ promotes to tier 2", domainuser.RoleProvider, 20000, 1500, 18500, 1, billing.RoleFreelance},
		{"freelance boundary — 1000€ promotes to tier 3", domainuser.RoleProvider, 100000, 2500, 97500, 2, billing.RoleFreelance},

		// Agency grid — 19 / 39 / 69 € with seuils 500 / 2500 €
		{"agency tier 1 — 400€", domainuser.RoleAgency, 40000, 1900, 38100, 0, billing.RoleAgency},
		{"agency tier 2 — 1500€", domainuser.RoleAgency, 150000, 3900, 146100, 1, billing.RoleAgency},
		{"agency tier 3 — 5000€", domainuser.RoleAgency, 500000, 6900, 493100, 2, billing.RoleAgency},
		{"agency boundary — 500€ promotes to tier 2", domainuser.RoleAgency, 50000, 3900, 46100, 1, billing.RoleAgency},
		{"agency boundary — 2500€ promotes to tier 3", domainuser.RoleAgency, 250000, 6900, 243100, 2, billing.RoleAgency},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc, records, stripeStub := newFeeTestService(tc.role)

			proposalID := uuid.New()
			milestoneID := uuid.New()
			clientID := uuid.New()
			providerID := uuid.New()

			out, err := svc.CreatePaymentIntent(context.Background(), service.PaymentIntentInput{
				ProposalID:     proposalID,
				MilestoneID:    milestoneID,
				ClientID:       clientID,
				ProviderID:     providerID,
				ProposalAmount: tc.amountCents,
			})
			require.NoError(t, err)
			require.NotNil(t, out)

			assert.Equal(t, tc.amountCents, out.ProposalAmount)
			assert.Equal(t, tc.wantFeeCents, out.PlatformFee, "platform fee must match the billing schedule")
			assert.Equal(t, tc.wantNetCents, out.ProviderPayout, "provider payout = amount - fee")
			assert.Equal(t, tc.amountCents+out.StripeFee, out.ClientTotal,
				"client total = amount + stripe fee (platform fee is deducted from provider, NOT added to client)")

			// Persisted record must mirror the DTO exactly — same fee frozen
			// into the row so historical audits are never drifted by a
			// schedule change.
			require.NotNil(t, records.persisted, "record must have been persisted")
			assert.Equal(t, tc.wantFeeCents, records.persisted.PlatformFeeAmount)
			assert.Equal(t, tc.wantNetCents, records.persisted.ProviderPayout)
			assert.Equal(t, tc.amountCents+out.StripeFee, records.persisted.ClientTotalAmount)

			// Stripe is called with the client-facing total, not the net.
			// A regression here would either over-charge the client or
			// under-charge the platform.
			require.Len(t, stripeStub.intentCalls, 1)
			assert.Equal(t, tc.amountCents+out.StripeFee, stripeStub.intentCalls[0].AmountCentimes)

			// Sanity: the billing schedule's view of this (role, amount)
			// matches what the service persisted — catches a subtle bug
			// where the service and the fee-preview endpoint might drift.
			scheduled := billing.Calculate(tc.wantBillingRole, tc.amountCents)
			assert.Equal(t, scheduled.FeeCents, records.persisted.PlatformFeeAmount)
		})
	}
}

// TestCreatePaymentIntent_ProviderLookupFailure documents the fail-fast
// behaviour: if the provider user cannot be resolved, we refuse to create
// a payment record rather than persist a zero fee (which would under-charge
// the platform) or a guessed fee (which would violate the Single Source of
// Truth for the schedule).
func TestCreatePaymentIntent_ProviderLookupFailure(t *testing.T) {
	records := &feeStubRecords{}
	svc := NewService(ServiceDeps{
		Records: records,
		Users:   &feeStubUserRepo{user: nil}, // GetByID returns error
		Stripe:  &feeStubStripe{},
	})

	_, err := svc.CreatePaymentIntent(context.Background(), service.PaymentIntentInput{
		ProposalID:     uuid.New(),
		MilestoneID:    uuid.New(),
		ClientID:       uuid.New(),
		ProviderID:     uuid.New(),
		ProposalAmount: 50000,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "compute platform fee")
	assert.Nil(t, records.persisted, "no record must be persisted on failure")
}

// TestCreatePaymentIntent_PremiumWaiver is the Phase B contract: whenever
// the provider has an active Premium subscription, the platform fee MUST
// be zero on EVERY (role, amount) combination. Covers both grids
// (freelance and agency) × every tier bracket + boundary cases so a
// regression in computePlatformFee / billing.Calculate wiring fails
// loudly and specifically.
func TestCreatePaymentIntent_PremiumWaiver(t *testing.T) {
	tests := []struct {
		name         string
		role         domainuser.Role
		amountCents  int64
		subscribed   bool
		wantFeeCents int64
		wantNetCents int64
	}{
		// Freelance — not subscribed (control): the normal grid applies.
		{"freelance tier1 free", domainuser.RoleProvider, 15000, false, 900, 14100},
		{"freelance tier2 free", domainuser.RoleProvider, 50000, false, 1500, 48500},
		{"freelance tier3 free", domainuser.RoleProvider, 200000, false, 2500, 197500},
		// Freelance — subscribed: fee waived across the whole grid.
		{"freelance tier1 premium", domainuser.RoleProvider, 15000, true, 0, 15000},
		{"freelance tier2 premium", domainuser.RoleProvider, 50000, true, 0, 50000},
		{"freelance tier3 premium", domainuser.RoleProvider, 200000, true, 0, 200000},
		// Freelance boundary 200€ exactly — not subscribed promotes tier.
		{"freelance boundary 200€ free", domainuser.RoleProvider, 20000, false, 1500, 18500},
		// Freelance boundary 200€ exactly — subscribed still waives.
		{"freelance boundary 200€ premium", domainuser.RoleProvider, 20000, true, 0, 20000},

		// Agency — not subscribed (control): agency grid applies.
		{"agency tier1 free", domainuser.RoleAgency, 40000, false, 1900, 38100},
		{"agency tier2 free", domainuser.RoleAgency, 150000, false, 3900, 146100},
		{"agency tier3 free", domainuser.RoleAgency, 500000, false, 6900, 493100},
		// Agency — subscribed: fee waived across the whole grid.
		{"agency tier1 premium", domainuser.RoleAgency, 40000, true, 0, 40000},
		{"agency tier2 premium", domainuser.RoleAgency, 150000, true, 0, 150000},
		{"agency tier3 premium", domainuser.RoleAgency, 500000, true, 0, 500000},
		// Agency boundary 2500€ promotes tier.
		{"agency boundary 2500€ free", domainuser.RoleAgency, 250000, false, 6900, 243100},
		{"agency boundary 2500€ premium", domainuser.RoleAgency, 250000, true, 0, 250000},

		// Edge: large amount (10k€) — waiver still applies.
		{"freelance 10k€ premium", domainuser.RoleProvider, 1_000_000, true, 0, 1_000_000},
		{"agency 10k€ premium", domainuser.RoleAgency, 1_000_000, true, 0, 1_000_000},

		// Edge: amount exactly equals fee tier upper bound. No matter
		// the tier, premium = 0 fee.
		{"freelance tier boundary premium", domainuser.RoleProvider, 100000, true, 0, 100000},
		{"agency tier boundary premium", domainuser.RoleAgency, 50000, true, 0, 50000},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc, records := newFeeTestServiceWithSubscription(tc.role, tc.subscribed)

			out, err := svc.CreatePaymentIntent(context.Background(), service.PaymentIntentInput{
				ProposalID:     uuid.New(),
				MilestoneID:    uuid.New(),
				ClientID:       uuid.New(),
				ProviderID:     uuid.New(),
				ProposalAmount: tc.amountCents,
			})
			require.NoError(t, err)

			assert.Equal(t, tc.wantFeeCents, out.PlatformFee, "fee must match waiver state")
			assert.Equal(t, tc.wantNetCents, out.ProviderPayout, "provider payout = amount - fee")

			require.NotNil(t, records.persisted)
			assert.Equal(t, tc.wantFeeCents, records.persisted.PlatformFeeAmount)
			assert.Equal(t, tc.wantNetCents, records.persisted.ProviderPayout)
		})
	}
}

// TestCreatePaymentIntent_SubscriptionReader_Absent_FullFee proves that
// removing the subscription feature (SetSubscriptionReader never called,
// reader stays nil) leaves the payment service in its pre-Premium state:
// every milestone gets the full grid fee. This is the "removable
// feature" invariant expressed as an executable test.
func TestCreatePaymentIntent_SubscriptionReader_Absent_FullFee(t *testing.T) {
	records := &feeStubRecords{}
	svc := NewService(ServiceDeps{
		Records: records,
		Users: &feeStubUserRepo{user: &domainuser.User{
			ID:   uuid.New(),
			Role: domainuser.RoleProvider,
		}},
		Stripe: &feeStubStripe{},
	})
	// NOTE: SetSubscriptionReader intentionally NOT called.

	_, err := svc.CreatePaymentIntent(context.Background(), service.PaymentIntentInput{
		ProposalID:     uuid.New(),
		MilestoneID:    uuid.New(),
		ClientID:       uuid.New(),
		ProviderID:     uuid.New(),
		ProposalAmount: 50000,
	})
	require.NoError(t, err)
	require.NotNil(t, records.persisted)
	assert.Equal(t, int64(1500), records.persisted.PlatformFeeAmount,
		"without subscription reader, full grid fee MUST apply")
}

// TestCreatePaymentIntent_SubscriptionReaderFailure_FailsToFullFee
// documents the conservative fallback: when the reader errors (cache
// down, DB blip), we apply the FULL fee rather than risk granting a
// free milestone to a potentially non-subscribed user. A genuinely
// subscribed user affected by this will be refunded via support.
func TestCreatePaymentIntent_SubscriptionReaderFailure_FailsToFullFee(t *testing.T) {
	records := &feeStubRecords{}
	svc := NewService(ServiceDeps{
		Records: records,
		Users: &feeStubUserRepo{user: &domainuser.User{
			ID:   uuid.New(),
			Role: domainuser.RoleProvider,
		}},
		Stripe: &feeStubStripe{},
	})
	svc.SetSubscriptionReader(&feeStubSubscriptionReader{
		err: errors.New("redis down"),
	})

	_, err := svc.CreatePaymentIntent(context.Background(), service.PaymentIntentInput{
		ProposalID:     uuid.New(),
		MilestoneID:    uuid.New(),
		ClientID:       uuid.New(),
		ProviderID:     uuid.New(),
		ProposalAmount: 50000,
	})
	require.NoError(t, err, "a transient subscription-reader failure must NOT block the payment")
	require.NotNil(t, records.persisted)
	assert.Equal(t, int64(1500), records.persisted.PlatformFeeAmount,
		"reader failure MUST apply the full fee (fail closed)")
}
