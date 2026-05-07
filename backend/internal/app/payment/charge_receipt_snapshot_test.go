package payment

// Tests for the receipt-snapshot hook on ChargeService.CreatePaymentIntent.
// The snapshot resolver is wired post-construction via
// SetReceiptSnapshotResolver — when set, every new payment_record row
// must carry a JSONB billing_snapshot. When unset, the field stays nil
// and payment creation must still succeed (best-effort guarantee).

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domain "marketplace-backend/internal/domain/payment"
	"marketplace-backend/internal/port/service"
)

// stubReceiptResolver captures the input and returns a canned snapshot.
type stubReceiptResolver struct {
	calls       int
	gotInput    service.ReceiptSnapshotInput
	out         service.ReceiptSnapshot
	resolveErr  error
	marshalErr  error
	marshalled  []byte
	marshalls   int
}

func (s *stubReceiptResolver) ResolveForPayment(_ context.Context, in service.ReceiptSnapshotInput) (service.ReceiptSnapshot, error) {
	s.calls++
	s.gotInput = in
	if s.resolveErr != nil {
		return service.ReceiptSnapshot{}, s.resolveErr
	}
	return s.out, nil
}

func (s *stubReceiptResolver) MarshalSnapshot(snap service.ReceiptSnapshot) ([]byte, error) {
	s.marshalls++
	if s.marshalErr != nil {
		return nil, s.marshalErr
	}
	if s.marshalled != nil {
		return s.marshalled, nil
	}
	return []byte(`{"client":{"organization_id":"00000000-0000-0000-0000-000000000000","name":"X"}}`), nil
}

func newSnapshotChargeFixture(t *testing.T) (*ChargeService, *chargeStubRecords) {
	t.Helper()
	records := &chargeStubRecords{byMilestoneErr: domain.ErrPaymentRecordNotFound}
	stripe := &chargeStubStripe{}
	c := NewChargeService(ChargeServiceDeps{
		Records:       records,
		Stripe:        stripe,
		FeeCalculator: &stubFeeCalc{fee: 100},
	})
	return c, records
}

func TestChargeService_CreatePaymentIntent_NoSnapshotResolver_NilSnapshotJSON(t *testing.T) {
	c, records := newSnapshotChargeFixture(t)
	// resolver intentionally not set

	_, err := c.CreatePaymentIntent(context.Background(), service.PaymentIntentInput{
		ProposalID:     uuid.New(),
		MilestoneID:    uuid.New(),
		ClientID:       uuid.New(),
		ProviderID:     uuid.New(),
		ProposalAmount: 1000,
	})
	require.NoError(t, err)
	require.NotNil(t, records.createdRec)
	assert.Nil(t, records.createdRec.BillingSnapshotJSON, "no resolver → nil JSONB")
}

func TestChargeService_CreatePaymentIntent_WithResolver_PopulatesSnapshotJSON(t *testing.T) {
	c, records := newSnapshotChargeFixture(t)
	clientOrg := uuid.New()
	resolver := &stubReceiptResolver{
		out: service.ReceiptSnapshot{
			Client: service.ReceiptSnapshotParty{OrganizationID: clientOrg, Name: "Client SAS"},
		},
		marshalled: []byte(`{"client":{"organization_id":"` + clientOrg.String() + `","name":"Client SAS"}}`),
	}
	c.SetReceiptSnapshotResolver(resolver)

	clientID := uuid.New()
	providerID := uuid.New()
	proposalID := uuid.New()
	_, err := c.CreatePaymentIntent(context.Background(), service.PaymentIntentInput{
		ProposalID:     proposalID,
		MilestoneID:    uuid.New(),
		ClientID:       clientID,
		ProviderID:     providerID,
		ProposalAmount: 1000,
	})
	require.NoError(t, err)
	require.NotNil(t, records.createdRec)
	assert.NotNil(t, records.createdRec.BillingSnapshotJSON, "resolver hooked → JSONB populated")
	assert.Contains(t, string(records.createdRec.BillingSnapshotJSON), clientOrg.String())

	// Resolver must have been called exactly once with the right keys.
	assert.Equal(t, 1, resolver.calls)
	assert.Equal(t, clientID, resolver.gotInput.ClientUserID)
	assert.Equal(t, providerID, resolver.gotInput.ProviderUserID)
	assert.Equal(t, proposalID, resolver.gotInput.ProposalID)
	assert.Equal(t, 1, resolver.marshalls)
}

func TestChargeService_CreatePaymentIntent_ResolverError_PaymentStillSucceeds(t *testing.T) {
	c, records := newSnapshotChargeFixture(t)
	resolver := &stubReceiptResolver{resolveErr: errors.New("boom")}
	c.SetReceiptSnapshotResolver(resolver)

	_, err := c.CreatePaymentIntent(context.Background(), service.PaymentIntentInput{
		ProposalID:     uuid.New(),
		MilestoneID:    uuid.New(),
		ClientID:       uuid.New(),
		ProviderID:     uuid.New(),
		ProposalAmount: 1000,
	})
	require.NoError(t, err)
	require.NotNil(t, records.createdRec)
	assert.Nil(t, records.createdRec.BillingSnapshotJSON, "resolver err → swallow + nil JSONB")
	assert.Equal(t, 1, resolver.calls)
	assert.Equal(t, 0, resolver.marshalls, "marshal must be skipped when resolve failed")
}

func TestChargeService_CreatePaymentIntent_MarshalError_PaymentStillSucceeds(t *testing.T) {
	c, records := newSnapshotChargeFixture(t)
	resolver := &stubReceiptResolver{
		out:        service.ReceiptSnapshot{Client: service.ReceiptSnapshotParty{OrganizationID: uuid.New()}},
		marshalErr: errors.New("invalid utf-8"),
	}
	c.SetReceiptSnapshotResolver(resolver)

	_, err := c.CreatePaymentIntent(context.Background(), service.PaymentIntentInput{
		ProposalID:     uuid.New(),
		MilestoneID:    uuid.New(),
		ClientID:       uuid.New(),
		ProviderID:     uuid.New(),
		ProposalAmount: 1000,
	})
	require.NoError(t, err)
	require.NotNil(t, records.createdRec)
	assert.Nil(t, records.createdRec.BillingSnapshotJSON, "marshal err → swallow + nil JSONB")
}
