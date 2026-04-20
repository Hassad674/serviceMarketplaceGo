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
