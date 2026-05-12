package payment

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domain "marketplace-backend/internal/domain/payment"
	"marketplace-backend/internal/port/service"
)

// stubPerMilestoneInvoicer records every IssueFromMilestone call and
// satisfies service.PerMilestoneInvoicer. Thread-safe so race tests
// (and the concurrent fixture mutation in the stub records repo)
// don't trip -race.
type stubPerMilestoneInvoicer struct {
	mu    sync.Mutex
	calls []uuid.UUID
	err   error
}

// Compile-time check — the stub MUST stay in lock-step with the port.
var _ service.PerMilestoneInvoicer = (*stubPerMilestoneInvoicer)(nil)

func (s *stubPerMilestoneInvoicer) IssueFromMilestone(_ context.Context, milestoneID uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls = append(s.calls, milestoneID)
	return s.err
}

func (s *stubPerMilestoneInvoicer) callCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.calls)
}

func (s *stubPerMilestoneInvoicer) lastMilestone() uuid.UUID {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.calls) == 0 {
		return uuid.Nil
	}
	return s.calls[len(s.calls)-1]
}

// ---------------------------------------------------------------------
// TransferMilestone — the per-milestone release path that auto-fires
// when CompleteProposal calls payments.TransferMilestone.
// ---------------------------------------------------------------------

func TestPayoutService_TransferMilestone_FiresInvoice_OnSuccess(t *testing.T) {
	rec := newSucceededPendingRecord()
	records := &payoutStubRecords{
		byMilestone: map[uuid.UUID]*domain.PaymentRecord{rec.MilestoneID: rec},
	}
	orgs := &payoutStubOrgs{stripeAccountID: "acct_test"}
	stripe := &payoutStubStripe{}
	inv := &stubPerMilestoneInvoicer{}

	p := NewPayoutService(PayoutServiceDeps{Records: records, Organizations: orgs, Stripe: stripe})
	p.SetPerMilestoneInvoicer(inv)

	err := p.TransferMilestone(context.Background(), rec.MilestoneID)
	require.NoError(t, err)
	assert.Len(t, stripe.transferCalls, 1, "stripe transfer must have fired")
	require.Equal(t, 1, inv.callCount(),
		"invoice must fire exactly once after a successful transfer")
	assert.Equal(t, rec.MilestoneID, inv.lastMilestone(),
		"invoicer must receive the just-transferred milestone id")
}

func TestPayoutService_TransferMilestone_DoesNotFireInvoice_OnStripeFail(t *testing.T) {
	rec := newSucceededPendingRecord()
	records := &payoutStubRecords{
		byMilestone: map[uuid.UUID]*domain.PaymentRecord{rec.MilestoneID: rec},
	}
	orgs := &payoutStubOrgs{stripeAccountID: "acct_test"}
	stripe := &payoutStubStripe{transferErr: errors.New("stripe down")}
	inv := &stubPerMilestoneInvoicer{}

	p := NewPayoutService(PayoutServiceDeps{Records: records, Organizations: orgs, Stripe: stripe})
	p.SetPerMilestoneInvoicer(inv)

	err := p.TransferMilestone(context.Background(), rec.MilestoneID)
	require.Error(t, err, "stripe failure must surface")
	assert.Equal(t, 0, inv.callCount(),
		"invoice must NOT fire when the transfer itself failed — the legal trigger is transfer.completed, not transfer.attempted")
}

func TestPayoutService_TransferMilestone_DoesNotFireInvoice_AlreadyDone(t *testing.T) {
	rec := newSucceededPendingRecord()
	rec.TransferStatus = domain.TransferCompleted
	records := &payoutStubRecords{
		byMilestone: map[uuid.UUID]*domain.PaymentRecord{rec.MilestoneID: rec},
	}
	inv := &stubPerMilestoneInvoicer{}

	p := NewPayoutService(PayoutServiceDeps{Records: records, Organizations: &payoutStubOrgs{}, Stripe: &payoutStubStripe{}})
	p.SetPerMilestoneInvoicer(inv)

	err := p.TransferMilestone(context.Background(), rec.MilestoneID)
	assert.ErrorIs(t, err, domain.ErrTransferAlreadyDone)
	assert.Equal(t, 0, inv.callCount(),
		"invoice must NOT fire on a no-op transfer — the original transfer.completed already fired it")
}

func TestPayoutService_TransferMilestone_DoesNotFireInvoice_NoStripeAccount(t *testing.T) {
	rec := newSucceededPendingRecord()
	records := &payoutStubRecords{
		byMilestone: map[uuid.UUID]*domain.PaymentRecord{rec.MilestoneID: rec},
	}
	orgs := &payoutStubOrgs{stripeAccountID: ""}
	inv := &stubPerMilestoneInvoicer{}

	p := NewPayoutService(PayoutServiceDeps{Records: records, Organizations: orgs, Stripe: &payoutStubStripe{}})
	p.SetPerMilestoneInvoicer(inv)

	err := p.TransferMilestone(context.Background(), rec.MilestoneID)
	assert.ErrorIs(t, err, domain.ErrStripeAccountNotFound)
	assert.Equal(t, 0, inv.callCount(),
		"invoice must NOT fire when the Stripe transfer can't even be attempted")
}

func TestPayoutService_TransferMilestone_InvoicerErrorIsSwallowed(t *testing.T) {
	rec := newSucceededPendingRecord()
	records := &payoutStubRecords{
		byMilestone: map[uuid.UUID]*domain.PaymentRecord{rec.MilestoneID: rec},
	}
	orgs := &payoutStubOrgs{stripeAccountID: "acct_test"}
	stripe := &payoutStubStripe{}
	inv := &stubPerMilestoneInvoicer{err: errors.New("invoicing service down")}

	p := NewPayoutService(PayoutServiceDeps{Records: records, Organizations: orgs, Stripe: stripe})
	p.SetPerMilestoneInvoicer(inv)

	err := p.TransferMilestone(context.Background(), rec.MilestoneID)
	require.NoError(t, err,
		"invoicer errors must be swallowed (best-effort, monthly safety-net retries)")
	assert.Equal(t, 1, inv.callCount(),
		"invoicer was attempted")
	require.NotEmpty(t, records.updates)
	assert.Equal(t, domain.TransferCompleted, records.updates[0].TransferStatus,
		"transfer state must remain Completed — the transfer succeeded even if invoicing failed")
}

// TestPayoutService_TransferMilestone_NilInvoicer_NoOp confirms the
// feature can be disabled at startup (legacy / partial wiring) without
// breaking the transfer path.
func TestPayoutService_TransferMilestone_NilInvoicer_NoOp(t *testing.T) {
	rec := newSucceededPendingRecord()
	records := &payoutStubRecords{
		byMilestone: map[uuid.UUID]*domain.PaymentRecord{rec.MilestoneID: rec},
	}
	orgs := &payoutStubOrgs{stripeAccountID: "acct_test"}
	stripe := &payoutStubStripe{}

	p := NewPayoutService(PayoutServiceDeps{Records: records, Organizations: orgs, Stripe: stripe})
	// Note: no SetPerMilestoneInvoicer call.

	err := p.TransferMilestone(context.Background(), rec.MilestoneID)
	require.NoError(t, err)
	require.NotEmpty(t, records.updates)
	assert.Equal(t, domain.TransferCompleted, records.updates[0].TransferStatus)
}

// ---------------------------------------------------------------------
// TransferCompletedRetry — replayed transfer must not double-invoice.
// ---------------------------------------------------------------------

// TestPayoutService_TransferCompletedRetry simulates the "replayed
// webhook" scenario: the same milestone receives two
// TransferMilestone calls in sequence. The first succeeds and fires
// the invoice; the second short-circuits via ErrTransferAlreadyDone
// and MUST NOT fire a second invoice. The DB-level partial UNIQUE
// index would catch a duplicate write at the persistence layer, but
// we want the app layer to bail BEFORE we even reach the invoicer to
// avoid a useless RPC.
func TestPayoutService_TransferCompletedRetry_DoesNotDoubleInvoice(t *testing.T) {
	rec := newSucceededPendingRecord()
	records := &payoutStubRecords{
		byMilestone: map[uuid.UUID]*domain.PaymentRecord{rec.MilestoneID: rec},
	}
	orgs := &payoutStubOrgs{stripeAccountID: "acct_test"}
	stripe := &payoutStubStripe{}
	inv := &stubPerMilestoneInvoicer{}

	p := NewPayoutService(PayoutServiceDeps{Records: records, Organizations: orgs, Stripe: stripe})
	p.SetPerMilestoneInvoicer(inv)

	// First call — happy path.
	require.NoError(t, p.TransferMilestone(context.Background(), rec.MilestoneID))
	require.Equal(t, 1, inv.callCount(),
		"first transfer must fire the invoice exactly once")

	// Second call — replayed event. The fixture's Update mirrors the
	// new state back into byMilestone, so the second call observes
	// TransferCompleted and short-circuits via ErrTransferAlreadyDone.
	err := p.TransferMilestone(context.Background(), rec.MilestoneID)
	assert.ErrorIs(t, err, domain.ErrTransferAlreadyDone)
	assert.Equal(t, 1, inv.callCount(),
		"replayed transfer must NOT fire a second invoice (idempotence guard)")
}

// ---------------------------------------------------------------------
// RequestPayout — wallet "Retirer" button: drains every completed
// milestone of the org.
// ---------------------------------------------------------------------

func TestPayoutService_RequestPayout_FiresInvoicePerTransferredRecord(t *testing.T) {
	completedProposalID := uuid.New()
	r1 := newSucceededPendingRecord()
	r1.ProposalID = completedProposalID
	r2 := newSucceededPendingRecord()
	r2.ProposalID = completedProposalID

	records := &payoutStubRecords{
		byOrganization: []*domain.PaymentRecord{r1, r2},
		byMilestone: map[uuid.UUID]*domain.PaymentRecord{
			r1.MilestoneID: r1,
			r2.MilestoneID: r2,
		},
		byID: map[uuid.UUID]*domain.PaymentRecord{r1.ID: r1, r2.ID: r2},
	}
	orgs := &payoutStubOrgs{stripeAccountID: "acct_test"}
	stripe := &payoutStubStripe{}
	inv := &stubPerMilestoneInvoicer{}
	statuses := &payoutStubProposalStatuses{
		statuses: map[uuid.UUID]string{completedProposalID: "completed"},
	}

	p := NewPayoutService(PayoutServiceDeps{Records: records, Organizations: orgs, Stripe: stripe})
	p.SetPerMilestoneInvoicer(inv)
	p.SetProposalStatusReader(statuses)

	result, err := p.RequestPayout(context.Background(), uuid.New(), uuid.New())
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "transferred", result.Status)
	assert.Equal(t, 2, inv.callCount(),
		"invoice must fire once per transferred record (2 milestones → 2 invoices)")
}

func TestPayoutService_RequestPayout_DoesNotFireForSkippedRecords(t *testing.T) {
	// Mixed batch: r1 completed (transferred + invoiced), r2 status
	// failed (skipped — no transfer, no invoice).
	completedProposalID := uuid.New()
	r1 := newSucceededPendingRecord()
	r1.ProposalID = completedProposalID
	r2 := newSucceededPendingRecord()
	r2.ProposalID = completedProposalID
	r2.TransferStatus = domain.TransferFailed // skipped by the loop

	records := &payoutStubRecords{
		byOrganization: []*domain.PaymentRecord{r1, r2},
		byMilestone: map[uuid.UUID]*domain.PaymentRecord{
			r1.MilestoneID: r1,
			r2.MilestoneID: r2,
		},
		byID: map[uuid.UUID]*domain.PaymentRecord{r1.ID: r1, r2.ID: r2},
	}
	orgs := &payoutStubOrgs{stripeAccountID: "acct_test"}
	stripe := &payoutStubStripe{}
	inv := &stubPerMilestoneInvoicer{}
	statuses := &payoutStubProposalStatuses{
		statuses: map[uuid.UUID]string{completedProposalID: "completed"},
	}

	p := NewPayoutService(PayoutServiceDeps{Records: records, Organizations: orgs, Stripe: stripe})
	p.SetPerMilestoneInvoicer(inv)
	p.SetProposalStatusReader(statuses)

	_, err := p.RequestPayout(context.Background(), uuid.New(), uuid.New())
	require.NoError(t, err)
	assert.Equal(t, 1, inv.callCount(),
		"invoice must fire only for records that actually transferred")
	assert.Equal(t, r1.MilestoneID, inv.lastMilestone(),
		"the invoice must target the record that was just transferred, not the skipped one")
}

// ---------------------------------------------------------------------
// RetryFailedTransfer — recovery path for stuck transfers.
// ---------------------------------------------------------------------

func TestPayoutService_RetryFailedTransfer_FiresInvoice_OnSuccess(t *testing.T) {
	providerOrgID := uuid.New()
	rec := newSucceededPendingRecord()
	rec.TransferStatus = domain.TransferFailed // prerequisite for retry

	records := &payoutStubRecords{
		byMilestone: map[uuid.UUID]*domain.PaymentRecord{rec.MilestoneID: rec},
		byID:        map[uuid.UUID]*domain.PaymentRecord{rec.ID: rec},
	}
	orgs := &payoutStubOrgs{
		stripeAccountID: "acct_test",
		providerOrgID:   providerOrgID,
	}
	stripe := &payoutStubStripe{}
	inv := &stubPerMilestoneInvoicer{}
	statuses := &payoutStubProposalStatuses{
		statuses: map[uuid.UUID]string{rec.ProposalID: "completed"},
	}

	p := NewPayoutService(PayoutServiceDeps{Records: records, Organizations: orgs, Stripe: stripe})
	p.SetPerMilestoneInvoicer(inv)
	p.SetProposalStatusReader(statuses)

	_, err := p.RetryFailedTransfer(context.Background(), uuid.New(), providerOrgID, rec.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, inv.callCount(),
		"retry must fire the invoice exactly once on a successful retry")
	assert.Equal(t, rec.MilestoneID, inv.lastMilestone())
}

func TestPayoutService_RetryFailedTransfer_DoesNotFire_OnStripeFail(t *testing.T) {
	providerOrgID := uuid.New()
	rec := newSucceededPendingRecord()
	rec.TransferStatus = domain.TransferFailed

	records := &payoutStubRecords{
		byMilestone: map[uuid.UUID]*domain.PaymentRecord{rec.MilestoneID: rec},
		byID:        map[uuid.UUID]*domain.PaymentRecord{rec.ID: rec},
	}
	orgs := &payoutStubOrgs{
		stripeAccountID: "acct_test",
		providerOrgID:   providerOrgID,
	}
	stripe := &payoutStubStripe{transferErr: errors.New("still failing")}
	inv := &stubPerMilestoneInvoicer{}
	statuses := &payoutStubProposalStatuses{
		statuses: map[uuid.UUID]string{rec.ProposalID: "completed"},
	}

	p := NewPayoutService(PayoutServiceDeps{Records: records, Organizations: orgs, Stripe: stripe})
	p.SetPerMilestoneInvoicer(inv)
	p.SetProposalStatusReader(statuses)

	_, err := p.RetryFailedTransfer(context.Background(), uuid.New(), providerOrgID, rec.ID)
	require.Error(t, err)
	assert.Equal(t, 0, inv.callCount(),
		"failed retry must NOT fire the invoice (the legal trigger is transfer.completed, not transfer.attempted)")
}

// ---------------------------------------------------------------------
// Edge case — zero milestone id never reaches the invoicer.
// ---------------------------------------------------------------------

// TestPayoutService_FirePerMilestoneInvoice_ZeroMilestoneIsNoOp is a
// direct unit on the internal helper. The defensive guard exists
// because the phase-4 migration forbids zero milestone_id on
// payment_records but a stale legacy row could still surface a
// zero value — the invoicer must not be called with uuid.Nil.
func TestPayoutService_FirePerMilestoneInvoice_ZeroMilestoneIsNoOp(t *testing.T) {
	inv := &stubPerMilestoneInvoicer{}
	p := NewPayoutService(PayoutServiceDeps{Records: &payoutStubRecords{}, Organizations: &payoutStubOrgs{}, Stripe: &payoutStubStripe{}})
	p.SetPerMilestoneInvoicer(inv)

	p.firePerMilestoneInvoice(context.Background(), uuid.Nil)
	assert.Equal(t, 0, inv.callCount(), "zero milestone id must short-circuit before the invoicer is called")
}

// ---------------------------------------------------------------------
// Concurrency — two transfers on different milestones can fire
// in parallel without contention.
// ---------------------------------------------------------------------

func TestPayoutService_TransferMilestone_ConcurrentInvoicerCalls(t *testing.T) {
	// Two distinct milestones, two goroutines, single invoicer.
	rec1 := newSucceededPendingRecord()
	rec2 := newSucceededPendingRecord()
	records := &payoutStubRecords{
		byMilestone: map[uuid.UUID]*domain.PaymentRecord{
			rec1.MilestoneID: rec1,
			rec2.MilestoneID: rec2,
		},
	}
	orgs := &payoutStubOrgs{stripeAccountID: "acct_test"}
	stripe := &payoutStubStripe{}
	inv := &stubPerMilestoneInvoicer{}
	p := NewPayoutService(PayoutServiceDeps{Records: records, Organizations: orgs, Stripe: stripe})
	p.SetPerMilestoneInvoicer(inv)

	var wg sync.WaitGroup
	var errs int32
	wg.Add(2)
	for _, mid := range []uuid.UUID{rec1.MilestoneID, rec2.MilestoneID} {
		go func(milestoneID uuid.UUID) {
			defer wg.Done()
			if err := p.TransferMilestone(context.Background(), milestoneID); err != nil {
				atomic.AddInt32(&errs, 1)
			}
		}(mid)
	}
	wg.Wait()

	assert.Zero(t, atomic.LoadInt32(&errs), "both concurrent transfers must succeed")
	assert.Equal(t, 2, inv.callCount(),
		"each concurrent transfer must independently fire its invoice")
}
