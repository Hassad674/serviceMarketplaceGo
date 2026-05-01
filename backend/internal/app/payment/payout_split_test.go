package payment

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/domain/organization"
	domain "marketplace-backend/internal/domain/payment"
	"marketplace-backend/internal/port/service"
)

// ---------------------------------------------------------------------------
// Phase 3.1 lock-down tests — verify the payout.go → payout_transfer.go +
// payout_request.go split preserves behaviour at the facade boundary.
// Every public PayoutService method routed via the parent Service must
// keep the exact same surface and return type.
// ---------------------------------------------------------------------------

// TestServiceFacade_DelegatesAllPayoutMethods exercises every public
// payout method on the facade in a single test so a future split of any
// method to a new file (or accidentally dropping it from one file
// without re-adding it) trips this test. The actual behaviour is
// covered in the per-method tests in payout_test.go — here we only
// assert the facade routes through to a non-nil sub-service AND that
// the method returns the type its signature promises.
func TestServiceFacade_DelegatesAllPayoutMethods(t *testing.T) {
	rec := newSucceededPendingRecord()
	rec.StripePaymentIntentID = "pi_for_refund"

	failed := newSucceededPendingRecord()
	failed.TransferStatus = domain.TransferFailed
	failed.ProviderID = uuid.New()

	orgID := uuid.New()
	s := NewService(ServiceDeps{
		Records:       &facadeRecords{byMilestone: map[uuid.UUID]*domain.PaymentRecord{rec.MilestoneID: rec}, byProposalRec: rec, byID: map[uuid.UUID]*domain.PaymentRecord{failed.ID: failed}},
		Users:         &facadeUsers{},
		Organizations: facadeOrgs{stripeAccountID: "acct_facade"},
		Stripe:        facadeStripe{},
	})

	t.Run("TransferToProvider", func(t *testing.T) {
		require.NoError(t, s.TransferToProvider(context.Background(), rec.ProposalID))
	})

	t.Run("TransferMilestone", func(t *testing.T) {
		// Need a fresh record — the previous test transferred the only one.
		fresh := newSucceededPendingRecord()
		fresh2 := NewService(ServiceDeps{
			Records:       &facadeRecords{byMilestone: map[uuid.UUID]*domain.PaymentRecord{fresh.MilestoneID: fresh}},
			Users:         &facadeUsers{},
			Organizations: facadeOrgs{stripeAccountID: "acct_facade"},
			Stripe:        facadeStripe{},
		})
		require.NoError(t, fresh2.TransferMilestone(context.Background(), fresh.MilestoneID))
	})

	t.Run("TransferPartialToProvider", func(t *testing.T) {
		fresh := newSucceededPendingRecord()
		s := NewService(ServiceDeps{
			Records:       &facadeRecords{byProposalRec: fresh},
			Users:         &facadeUsers{},
			Organizations: facadeOrgs{stripeAccountID: "acct_facade"},
			Stripe:        facadeStripe{},
		})
		require.NoError(t, s.TransferPartialToProvider(context.Background(), fresh.ProposalID, 500))
	})

	t.Run("RefundToClient", func(t *testing.T) {
		require.NoError(t, s.RefundToClient(context.Background(), rec.ProposalID, 100))
	})

	t.Run("RequestPayout", func(t *testing.T) {
		out, err := s.RequestPayout(context.Background(), uuid.New(), orgID)
		require.NoError(t, err)
		require.NotNil(t, out)
	})

	t.Run("RetryFailedTransfer_RoutesToPayout", func(t *testing.T) {
		// Even though the call returns ErrTransferNotRetriable (no
		// matching org), the IMPORTANT lock-down here is "does the
		// method exist on the facade and route to PayoutService?". A
		// nil panic would mean the facade lost its delegation.
		_, err := s.RetryFailedTransfer(context.Background(), uuid.New(), uuid.New(), failed.ID)
		require.Error(t, err)
	})

	t.Run("CanProviderReceivePayouts", func(t *testing.T) {
		ok, err := s.CanProviderReceivePayouts(context.Background(), orgID)
		require.NoError(t, err)
		_ = ok // value-shape check; behaviour tested elsewhere
	})

	t.Run("HasAutoPayoutConsent", func(t *testing.T) {
		ok, err := s.HasAutoPayoutConsent(context.Background(), orgID)
		require.NoError(t, err)
		_ = ok
	})

	t.Run("WaivePlatformFeeOnActiveRecords", func(t *testing.T) {
		require.NoError(t, s.WaivePlatformFeeOnActiveRecords(context.Background(), orgID))
	})

	// Setters routed through the facade must reach the underlying
	// PayoutService — calling them with nil must not panic.
	t.Run("SetReferralDistributor_NilSafe", func(t *testing.T) {
		s.SetReferralDistributor(nil)
	})
	t.Run("SetProposalStatusReader_NilSafe", func(t *testing.T) {
		s.SetProposalStatusReader(nil)
	})
}

// TestServiceFacade_SatisfiesPaymentProcessorContract is a compile-time
// + runtime check that every PaymentProcessor method (the contract
// proposal depends on) is reachable through the facade after the split.
func TestServiceFacade_SatisfiesPaymentProcessorContract(t *testing.T) {
	var _ service.PaymentProcessor = (*Service)(nil)
	s := NewService(ServiceDeps{
		Records:       &facadeRecords{},
		Users:         &facadeUsers{},
		Organizations: facadeOrgs{stripeAccountID: "acct_x"},
		Stripe:        facadeStripe{},
	})
	var processor service.PaymentProcessor = s
	require.NotNil(t, processor)

	// PayoutService accessor must remain wired so future call sites
	// can apply ISP by depending on the focused sub-service rather
	// than the whole facade.
	require.NotNil(t, s.Payout())
}

// ---------------------------------------------------------------------------
// Edge-case coverage on helpers split off into payout_request.go so the
// transition does not lose any pre-split coverage. Targets the previously
// untested branches of recordAutoPayoutConsent + maybeStampRetryConsent
// + HasAutoPayoutConsent.
// ---------------------------------------------------------------------------

// payoutCoverageOrgs is a controllable OrganizationRepository stub for
// the lock-down + edge-case tests. The pre-existing payoutStubOrgs has
// hardcoded behaviour (FindByID always returns a non-nil org) which
// blocks tests for the nil-org and FindByID-error branches.
type payoutCoverageOrgs struct {
	payoutStubOrgs
	findByIDErr error
	findByIDNil bool
	updateErr   error
}

func (o *payoutCoverageOrgs) FindByID(ctx context.Context, id uuid.UUID) (*organization.Organization, error) {
	if o.findByIDErr != nil {
		return nil, o.findByIDErr
	}
	if o.findByIDNil {
		return nil, nil
	}
	return o.payoutStubOrgs.FindByID(ctx, id)
}

func (o *payoutCoverageOrgs) Update(_ context.Context, _ *organization.Organization) error {
	o.payoutStubOrgs.mu.Lock()
	o.payoutStubOrgs.updateCalls++
	o.payoutStubOrgs.mu.Unlock()
	return o.updateErr
}

func TestPayoutService_HasAutoPayoutConsent_FindErr_Wrapped(t *testing.T) {
	orgs := &payoutCoverageOrgs{findByIDErr: errors.New("db blip")}
	p := NewPayoutService(PayoutServiceDeps{Records: &payoutStubRecords{}, Organizations: orgs})

	got, err := p.HasAutoPayoutConsent(context.Background(), uuid.New())
	require.Error(t, err, "infra failure must surface — never silently return false")
	assert.Contains(t, err.Error(), "find org for auto-payout consent")
	assert.False(t, got)
}

func TestPayoutService_HasAutoPayoutConsent_OrgNil_FalseNoErr(t *testing.T) {
	orgs := &payoutCoverageOrgs{findByIDNil: true}
	p := NewPayoutService(PayoutServiceDeps{Records: &payoutStubRecords{}, Organizations: orgs})

	got, err := p.HasAutoPayoutConsent(context.Background(), uuid.New())
	require.NoError(t, err, "missing org fails closed but is not an error")
	assert.False(t, got)
}

// recordAutoPayoutConsent is reached only through fireBankPayout. We
// drive it via the public RequestPayout path with an org config that
// hits each branch.

func TestPayoutService_RequestPayout_RecordConsent_OrgFindErr_Swallowed(t *testing.T) {
	orgID := uuid.New()
	rec := newSucceededPendingRecord()
	records := &payoutStubRecords{byOrganization: []*domain.PaymentRecord{rec}}
	orgs := &payoutCoverageOrgs{
		payoutStubOrgs: payoutStubOrgs{stripeAccountID: "acct_test"},
		findByIDErr:    errors.New("transient db blip"),
	}
	stripe := &payoutStubStripe{}
	p := NewPayoutService(PayoutServiceDeps{Records: records, Organizations: orgs, Stripe: stripe})
	p.SetProposalStatusReader(&payoutStubProposalStatuses{statuses: map[uuid.UUID]string{rec.ProposalID: "completed"}})

	out, err := p.RequestPayout(context.Background(), uuid.New(), orgID)
	require.NoError(t, err, "consent FindByID failure must never break the payout (already succeeded)")
	require.NotNil(t, out)
	assert.Equal(t, "transferred", out.Status)
}

func TestPayoutService_RequestPayout_RecordConsent_AlreadyConsented_NoUpdate(t *testing.T) {
	orgID := uuid.New()
	rec := newSucceededPendingRecord()
	records := &payoutStubRecords{byOrganization: []*domain.PaymentRecord{rec}}
	orgs := &payoutCoverageOrgs{
		payoutStubOrgs: payoutStubOrgs{
			stripeAccountID: "acct_test",
			consentForOrg:   map[uuid.UUID]bool{orgID: true},
		},
	}
	stripe := &payoutStubStripe{}
	p := NewPayoutService(PayoutServiceDeps{Records: records, Organizations: orgs, Stripe: stripe})
	p.SetProposalStatusReader(&payoutStubProposalStatuses{statuses: map[uuid.UUID]string{rec.ProposalID: "completed"}})

	_, err := p.RequestPayout(context.Background(), uuid.New(), orgID)
	require.NoError(t, err)
	assert.Zero(t, orgs.updateCalls, "already-consented orgs must not be re-stamped")
}

func TestPayoutService_RequestPayout_RecordConsent_UpdateErr_Swallowed(t *testing.T) {
	orgID := uuid.New()
	rec := newSucceededPendingRecord()
	records := &payoutStubRecords{byOrganization: []*domain.PaymentRecord{rec}}
	orgs := &payoutCoverageOrgs{
		payoutStubOrgs: payoutStubOrgs{stripeAccountID: "acct_test"},
		updateErr:      errors.New("write conflict"),
	}
	stripe := &payoutStubStripe{}
	p := NewPayoutService(PayoutServiceDeps{Records: records, Organizations: orgs, Stripe: stripe})
	p.SetProposalStatusReader(&payoutStubProposalStatuses{statuses: map[uuid.UUID]string{rec.ProposalID: "completed"}})

	out, err := p.RequestPayout(context.Background(), uuid.New(), orgID)
	require.NoError(t, err, "consent stamp failure must never break the (already-successful) payout")
	require.NotNil(t, out)
	assert.Equal(t, "transferred", out.Status, "the funds were transferred — a stamp glitch is not a failure")
}

// RetryFailedTransfer drives maybeStampRetryConsent through its three
// branches: nil org (already covered upstream by ErrTransferNotRetriable),
// already-consented (no update), and update error (logged + swallowed).

func TestPayoutService_RetryFailedTransfer_AlreadyConsented_NoOrgUpdate(t *testing.T) {
	providerUserID := uuid.New()
	orgID := uuid.New()
	rec := newSucceededPendingRecord()
	rec.ProviderID = providerUserID
	rec.TransferStatus = domain.TransferFailed
	records := &payoutStubRecords{
		byID: map[uuid.UUID]*domain.PaymentRecord{rec.ID: rec},
	}
	orgs := &payoutCoverageOrgs{
		payoutStubOrgs: payoutStubOrgs{
			stripeAccountID: "acct_test",
			providerOrgID:   orgID,
			consentForOrg:   map[uuid.UUID]bool{orgID: true},
		},
	}
	stripe := &payoutStubStripe{}
	p := NewPayoutService(PayoutServiceDeps{Records: records, Organizations: orgs, Stripe: stripe})
	p.SetProposalStatusReader(&payoutStubProposalStatuses{statuses: map[uuid.UUID]string{rec.ProposalID: "completed"}})

	_, err := p.RetryFailedTransfer(context.Background(), uuid.New(), orgID, rec.ID)
	require.NoError(t, err)
	assert.Zero(t, orgs.updateCalls, "already-consented orgs are skipped without an Update")
}

func TestPayoutService_RetryFailedTransfer_StampConsentUpdateErr_Swallowed(t *testing.T) {
	providerUserID := uuid.New()
	orgID := uuid.New()
	rec := newSucceededPendingRecord()
	rec.ProviderID = providerUserID
	rec.TransferStatus = domain.TransferFailed
	records := &payoutStubRecords{
		byID: map[uuid.UUID]*domain.PaymentRecord{rec.ID: rec},
	}
	orgs := &payoutCoverageOrgs{
		payoutStubOrgs: payoutStubOrgs{
			stripeAccountID: "acct_test",
			providerOrgID:   orgID,
		},
		updateErr: errors.New("stamp save conflict"),
	}
	stripe := &payoutStubStripe{}
	p := NewPayoutService(PayoutServiceDeps{Records: records, Organizations: orgs, Stripe: stripe})
	p.SetProposalStatusReader(&payoutStubProposalStatuses{statuses: map[uuid.UUID]string{rec.ProposalID: "completed"}})

	out, err := p.RetryFailedTransfer(context.Background(), uuid.New(), orgID, rec.ID)
	require.NoError(t, err, "consent save failure must NOT break the retry")
	require.NotNil(t, out)
	assert.Equal(t, "transferred", out.Status)
}

// assertRetryAllowed: previously untested proposal-status error path.

func TestPayoutService_RetryFailedTransfer_StatusLookupErr_Wrapped(t *testing.T) {
	providerUserID := uuid.New()
	orgID := uuid.New()
	rec := newSucceededPendingRecord()
	rec.ProviderID = providerUserID
	rec.TransferStatus = domain.TransferFailed
	records := &payoutStubRecords{
		byID: map[uuid.UUID]*domain.PaymentRecord{rec.ID: rec},
	}
	orgs := &payoutCoverageOrgs{
		payoutStubOrgs: payoutStubOrgs{
			stripeAccountID: "acct_test",
			providerOrgID:   orgID,
		},
	}
	stripe := &payoutStubStripe{}
	p := NewPayoutService(PayoutServiceDeps{Records: records, Organizations: orgs, Stripe: stripe})
	p.SetProposalStatusReader(&payoutStubProposalStatuses{err: errors.New("proposal lookup down")})

	_, err := p.RetryFailedTransfer(context.Background(), uuid.New(), orgID, rec.ID)
	require.Error(t, err, "infra failure on the status gate must surface — never default to allowed")
	assert.Contains(t, err.Error(), "lookup proposal status")
}

// loadRetryRecord: previously untested "wrap non-sql error" branch.

type errRecords struct {
	*payoutStubRecords
	getByIDErr error
}

func (e *errRecords) GetByID(ctx context.Context, id uuid.UUID) (*domain.PaymentRecord, error) {
	if e.getByIDErr != nil {
		return nil, e.getByIDErr
	}
	return e.payoutStubRecords.GetByID(ctx, id)
}

// GetByIDForOrg routes through the same overridable error
// channel — the migrated loadRetryRecord path now reads through
// GetByIDForOrg, so the infra-error test case has to surface the
// override here.
func (e *errRecords) GetByIDForOrg(ctx context.Context, id, _ uuid.UUID) (*domain.PaymentRecord, error) {
	if e.getByIDErr != nil {
		return nil, e.getByIDErr
	}
	return e.payoutStubRecords.GetByID(ctx, id)
}

func TestPayoutService_RetryFailedTransfer_GetByIDInfraErr_Wrapped(t *testing.T) {
	records := &errRecords{
		payoutStubRecords: &payoutStubRecords{byID: map[uuid.UUID]*domain.PaymentRecord{}},
		getByIDErr:        errors.New("connection reset"),
	}
	p := NewPayoutService(PayoutServiceDeps{Records: records, Organizations: &payoutStubOrgs{}, Stripe: &payoutStubStripe{}})

	_, err := p.RetryFailedTransfer(context.Background(), uuid.New(), uuid.New(), uuid.New())
	require.Error(t, err)
	// Generic infra failures must NOT collapse to ErrPaymentRecordNotFound —
	// otherwise the handler returns 404 instead of 500 and the user gets
	// a misleading message ("not found" when the DB is actually down).
	assert.NotErrorIs(t, err, domain.ErrPaymentRecordNotFound)
	assert.Contains(t, err.Error(), "find payment record")
}

// ---------------------------------------------------------------------------
// Benchmarks — proves the file split does not regress hot-path
// performance. RequestPayout + TransferMilestone are the two hottest
// payment-side methods; if either one slows down measurably, the cause
// is here. These run quickly in -short mode and add <0.5s to CI.
//
// We re-route slog to a discard handler for the duration of the bench
// so the per-op log lines don't dwarf the actual measured work.
// ---------------------------------------------------------------------------

// silenceSlog swaps the default slog handler for an io.Discard one for
// the lifetime of the test, restoring the original on cleanup. Bench
// noise on the order of "INFO payout: bank transfer initiated" per op
// is irrelevant for correctness but it makes the bench output unreadable
// in CI logs.
func silenceSlog(tb testing.TB) {
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	tb.Cleanup(func() { slog.SetDefault(prev) })
}

func BenchmarkPayoutService_TransferMilestone_HappyPath(b *testing.B) {
	silenceSlog(b)
	for i := 0; i < b.N; i++ {
		rec := newSucceededPendingRecord()
		records := &payoutStubRecords{byMilestone: map[uuid.UUID]*domain.PaymentRecord{rec.MilestoneID: rec}}
		orgs := &payoutStubOrgs{stripeAccountID: "acct_b"}
		stripe := &payoutStubStripe{}
		p := NewPayoutService(PayoutServiceDeps{Records: records, Organizations: orgs, Stripe: stripe})

		_ = p.TransferMilestone(context.Background(), rec.MilestoneID)
	}
}

func BenchmarkPayoutService_RequestPayout_HappyPath(b *testing.B) {
	silenceSlog(b)
	for i := 0; i < b.N; i++ {
		orgID := uuid.New()
		rec := newSucceededPendingRecord()
		records := &payoutStubRecords{byOrganization: []*domain.PaymentRecord{rec}}
		orgs := &payoutStubOrgs{stripeAccountID: "acct_b"}
		stripe := &payoutStubStripe{}
		p := NewPayoutService(PayoutServiceDeps{Records: records, Organizations: orgs, Stripe: stripe})
		p.SetProposalStatusReader(&payoutStubProposalStatuses{statuses: map[uuid.UUID]string{rec.ProposalID: "completed"}})

		_, _ = p.RequestPayout(context.Background(), uuid.New(), orgID)
	}
}
