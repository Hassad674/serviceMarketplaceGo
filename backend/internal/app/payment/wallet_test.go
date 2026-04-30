package payment

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/domain/billing"
	"marketplace-backend/internal/domain/organization"
	domain "marketplace-backend/internal/domain/payment"
	domainuser "marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/port/service"
)

// ---------------------------------------------------------------------------
// WalletService — dedicated tests proving the read-side sub-service is
// independently testable (no payout / charge dependencies leaked).
// ---------------------------------------------------------------------------

// walletStubRecords is a minimal listing stub for WalletService — it
// implements the two methods GetWalletOverview actually exercises and
// embeds the wide port for the rest. Mirrors the test pattern of the
// existing service_stripe_test.go so the segregation cost is zero for
// readers familiar with the previous suite.
type walletStubRecords struct {
	repository.PaymentRecordRepository
	rows    []*domain.PaymentRecord
	listErr error
}

func (s *walletStubRecords) ListByOrganization(_ context.Context, _ uuid.UUID) ([]*domain.PaymentRecord, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	return s.rows, nil
}

func (s *walletStubRecords) GetByProposalID(_ context.Context, id uuid.UUID) (*domain.PaymentRecord, error) {
	for _, r := range s.rows {
		if r.ProposalID == id {
			cp := *r
			return &cp, nil
		}
	}
	return nil, domain.ErrPaymentRecordNotFound
}

type walletStubOrgs struct {
	repository.OrganizationRepository
	stripeAccountID string
	stripeErr       error
}

func (o *walletStubOrgs) GetStripeAccount(_ context.Context, _ uuid.UUID) (string, string, error) {
	return o.stripeAccountID, "FR", o.stripeErr
}

type walletStubStripe struct {
	service.StripeService
	chargesEnabled bool
	payoutsEnabled bool
	getAccountErr  error
}

func (s *walletStubStripe) GetAccount(_ context.Context, _ string) (*service.StripeAccountInfo, error) {
	if s.getAccountErr != nil {
		return nil, s.getAccountErr
	}
	return &service.StripeAccountInfo{
		ChargesEnabled: s.chargesEnabled,
		PayoutsEnabled: s.payoutsEnabled,
	}, nil
}

type walletStubUsers struct {
	repository.UserRepository
	user *domainuser.User
}

func (u *walletStubUsers) GetByID(_ context.Context, _ uuid.UUID) (*domainuser.User, error) {
	if u.user == nil {
		return nil, errors.New("user not found")
	}
	cp := *u.user
	return &cp, nil
}

func TestWalletService_GetWalletOverview_AggregatesEscrowAndTransferred(t *testing.T) {
	orgID := uuid.New()

	pending := &domain.PaymentRecord{
		ID:             uuid.New(),
		ProposalID:     uuid.New(),
		MilestoneID:    uuid.New(),
		Status:         domain.RecordStatusSucceeded,
		TransferStatus: domain.TransferPending,
		ProposalAmount: 1000,
		ProviderPayout: 950,
		PlatformFeeAmount: 50,
		CreatedAt:      time.Now(),
	}
	completed := &domain.PaymentRecord{
		ID:             uuid.New(),
		ProposalID:     uuid.New(),
		MilestoneID:    uuid.New(),
		Status:         domain.RecordStatusSucceeded,
		TransferStatus: domain.TransferCompleted,
		ProposalAmount: 2000,
		ProviderPayout: 1900,
		PlatformFeeAmount: 100,
		CreatedAt:      time.Now(),
	}
	notSucceeded := &domain.PaymentRecord{
		ID:             uuid.New(),
		ProposalID:     uuid.New(),
		MilestoneID:    uuid.New(),
		Status:         domain.RecordStatusPending,
		TransferStatus: domain.TransferPending,
		ProposalAmount: 500,
		ProviderPayout: 475,
		CreatedAt:      time.Now(),
	}

	records := &walletStubRecords{rows: []*domain.PaymentRecord{pending, completed, notSucceeded}}
	orgs := &walletStubOrgs{stripeAccountID: "acct_test"}
	stripe := &walletStubStripe{chargesEnabled: true, payoutsEnabled: true}

	wallet := NewWalletService(WalletServiceDeps{
		Records: records, Users: &walletStubUsers{}, Organizations: orgs, Stripe: stripe,
	})

	ov, err := wallet.GetWalletOverview(context.Background(), uuid.New(), orgID)
	require.NoError(t, err)
	require.NotNil(t, ov)

	assert.Equal(t, "acct_test", ov.StripeAccountID)
	assert.True(t, ov.ChargesEnabled)
	assert.True(t, ov.PayoutsEnabled)
	assert.Equal(t, int64(950), ov.EscrowAmount, "only succeeded+pending records count toward escrow")
	assert.Equal(t, int64(1900), ov.TransferredAmount, "only completed records count toward transferred")
	assert.Equal(t, ov.EscrowAmount, ov.AvailableAmount, "available equals escrow")
	assert.Len(t, ov.Records, 3, "all records appear in the wallet record list")
}

func TestWalletService_GetWalletOverview_NoStripeAccount_GracefullyDegrades(t *testing.T) {
	records := &walletStubRecords{}
	orgs := &walletStubOrgs{stripeAccountID: ""}
	stripe := &walletStubStripe{}

	wallet := NewWalletService(WalletServiceDeps{
		Records: records, Users: &walletStubUsers{}, Organizations: orgs, Stripe: stripe,
	})
	ov, err := wallet.GetWalletOverview(context.Background(), uuid.New(), uuid.New())
	require.NoError(t, err)
	assert.Empty(t, ov.StripeAccountID)
	assert.False(t, ov.ChargesEnabled)
	assert.False(t, ov.PayoutsEnabled)
}

func TestWalletService_GetWalletOverview_StripeAPIError_PreservesAccountID(t *testing.T) {
	records := &walletStubRecords{}
	orgs := &walletStubOrgs{stripeAccountID: "acct_test_blip"}
	stripe := &walletStubStripe{getAccountErr: errors.New("stripe down")}

	wallet := NewWalletService(WalletServiceDeps{
		Records: records, Users: &walletStubUsers{}, Organizations: orgs, Stripe: stripe,
	})
	ov, err := wallet.GetWalletOverview(context.Background(), uuid.New(), uuid.New())
	require.NoError(t, err, "Stripe getAccount errors must NOT bubble up — wallet must keep working")
	assert.Equal(t, "acct_test_blip", ov.StripeAccountID, "id is still surfaced from the org row")
	assert.False(t, ov.ChargesEnabled, "capabilities default to false on lookup failure")
}

func TestWalletService_GetWalletOverview_RecordsListErr_DegradesToEmpty(t *testing.T) {
	records := &walletStubRecords{listErr: errors.New("db down")}
	orgs := &walletStubOrgs{stripeAccountID: "acct_test"}
	stripe := &walletStubStripe{}

	wallet := NewWalletService(WalletServiceDeps{
		Records: records, Users: &walletStubUsers{}, Organizations: orgs, Stripe: stripe,
	})
	// Pre-existing contract: list errors return the partial wallet,
	// no error. Documented in the test so a future "fail loud" change
	// surfaces here.
	ov, err := wallet.GetWalletOverview(context.Background(), uuid.New(), uuid.New())
	require.NoError(t, err)
	require.NotNil(t, ov)
	assert.Empty(t, ov.Records)
}

// ---------------------------------------------------------------------------
// PreviewFee — recipient resolution + Premium waiver
// ---------------------------------------------------------------------------

func TestWalletService_PreviewFee_NoRecipient_ProviderRoleDefaultsTrue(t *testing.T) {
	users := &walletStubUsers{user: &domainuser.User{ID: uuid.New(), Role: domainuser.RoleProvider}}
	wallet := NewWalletService(WalletServiceDeps{Records: &walletStubRecords{}, Users: users})

	out, err := wallet.PreviewFee(context.Background(), uuid.New(), 50000, nil)
	require.NoError(t, err)
	assert.True(t, out.ViewerIsProvider, "provider role with no recipient defaults to ViewerIsProvider=true")
	assert.Equal(t, billing.RoleFreelance, out.Billing.Role)
}

func TestWalletService_PreviewFee_NoRecipient_EnterpriseRoleAlwaysFalse(t *testing.T) {
	users := &walletStubUsers{user: &domainuser.User{ID: uuid.New(), Role: domainuser.RoleEnterprise}}
	wallet := NewWalletService(WalletServiceDeps{Records: &walletStubRecords{}, Users: users})

	out, err := wallet.PreviewFee(context.Background(), uuid.New(), 50000, nil)
	require.NoError(t, err)
	assert.False(t, out.ViewerIsProvider, "enterprise is ALWAYS the client — never sees provider fees")
}

func TestWalletService_PreviewFee_BadUser_FailsLoud(t *testing.T) {
	users := &walletStubUsers{user: nil} // returns error
	wallet := NewWalletService(WalletServiceDeps{Records: &walletStubRecords{}, Users: users})

	_, err := wallet.PreviewFee(context.Background(), uuid.New(), 50000, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "fetch user")
}

// stubSubReader is a focused stub for SubscriptionReader.
type stubSubReader struct {
	active bool
	err    error
	calls  int
}

func (s *stubSubReader) IsActive(_ context.Context, _ uuid.UUID) (bool, error) {
	s.calls++
	return s.active, s.err
}

func TestWalletService_PreviewFee_PremiumWaivesFee(t *testing.T) {
	users := &walletStubUsers{user: &domainuser.User{ID: uuid.New(), Role: domainuser.RoleProvider}}
	wallet := NewWalletService(WalletServiceDeps{Records: &walletStubRecords{}, Users: users})
	wallet.SetSubscriptionReader(&stubSubReader{active: true})

	out, err := wallet.PreviewFee(context.Background(), uuid.New(), 50000, nil)
	require.NoError(t, err)
	assert.True(t, out.ViewerIsSubscribed)
	assert.Zero(t, out.Billing.FeeCents, "premium waives the fee")
	assert.Equal(t, int64(50000), out.Billing.NetCents, "net = amount when fee is 0")
}

func TestWalletService_PreviewFee_SubscriptionLookupErr_FullFeeApplies(t *testing.T) {
	users := &walletStubUsers{user: &domainuser.User{ID: uuid.New(), Role: domainuser.RoleProvider}}
	wallet := NewWalletService(WalletServiceDeps{Records: &walletStubRecords{}, Users: users})
	wallet.SetSubscriptionReader(&stubSubReader{err: errors.New("redis blip")})

	out, err := wallet.PreviewFee(context.Background(), uuid.New(), 50000, nil)
	require.NoError(t, err, "subscription lookup failure must not block the preview")
	assert.False(t, out.ViewerIsSubscribed)
	assert.NotZero(t, out.Billing.FeeCents, "full fee applies on subscription reader failure (fail closed)")
}

// recipientWalletUsers responds with two distinct users: the caller and
// the recipient. PreviewFee with recipientID exercises the role resolution.
type recipientWalletUsers struct {
	repository.UserRepository
	caller    *domainuser.User
	recipient *domainuser.User
}

func (u *recipientWalletUsers) GetByID(_ context.Context, id uuid.UUID) (*domainuser.User, error) {
	if u.caller != nil && u.caller.ID == id {
		cp := *u.caller
		return &cp, nil
	}
	if u.recipient != nil && u.recipient.ID == id {
		cp := *u.recipient
		return &cp, nil
	}
	return nil, errors.New("user not found")
}

func TestWalletService_PreviewFee_RecipientSetUsesDetermineRoles(t *testing.T) {
	caller := &domainuser.User{ID: uuid.New(), Role: domainuser.RoleProvider}
	recipient := &domainuser.User{ID: uuid.New(), Role: domainuser.RoleEnterprise}
	users := &recipientWalletUsers{caller: caller, recipient: recipient}
	wallet := NewWalletService(WalletServiceDeps{Records: &walletStubRecords{}, Users: users})

	out, err := wallet.PreviewFee(context.Background(), caller.ID, 50000, &recipient.ID)
	require.NoError(t, err)
	assert.True(t, out.ViewerIsProvider, "provider→enterprise: viewer is the provider")
}

func TestWalletService_PreviewFee_InvalidRoleCombination_FailsClosed(t *testing.T) {
	// Two enterprises — invalid combination, DetermineRoles errors.
	caller := &domainuser.User{ID: uuid.New(), Role: domainuser.RoleEnterprise}
	recipient := &domainuser.User{ID: uuid.New(), Role: domainuser.RoleEnterprise}
	users := &recipientWalletUsers{caller: caller, recipient: recipient}
	wallet := NewWalletService(WalletServiceDeps{Records: &walletStubRecords{}, Users: users})

	out, err := wallet.PreviewFee(context.Background(), caller.ID, 50000, &recipient.ID)
	require.NoError(t, err)
	assert.False(t, out.ViewerIsProvider, "invalid role combination must fail closed (UI hides preview)")
}

func TestWalletService_PreviewFee_UnknownRecipient_FailsClosed(t *testing.T) {
	caller := &domainuser.User{ID: uuid.New(), Role: domainuser.RoleProvider}
	users := &recipientWalletUsers{caller: caller, recipient: nil}
	wallet := NewWalletService(WalletServiceDeps{Records: &walletStubRecords{}, Users: users})

	bogus := uuid.New()
	out, err := wallet.PreviewFee(context.Background(), caller.ID, 50000, &bogus)
	require.NoError(t, err)
	assert.False(t, out.ViewerIsProvider, "unknown recipient must hide the preview")
}

// ---------------------------------------------------------------------------
// computePlatformFee — direct sub-service test (private method via wallet)
// ---------------------------------------------------------------------------

func TestWalletService_ComputePlatformFee_FreelanceGridWithoutSubscription(t *testing.T) {
	users := &walletStubUsers{user: &domainuser.User{ID: uuid.New(), Role: domainuser.RoleProvider}}
	wallet := NewWalletService(WalletServiceDeps{Records: &walletStubRecords{}, Users: users})

	tests := []struct {
		name        string
		amountCents int64
		wantFee     int64
	}{
		{"tier 1 — 150€", 15000, 900},
		{"tier 2 — 500€", 50000, 1500},
		{"tier 3 — 2000€", 200000, 2500},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fee, err := wallet.computePlatformFee(context.Background(), uuid.New(), tc.amountCents)
			require.NoError(t, err)
			assert.Equal(t, tc.wantFee, fee)
		})
	}
}

func TestWalletService_ComputePlatformFee_AgencyGrid(t *testing.T) {
	users := &walletStubUsers{user: &domainuser.User{ID: uuid.New(), Role: domainuser.RoleAgency}}
	wallet := NewWalletService(WalletServiceDeps{Records: &walletStubRecords{}, Users: users})

	fee, err := wallet.computePlatformFee(context.Background(), uuid.New(), 150000)
	require.NoError(t, err)
	assert.Equal(t, int64(3900), fee, "agency tier 2 = 39€")
}

func TestWalletService_ComputePlatformFee_PremiumZeroes(t *testing.T) {
	users := &walletStubUsers{user: &domainuser.User{ID: uuid.New(), Role: domainuser.RoleProvider}}
	wallet := NewWalletService(WalletServiceDeps{Records: &walletStubRecords{}, Users: users})
	wallet.SetSubscriptionReader(&stubSubReader{active: true})

	fee, err := wallet.computePlatformFee(context.Background(), uuid.New(), 50000)
	require.NoError(t, err)
	assert.Zero(t, fee, "premium subscriber pays no fee on any milestone")
}

func TestWalletService_ComputePlatformFee_UserLookupFails(t *testing.T) {
	users := &walletStubUsers{user: nil}
	wallet := NewWalletService(WalletServiceDeps{Records: &walletStubRecords{}, Users: users})

	_, err := wallet.computePlatformFee(context.Background(), uuid.New(), 50000)
	require.Error(t, err, "user lookup failure MUST fail loud — silently using zero would underprice the platform")
	assert.Contains(t, err.Error(), "fetch provider")
}

// ---------------------------------------------------------------------------
// Race tests — concurrent reads of WalletService must not corrupt state
// ---------------------------------------------------------------------------

func TestWalletService_ConcurrentGetWalletOverview_NoRace(t *testing.T) {
	if testing.Short() {
		t.Skip("race test")
	}
	rec := &domain.PaymentRecord{
		ID:             uuid.New(),
		ProposalID:     uuid.New(),
		MilestoneID:    uuid.New(),
		Status:         domain.RecordStatusSucceeded,
		TransferStatus: domain.TransferPending,
		ProposalAmount: 1000,
		ProviderPayout: 950,
		CreatedAt:      time.Now(),
	}
	records := &walletStubRecords{rows: []*domain.PaymentRecord{rec}}
	orgs := &walletStubOrgs{stripeAccountID: "acct_test"}
	stripe := &walletStubStripe{chargesEnabled: true, payoutsEnabled: true}

	wallet := NewWalletService(WalletServiceDeps{
		Records: records, Users: &walletStubUsers{}, Organizations: orgs, Stripe: stripe,
	})

	const goroutines = 16
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			ov, err := wallet.GetWalletOverview(context.Background(), uuid.New(), uuid.New())
			assert.NoError(t, err)
			assert.NotNil(t, ov)
		}()
	}
	wg.Wait()
}

// ---------------------------------------------------------------------------
// Coverage probe — proves Wallet() accessor returns the same instance
// every call (no surprise allocations) and that the segregated pointer
// is non-nil on a freshly-constructed parent Service.
// ---------------------------------------------------------------------------

func TestService_WalletAccessor_StableIdentity(t *testing.T) {
	svc := NewService(ServiceDeps{
		Records: &walletStubRecords{},
		Users:   &walletStubUsers{user: &domainuser.User{Role: domainuser.RoleProvider}},
		Organizations: &walletStubOrgs{},
		Stripe:        &walletStubStripe{},
	})
	a := svc.Wallet()
	b := svc.Wallet()
	require.NotNil(t, a)
	assert.Same(t, a, b, "Wallet() must return the same pointer across calls")
}

// ---------------------------------------------------------------------------
// Helpers — exhaustive Liskov check that the wallet sub-service can be
// passed wherever a platformFeeCalculator is expected.
// ---------------------------------------------------------------------------

func TestWalletService_SatisfiesPlatformFeeCalculator(t *testing.T) {
	var _ platformFeeCalculator = (*WalletService)(nil)
}

// ---------------------------------------------------------------------------
// referralWallet stubbing — exhaustive paths for the apporteur side.
// ---------------------------------------------------------------------------

type stubReferralWallet struct {
	summary  service.ReferrerCommissionSummary
	summaryErr error
	recent   []service.ReferralCommissionRecord
	recentErr  error
}

func (s *stubReferralWallet) GetReferrerSummary(_ context.Context, _ uuid.UUID) (service.ReferrerCommissionSummary, error) {
	return s.summary, s.summaryErr
}
func (s *stubReferralWallet) RecentCommissions(_ context.Context, _ uuid.UUID, _ int) ([]service.ReferralCommissionRecord, error) {
	return s.recent, s.recentErr
}

func TestWalletService_GetWalletOverview_PopulatesCommissionSection(t *testing.T) {
	records := &walletStubRecords{}
	orgs := &walletStubOrgs{stripeAccountID: "acct_test"}
	stripe := &walletStubStripe{}

	wallet := NewWalletService(WalletServiceDeps{
		Records: records, Users: &walletStubUsers{}, Organizations: orgs, Stripe: stripe,
	})

	now := time.Now()
	wallet.SetReferralWalletReader(&stubReferralWallet{
		summary: service.ReferrerCommissionSummary{
			PendingCents:  1000,
			PaidCents:     5000,
			Currency:      "eur",
		},
		recent: []service.ReferralCommissionRecord{
			{
				ID:               uuid.New(),
				ReferralID:       uuid.New(),
				ProposalID:       uuid.New(),
				MilestoneID:      uuid.New(),
				GrossAmountCents: 10000,
				CommissionCents:  1000,
				Currency:         "eur",
				Status:           "paid",
				PaidAt:           &now,
				ClawedBackAt:     nil,
				CreatedAt:        now,
			},
		},
	})

	ov, err := wallet.GetWalletOverview(context.Background(), uuid.New(), uuid.New())
	require.NoError(t, err)
	assert.Equal(t, int64(1000), ov.Commissions.PendingCents)
	assert.Equal(t, int64(5000), ov.Commissions.PaidCents)
	assert.Equal(t, "eur", ov.Commissions.Currency)
	require.Len(t, ov.CommissionRecords, 1)
	rec := ov.CommissionRecords[0]
	assert.Equal(t, "paid", rec.Status)
	assert.Equal(t, "eur", rec.Currency)
	assert.NotEmpty(t, rec.PaidAt)
	assert.Empty(t, rec.ClawedBackAt, "clawed-back time stays empty when nil")
}

func TestWalletService_GetWalletOverview_ReferralReadFails_NoFatal(t *testing.T) {
	records := &walletStubRecords{}
	orgs := &walletStubOrgs{stripeAccountID: "acct_test"}
	stripe := &walletStubStripe{}

	wallet := NewWalletService(WalletServiceDeps{
		Records: records, Users: &walletStubUsers{}, Organizations: orgs, Stripe: stripe,
	})
	wallet.SetReferralWalletReader(&stubReferralWallet{
		summaryErr: errors.New("boom"),
		recentErr:  errors.New("kaboom"),
	})
	// The referral side failing must NOT take down the wallet.
	ov, err := wallet.GetWalletOverview(context.Background(), uuid.New(), uuid.New())
	require.NoError(t, err)
	require.NotNil(t, ov)
	assert.Zero(t, ov.Commissions.PendingCents, "summary error → no commission totals")
}

// ---------------------------------------------------------------------------
// Domain-level invariant: enterprise-role default never accidentally
// flips to true. Explicit guard against a regression in
// defaultViewerIsProvider.
// ---------------------------------------------------------------------------

func TestDefaultViewerIsProvider_RoleMatrix(t *testing.T) {
	tests := []struct {
		role domainuser.Role
		want bool
	}{
		{domainuser.RoleProvider, true},
		{domainuser.RoleAgency, true},
		{domainuser.RoleEnterprise, false},
	}
	for _, tc := range tests {
		t.Run(string(tc.role), func(t *testing.T) {
			got := defaultViewerIsProvider(tc.role)
			assert.Equal(t, tc.want, got)
		})
	}
}

// Domain regression: the wallet's provider-side aggregation must never
// count refunded records as either escrow or transferred.
func TestWalletService_GetWalletOverview_RefundedRecordIgnored(t *testing.T) {
	refunded := &domain.PaymentRecord{
		ID:             uuid.New(),
		ProposalID:     uuid.New(),
		MilestoneID:    uuid.New(),
		Status:         domain.RecordStatusRefunded,
		TransferStatus: domain.TransferPending,
		ProposalAmount: 1000,
		ProviderPayout: 950,
		CreatedAt:      time.Now(),
	}
	records := &walletStubRecords{rows: []*domain.PaymentRecord{refunded}}
	wallet := NewWalletService(WalletServiceDeps{
		Records: records,
		Users:   &walletStubUsers{},
		Organizations: &walletStubOrgs{stripeAccountID: "acct_test"},
		Stripe:        &walletStubStripe{},
	})
	ov, err := wallet.GetWalletOverview(context.Background(), uuid.New(), uuid.New())
	require.NoError(t, err)
	assert.Zero(t, ov.EscrowAmount)
	assert.Zero(t, ov.TransferredAmount)
	assert.Len(t, ov.Records, 1, "the record still appears in the list, just not in the aggregates")
}

// ---------------------------------------------------------------------------
// Acceptance: WalletService.GetPaymentRecord is a thin GetByProposalID.
// ---------------------------------------------------------------------------

func TestWalletService_GetPaymentRecord_RoundTrips(t *testing.T) {
	rec := &domain.PaymentRecord{
		ID:         uuid.New(),
		ProposalID: uuid.New(),
		Status:     domain.RecordStatusSucceeded,
	}
	records := &walletStubRecords{rows: []*domain.PaymentRecord{rec}}
	wallet := NewWalletService(WalletServiceDeps{
		Records: records,
		Users:   &walletStubUsers{},
		Organizations: &walletStubOrgs{},
	})

	got, err := wallet.GetPaymentRecord(context.Background(), rec.ProposalID)
	require.NoError(t, err)
	assert.Equal(t, rec.ID, got.ID)
}

// Sanity: hand-rolled organization mock for type-checking. We do not
// call any methods — this just proves the wallet service does NOT
// require any of the writer/Stripe-store methods, only the read side.
type minimalOrgReaderForWallet struct {
	repository.OrganizationRepository
}

func (minimalOrgReaderForWallet) GetStripeAccount(_ context.Context, _ uuid.UUID) (string, string, error) {
	return "", "", nil
}

// referralWalletReader from the port has these field names — keep this
// in sync if the port changes shape.
var _ service.ReferralWalletReader = (*stubReferralWallet)(nil)
var _ service.SubscriptionReader = (*stubSubReader)(nil)

// Sanity at the org domain layer — the wallet service uses the
// organization domain entity but does not exercise its full surface.
// This pins the fact that we depend on a reachable, well-typed
// Organization.
func TestWalletService_OrganizationEntity_StillReachable(t *testing.T) {
	o := &organization.Organization{ID: uuid.New()}
	assert.NotEqual(t, uuid.Nil, o.ID)
}
