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

// fakeOrgs stubs OrganizationRepository; only GetStripeAccountByUserID
// is exercised by the payment service tests.
type fakeOrgs struct {
	repository.OrganizationRepository
	stripeAccountID string
}

func (f *fakeOrgs) GetStripeAccountByUserID(_ context.Context, _ uuid.UUID) (string, string, error) {
	return f.stripeAccountID, "FR", nil
}
func (f *fakeOrgs) GetStripeAccount(_ context.Context, _ uuid.UUID) (string, string, error) {
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
	orgs := &fakeOrgs{stripeAccountID: ""} // KYC not done
	stripe := &fakeStripe{}

	svc := NewService(ServiceDeps{
		Records: records,
		Organizations: orgs,
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
	orgs := &fakeOrgs{stripeAccountID: ""} // KYC status is irrelevant here
	stripe := &fakeStripe{}

	svc := NewService(ServiceDeps{
		Records: records,
		Organizations: orgs,
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
	orgs := &fakeOrgs{stripeAccountID: "acct_test_123"}
	stripe := &fakeStripe{}

	svc := NewService(ServiceDeps{
		Records: records,
		Organizations: orgs,
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

// fakeProposalStatuses stubs ProposalStatusReader. The map keys are the
// proposal IDs; missing entries return "" with nil error (the contract's
// "unknown — do not transfer" sentinel).
type fakeProposalStatuses struct {
	statuses map[uuid.UUID]string
	calls    []uuid.UUID
}

func (f *fakeProposalStatuses) GetProposalStatus(_ context.Context, id uuid.UUID) (string, error) {
	f.calls = append(f.calls, id)
	return f.statuses[id], nil
}

// listingRecords exercises ListByOrganization for RequestPayout tests.
// Embedded PaymentRecordRepository keeps the unused methods panic-free.
type listingRecords struct {
	repository.PaymentRecordRepository
	records  []*domain.PaymentRecord
	updated  []*domain.PaymentRecord
}

func (l *listingRecords) ListByOrganization(_ context.Context, _ uuid.UUID) ([]*domain.PaymentRecord, error) {
	out := make([]*domain.PaymentRecord, 0, len(l.records))
	for _, r := range l.records {
		copy := *r
		out = append(out, &copy)
	}
	return out, nil
}

func (l *listingRecords) Update(_ context.Context, r *domain.PaymentRecord) error {
	l.updated = append(l.updated, r)
	return nil
}

// The bug: RequestPayout filtered on Status=succeeded + TransferStatus=pending
// but did NOT check the proposal's mission_status. The UI correctly hid
// escrow funds (missions still active) from AvailableAmount, but a button
// click still pulled everything. The fix gates transfers on
// mission_status=="completed" via a ProposalStatusReader port.
func TestRequestPayout_SkipsEscrowWhenMissionNotCompleted(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	activeProposal := uuid.New()
	completedProposal := uuid.New()

	rec1 := baseRecord()
	rec1.ProposalID = activeProposal
	rec1.ProviderPayout = 400

	rec2 := baseRecord()
	rec2.ProposalID = completedProposal
	rec2.ProviderPayout = 600

	records := &listingRecords{records: []*domain.PaymentRecord{rec1, rec2}}
	orgs := &fakeOrgs{stripeAccountID: "acct_test_123"}
	stripe := &fakeStripe{}
	statuses := &fakeProposalStatuses{statuses: map[uuid.UUID]string{
		activeProposal:    "active",
		completedProposal: "completed",
	}}

	svc := NewService(ServiceDeps{
		Records:       records,
		Organizations: orgs,
		Stripe:        stripe,
	})
	svc.SetProposalStatusReader(statuses)

	result, err := svc.RequestPayout(context.Background(), userID, orgID)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Only the completed proposal must hit Stripe. The active one stays
	// in escrow even though its payment record is succeeded+pending.
	assert.Len(t, stripe.transferCalls, 1, "exactly one transfer for the completed mission")
	assert.Equal(t, int64(600), stripe.transferCalls[0].Amount)
	assert.Equal(t, completedProposal.String(), stripe.transferCalls[0].TransferGroup)

	// Both records are looked up for their status, only the completed one is updated.
	assert.Contains(t, statuses.calls, activeProposal)
	assert.Contains(t, statuses.calls, completedProposal)
	assert.Len(t, records.updated, 1, "only the transferred record is persisted")
	assert.Equal(t, completedProposal, records.updated[0].ProposalID)
	assert.Equal(t, domain.TransferCompleted, records.updated[0].TransferStatus)
}

// When the ProposalStatusReader is not wired (legacy / test bootstraps),
// the service falls back to the pre-fix behaviour: transfer everything
// succeeded+pending. This keeps the feature bootable without proposal
// but MUST log a warning so the degraded mode is never silent in prod.
func TestRequestPayout_NoStatusReader_FallsBackToLegacyBehaviour(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()

	rec := baseRecord()
	rec.ProviderPayout = 500

	records := &listingRecords{records: []*domain.PaymentRecord{rec}}
	orgs := &fakeOrgs{stripeAccountID: "acct_test_123"}
	stripe := &fakeStripe{}

	svc := NewService(ServiceDeps{
		Records:       records,
		Organizations: orgs,
		Stripe:        stripe,
	})
	// NOTE: no SetProposalStatusReader — the service must not crash
	// and must preserve the legacy behaviour so existing tests keep
	// passing until every caller wires the reader.

	_, err := svc.RequestPayout(context.Background(), userID, orgID)
	assert.NoError(t, err)
	assert.Len(t, stripe.transferCalls, 1, "fallback mode: transfer still happens")
}

// Not-succeeded records must be rejected — this preserves the invariant
// that you cannot distribute funds that are not held in escrow.
func TestTransferPartialToProvider_RecordNotSucceeded_Rejected(t *testing.T) {
	rec := baseRecord()
	rec.Status = domain.RecordStatusPending
	records := &fakeRecords{record: rec}
	orgs := &fakeOrgs{stripeAccountID: "acct_test_123"}
	stripe := &fakeStripe{}

	svc := NewService(ServiceDeps{
		Records: records,
		Organizations: orgs,
		Stripe:  stripe,
	})

	err := svc.TransferPartialToProvider(context.Background(), rec.ProposalID, 700)
	assert.ErrorIs(t, err, domain.ErrPaymentNotSucceeded)
	assert.Nil(t, records.updatedRec, "nothing must be persisted on a rejected record")
}
