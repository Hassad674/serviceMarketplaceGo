package payment

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/domain/organization"
	domain "marketplace-backend/internal/domain/payment"
	domainuser "marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/port/service"
)

// ---------------------------------------------------------------------------
// Service facade tests — every delegation method on the parent Service
// must call its underlying sub-service. We can't easily black-box this
// (the sub-services are unexported), so we test the WIDTH of the facade:
// each delegation compiles, returns the right type, and propagates
// errors properly. Behaviour tests live next to each sub-service.
// ---------------------------------------------------------------------------

// facadeRecords is a one-method-or-less stub for each backend method
// the facade tests exercise. We keep it tiny because the per-method
// behaviour was already covered in wallet_test/charge_test/payout_test —
// here we only care that the facade routes to the right sub-service.
type facadeRecords struct {
	walletStubRecords
	byMilestone   map[uuid.UUID]*domain.PaymentRecord
	byProposalRec *domain.PaymentRecord
	byPI          map[string]*domain.PaymentRecord
	byID          map[uuid.UUID]*domain.PaymentRecord
}

func (f *facadeRecords) GetByMilestoneID(_ context.Context, id uuid.UUID) (*domain.PaymentRecord, error) {
	if r, ok := f.byMilestone[id]; ok {
		cp := *r
		return &cp, nil
	}
	return nil, domain.ErrPaymentRecordNotFound
}

func (f *facadeRecords) GetByProposalID(_ context.Context, _ uuid.UUID) (*domain.PaymentRecord, error) {
	if f.byProposalRec == nil {
		return nil, domain.ErrPaymentRecordNotFound
	}
	cp := *f.byProposalRec
	return &cp, nil
}

func (f *facadeRecords) GetByPaymentIntentID(_ context.Context, id string) (*domain.PaymentRecord, error) {
	if r, ok := f.byPI[id]; ok {
		cp := *r
		return &cp, nil
	}
	return nil, domain.ErrPaymentRecordNotFound
}

func (f *facadeRecords) GetByID(_ context.Context, id uuid.UUID) (*domain.PaymentRecord, error) {
	if r, ok := f.byID[id]; ok {
		cp := *r
		return &cp, nil
	}
	return nil, domain.ErrPaymentRecordNotFound
}

// GetByIDForOrg delegates to GetByID — the facade routing test
// only cares that the right sub-service is invoked.
func (f *facadeRecords) GetByIDForOrg(ctx context.Context, id, _ uuid.UUID) (*domain.PaymentRecord, error) {
	return f.GetByID(ctx, id)
}

func (f *facadeRecords) ListByProposalID(_ context.Context, id uuid.UUID) ([]*domain.PaymentRecord, error) {
	for _, r := range f.byMilestone {
		if r.ProposalID == id {
			return []*domain.PaymentRecord{r}, nil
		}
	}
	return nil, nil
}

func (f *facadeRecords) ListByOrganization(_ context.Context, _ uuid.UUID) ([]*domain.PaymentRecord, error) {
	out := make([]*domain.PaymentRecord, 0, len(f.byMilestone))
	for _, r := range f.byMilestone {
		out = append(out, r)
	}
	return out, nil
}

func (f *facadeRecords) Create(_ context.Context, _ *domain.PaymentRecord) error { return nil }
func (f *facadeRecords) Update(_ context.Context, _ *domain.PaymentRecord) error { return nil }

type facadeOrgs struct {
	stripeAccountID string
}

func (facadeOrgs) Create(_ context.Context, _ *organization.Organization) error { return nil }
func (facadeOrgs) CreateWithOwnerMembership(_ context.Context, _ *organization.Organization, _ *organization.Member) error {
	return nil
}
func (f facadeOrgs) FindByID(_ context.Context, id uuid.UUID) (*organization.Organization, error) {
	return &organization.Organization{ID: id}, nil
}
func (facadeOrgs) FindByOwnerUserID(_ context.Context, _ uuid.UUID) (*organization.Organization, error) {
	return nil, nil
}
func (facadeOrgs) FindByUserID(_ context.Context, _ uuid.UUID) (*organization.Organization, error) {
	return &organization.Organization{ID: uuid.New()}, nil
}
func (facadeOrgs) Update(_ context.Context, _ *organization.Organization) error { return nil }
func (facadeOrgs) Delete(_ context.Context, _ uuid.UUID) error                  { return nil }
func (facadeOrgs) SaveRoleOverrides(_ context.Context, _ uuid.UUID, _ organization.RoleOverrides) error {
	return nil
}
func (facadeOrgs) CountAll(_ context.Context) (int, error) { return 0, nil }
func (facadeOrgs) FindByStripeAccountID(_ context.Context, _ string) (*organization.Organization, error) {
	return nil, nil
}
func (facadeOrgs) ListKYCPending(_ context.Context) ([]*organization.Organization, error) {
	return nil, nil
}
func (facadeOrgs) ListWithStripeAccount(_ context.Context) ([]uuid.UUID, error) { return nil, nil }
func (f facadeOrgs) GetStripeAccount(_ context.Context, _ uuid.UUID) (string, string, error) {
	return f.stripeAccountID, "FR", nil
}
func (f facadeOrgs) GetStripeAccountByUserID(_ context.Context, _ uuid.UUID) (string, string, error) {
	return f.stripeAccountID, "FR", nil
}
func (facadeOrgs) SetStripeAccount(_ context.Context, _ uuid.UUID, _, _ string) error { return nil }
func (facadeOrgs) ClearStripeAccount(_ context.Context, _ uuid.UUID) error            { return nil }
func (facadeOrgs) GetStripeLastState(_ context.Context, _ uuid.UUID) ([]byte, error) {
	return nil, nil
}
func (facadeOrgs) SaveStripeLastState(_ context.Context, _ uuid.UUID, _ []byte) error { return nil }
func (facadeOrgs) SetKYCFirstEarning(_ context.Context, _ uuid.UUID, _ time.Time) error {
	return nil
}
func (facadeOrgs) SaveKYCNotificationState(_ context.Context, _ uuid.UUID, _ map[string]time.Time) error {
	return nil
}

type facadeStripe struct {
	service.StripeService
}

func (facadeStripe) CreatePaymentIntent(_ context.Context, in service.CreatePaymentIntentInput) (*service.PaymentIntentResult, error) {
	return &service.PaymentIntentResult{
		PaymentIntentID: "pi_facade",
		ClientSecret:    "cs_facade",
	}, nil
}
func (facadeStripe) GetPaymentIntent(_ context.Context, _ string) (*service.PaymentIntentStatus, error) {
	return &service.PaymentIntentStatus{Status: "succeeded"}, nil
}
func (facadeStripe) CreateTransfer(_ context.Context, _ service.CreateTransferInput) (string, error) {
	return "tr_facade", nil
}
func (facadeStripe) CreatePayout(_ context.Context, _ service.CreatePayoutInput) (string, error) {
	return "po_facade", nil
}
func (facadeStripe) CreateRefund(_ context.Context, _ string, _ int64) (string, error) {
	return "re_facade", nil
}
func (facadeStripe) GetAccount(_ context.Context, _ string) (*service.StripeAccountInfo, error) {
	return &service.StripeAccountInfo{ChargesEnabled: true, PayoutsEnabled: true}, nil
}
func (facadeStripe) ConstructWebhookEvent(_ []byte, _ string) (*service.StripeWebhookEvent, error) {
	return &service.StripeWebhookEvent{Type: "test"}, nil
}

type facadeUsers struct {
	walletStubUsers
}

func (f facadeUsers) GetByID(_ context.Context, _ uuid.UUID) (*domainuser.User, error) {
	return &domainuser.User{ID: uuid.New(), Role: domainuser.RoleProvider}, nil
}

func newFacadeService() *Service {
	return NewService(ServiceDeps{
		Records:       &facadeRecords{},
		Users:         &facadeUsers{},
		Organizations: facadeOrgs{stripeAccountID: "acct_facade"},
		Stripe:        facadeStripe{},
	})
}

func TestService_StripeConfigured_TrueWhenWired(t *testing.T) {
	s := newFacadeService()
	assert.True(t, s.StripeConfigured())
}

func TestService_StripeConfigured_FalseWhenNotWired(t *testing.T) {
	s := NewService(ServiceDeps{
		Records: &facadeRecords{}, Users: &facadeUsers{}, Organizations: facadeOrgs{},
	})
	assert.False(t, s.StripeConfigured())
}

func TestService_FacadeAccessors_ReturnSubServices(t *testing.T) {
	s := newFacadeService()
	assert.NotNil(t, s.Wallet())
	assert.NotNil(t, s.Charge())
	assert.NotNil(t, s.Payout())
}

func TestService_GetWalletOverview_DelegatesToWallet(t *testing.T) {
	s := newFacadeService()
	ov, err := s.GetWalletOverview(context.Background(), uuid.New(), uuid.New())
	require.NoError(t, err)
	assert.Equal(t, "acct_facade", ov.StripeAccountID)
}

func TestService_GetPaymentRecord_DelegatesToWallet(t *testing.T) {
	rec := newSucceededPendingRecord()
	s := NewService(ServiceDeps{
		Records:       &facadeRecords{byProposalRec: rec},
		Users:         &facadeUsers{},
		Organizations: facadeOrgs{},
		Stripe:        facadeStripe{},
	})
	got, err := s.GetPaymentRecord(context.Background(), rec.ProposalID)
	require.NoError(t, err)
	assert.Equal(t, rec.ID, got.ID)
}

func TestService_PreviewFee_DelegatesToWallet(t *testing.T) {
	s := newFacadeService()
	out, err := s.PreviewFee(context.Background(), uuid.New(), 50000, nil)
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.True(t, out.ViewerIsProvider, "provider role default")
}

func TestService_CreatePaymentIntent_DelegatesToCharge(t *testing.T) {
	s := newFacadeService()
	out, err := s.CreatePaymentIntent(context.Background(), service.PaymentIntentInput{
		ProposalID:     uuid.New(),
		MilestoneID:    uuid.New(),
		ClientID:       uuid.New(),
		ProviderID:     uuid.New(),
		ProposalAmount: 50000,
	})
	require.NoError(t, err)
	assert.Equal(t, "cs_facade", out.ClientSecret)
}

func TestService_HandlePaymentSucceeded_DelegatesToCharge(t *testing.T) {
	rec := &domain.PaymentRecord{
		ID:                    uuid.New(),
		ProposalID:            uuid.New(),
		StripePaymentIntentID: "pi_facade",
		Status:                domain.RecordStatusPending,
	}
	s := NewService(ServiceDeps{
		Records:       &facadeRecords{byPI: map[string]*domain.PaymentRecord{"pi_facade": rec}},
		Users:         &facadeUsers{},
		Organizations: facadeOrgs{},
		Stripe:        facadeStripe{},
	})
	gotID, err := s.HandlePaymentSucceeded(context.Background(), "pi_facade")
	require.NoError(t, err)
	assert.Equal(t, rec.ProposalID, gotID)
}

func TestService_VerifyWebhook_DelegatesToCharge(t *testing.T) {
	s := newFacadeService()
	ev, err := s.VerifyWebhook([]byte("payload"), "sig")
	require.NoError(t, err)
	assert.Equal(t, "test", ev.Type)
}

func TestService_TransferToProvider_DelegatesToPayout(t *testing.T) {
	rec := newSucceededPendingRecord()
	s := NewService(ServiceDeps{
		Records:       &facadeRecords{byMilestone: map[uuid.UUID]*domain.PaymentRecord{rec.MilestoneID: rec}},
		Users:         &facadeUsers{},
		Organizations: facadeOrgs{stripeAccountID: "acct_facade"},
		Stripe:        facadeStripe{},
	})
	err := s.TransferToProvider(context.Background(), rec.ProposalID)
	require.NoError(t, err)
}

func TestService_RefundToClient_DelegatesToPayout(t *testing.T) {
	rec := newSucceededPendingRecord()
	rec.StripePaymentIntentID = "pi_xxx"
	s := NewService(ServiceDeps{
		Records:       &facadeRecords{byProposalRec: rec},
		Users:         &facadeUsers{},
		Organizations: facadeOrgs{},
		Stripe:        facadeStripe{},
	})
	err := s.RefundToClient(context.Background(), rec.ProposalID, 500)
	require.NoError(t, err)
}

func TestService_CanProviderReceivePayouts_DelegatesToPayout(t *testing.T) {
	s := newFacadeService()
	ok, err := s.CanProviderReceivePayouts(context.Background(), uuid.New())
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestService_HasAutoPayoutConsent_DelegatesToPayout(t *testing.T) {
	s := newFacadeService()
	got, err := s.HasAutoPayoutConsent(context.Background(), uuid.New())
	require.NoError(t, err)
	assert.False(t, got, "facade default org has no consent stamped")
}

func TestService_SetReferralDistributor_NilSafe(t *testing.T) {
	s := newFacadeService()
	s.SetReferralDistributor(nil) // must not panic
}

func TestService_SetReferralClawback_StoresOnFacade(t *testing.T) {
	s := newFacadeService()
	s.SetReferralClawback(nil) // setter must compile and not panic
	assert.Nil(t, s.referralClawback, "nil setter must keep field nil")
}

func TestService_SetReferralWalletReader_DelegatesToWallet(t *testing.T) {
	s := newFacadeService()
	s.SetReferralWalletReader(nil) // must not panic
}

// ---------------------------------------------------------------------------
// PaymentProcessor port — the facade must satisfy the public contract
// proposal depends on. Compile-time + runtime check.
// ---------------------------------------------------------------------------

func TestService_SatisfiesPaymentProcessorPort(t *testing.T) {
	var _ service.PaymentProcessor = (*Service)(nil)
	s := newFacadeService()
	var ifaced service.PaymentProcessor = s
	assert.NotNil(t, ifaced)
}
