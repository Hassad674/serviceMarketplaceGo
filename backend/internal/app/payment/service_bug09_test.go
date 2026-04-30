package payment

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domain "marketplace-backend/internal/domain/payment"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/port/service"
)

// ---------------------------------------------------------------------------
// BUG-09 — payment service swallows update errors after re-fetching a
// Stripe PaymentIntent / after MarkTransferFailed.
//
// The two patched locations:
//   1. createPaymentIntentFromExisting: when the existing record had no
//      StripePaymentIntentID, we set it from the Stripe response and
//      called `_ = s.records.Update(...)`. If the DB write failed, the
//      caller got a working ClientSecret backed by a record with the
//      OLD (empty) PI id — every subsequent transfer/refund targeted a
//      phantom PI.
//   2. TransferMilestone: when the Stripe CreateTransfer returned an
//      error, we set MarkTransferFailed and `_ = s.records.Update(...)`.
//      A DB write failure left the record as Succeeded+TransferPending,
//      so the wallet showed it ready to retry while the Stripe transfer
//      had permanently failed.
// ---------------------------------------------------------------------------

// existingPIRecords stubs only GetByMilestoneID / Update — the methods
// createPaymentIntentFromExisting actually exercises. Update can be
// configured to fail to simulate a DB blip.
type existingPIRecords struct {
	repository.PaymentRecordRepository
	record    *domain.PaymentRecord
	updateErr error
	updateCalls int
	updated   *domain.PaymentRecord
}

func (e *existingPIRecords) GetByMilestoneID(_ context.Context, _ uuid.UUID) (*domain.PaymentRecord, error) {
	if e.record == nil {
		return nil, domain.ErrPaymentRecordNotFound
	}
	cp := *e.record
	return &cp, nil
}

func (e *existingPIRecords) Update(_ context.Context, r *domain.PaymentRecord) error {
	e.updateCalls++
	if e.updateErr != nil {
		return e.updateErr
	}
	cp := *r
	e.updated = &cp
	return nil
}

func (e *existingPIRecords) Create(_ context.Context, _ *domain.PaymentRecord) error {
	return nil
}

// piRefreshStripe captures the CreatePaymentIntent calls so a test
// can assert which PI id the service tried to persist.
type piRefreshStripe struct {
	service.StripeService
	piResult *service.PaymentIntentResult
	piErr    error
}

func (p *piRefreshStripe) CreatePaymentIntent(_ context.Context, _ service.CreatePaymentIntentInput) (*service.PaymentIntentResult, error) {
	return p.piResult, p.piErr
}

// TestCreatePaymentIntentFromExisting_DBBlipSurfacesError pins the
// BUG-09 fix on the first patched location. Before the fix, a DB
// failure on the Update call was silently swallowed and the caller
// received a working ClientSecret backed by a desynced record.
func TestCreatePaymentIntentFromExisting_DBBlipSurfacesError(t *testing.T) {
	dbBlip := errors.New("connection reset by peer")

	rec := baseRecord()
	rec.StripePaymentIntentID = "" // empty → triggers the patched branch
	rec.MilestoneID = uuid.New()

	records := &existingPIRecords{record: rec, updateErr: dbBlip}
	stripeStub := &piRefreshStripe{
		// PaymentIntentID is what the service tries to persist on the
		// record. ClientSecret is what would be returned to the caller
		// when the persist succeeds.
		piResult: &service.PaymentIntentResult{
			PaymentIntentID: "pi_new_after_refetch",
			ClientSecret:    "cs_test_xxx",
		},
	}

	svc := NewService(ServiceDeps{Records: records, Stripe: stripeStub})

	// The service must call createPaymentIntentFromExisting because
	// existingPIRecords.GetByMilestoneID returns a non-nil record.
	out, err := svc.CreatePaymentIntent(context.Background(), service.PaymentIntentInput{
		ProposalID:  rec.ProposalID,
		MilestoneID: rec.MilestoneID,
		ClientID:    rec.ClientID,
		ProviderID:  rec.ProviderID,
	})

	require.Error(t, err, "BUG-09: DB blip on PI re-fetch persistence must NOT be swallowed")
	assert.ErrorIs(t, err, dbBlip, "the original DB error must be preserved")
	assert.Nil(t, out, "no working ClientSecret may be returned when the record is desynced")
	assert.Equal(t, 1, records.updateCalls, "Update was attempted once")
}

// TestCreatePaymentIntentFromExisting_HappyPath_PersistsAndReturns
// verifies the fix doesn't break the success path. The new PI id is
// persisted on the record; the call returns the ClientSecret.
func TestCreatePaymentIntentFromExisting_HappyPath_PersistsAndReturns(t *testing.T) {
	rec := baseRecord()
	rec.StripePaymentIntentID = "" // empty → triggers the persist branch
	rec.MilestoneID = uuid.New()

	records := &existingPIRecords{record: rec}
	stripeStub := &piRefreshStripe{
		piResult: &service.PaymentIntentResult{
			PaymentIntentID: "pi_new_yyy",
			ClientSecret:    "cs_test_yyy",
		},
	}

	svc := NewService(ServiceDeps{Records: records, Stripe: stripeStub})

	out, err := svc.CreatePaymentIntent(context.Background(), service.PaymentIntentInput{
		ProposalID:  rec.ProposalID,
		MilestoneID: rec.MilestoneID,
		ClientID:    rec.ClientID,
		ProviderID:  rec.ProviderID,
	})
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, "cs_test_yyy", out.ClientSecret)
	require.NotNil(t, records.updated)
}

// ---------------------------------------------------------------------------
// BUG-09 location 2: TransferMilestone MarkTransferFailed save failure.
// ---------------------------------------------------------------------------

// failingTransferStripe mimics Stripe rejecting CreateTransfer so the
// MarkTransferFailed path runs. Combined with a failing records.Update
// (orgRetryRecords), we exercise the BUG-09 location 2 fix.
type failingTransferStripe struct {
	service.StripeService
	transferErr error
}

func (f *failingTransferStripe) CreateTransfer(_ context.Context, _ service.CreateTransferInput) (string, error) {
	return "", f.transferErr
}

func (f *failingTransferStripe) GetAccount(_ context.Context, _ string) (*service.StripeAccountInfo, error) {
	return &service.StripeAccountInfo{ChargesEnabled: true, PayoutsEnabled: true}, nil
}

// transferMilestoneRecords stubs the milestoneID lookup + Update for
// the TransferMilestone path. updateErr is returned ONLY on the second
// call (the MarkTransferFailed save) — the first Update on success
// is never reached because Stripe fails first.
type transferMilestoneRecords struct {
	repository.PaymentRecordRepository
	record    *domain.PaymentRecord
	updateErr error
	updateCalls int
}

func (t *transferMilestoneRecords) GetByMilestoneID(_ context.Context, _ uuid.UUID) (*domain.PaymentRecord, error) {
	cp := *t.record
	return &cp, nil
}

func (t *transferMilestoneRecords) Update(_ context.Context, _ *domain.PaymentRecord) error {
	t.updateCalls++
	if t.updateErr != nil {
		return t.updateErr
	}
	return nil
}

// TestTransferMilestone_StripeFails_DBBlipOnFailMark_BothErrorsReported
// is the targeted regression test for BUG-09 location 2. Before the
// fix, the DB blip on the MarkTransferFailed save was silently swallowed.
// After the fix, both errors are reported in the wrapped error so ops
// can see the desync.
func TestTransferMilestone_StripeFails_DBBlipOnFailMark_BothErrorsReported(t *testing.T) {
	rec := baseRecord()
	rec.MilestoneID = uuid.New()
	rec.Status = domain.RecordStatusSucceeded
	rec.TransferStatus = domain.TransferPending
	rec.ProviderPayout = 1500

	stripeFailure := errors.New("stripe: rate limited")
	dbBlip := errors.New("write conflict on payment_records")

	records := &transferMilestoneRecords{
		record:    rec,
		updateErr: dbBlip,
	}
	orgs := &fakeOrgs{stripeAccountID: "acct_test_123"}
	stripeStub := &failingTransferStripe{transferErr: stripeFailure}

	svc := NewService(ServiceDeps{Records: records, Organizations: orgs, Stripe: stripeStub})

	err := svc.TransferMilestone(context.Background(), rec.MilestoneID)
	require.Error(t, err)
	// Stripe error wrapped (primary).
	assert.ErrorIs(t, err, stripeFailure)
	// DB save failure mentioned in the message so the caller / log
	// reader sees both sides of the desync.
	assert.Contains(t, err.Error(), "mark failed save also failed",
		"BUG-09 location 2: DB save failure must be surfaced alongside the Stripe error")
	assert.Contains(t, err.Error(), dbBlip.Error())

	// Update was attempted exactly once (the MarkTransferFailed save).
	assert.Equal(t, 1, records.updateCalls)
}

// TestTransferMilestone_StripeFails_DBOK_StripeErrorOnly is the
// normal path: Stripe failed, DB save OK. The Stripe error must be
// wrapped without any "save also failed" wording so existing log
// consumers keep working.
func TestTransferMilestone_StripeFails_DBOK_StripeErrorOnly(t *testing.T) {
	rec := baseRecord()
	rec.MilestoneID = uuid.New()
	rec.Status = domain.RecordStatusSucceeded
	rec.TransferStatus = domain.TransferPending
	rec.ProviderPayout = 1500

	stripeFailure := errors.New("stripe: bad request")

	records := &transferMilestoneRecords{record: rec}
	orgs := &fakeOrgs{stripeAccountID: "acct_test_123"}
	stripeStub := &failingTransferStripe{transferErr: stripeFailure}

	svc := NewService(ServiceDeps{Records: records, Organizations: orgs, Stripe: stripeStub})

	err := svc.TransferMilestone(context.Background(), rec.MilestoneID)
	require.Error(t, err)
	assert.ErrorIs(t, err, stripeFailure)
	assert.NotContains(t, err.Error(), "mark failed save also failed",
		"normal Stripe failure path must not mention the save error")
	assert.Equal(t, 1, records.updateCalls)
}

// TestTransferMilestone_HappyPath_StripeOK_DBOK regression — the BUG-09
// fix must not break the happy path. CreateTransfer succeeds, Update
// is called twice (once for MarkTransferred, no extra path).
func TestTransferMilestone_HappyPath_StripeOK_DBOK(t *testing.T) {
	rec := baseRecord()
	rec.MilestoneID = uuid.New()
	rec.Status = domain.RecordStatusSucceeded
	rec.TransferStatus = domain.TransferPending
	rec.ProviderPayout = 1500

	records := &milestoneRecords{
		byMilestone: map[uuid.UUID]*domain.PaymentRecord{rec.MilestoneID: rec},
	}
	orgs := &fakeOrgs{stripeAccountID: "acct_test_123"}
	stripeStub := &fakeStripe{}

	svc := NewService(ServiceDeps{Records: records, Organizations: orgs, Stripe: stripeStub})

	err := svc.TransferMilestone(context.Background(), rec.MilestoneID)
	require.NoError(t, err)
	require.Len(t, records.updated, 1, "exactly one Update on the success path")
	assert.Equal(t, domain.TransferCompleted, records.updated[0].TransferStatus)
}
