package payment

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	domain "marketplace-backend/internal/domain/payment"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/port/service"
)

// fakeRecords is a minimal PaymentRecordRepository stub. Only GetByProposalID
// and Update are overridden — other methods will panic if called, which is
// fine because TransferPartialToProvider only touches these two.
type fakeRecords struct {
	repository.PaymentRecordRepository
	record     *domain.PaymentRecord
	updatedRec *domain.PaymentRecord
}

func (f *fakeRecords) GetByProposalID(_ context.Context, _ uuid.UUID) (*domain.PaymentRecord, error) {
	copy := *f.record
	return &copy, nil
}

func (f *fakeRecords) Update(_ context.Context, r *domain.PaymentRecord) error {
	f.updatedRec = r
	return nil
}

// fakeUsers stubs UserRepository; only GetStripeAccount is exercised.
type fakeUsers struct {
	repository.UserRepository
	stripeAccountID string
}

func (f *fakeUsers) GetStripeAccount(_ context.Context, _ uuid.UUID) (string, string, error) {
	return f.stripeAccountID, "FR", nil
}

// fakeStripe stubs StripeService; only CreateTransfer and CreateRefund are
// touched by the partial/refund flows, but the tests below target
// TransferPartialToProvider so only CreateTransfer matters.
type fakeStripe struct {
	service.StripeService
	transferCalls []service.CreateTransferInput
}

func (f *fakeStripe) CreateTransfer(_ context.Context, in service.CreateTransferInput) (string, error) {
	f.transferCalls = append(f.transferCalls, in)
	return "tr_test_" + in.IdempotencyKey, nil
}

func baseRecord() *domain.PaymentRecord {
	return &domain.PaymentRecord{
		ID:             uuid.New(),
		ProposalID:     uuid.New(),
		ProviderID:     uuid.New(),
		ClientID:       uuid.New(),
		Currency:       "eur",
		ProposalAmount: 1000,
		ProviderPayout: 1000, // original payout before split
		Status:         domain.RecordStatusSucceeded,
		TransferStatus: domain.TransferPending,
	}
}

// The bug: when the provider has no Stripe account (KYC not done), the old
// code returned ErrStripeAccountNotFound before updating the record. The
// wallet kept the original payout, so a later RequestPayout transferred the
// full amount instead of the dispute split — over-paying the provider.
//
// The fix persists the new ProviderPayout with TransferPending so a post-KYC
// RequestPayout picks up the correct (split) amount.
func TestTransferPartialToProvider_NoStripeAccount_PersistsSplitAmount(t *testing.T) {
	rec := baseRecord()
	records := &fakeRecords{record: rec}
	users := &fakeUsers{stripeAccountID: ""} // KYC not done
	stripe := &fakeStripe{}

	svc := NewService(ServiceDeps{
		Records: records,
		Users:   users,
		Stripe:  stripe,
	})

	// Admin decides the provider gets 700 (original was 1000).
	err := svc.TransferPartialToProvider(context.Background(), rec.ProposalID, 700)
	assert.NoError(t, err, "no-KYC case is a valid state, not an error")

	assert.NotNil(t, records.updatedRec, "record must be persisted even without Stripe transfer")
	assert.Equal(t, int64(700), records.updatedRec.ProviderPayout,
		"ProviderPayout must reflect the admin split, not the original amount")
	assert.Equal(t, domain.TransferPending, records.updatedRec.TransferStatus,
		"TransferStatus must stay Pending so RequestPayout retries with the correct amount")
	assert.Empty(t, stripe.transferCalls, "no Stripe transfer must be attempted without an account")
}

// Full refund (amount == 0) must mark the record completed with zero payout
// regardless of KYC status — nothing to transfer means nothing to defer.
func TestTransferPartialToProvider_FullRefund_MarksCompletedZeroPayout(t *testing.T) {
	rec := baseRecord()
	records := &fakeRecords{record: rec}
	users := &fakeUsers{stripeAccountID: ""} // KYC status is irrelevant here
	stripe := &fakeStripe{}

	svc := NewService(ServiceDeps{
		Records: records,
		Users:   users,
		Stripe:  stripe,
	})

	err := svc.TransferPartialToProvider(context.Background(), rec.ProposalID, 0)
	assert.NoError(t, err)
	assert.NotNil(t, records.updatedRec)
	assert.Equal(t, int64(0), records.updatedRec.ProviderPayout)
	assert.Equal(t, domain.TransferCompleted, records.updatedRec.TransferStatus)
	assert.Empty(t, stripe.transferCalls, "zero-amount must not hit Stripe")
}

// Happy path: KYC complete, partial split. Stripe transfer is called with
// the split amount and the record is marked completed.
func TestTransferPartialToProvider_KYCReady_TransfersAndMarksCompleted(t *testing.T) {
	rec := baseRecord()
	records := &fakeRecords{record: rec}
	users := &fakeUsers{stripeAccountID: "acct_test_123"}
	stripe := &fakeStripe{}

	svc := NewService(ServiceDeps{
		Records: records,
		Users:   users,
		Stripe:  stripe,
	})

	err := svc.TransferPartialToProvider(context.Background(), rec.ProposalID, 700)
	assert.NoError(t, err)
	assert.NotNil(t, records.updatedRec)
	assert.Equal(t, int64(700), records.updatedRec.ProviderPayout)
	assert.Equal(t, domain.TransferCompleted, records.updatedRec.TransferStatus)
	assert.Len(t, stripe.transferCalls, 1)
	assert.Equal(t, int64(700), stripe.transferCalls[0].Amount)
	assert.Equal(t, "acct_test_123", stripe.transferCalls[0].DestinationAccount)
}

// Not-succeeded records must be rejected — this preserves the invariant
// that you cannot distribute funds that are not held in escrow.
func TestTransferPartialToProvider_RecordNotSucceeded_Rejected(t *testing.T) {
	rec := baseRecord()
	rec.Status = domain.RecordStatusPending
	records := &fakeRecords{record: rec}
	users := &fakeUsers{stripeAccountID: "acct_test_123"}
	stripe := &fakeStripe{}

	svc := NewService(ServiceDeps{
		Records: records,
		Users:   users,
		Stripe:  stripe,
	})

	err := svc.TransferPartialToProvider(context.Background(), rec.ProposalID, 700)
	assert.ErrorIs(t, err, domain.ErrPaymentNotSucceeded)
	assert.Nil(t, records.updatedRec, "nothing must be persisted on a rejected record")
}
