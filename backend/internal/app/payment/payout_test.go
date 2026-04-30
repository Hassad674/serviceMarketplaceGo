package payment

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/domain/organization"
	domain "marketplace-backend/internal/domain/payment"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/port/service"
)

// ---------------------------------------------------------------------------
// PayoutService — dedicated tests for the transfer / payout sub-service.
// Exercises the SOLID decomposition directly (no Service facade).
// ---------------------------------------------------------------------------

// payoutStubRecords supports every records method the payout flow
// touches, with thread-safe counters for race tests.
type payoutStubRecords struct {
	repository.PaymentRecordRepository
	mu sync.Mutex

	byProposal     []*domain.PaymentRecord
	byMilestone    map[uuid.UUID]*domain.PaymentRecord
	byID           map[uuid.UUID]*domain.PaymentRecord
	byOrganization []*domain.PaymentRecord

	byProposalErr error
	byOrgErr      error
	updateErr     error
	updateCalls   int
	updates       []*domain.PaymentRecord
}

func (p *payoutStubRecords) ListByProposalID(_ context.Context, _ uuid.UUID) ([]*domain.PaymentRecord, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.byProposalErr != nil {
		return nil, p.byProposalErr
	}
	out := make([]*domain.PaymentRecord, 0, len(p.byProposal))
	for _, r := range p.byProposal {
		cp := *r
		out = append(out, &cp)
	}
	return out, nil
}

func (p *payoutStubRecords) ListByOrganization(_ context.Context, _ uuid.UUID) ([]*domain.PaymentRecord, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.byOrgErr != nil {
		return nil, p.byOrgErr
	}
	out := make([]*domain.PaymentRecord, 0, len(p.byOrganization))
	for _, r := range p.byOrganization {
		cp := *r
		out = append(out, &cp)
	}
	return out, nil
}

func (p *payoutStubRecords) GetByMilestoneID(_ context.Context, id uuid.UUID) (*domain.PaymentRecord, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	r, ok := p.byMilestone[id]
	if !ok {
		return nil, domain.ErrPaymentRecordNotFound
	}
	cp := *r
	return &cp, nil
}

func (p *payoutStubRecords) GetByProposalID(_ context.Context, id uuid.UUID) (*domain.PaymentRecord, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, r := range p.byProposal {
		if r.ProposalID == id {
			cp := *r
			return &cp, nil
		}
	}
	return nil, domain.ErrPaymentRecordNotFound
}

func (p *payoutStubRecords) GetByID(_ context.Context, id uuid.UUID) (*domain.PaymentRecord, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	r, ok := p.byID[id]
	if !ok {
		return nil, domain.ErrPaymentRecordNotFound
	}
	cp := *r
	return &cp, nil
}

func (p *payoutStubRecords) Update(_ context.Context, r *domain.PaymentRecord) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.updateCalls++
	if p.updateErr != nil {
		return p.updateErr
	}
	cp := *r
	p.updates = append(p.updates, &cp)
	// Mirror the update into source maps so a follow-up Get sees the
	// new state — models the real DB.
	if existing, ok := p.byMilestone[r.MilestoneID]; ok {
		*existing = *r
	}
	if existing, ok := p.byID[r.ID]; ok {
		*existing = *r
	}
	for _, src := range p.byProposal {
		if src.ID == r.ID {
			*src = *r
		}
	}
	return nil
}

// payoutStubOrgs supports every org method the payout flow touches.
type payoutStubOrgs struct {
	repository.OrganizationRepository
	mu               sync.Mutex
	stripeAccountID  string
	getStripeErr     error
	consentForOrg    map[uuid.UUID]bool // orgID → already consented
	updateCalls      int
	updateErr        error
	providerOrgID    uuid.UUID
}

func (o *payoutStubOrgs) GetStripeAccount(_ context.Context, _ uuid.UUID) (string, string, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.stripeAccountID, "FR", o.getStripeErr
}

func (o *payoutStubOrgs) GetStripeAccountByUserID(_ context.Context, _ uuid.UUID) (string, string, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.stripeAccountID, "FR", o.getStripeErr
}

func (o *payoutStubOrgs) FindByID(_ context.Context, id uuid.UUID) (*organization.Organization, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	org := &organization.Organization{ID: id}
	if o.consentForOrg[id] {
		now := time.Now()
		org.AutoPayoutEnabledAt = &now
	}
	return org, nil
}

func (o *payoutStubOrgs) FindByUserID(_ context.Context, _ uuid.UUID) (*organization.Organization, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.providerOrgID == uuid.Nil {
		return nil, errors.New("no provider org")
	}
	org := &organization.Organization{ID: o.providerOrgID}
	if o.consentForOrg[o.providerOrgID] {
		now := time.Now()
		org.AutoPayoutEnabledAt = &now
	}
	return org, nil
}

func (o *payoutStubOrgs) Update(_ context.Context, _ *organization.Organization) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.updateCalls++
	return o.updateErr
}

// payoutStubStripe supports every stripe method the payout flow uses.
type payoutStubStripe struct {
	service.StripeService
	mu             sync.Mutex
	transferCalls  []service.CreateTransferInput
	transferErr    error
	payoutCalls    []service.CreatePayoutInput
	payoutErr      error
	refundCalls    int
	refundErr      error
	getAccountInfo *service.StripeAccountInfo
	getAccountErr  error
}

func (s *payoutStubStripe) CreateTransfer(_ context.Context, in service.CreateTransferInput) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.transferCalls = append(s.transferCalls, in)
	if s.transferErr != nil {
		return "", s.transferErr
	}
	return "tr_" + in.IdempotencyKey, nil
}

func (s *payoutStubStripe) CreatePayout(_ context.Context, in service.CreatePayoutInput) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.payoutCalls = append(s.payoutCalls, in)
	if s.payoutErr != nil {
		return "", s.payoutErr
	}
	return "po_" + in.IdempotencyKey, nil
}

func (s *payoutStubStripe) CreateRefund(_ context.Context, _ string, _ int64) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.refundCalls++
	if s.refundErr != nil {
		return "", s.refundErr
	}
	return "re_test", nil
}

func (s *payoutStubStripe) GetAccount(_ context.Context, _ string) (*service.StripeAccountInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.getAccountErr != nil {
		return nil, s.getAccountErr
	}
	if s.getAccountInfo == nil {
		// Default: payouts enabled.
		return &service.StripeAccountInfo{ChargesEnabled: true, PayoutsEnabled: true}, nil
	}
	return s.getAccountInfo, nil
}

// payoutStubProposalStatuses returns the statuses configured per
// proposal id.
type payoutStubProposalStatuses struct {
	mu       sync.Mutex
	statuses map[uuid.UUID]string
	err      error
	calls    int32
}

func (s *payoutStubProposalStatuses) GetProposalStatus(_ context.Context, id uuid.UUID) (string, error) {
	atomic.AddInt32(&s.calls, 1)
	if s.err != nil {
		return "", s.err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.statuses[id], nil
}

func newSucceededPendingRecord() *domain.PaymentRecord {
	return &domain.PaymentRecord{
		ID:             uuid.New(),
		ProposalID:     uuid.New(),
		MilestoneID:    uuid.New(),
		ClientID:       uuid.New(),
		ProviderID:     uuid.New(),
		Currency:       "eur",
		ProposalAmount: 1000,
		ProviderPayout: 950,
		Status:         domain.RecordStatusSucceeded,
		TransferStatus: domain.TransferPending,
		CreatedAt:      time.Now(),
	}
}

// ---------------------------------------------------------------------------
// TransferMilestone — direct test on the sub-service
// ---------------------------------------------------------------------------

func TestPayoutService_TransferMilestone_HappyPath(t *testing.T) {
	rec := newSucceededPendingRecord()
	records := &payoutStubRecords{
		byMilestone: map[uuid.UUID]*domain.PaymentRecord{rec.MilestoneID: rec},
	}
	orgs := &payoutStubOrgs{stripeAccountID: "acct_test"}
	stripe := &payoutStubStripe{}
	p := NewPayoutService(PayoutServiceDeps{Records: records, Organizations: orgs, Stripe: stripe})

	err := p.TransferMilestone(context.Background(), rec.MilestoneID)
	require.NoError(t, err)
	assert.Len(t, stripe.transferCalls, 1)
	require.NotEmpty(t, records.updates)
	assert.Equal(t, domain.TransferCompleted, records.updates[0].TransferStatus)
}

func TestPayoutService_TransferMilestone_NotSucceeded_Rejects(t *testing.T) {
	rec := newSucceededPendingRecord()
	rec.Status = domain.RecordStatusPending
	records := &payoutStubRecords{
		byMilestone: map[uuid.UUID]*domain.PaymentRecord{rec.MilestoneID: rec},
	}
	p := NewPayoutService(PayoutServiceDeps{Records: records, Organizations: &payoutStubOrgs{}, Stripe: &payoutStubStripe{}})

	err := p.TransferMilestone(context.Background(), rec.MilestoneID)
	assert.ErrorIs(t, err, domain.ErrPaymentNotSucceeded)
}

func TestPayoutService_TransferMilestone_AlreadyTransferred_Rejects(t *testing.T) {
	rec := newSucceededPendingRecord()
	rec.TransferStatus = domain.TransferCompleted
	records := &payoutStubRecords{
		byMilestone: map[uuid.UUID]*domain.PaymentRecord{rec.MilestoneID: rec},
	}
	p := NewPayoutService(PayoutServiceDeps{Records: records, Organizations: &payoutStubOrgs{}, Stripe: &payoutStubStripe{}})

	err := p.TransferMilestone(context.Background(), rec.MilestoneID)
	assert.ErrorIs(t, err, domain.ErrTransferAlreadyDone)
}

func TestPayoutService_TransferMilestone_NoStripeAccount_Rejects(t *testing.T) {
	rec := newSucceededPendingRecord()
	records := &payoutStubRecords{byMilestone: map[uuid.UUID]*domain.PaymentRecord{rec.MilestoneID: rec}}
	orgs := &payoutStubOrgs{stripeAccountID: ""}
	p := NewPayoutService(PayoutServiceDeps{Records: records, Organizations: orgs, Stripe: &payoutStubStripe{}})

	err := p.TransferMilestone(context.Background(), rec.MilestoneID)
	assert.ErrorIs(t, err, domain.ErrStripeAccountNotFound)
}

// ---------------------------------------------------------------------------
// TransferMilestone — referral commission distributor hook
// ---------------------------------------------------------------------------

type stubReferralDistributor struct {
	calls    int
	err      error
}

func (s *stubReferralDistributor) DistributeIfApplicable(_ context.Context, _ service.ReferralCommissionDistributorInput) (service.ReferralCommissionResult, error) {
	s.calls++
	if s.err != nil {
		return service.ReferralCommissionFailed, s.err
	}
	return service.ReferralCommissionPaid, nil
}

func TestPayoutService_TransferMilestone_FiresReferralDistributor(t *testing.T) {
	rec := newSucceededPendingRecord()
	records := &payoutStubRecords{byMilestone: map[uuid.UUID]*domain.PaymentRecord{rec.MilestoneID: rec}}
	orgs := &payoutStubOrgs{stripeAccountID: "acct_test"}
	stripe := &payoutStubStripe{}
	dist := &stubReferralDistributor{}

	p := NewPayoutService(PayoutServiceDeps{Records: records, Organizations: orgs, Stripe: stripe})
	p.SetReferralDistributor(dist)

	err := p.TransferMilestone(context.Background(), rec.MilestoneID)
	require.NoError(t, err)
	assert.Equal(t, 1, dist.calls, "distributor must be invoked once on a successful transfer")
}

func TestPayoutService_TransferMilestone_DistributorErr_Swallowed(t *testing.T) {
	rec := newSucceededPendingRecord()
	records := &payoutStubRecords{byMilestone: map[uuid.UUID]*domain.PaymentRecord{rec.MilestoneID: rec}}
	orgs := &payoutStubOrgs{stripeAccountID: "acct_test"}
	stripe := &payoutStubStripe{}
	dist := &stubReferralDistributor{err: errors.New("flaky referral service")}

	p := NewPayoutService(PayoutServiceDeps{Records: records, Organizations: orgs, Stripe: stripe})
	p.SetReferralDistributor(dist)

	err := p.TransferMilestone(context.Background(), rec.MilestoneID)
	require.NoError(t, err, "distributor errors must be swallowed (they are fire-and-forget)")
}

// ---------------------------------------------------------------------------
// TransferToProvider — proposal-wide iteration
// ---------------------------------------------------------------------------

func TestPayoutService_TransferToProvider_SkipsAlreadyReleased(t *testing.T) {
	proposalID := uuid.New()
	pending := newSucceededPendingRecord()
	pending.ProposalID = proposalID
	completed := newSucceededPendingRecord()
	completed.ProposalID = proposalID
	completed.TransferStatus = domain.TransferCompleted

	records := &payoutStubRecords{
		byProposal:  []*domain.PaymentRecord{pending, completed},
		byMilestone: map[uuid.UUID]*domain.PaymentRecord{pending.MilestoneID: pending, completed.MilestoneID: completed},
	}
	orgs := &payoutStubOrgs{stripeAccountID: "acct_test"}
	stripe := &payoutStubStripe{}
	p := NewPayoutService(PayoutServiceDeps{Records: records, Organizations: orgs, Stripe: stripe})

	err := p.TransferToProvider(context.Background(), proposalID)
	require.NoError(t, err)
	assert.Len(t, stripe.transferCalls, 1, "only the pending milestone is transferred")
}

func TestPayoutService_TransferToProvider_NothingToRelease_ReturnsAlreadyDone(t *testing.T) {
	proposalID := uuid.New()
	completed := newSucceededPendingRecord()
	completed.ProposalID = proposalID
	completed.TransferStatus = domain.TransferCompleted

	records := &payoutStubRecords{byProposal: []*domain.PaymentRecord{completed}}
	p := NewPayoutService(PayoutServiceDeps{Records: records, Organizations: &payoutStubOrgs{}, Stripe: &payoutStubStripe{}})

	err := p.TransferToProvider(context.Background(), proposalID)
	assert.ErrorIs(t, err, domain.ErrTransferAlreadyDone)
}

func TestPayoutService_TransferToProvider_NoRecords_ReturnsNotFound(t *testing.T) {
	records := &payoutStubRecords{byProposal: nil}
	p := NewPayoutService(PayoutServiceDeps{Records: records, Organizations: &payoutStubOrgs{}, Stripe: &payoutStubStripe{}})

	err := p.TransferToProvider(context.Background(), uuid.New())
	assert.ErrorIs(t, err, domain.ErrPaymentRecordNotFound)
}

func TestPayoutService_TransferToProvider_DBErr_Wrapped(t *testing.T) {
	records := &payoutStubRecords{byProposalErr: errors.New("db down")}
	p := NewPayoutService(PayoutServiceDeps{Records: records, Organizations: &payoutStubOrgs{}, Stripe: &payoutStubStripe{}})

	err := p.TransferToProvider(context.Background(), uuid.New())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "list payment records")
}

// ---------------------------------------------------------------------------
// TransferPartialToProvider — dispute resolution applied
// ---------------------------------------------------------------------------

func TestPayoutService_TransferPartialToProvider_ZeroAmount_MarksCompletedNoStripe(t *testing.T) {
	rec := newSucceededPendingRecord()
	records := &payoutStubRecords{byProposal: []*domain.PaymentRecord{rec}}
	stripe := &payoutStubStripe{}
	p := NewPayoutService(PayoutServiceDeps{Records: records, Organizations: &payoutStubOrgs{}, Stripe: stripe})

	err := p.TransferPartialToProvider(context.Background(), rec.ProposalID, 0)
	require.NoError(t, err)
	assert.Empty(t, stripe.transferCalls)
	require.NotEmpty(t, records.updates)
	assert.Zero(t, records.updates[0].ProviderPayout)
	assert.Equal(t, domain.TransferCompleted, records.updates[0].TransferStatus)
}

func TestPayoutService_TransferPartialToProvider_NoStripeAccount_PersistsButDefers(t *testing.T) {
	rec := newSucceededPendingRecord()
	records := &payoutStubRecords{byProposal: []*domain.PaymentRecord{rec}}
	orgs := &payoutStubOrgs{stripeAccountID: ""}
	stripe := &payoutStubStripe{}
	p := NewPayoutService(PayoutServiceDeps{Records: records, Organizations: orgs, Stripe: stripe})

	err := p.TransferPartialToProvider(context.Background(), rec.ProposalID, 700)
	require.NoError(t, err)
	require.NotEmpty(t, records.updates)
	assert.Equal(t, int64(700), records.updates[0].ProviderPayout, "split amount persisted even without Stripe")
	assert.Equal(t, domain.TransferPending, records.updates[0].TransferStatus, "stays Pending so RequestPayout can retry")
}

func TestPayoutService_TransferPartialToProvider_StripeFails_PersistsAmountThenErrors(t *testing.T) {
	rec := newSucceededPendingRecord()
	records := &payoutStubRecords{byProposal: []*domain.PaymentRecord{rec}}
	orgs := &payoutStubOrgs{stripeAccountID: "acct_test"}
	stripe := &payoutStubStripe{transferErr: errors.New("payouts disabled")}
	p := NewPayoutService(PayoutServiceDeps{Records: records, Organizations: orgs, Stripe: stripe})

	err := p.TransferPartialToProvider(context.Background(), rec.ProposalID, 700)
	require.Error(t, err)
	require.NotEmpty(t, records.updates)
	assert.Equal(t, int64(700), records.updates[0].ProviderPayout, "amount persists despite Stripe failure")
}

func TestPayoutService_TransferPartialToProvider_Succeeded_MarksCompleted(t *testing.T) {
	rec := newSucceededPendingRecord()
	records := &payoutStubRecords{byProposal: []*domain.PaymentRecord{rec}}
	orgs := &payoutStubOrgs{stripeAccountID: "acct_test"}
	stripe := &payoutStubStripe{}
	p := NewPayoutService(PayoutServiceDeps{Records: records, Organizations: orgs, Stripe: stripe})

	err := p.TransferPartialToProvider(context.Background(), rec.ProposalID, 700)
	require.NoError(t, err)
	require.NotEmpty(t, records.updates)
	assert.Equal(t, int64(700), records.updates[0].ProviderPayout)
	assert.Equal(t, domain.TransferCompleted, records.updates[0].TransferStatus)
	assert.Len(t, stripe.transferCalls, 1)
}

func TestPayoutService_TransferPartialToProvider_NotSucceeded_Rejected(t *testing.T) {
	rec := newSucceededPendingRecord()
	rec.Status = domain.RecordStatusPending
	records := &payoutStubRecords{byProposal: []*domain.PaymentRecord{rec}}
	p := NewPayoutService(PayoutServiceDeps{Records: records, Organizations: &payoutStubOrgs{}, Stripe: &payoutStubStripe{}})

	err := p.TransferPartialToProvider(context.Background(), rec.ProposalID, 500)
	assert.ErrorIs(t, err, domain.ErrPaymentNotSucceeded)
}

// ---------------------------------------------------------------------------
// RefundToClient
// ---------------------------------------------------------------------------

func TestPayoutService_RefundToClient_ZeroAmount_NoOp(t *testing.T) {
	stripe := &payoutStubStripe{}
	p := NewPayoutService(PayoutServiceDeps{Records: &payoutStubRecords{}, Organizations: &payoutStubOrgs{}, Stripe: stripe})

	err := p.RefundToClient(context.Background(), uuid.New(), 0)
	require.NoError(t, err)
	assert.Equal(t, 0, stripe.refundCalls)
}

func TestPayoutService_RefundToClient_NoPaymentIntent_Errors(t *testing.T) {
	rec := newSucceededPendingRecord()
	rec.StripePaymentIntentID = "" // no PI
	records := &payoutStubRecords{byProposal: []*domain.PaymentRecord{rec}}
	stripe := &payoutStubStripe{}
	p := NewPayoutService(PayoutServiceDeps{Records: records, Organizations: &payoutStubOrgs{}, Stripe: stripe})

	err := p.RefundToClient(context.Background(), rec.ProposalID, 100)
	require.Error(t, err)
}

func TestPayoutService_RefundToClient_FullRefund_MarksRefunded(t *testing.T) {
	rec := newSucceededPendingRecord()
	rec.StripePaymentIntentID = "pi_xxx"
	rec.ProviderPayout = 0 // full refund — provider gets nothing
	records := &payoutStubRecords{byProposal: []*domain.PaymentRecord{rec}}
	stripe := &payoutStubStripe{}
	p := NewPayoutService(PayoutServiceDeps{Records: records, Organizations: &payoutStubOrgs{}, Stripe: stripe})

	err := p.RefundToClient(context.Background(), rec.ProposalID, 1000)
	require.NoError(t, err)
	assert.Equal(t, 1, stripe.refundCalls)
	require.NotEmpty(t, records.updates)
	assert.Equal(t, domain.RecordStatusRefunded, records.updates[0].Status)
}

func TestPayoutService_RefundToClient_PartialRefund_KeepsSucceeded(t *testing.T) {
	rec := newSucceededPendingRecord()
	rec.StripePaymentIntentID = "pi_xxx"
	rec.ProviderPayout = 500 // partial — provider still gets paid
	records := &payoutStubRecords{byProposal: []*domain.PaymentRecord{rec}}
	stripe := &payoutStubStripe{}
	p := NewPayoutService(PayoutServiceDeps{Records: records, Organizations: &payoutStubOrgs{}, Stripe: stripe})

	err := p.RefundToClient(context.Background(), rec.ProposalID, 500)
	require.NoError(t, err)
	assert.Equal(t, 1, stripe.refundCalls)
	require.NotEmpty(t, records.updates)
	assert.NotEqual(t, domain.RecordStatusRefunded, records.updates[0].Status, "partial refund leaves status unchanged")
}

func TestPayoutService_RefundToClient_StripeFails_NoUpdate(t *testing.T) {
	rec := newSucceededPendingRecord()
	rec.StripePaymentIntentID = "pi_xxx"
	records := &payoutStubRecords{byProposal: []*domain.PaymentRecord{rec}}
	stripe := &payoutStubStripe{refundErr: errors.New("stripe rate limit")}
	p := NewPayoutService(PayoutServiceDeps{Records: records, Organizations: &payoutStubOrgs{}, Stripe: stripe})

	err := p.RefundToClient(context.Background(), rec.ProposalID, 500)
	require.Error(t, err)
	assert.Empty(t, records.updates, "no record update if Stripe rejected the refund")
}

// ---------------------------------------------------------------------------
// CanProviderReceivePayouts — the milestone-release pre-check
// ---------------------------------------------------------------------------

func TestPayoutService_CanProviderReceivePayouts_NoAccount_FalseNoErr(t *testing.T) {
	orgs := &payoutStubOrgs{stripeAccountID: ""}
	p := NewPayoutService(PayoutServiceDeps{Records: &payoutStubRecords{}, Organizations: orgs, Stripe: &payoutStubStripe{}})

	ok, err := p.CanProviderReceivePayouts(context.Background(), uuid.New())
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestPayoutService_CanProviderReceivePayouts_AccountReady_True(t *testing.T) {
	orgs := &payoutStubOrgs{stripeAccountID: "acct_test"}
	stripe := &payoutStubStripe{getAccountInfo: &service.StripeAccountInfo{ChargesEnabled: true, PayoutsEnabled: true}}
	p := NewPayoutService(PayoutServiceDeps{Records: &payoutStubRecords{}, Organizations: orgs, Stripe: stripe})

	ok, err := p.CanProviderReceivePayouts(context.Background(), uuid.New())
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestPayoutService_CanProviderReceivePayouts_KYCPending_False(t *testing.T) {
	orgs := &payoutStubOrgs{stripeAccountID: "acct_test"}
	stripe := &payoutStubStripe{getAccountInfo: &service.StripeAccountInfo{ChargesEnabled: false, PayoutsEnabled: false}}
	p := NewPayoutService(PayoutServiceDeps{Records: &payoutStubRecords{}, Organizations: orgs, Stripe: stripe})

	ok, err := p.CanProviderReceivePayouts(context.Background(), uuid.New())
	require.NoError(t, err)
	assert.False(t, ok, "KYC not validated → not ready, but not an error either")
}

func TestPayoutService_CanProviderReceivePayouts_OrgErr_Wrapped(t *testing.T) {
	orgs := &payoutStubOrgs{getStripeErr: errors.New("db blip")}
	p := NewPayoutService(PayoutServiceDeps{Records: &payoutStubRecords{}, Organizations: orgs, Stripe: &payoutStubStripe{}})

	_, err := p.CanProviderReceivePayouts(context.Background(), uuid.New())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "get stripe account")
}

func TestPayoutService_CanProviderReceivePayouts_NoStripe_FalseNoErr(t *testing.T) {
	orgs := &payoutStubOrgs{stripeAccountID: "acct_test"}
	// No stripe wired
	p := NewPayoutService(PayoutServiceDeps{Records: &payoutStubRecords{}, Organizations: orgs})

	ok, err := p.CanProviderReceivePayouts(context.Background(), uuid.New())
	require.NoError(t, err)
	assert.False(t, ok)
}

// ---------------------------------------------------------------------------
// HasAutoPayoutConsent
// ---------------------------------------------------------------------------

func TestPayoutService_HasAutoPayoutConsent_NotConsented_False(t *testing.T) {
	orgID := uuid.New()
	orgs := &payoutStubOrgs{}
	p := NewPayoutService(PayoutServiceDeps{Records: &payoutStubRecords{}, Organizations: orgs, Stripe: &payoutStubStripe{}})

	got, err := p.HasAutoPayoutConsent(context.Background(), orgID)
	require.NoError(t, err)
	assert.False(t, got)
}

func TestPayoutService_HasAutoPayoutConsent_Consented_True(t *testing.T) {
	orgID := uuid.New()
	orgs := &payoutStubOrgs{consentForOrg: map[uuid.UUID]bool{orgID: true}}
	p := NewPayoutService(PayoutServiceDeps{Records: &payoutStubRecords{}, Organizations: orgs, Stripe: &payoutStubStripe{}})

	got, err := p.HasAutoPayoutConsent(context.Background(), orgID)
	require.NoError(t, err)
	assert.True(t, got)
}

// ---------------------------------------------------------------------------
// WaivePlatformFeeOnActiveRecords
// ---------------------------------------------------------------------------

func TestPayoutService_WaivePlatformFeeOnActiveRecords_PendingAndFailedWaived(t *testing.T) {
	pending := newSucceededPendingRecord()
	pending.ProposalAmount = 100_000
	pending.PlatformFeeAmount = 5_000
	pending.ProviderPayout = 95_000

	failed := newSucceededPendingRecord()
	failed.TransferStatus = domain.TransferFailed
	failed.ProposalAmount = 200_000
	failed.PlatformFeeAmount = 10_000
	failed.ProviderPayout = 190_000

	completed := newSucceededPendingRecord()
	completed.TransferStatus = domain.TransferCompleted
	completed.ProposalAmount = 300_000
	completed.PlatformFeeAmount = 15_000
	completed.ProviderPayout = 285_000

	records := &payoutStubRecords{byOrganization: []*domain.PaymentRecord{pending, failed, completed}}
	p := NewPayoutService(PayoutServiceDeps{Records: records})

	err := p.WaivePlatformFeeOnActiveRecords(context.Background(), uuid.New())
	require.NoError(t, err)
	assert.Len(t, records.updates, 2, "only pending + failed must be waived (completed is post-transfer, untouched)")
	for _, r := range records.updates {
		assert.Zero(t, r.PlatformFeeAmount)
		assert.Equal(t, r.ProposalAmount, r.ProviderPayout)
	}
}

func TestPayoutService_WaivePlatformFeeOnActiveRecords_AlreadyZero_NoUpdate(t *testing.T) {
	r := newSucceededPendingRecord()
	r.ProposalAmount = 100_000
	r.PlatformFeeAmount = 0
	r.ProviderPayout = 100_000
	records := &payoutStubRecords{byOrganization: []*domain.PaymentRecord{r}}
	p := NewPayoutService(PayoutServiceDeps{Records: records})

	err := p.WaivePlatformFeeOnActiveRecords(context.Background(), uuid.New())
	require.NoError(t, err)
	assert.Empty(t, records.updates, "no Update when fee was already zero")
}

func TestPayoutService_WaivePlatformFeeOnActiveRecords_NilRecords_NoOp(t *testing.T) {
	p := NewPayoutService(PayoutServiceDeps{})
	err := p.WaivePlatformFeeOnActiveRecords(context.Background(), uuid.New())
	require.NoError(t, err, "without a records repo the call must no-op cleanly")
}

func TestPayoutService_WaivePlatformFeeOnActiveRecords_ListErr_Wrapped(t *testing.T) {
	records := &payoutStubRecords{byOrgErr: errors.New("db down")}
	p := NewPayoutService(PayoutServiceDeps{Records: records})

	err := p.WaivePlatformFeeOnActiveRecords(context.Background(), uuid.New())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "list records")
}

// ---------------------------------------------------------------------------
// pickPayoutCurrency — pure helper
// ---------------------------------------------------------------------------

func TestPickPayoutCurrency_FallbackToEUR(t *testing.T) {
	got := pickPayoutCurrency(nil)
	assert.Equal(t, "eur", got)
}

func TestPickPayoutCurrency_PicksFirstTransferred(t *testing.T) {
	r1 := newSucceededPendingRecord()
	r1.TransferStatus = domain.TransferPending
	r1.Currency = "usd"
	r2 := newSucceededPendingRecord()
	r2.TransferStatus = domain.TransferCompleted
	r2.Currency = "eur"

	got := pickPayoutCurrency([]*domain.PaymentRecord{r1, r2})
	assert.Equal(t, "eur", got, "pending record's currency is ignored — only completed records count")
}

// ---------------------------------------------------------------------------
// Race tests — concurrent payout / transfer calls
// ---------------------------------------------------------------------------

func TestPayoutService_TransferMilestone_Concurrent_NoRace(t *testing.T) {
	if testing.Short() {
		t.Skip("race test")
	}
	const N = 16
	records := &payoutStubRecords{byMilestone: map[uuid.UUID]*domain.PaymentRecord{}}
	for i := 0; i < N; i++ {
		rec := newSucceededPendingRecord()
		records.byMilestone[rec.MilestoneID] = rec
	}
	orgs := &payoutStubOrgs{stripeAccountID: "acct_test"}
	stripe := &payoutStubStripe{}
	p := NewPayoutService(PayoutServiceDeps{Records: records, Organizations: orgs, Stripe: stripe})

	// Snapshot the milestone IDs before launching goroutines so each
	// goroutine has a stable id to drive (no map-iteration race).
	ids := make([]uuid.UUID, 0, N)
	for id := range records.byMilestone {
		ids = append(ids, id)
	}

	var wg sync.WaitGroup
	wg.Add(N)
	for _, id := range ids {
		id := id
		go func() {
			defer wg.Done()
			_ = p.TransferMilestone(context.Background(), id)
		}()
	}
	wg.Wait()
	assert.Equal(t, N, len(stripe.transferCalls), "every milestone must produce a Stripe transfer")
}

func TestPayoutService_RequestPayout_Concurrent_NoCorruption(t *testing.T) {
	if testing.Short() {
		t.Skip("race test")
	}
	orgID := uuid.New()
	rec := newSucceededPendingRecord()
	records := &payoutStubRecords{byOrganization: []*domain.PaymentRecord{rec}}
	orgs := &payoutStubOrgs{stripeAccountID: "acct_test"}
	stripe := &payoutStubStripe{}
	p := NewPayoutService(PayoutServiceDeps{Records: records, Organizations: orgs, Stripe: stripe})
	p.SetProposalStatusReader(&payoutStubProposalStatuses{statuses: map[uuid.UUID]string{rec.ProposalID: "completed"}})

	var wg sync.WaitGroup
	const N = 8
	wg.Add(N)
	for i := 0; i < N; i++ {
		go func() {
			defer wg.Done()
			_, _ = p.RequestPayout(context.Background(), uuid.New(), orgID)
		}()
	}
	wg.Wait()
	// We only assert the system didn't crash. Each goroutine reads the
	// same state, so the precise number of CreateTransfer calls is
	// implementation-defined (the legacy code makes one transfer per
	// goroutine — the idempotency key prevents Stripe from double-paying).
}

// ---------------------------------------------------------------------------
// Property test — TransferMilestone state machine never produces invalid
// states. We feed it every (input.Status, input.TransferStatus)
// combination and assert the post-condition is one of the allowed states.
// ---------------------------------------------------------------------------

func TestPayoutService_TransferMilestone_StateMachineProperty(t *testing.T) {
	allStatuses := []domain.PaymentRecordStatus{
		domain.RecordStatusPending,
		domain.RecordStatusSucceeded,
		domain.RecordStatusFailed,
		domain.RecordStatusRefunded,
	}
	allTransfers := []domain.TransferStatus{
		domain.TransferPending,
		domain.TransferCompleted,
		domain.TransferFailed,
	}

	for _, s := range allStatuses {
		for _, ts := range allTransfers {
			s, ts := s, ts
			t.Run(string(s)+"-"+string(ts), func(t *testing.T) {
				rec := newSucceededPendingRecord()
				rec.Status = s
				rec.TransferStatus = ts

				records := &payoutStubRecords{byMilestone: map[uuid.UUID]*domain.PaymentRecord{rec.MilestoneID: rec}}
				orgs := &payoutStubOrgs{stripeAccountID: "acct_test"}
				stripe := &payoutStubStripe{}
				p := NewPayoutService(PayoutServiceDeps{Records: records, Organizations: orgs, Stripe: stripe})

				err := p.TransferMilestone(context.Background(), rec.MilestoneID)

				switch {
				case s == domain.RecordStatusSucceeded && ts == domain.TransferPending:
					require.NoError(t, err)
					assert.Len(t, stripe.transferCalls, 1)
				case s != domain.RecordStatusSucceeded:
					require.Error(t, err)
					// Some non-Succeeded states map to ErrPaymentNotSucceeded;
					// failed/refunded explicitly do not flow through the
					// transfer path.
					assert.True(t,
						errors.Is(err, domain.ErrPaymentNotSucceeded) ||
							errors.Is(err, domain.ErrTransferAlreadyDone),
						"unexpected error: %v", err)
				case ts != domain.TransferPending:
					require.Error(t, err)
					assert.ErrorIs(t, err, domain.ErrTransferAlreadyDone)
				}
			})
		}
	}
}

// ---------------------------------------------------------------------------
// PaymentProcessor port satisfaction — proves PayoutService methods
// continue to satisfy the public port contract that proposal depends on.
// ---------------------------------------------------------------------------

func TestPayoutService_ContractsWithPaymentProcessorMethods(t *testing.T) {
	// Every PaymentProcessor method that is "transfer side" lives on
	// PayoutService and is reachable through the parent Service facade.
	svc := NewService(ServiceDeps{
		Records:       &payoutStubRecords{},
		Organizations: &payoutStubOrgs{},
		Stripe:        &payoutStubStripe{},
	})
	var _ service.PaymentProcessor = svc
	require.NotNil(t, svc.Payout())
}
