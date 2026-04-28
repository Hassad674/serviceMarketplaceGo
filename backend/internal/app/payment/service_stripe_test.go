package payment

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"marketplace-backend/internal/domain/organization"
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
	stripeAccountID    string
	hasConsent         bool
	updateCalls        int
	consentStampedHere bool
}

func (f *fakeOrgs) GetStripeAccountByUserID(_ context.Context, _ uuid.UUID) (string, string, error) {
	return f.stripeAccountID, "FR", nil
}
func (f *fakeOrgs) GetStripeAccount(_ context.Context, _ uuid.UUID) (string, string, error) {
	return f.stripeAccountID, "FR", nil
}
func (f *fakeOrgs) FindByID(_ context.Context, id uuid.UUID) (*organization.Organization, error) {
	org := &organization.Organization{ID: id}
	if f.hasConsent {
		now := time.Now()
		org.AutoPayoutEnabledAt = &now
	}
	return org, nil
}
func (f *fakeOrgs) FindByUserID(_ context.Context, userID uuid.UUID) (*organization.Organization, error) {
	org := &organization.Organization{ID: userID}
	if f.hasConsent {
		now := time.Now()
		org.AutoPayoutEnabledAt = &now
	}
	return org, nil
}
func (f *fakeOrgs) Update(_ context.Context, org *organization.Organization) error {
	f.updateCalls++
	if org.AutoPayoutEnabledAt != nil {
		f.consentStampedHere = true
	}
	return nil
}

// fakeStripe stubs StripeService; only CreateTransfer and CreateRefund are
// touched by the partial/refund flows, but the tests below target
// TransferPartialToProvider so only CreateTransfer matters.
type fakeStripe struct {
	service.StripeService
	transferCalls []service.CreateTransferInput
	payoutCalls   []service.CreatePayoutInput
	failPayout    bool
}

func (f *fakeStripe) CreateTransfer(_ context.Context, in service.CreateTransferInput) (string, error) {
	f.transferCalls = append(f.transferCalls, in)
	return "tr_test_" + in.IdempotencyKey, nil
}

func (f *fakeStripe) CreatePayout(_ context.Context, in service.CreatePayoutInput) (string, error) {
	f.payoutCalls = append(f.payoutCalls, in)
	if f.failPayout {
		return "", errors.New("stripe payout boom")
	}
	return "po_test_" + in.IdempotencyKey, nil
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

// Connected accounts are now created on a manual payout schedule (see
// adapter/stripe/account.go), so RequestPayout must explicitly fire a
// Stripe payout from the connected account → bank after the
// platform→connected transfer. Without this call, funds would sit on
// the connected account's Stripe balance and never reach the user's
// bank — exactly the surprise-no-payout bug the user reported on /api.
func TestRequestPayout_FiresExplicitStripePayoutAfterTransfer(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()

	rec := baseRecord()
	rec.ProviderPayout = 1500
	rec.Currency = "eur"

	records := &listingRecords{records: []*domain.PaymentRecord{rec}}
	orgs := &fakeOrgs{stripeAccountID: "acct_test_pay"}
	stripe := &fakeStripe{}

	svc := NewService(ServiceDeps{
		Records:       records,
		Organizations: orgs,
		Stripe:        stripe,
	})

	res, err := svc.RequestPayout(context.Background(), userID, orgID)
	assert.NoError(t, err)
	assert.Equal(t, "transferred", res.Status)

	// The Stripe payout must run on the connected account, in the
	// record's currency, for exactly the transferred amount, with an
	// idempotency key so a retry can't double-debit the balance.
	assert.Len(t, stripe.payoutCalls, 1, "payout must be triggered exactly once")
	po := stripe.payoutCalls[0]
	assert.Equal(t, "acct_test_pay", po.ConnectedAccountID)
	assert.Equal(t, int64(1500), po.Amount)
	assert.Equal(t, "eur", po.Currency)
	assert.NotEmpty(t, po.IdempotencyKey, "idempotency key required for safe retries")
}

// When CreateTransfer never succeeded, there is nothing to bank-pay
// out, so CreatePayout MUST NOT fire — otherwise we'd debit a balance
// that never received funds and Stripe would error or worse, succeed
// from an unrelated balance.
func TestRequestPayout_SkipsBankPayoutWhenNoTransferSucceeded(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()

	records := &listingRecords{records: nil} // no records → nothing transferred
	orgs := &fakeOrgs{stripeAccountID: "acct_test_skip"}
	stripe := &fakeStripe{}

	svc := NewService(ServiceDeps{
		Records:       records,
		Organizations: orgs,
		Stripe:        stripe,
	})

	res, err := svc.RequestPayout(context.Background(), userID, orgID)
	assert.NoError(t, err)
	assert.Equal(t, "nothing_to_transfer", res.Status)
	assert.Empty(t, stripe.payoutCalls, "no transfer = no bank payout")
}

// If the bank-leg payout itself fails, the transfers already
// succeeded — funds sit safely on the connected account. The handler
// should report a transferred-pending-bank status (informative, not
// a hard 500) so the user knows the platform side worked and the bank
// transfer is the part that needs follow-up.
func TestRequestPayout_PayoutFailureSurfacesAsTransferredPendingBank(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()

	rec := baseRecord()
	rec.ProviderPayout = 800
	rec.Currency = "eur"

	records := &listingRecords{records: []*domain.PaymentRecord{rec}}
	orgs := &fakeOrgs{stripeAccountID: "acct_test_fail"}
	stripe := &fakeStripe{failPayout: true}

	svc := NewService(ServiceDeps{
		Records:       records,
		Organizations: orgs,
		Stripe:        stripe,
	})

	res, err := svc.RequestPayout(context.Background(), userID, orgID)
	assert.NoError(t, err, "bank-leg failure must not roll back the transfers")
	assert.Equal(t, "transferred_pending_bank", res.Status)
	assert.Len(t, stripe.payoutCalls, 1, "payout was attempted")
}

// milestoneRecords lets us drive TransferMilestone + TransferToProvider
// iteration paths independently. It looks up records by milestone_id
// (primary path) and by proposal_id (iterator path). Update writes are
// collected in order so the test can assert which records were touched.
type milestoneRecords struct {
	repository.PaymentRecordRepository
	byMilestone map[uuid.UUID]*domain.PaymentRecord
	byProposal  map[uuid.UUID][]*domain.PaymentRecord
	updated     []*domain.PaymentRecord
}

func (m *milestoneRecords) GetByMilestoneID(_ context.Context, id uuid.UUID) (*domain.PaymentRecord, error) {
	r, ok := m.byMilestone[id]
	if !ok {
		return nil, domain.ErrPaymentRecordNotFound
	}
	cp := *r
	return &cp, nil
}

func (m *milestoneRecords) ListByProposalID(_ context.Context, id uuid.UUID) ([]*domain.PaymentRecord, error) {
	out := make([]*domain.PaymentRecord, 0, len(m.byProposal[id]))
	for _, r := range m.byProposal[id] {
		cp := *r
		out = append(out, &cp)
	}
	return out, nil
}

func (m *milestoneRecords) Update(_ context.Context, r *domain.PaymentRecord) error {
	m.updated = append(m.updated, r)
	// Mirror the update into the source maps so a follow-up call sees
	// the new state — models the real DB behaviour.
	if existing, ok := m.byMilestone[r.MilestoneID]; ok {
		*existing = *r
	}
	return nil
}

// Bug A: TransferToProvider(proposalID) used GetByProposalID which returns
// the most recent record by created_at. On a multi-milestone proposal that
// means the wrong record was always released. TransferMilestone targets the
// correct record by milestone_id; this test proves the fix.
func TestTransferMilestone_ReleasesSpecificRecord(t *testing.T) {
	providerID := uuid.New()
	proposalID := uuid.New()

	// Jalon 1: succeeded + pending (the one we want to release).
	rec1 := baseRecord()
	rec1.ProposalID = proposalID
	rec1.MilestoneID = uuid.New()
	rec1.ProviderID = providerID
	rec1.ProposalAmount = 1000
	rec1.ProviderPayout = 950

	// Jalon 2: also succeeded + pending but CREATED LATER — the legacy
	// GetByProposalID would pick this one and leave jalon 1 stuck in
	// escrow. The explicit milestone-scoped call must ignore it.
	rec2 := baseRecord()
	rec2.ProposalID = proposalID
	rec2.MilestoneID = uuid.New()
	rec2.ProviderID = providerID
	rec2.ProposalAmount = 2000
	rec2.ProviderPayout = 1900

	records := &milestoneRecords{
		byMilestone: map[uuid.UUID]*domain.PaymentRecord{
			rec1.MilestoneID: rec1,
			rec2.MilestoneID: rec2,
		},
		byProposal: map[uuid.UUID][]*domain.PaymentRecord{
			proposalID: {rec1, rec2},
		},
	}
	orgs := &fakeOrgs{stripeAccountID: "acct_test_123"}
	stripe := &fakeStripe{}

	svc := NewService(ServiceDeps{
		Records:       records,
		Organizations: orgs,
		Stripe:        stripe,
	})

	err := svc.TransferMilestone(context.Background(), rec1.MilestoneID)
	assert.NoError(t, err)

	// Exactly ONE Stripe transfer for jalon 1's amount — never jalon 2.
	assert.Len(t, stripe.transferCalls, 1, "exactly one transfer for the targeted milestone")
	assert.Equal(t, rec1.ProviderPayout, stripe.transferCalls[0].Amount,
		"amount must match the targeted milestone, not the most recent record")
	assert.Equal(t, proposalID.String(), stripe.transferCalls[0].TransferGroup)

	// Record 1 got persisted, record 2 must be untouched so the client
	// can still release it via its own milestone_id call.
	assert.Len(t, records.updated, 1, "only the targeted record is updated")
	assert.Equal(t, rec1.MilestoneID, records.updated[0].MilestoneID,
		"the persisted record must be jalon 1, proving we didn't fall back to the newest record")
	assert.Equal(t, domain.TransferCompleted, records.updated[0].TransferStatus)
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

// ---------------------------------------------------------------------------
// RetryFailedTransfer — recovery path for stuck payment records
// ---------------------------------------------------------------------------

// retryRecords is a focused stub for RetryFailedTransfer tests. It looks
// records up by id (the actual call path) and records the post-call state
// so tests can assert what was persisted (TransferPending vs TransferFailed
// vs TransferCompleted).
type retryRecords struct {
	repository.PaymentRecordRepository
	record   *domain.PaymentRecord
	notFound bool
	updates  []*domain.PaymentRecord
}

func (r *retryRecords) GetByID(_ context.Context, _ uuid.UUID) (*domain.PaymentRecord, error) {
	if r.notFound {
		return nil, domain.ErrPaymentRecordNotFound
	}
	cp := *r.record
	return &cp, nil
}

func (r *retryRecords) Update(_ context.Context, rec *domain.PaymentRecord) error {
	cp := *rec
	r.updates = append(r.updates, &cp)
	*r.record = *rec
	return nil
}

// retryOrgs adds FindByUserID on top of the existing fakeOrgs so the
// retry handler's auth check (provider's org == caller's org) can be
// exercised without dragging in the real postgres adapter.
type retryOrgs struct {
	repository.OrganizationRepository
	stripeAccountID string
	providerOrgID   uuid.UUID
}

func (o *retryOrgs) FindByUserID(_ context.Context, _ uuid.UUID) (*organization.Organization, error) {
	return &organization.Organization{ID: o.providerOrgID}, nil
}

func (o *retryOrgs) FindByID(_ context.Context, id uuid.UUID) (*organization.Organization, error) {
	return &organization.Organization{ID: id}, nil
}

func (o *retryOrgs) Update(_ context.Context, _ *organization.Organization) error {
	return nil
}

func (o *retryOrgs) GetStripeAccountByUserID(_ context.Context, _ uuid.UUID) (string, string, error) {
	return o.stripeAccountID, "FR", nil
}

func (o *retryOrgs) GetStripeAccount(_ context.Context, _ uuid.UUID) (string, string, error) {
	return o.stripeAccountID, "FR", nil
}

// retryStripe exposes both CreateTransfer and GetAccount so the KYC
// readiness probe (payouts_enabled) and the actual transfer can be
// independently controlled per test case.
type retryStripe struct {
	service.StripeService
	transferCalls  []service.CreateTransferInput
	transferErr    error
	payoutsEnabled bool
	getAccountErr  error
}

func (s *retryStripe) CreateTransfer(_ context.Context, in service.CreateTransferInput) (string, error) {
	s.transferCalls = append(s.transferCalls, in)
	if s.transferErr != nil {
		return "", s.transferErr
	}
	return "tr_retry_" + in.IdempotencyKey, nil
}

func (s *retryStripe) GetAccount(_ context.Context, _ string) (*service.StripeAccountInfo, error) {
	if s.getAccountErr != nil {
		return nil, s.getAccountErr
	}
	return &service.StripeAccountInfo{
		ChargesEnabled: s.payoutsEnabled,
		PayoutsEnabled: s.payoutsEnabled,
	}, nil
}

func failedRetryRecord(orgUserID uuid.UUID) *domain.PaymentRecord {
	r := baseRecord()
	r.ProviderID = orgUserID
	r.Status = domain.RecordStatusSucceeded
	r.TransferStatus = domain.TransferFailed
	r.ProviderPayout = 21_188_00 // the real-world 21 188 € case
	return r
}

// Happy path: record is failed, mission is completed, KYC is now done.
// The retry must reset to Pending, hit Stripe, and persist Completed.
func TestRetryFailedTransfer_HappyPath_TransfersAndMarksCompleted(t *testing.T) {
	providerOrgID := uuid.New()
	rec := failedRetryRecord(uuid.New())

	records := &retryRecords{record: rec}
	orgs := &retryOrgs{stripeAccountID: "acct_test", providerOrgID: providerOrgID}
	stripe := &retryStripe{payoutsEnabled: true}

	svc := NewService(ServiceDeps{Records: records, Organizations: orgs, Stripe: stripe})
	svc.SetProposalStatusReader(&fakeProposalStatuses{statuses: map[uuid.UUID]string{
		rec.ProposalID: "completed",
	}})

	result, err := svc.RetryFailedTransfer(context.Background(), uuid.New(), providerOrgID, rec.ID)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "transferred", result.Status)
	assert.Len(t, stripe.transferCalls, 1, "exactly one Stripe transfer attempted")
	assert.Equal(t, rec.ProviderPayout, stripe.transferCalls[0].Amount)
	// Two updates: reset to Pending, then mark Completed on success.
	assert.GreaterOrEqual(t, len(records.updates), 2)
	assert.Equal(t, domain.TransferCompleted, records.updates[len(records.updates)-1].TransferStatus)
}

// 412 path: the provider has an account on file BUT payouts_enabled=false
// (KYC not yet validated). The service must short-circuit with
// ErrProviderPayoutsDisabled BEFORE burning the Stripe transfer
// idempotency key — that's the whole point of the pre-check.
func TestRetryFailedTransfer_KYCPending_ReturnsProviderPayoutsDisabled(t *testing.T) {
	providerOrgID := uuid.New()
	rec := failedRetryRecord(uuid.New())

	records := &retryRecords{record: rec}
	orgs := &retryOrgs{stripeAccountID: "acct_test", providerOrgID: providerOrgID}
	stripe := &retryStripe{payoutsEnabled: false} // KYC pending

	svc := NewService(ServiceDeps{Records: records, Organizations: orgs, Stripe: stripe})
	svc.SetProposalStatusReader(&fakeProposalStatuses{statuses: map[uuid.UUID]string{
		rec.ProposalID: "completed",
	}})

	_, err := svc.RetryFailedTransfer(context.Background(), uuid.New(), providerOrgID, rec.ID)
	assert.ErrorIs(t, err, domain.ErrProviderPayoutsDisabled)
	assert.Empty(t, stripe.transferCalls, "no Stripe transfer must be attempted when payouts are disabled")
	// Nothing was persisted because the KYC gate fires before the
	// "reset to Pending" write.
	assert.Empty(t, records.updates)
}

// 403 path: no Stripe account on file at all. Distinct from the 412
// "payouts disabled" case so the UI can route the user to the right
// onboarding step.
func TestRetryFailedTransfer_NoStripeAccount_ReturnsAccountNotFound(t *testing.T) {
	providerOrgID := uuid.New()
	rec := failedRetryRecord(uuid.New())

	records := &retryRecords{record: rec}
	orgs := &retryOrgs{stripeAccountID: "", providerOrgID: providerOrgID}
	stripe := &retryStripe{payoutsEnabled: true}

	svc := NewService(ServiceDeps{Records: records, Organizations: orgs, Stripe: stripe})
	svc.SetProposalStatusReader(&fakeProposalStatuses{statuses: map[uuid.UUID]string{
		rec.ProposalID: "completed",
	}})

	_, err := svc.RetryFailedTransfer(context.Background(), uuid.New(), providerOrgID, rec.ID)
	assert.ErrorIs(t, err, domain.ErrStripeAccountNotFound)
	assert.Empty(t, stripe.transferCalls)
}

// 403 path: caller's org is NOT the provider's org. Must reject with
// ErrTransferNotRetriable (handler maps that to 409). This is the
// cross-tenant safeguard — a malicious user with a record id from a
// different org must NEVER succeed at retrying that org's transfer.
func TestRetryFailedTransfer_NotProviderOrg_Rejected(t *testing.T) {
	providerOrgID := uuid.New()
	otherOrgID := uuid.New() // attacker's org
	rec := failedRetryRecord(uuid.New())

	records := &retryRecords{record: rec}
	orgs := &retryOrgs{stripeAccountID: "acct_test", providerOrgID: providerOrgID}
	stripe := &retryStripe{payoutsEnabled: true}

	svc := NewService(ServiceDeps{Records: records, Organizations: orgs, Stripe: stripe})

	_, err := svc.RetryFailedTransfer(context.Background(), uuid.New(), otherOrgID, rec.ID)
	assert.ErrorIs(t, err, domain.ErrTransferNotRetriable)
	assert.Empty(t, stripe.transferCalls)
}

// 404 path: the payment record id doesn't exist. The service must
// surface the typed sentinel so the handler can return 404 instead of
// swallowing into a generic 500.
func TestRetryFailedTransfer_RecordNotFound_ReturnsTypedError(t *testing.T) {
	records := &retryRecords{notFound: true}
	orgs := &retryOrgs{stripeAccountID: "acct_test", providerOrgID: uuid.New()}
	stripe := &retryStripe{payoutsEnabled: true}

	svc := NewService(ServiceDeps{Records: records, Organizations: orgs, Stripe: stripe})

	_, err := svc.RetryFailedTransfer(context.Background(), uuid.New(), uuid.New(), uuid.New())
	assert.ErrorIs(t, err, domain.ErrPaymentRecordNotFound)
}

// 409 path: the record is succeeded+pending (already transferred or
// never failed). Calling Retry on a non-failed record is a no-op and
// must surface ErrTransferNotRetriable so the handler returns 409.
func TestRetryFailedTransfer_RecordNotFailed_Rejected(t *testing.T) {
	providerOrgID := uuid.New()
	rec := failedRetryRecord(uuid.New())
	rec.TransferStatus = domain.TransferCompleted // already done

	records := &retryRecords{record: rec}
	orgs := &retryOrgs{stripeAccountID: "acct_test", providerOrgID: providerOrgID}
	stripe := &retryStripe{payoutsEnabled: true}

	svc := NewService(ServiceDeps{Records: records, Organizations: orgs, Stripe: stripe})

	_, err := svc.RetryFailedTransfer(context.Background(), uuid.New(), providerOrgID, rec.ID)
	assert.ErrorIs(t, err, domain.ErrTransferNotRetriable)
	assert.Empty(t, stripe.transferCalls)
}

// 502 path: the KYC gate passes but Stripe rejects the transfer (e.g.
// rate-limited, network blip). The service must re-mark the record as
// TransferFailed so a future retry can pick it up — otherwise it would
// be stuck in TransferPending forever.
func TestRetryFailedTransfer_StripeError_MarksFailedAndPropagates(t *testing.T) {
	providerOrgID := uuid.New()
	rec := failedRetryRecord(uuid.New())

	records := &retryRecords{record: rec}
	orgs := &retryOrgs{stripeAccountID: "acct_test", providerOrgID: providerOrgID}
	stripe := &retryStripe{
		payoutsEnabled: true,
		transferErr:    errors.New("stripe: rate limited"),
	}

	svc := NewService(ServiceDeps{Records: records, Organizations: orgs, Stripe: stripe})
	svc.SetProposalStatusReader(&fakeProposalStatuses{statuses: map[uuid.UUID]string{
		rec.ProposalID: "completed",
	}})

	_, err := svc.RetryFailedTransfer(context.Background(), uuid.New(), providerOrgID, rec.ID)
	assert.Error(t, err)
	assert.NotErrorIs(t, err, domain.ErrTransferNotRetriable)
	assert.NotErrorIs(t, err, domain.ErrProviderPayoutsDisabled)
	// Last persisted state must be Failed so the row keeps showing
	// "Échec — Réessayer" in the wallet UI.
	assert.NotEmpty(t, records.updates)
	assert.Equal(t, domain.TransferFailed, records.updates[len(records.updates)-1].TransferStatus)
}

// Bug: when GetStripeAccountByUserID returns an empty account id (e.g.
// the lookup path drifted from the JWT/wallet path and no longer points
// to a KYC-completed org), TransferMilestone must surface the typed
// ErrStripeAccountNotFound — not panic, not return a generic error —
// so the outbox worker logs the diagnostic and the wallet shows
// "Échec — Réessayer" instead of silently dropping the transfer.
//
// Pinning this here also guards against accidental regressions in the
// repository's resolution chain: any future refactor that breaks the
// "empty account id ⇒ typed sentinel" invariant must update this test.
func TestTransferMilestone_NoStripeAccount_ReturnsTypedSentinel(t *testing.T) {
	rec := baseRecord()
	rec.MilestoneID = uuid.New()

	records := &milestoneRecords{
		byMilestone: map[uuid.UUID]*domain.PaymentRecord{
			rec.MilestoneID: rec,
		},
	}
	orgs := &fakeOrgs{stripeAccountID: ""} // resolution returns empty
	stripe := &fakeStripe{}

	svc := NewService(ServiceDeps{
		Records:       records,
		Organizations: orgs,
		Stripe:        stripe,
	})

	err := svc.TransferMilestone(context.Background(), rec.MilestoneID)
	assert.ErrorIs(t, err, domain.ErrStripeAccountNotFound,
		"empty stripe account id must surface as ErrStripeAccountNotFound")
	assert.Empty(t, stripe.transferCalls,
		"no Stripe API call must be issued when the destination is unknown")
	assert.Empty(t, records.updated,
		"the record must not be mutated when the resolution fails")
}
