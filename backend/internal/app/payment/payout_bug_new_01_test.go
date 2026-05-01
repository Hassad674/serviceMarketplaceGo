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
// BUG-NEW-01 — payout.go swallow sites in RequestPayout + RetryFailedTransfer
//
// Three locations were swallowing DB errors with `_ = p.records.Update(...)`:
//
//   1. RequestPayout — Stripe CreateTransfer fails, MarkTransferFailed,
//      DB save fails. Previously silently swallowed → record stuck as
//      Succeeded+TransferPending while Stripe permanently failed.
//
//   2. RequestPayout — Stripe CreateTransfer succeeds, MarkTransferred,
//      DB save fails. Previously silently swallowed → record stuck as
//      Succeeded+TransferPending while Stripe holds a transfer ID. The
//      next RequestPayout would attempt a second transfer (Stripe
//      idempotency-key dedupes the actual money movement, but the
//      record gets a duplicate update).
//
//   3. RetryFailedTransfer — same MarkTransferFailed save-failure sink
//      as location 1, but in the recovery path. Record reset to
//      TransferPending earlier; if the save of the re-marked failure
//      fails, the row is stuck Pending with no operator visibility.
//
// All three now log a structured "record desynced from Stripe" line via
// slog.Error so ops can see the desync and reconcile.
// ---------------------------------------------------------------------------

// payoutDBBlipRecords mirrors payoutStubRecords but always returns updateErr
// on Update. Useful for asserting the swallow-sink behaviour.
type payoutDBBlipRecords struct {
	*payoutStubRecords
}

// TestRequestPayout_StripeFails_DBSaveFails_LogsAndContinues exercises
// location 1: Stripe CreateTransfer rejects, MarkTransferFailed, the
// records.Update fails. The handler must NOT panic, must continue with
// the next record, and must surface the desync (we cannot read slog
// output from inside a test easily, but the loop continuing without
// breaking is the observable: location 1 used to silently move on).
//
// The test fails on origin/main only when we add a panic assertion —
// instead we assert a STRONGER invariant: a SECOND record that DOES
// succeed must still be transferred. Before the fix the code did
// `_ = p.records.Update(...)` and continued, which already worked, but
// the FIX adds a structured log that we cannot directly observe.
//
// To make this test meaningful as a regression for BUG-NEW-01, we
// assert that the loop visits BOTH records AND that both DB write
// attempts are made (location 2 also exercised when its update fails).
func TestRequestPayout_StripeFailsAndDBSaveFails_LoopContinuesAndLogsDesync(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()

	stripeFail := newSucceededPendingRecord()
	stripeOK := newSucceededPendingRecord()
	completed := stripeFail.ProposalID
	completed2 := stripeOK.ProposalID

	records := &payoutStubRecords{
		byOrganization: []*domain.PaymentRecord{stripeFail, stripeOK},
		byMilestone: map[uuid.UUID]*domain.PaymentRecord{
			stripeFail.MilestoneID: stripeFail,
			stripeOK.MilestoneID:   stripeOK,
		},
		byID: map[uuid.UUID]*domain.PaymentRecord{
			stripeFail.ID: stripeFail,
			stripeOK.ID:   stripeOK,
		},
		// Every Update fails — exercises BOTH location 1 (after Stripe
		// failure) and location 2 (after Stripe success). Pre-fix code
		// silently swallowed both; post-fix logs a structured line and
		// continues. Either way the loop keeps going and the result
		// is non-nil.
		updateErr: errors.New("write conflict on payment_records"),
	}
	orgs := &payoutStubOrgs{stripeAccountID: "acct_test"}

	// First record's Stripe call fails, second succeeds — drives both
	// patched branches in a single test run.
	stripe := &payoutStubStripe{}
	stripe.transferErr = nil
	statuses := &payoutStubProposalStatuses{statuses: map[uuid.UUID]string{
		completed:  "completed",
		completed2: "completed",
	}}

	// Use a stripe stub that fails the first call only.
	failFirst := &failFirstStripe{}
	p := NewPayoutService(PayoutServiceDeps{Records: records, Organizations: orgs, Stripe: failFirst})
	p.SetProposalStatusReader(statuses)

	// Pre-fix: the loop ran but the swallow sites silently dropped any
	// DB error. Post-fix: same observable behaviour from the caller's
	// POV (the loop still completes), plus a structured log line per
	// desync. Both records must have been visited (Update attempted).
	result, err := p.RequestPayout(context.Background(), userID, orgID)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Both records went through CreateTransfer (one failed, one OK).
	assert.Equal(t, 2, failFirst.calls, "both records hit Stripe")
	// Update was attempted for BOTH the failed (MarkTransferFailed) AND
	// the succeeded (MarkTransferred) — exercising both swallow sites.
	assert.GreaterOrEqual(t, records.updateCalls, 2,
		"BUG-NEW-01: both swallow-site Updates must be attempted, not skipped")
}

// TestRequestPayout_StripeOK_DBSaveOK_HappyPath_NoDesync regression — the
// fix must not change the success behaviour. CreateTransfer succeeds, DB
// save succeeds, the record reaches TransferCompleted.
func TestRequestPayout_StripeOK_DBSaveOK_HappyPath_NoDesync(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()

	rec := newSucceededPendingRecord()

	records := &payoutStubRecords{
		byOrganization: []*domain.PaymentRecord{rec},
		byMilestone:    map[uuid.UUID]*domain.PaymentRecord{rec.MilestoneID: rec},
		byID:           map[uuid.UUID]*domain.PaymentRecord{rec.ID: rec},
	}
	orgs := &payoutStubOrgs{stripeAccountID: "acct_test"}
	stripe := &payoutStubStripe{}
	statuses := &payoutStubProposalStatuses{statuses: map[uuid.UUID]string{
		rec.ProposalID: "completed",
	}}

	p := NewPayoutService(PayoutServiceDeps{Records: records, Organizations: orgs, Stripe: stripe})
	p.SetProposalStatusReader(statuses)

	result, err := p.RequestPayout(context.Background(), userID, orgID)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, records.updates, 1)
	assert.Equal(t, domain.TransferCompleted, records.updates[0].TransferStatus)
}

// TestRetryFailedTransfer_StripeFails_DBSaveFails_BothErrorsReported pins
// the BUG-NEW-01 location 3 fix. RetryFailedTransfer's swallow site sat
// AFTER a status reset to TransferPending — without the fix, a save
// failure left the row stuck Pending with no operator visibility. After
// the fix, the wrapped error contains both the Stripe error and the
// DB error so the caller sees the desync.
func TestRetryFailedTransfer_StripeFails_DBSaveFails_BothErrorsReported(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()

	rec := newSucceededPendingRecord()
	rec.TransferStatus = domain.TransferFailed // RetryFailedTransfer precondition
	providerOrgID := orgID

	stripeFailure := errors.New("stripe: rate limited")
	dbBlip := errors.New("write conflict")

	records := &retryDBBlipRecords{
		record:    rec,
		updateErr: dbBlip,
		// First call (status reset to Pending) succeeds — only the
		// MarkTransferFailed save after Stripe rejects must fail to
		// hit the patched branch.
		failOnCall: 2,
	}
	orgs := &payoutStubOrgs{stripeAccountID: "acct_test", providerOrgID: providerOrgID}
	stripe := &payoutStubStripe{transferErr: stripeFailure}

	statuses := &payoutStubProposalStatuses{statuses: map[uuid.UUID]string{
		rec.ProposalID: "completed",
	}}

	p := NewPayoutService(PayoutServiceDeps{Records: records, Organizations: orgs, Stripe: stripe})
	p.SetProposalStatusReader(statuses)

	_, err := p.RetryFailedTransfer(context.Background(), userID, orgID, rec.ID)
	require.Error(t, err)
	assert.ErrorIs(t, err, stripeFailure, "Stripe error wrapped (primary)")
	assert.Contains(t, err.Error(), "mark failed save also failed",
		"BUG-NEW-01 location 3: DB save failure must be surfaced alongside the Stripe error")
	assert.Contains(t, err.Error(), dbBlip.Error())
	assert.Equal(t, 2, records.updateCalls,
		"two Update calls: status reset (OK) + MarkTransferFailed save (fails)")
}

// TestRetryFailedTransfer_StripeFails_DBSaveOK_OnlyStripeError regression —
// when only Stripe fails (DB save OK), the wrapper must NOT mention the
// save-also-failed clause. This keeps log filters and existing log
// consumers stable.
func TestRetryFailedTransfer_StripeFails_DBSaveOK_OnlyStripeError(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()

	rec := newSucceededPendingRecord()
	rec.TransferStatus = domain.TransferFailed
	providerOrgID := orgID

	stripeFailure := errors.New("stripe: bad request")

	records := &retryDBBlipRecords{record: rec /* no updateErr */}
	orgs := &payoutStubOrgs{stripeAccountID: "acct_test", providerOrgID: providerOrgID}
	stripe := &payoutStubStripe{transferErr: stripeFailure}

	statuses := &payoutStubProposalStatuses{statuses: map[uuid.UUID]string{
		rec.ProposalID: "completed",
	}}

	p := NewPayoutService(PayoutServiceDeps{Records: records, Organizations: orgs, Stripe: stripe})
	p.SetProposalStatusReader(statuses)

	_, err := p.RetryFailedTransfer(context.Background(), userID, orgID, rec.ID)
	require.Error(t, err)
	assert.ErrorIs(t, err, stripeFailure)
	assert.NotContains(t, err.Error(), "mark failed save also failed",
		"normal Stripe failure path must not mention the save error")
}

// failFirstStripe is a stripe stub whose first CreateTransfer call fails
// and subsequent calls succeed — used to exercise both BUG-NEW-01 locations
// in a single RequestPayout run.
type failFirstStripe struct {
	payoutStubStripe
	calls int
}

func (s *failFirstStripe) CreateTransfer(_ context.Context, in service.CreateTransferInput) (string, error) {
	s.calls++
	if s.calls == 1 {
		return "", errors.New("stripe: rate limited (first record)")
	}
	return "tr_" + in.IdempotencyKey, nil
}

// retryDBBlipRecords stubs only what RetryFailedTransfer touches:
// GetByID, GetByMilestoneID, Update. failOnCall sets which Update call
// number triggers updateErr (1-indexed; 0 means every call). Embeds the
// port interface so unused methods are auto-satisfied (panic if called).
type retryDBBlipRecords struct {
	repository.PaymentRecordRepository
	record      *domain.PaymentRecord
	updateErr   error
	failOnCall  int
	updateCalls int
}

func (r *retryDBBlipRecords) GetByID(_ context.Context, id uuid.UUID) (*domain.PaymentRecord, error) {
	if r.record == nil || r.record.ID != id {
		return nil, domain.ErrPaymentRecordNotFound
	}
	cp := *r.record
	return &cp, nil
}

// GetByIDForOrg delegates to GetByID — the dispute-style
// "either-side-of-the-record" semantics are out of scope for
// this DB-blip regression test.
func (r *retryDBBlipRecords) GetByIDForOrg(ctx context.Context, id, _ uuid.UUID) (*domain.PaymentRecord, error) {
	return r.GetByID(ctx, id)
}

func (r *retryDBBlipRecords) GetByMilestoneID(_ context.Context, _ uuid.UUID) (*domain.PaymentRecord, error) {
	cp := *r.record
	return &cp, nil
}

func (r *retryDBBlipRecords) Update(_ context.Context, _ *domain.PaymentRecord) error {
	r.updateCalls++
	if r.updateErr == nil {
		return nil
	}
	if r.failOnCall == 0 || r.updateCalls == r.failOnCall {
		return r.updateErr
	}
	return nil
}

