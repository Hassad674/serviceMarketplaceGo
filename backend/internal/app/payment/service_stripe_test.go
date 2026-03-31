package payment

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domain "marketplace-backend/internal/domain/payment"
	"marketplace-backend/internal/port/repository"
	portservice "marketplace-backend/internal/port/service"
)

// --- helpers ---

func newPendingRecord(proposalID, clientID, providerID uuid.UUID) *domain.PaymentRecord {
	return domain.NewPaymentRecord(proposalID, clientID, providerID, 10000, 175)
}

func succeededRecord(proposalID, clientID, providerID uuid.UUID) *domain.PaymentRecord {
	r := newPendingRecord(proposalID, clientID, providerID)
	r.StripePaymentIntentID = "pi_existing"
	_ = r.MarkPaid()
	return r
}

func newTestService(
	infoRepo *mockPaymentInfoRepo,
	recordRepo *mockPaymentRecordRepo,
	stripe *mockStripeService,
) *Service {
	return NewService(infoRepo, recordRepo, stripe, nil, "")
}

// --- CreatePaymentIntent tests ---

func TestCreatePaymentIntent_NewPayment(t *testing.T) {
	proposalID := uuid.New()
	records := &mockPaymentRecordRepo{
		getByProposalIDFn: func(_ context.Context, _ uuid.UUID) (*domain.PaymentRecord, error) {
			return nil, domain.ErrPaymentRecordNotFound
		},
	}
	stripe := &mockStripeService{
		createPaymentIntentFn: func(_ context.Context, _ portservice.CreatePaymentIntentInput) (*portservice.PaymentIntentResult, error) {
			return &portservice.PaymentIntentResult{
				PaymentIntentID: "pi_new",
				ClientSecret:    "pi_new_secret",
			}, nil
		},
	}
	svc := newTestService(&mockPaymentInfoRepo{}, records, stripe)

	out, err := svc.CreatePaymentIntent(context.Background(), portservice.PaymentIntentInput{
		ProposalID:     proposalID,
		ClientID:       uuid.New(),
		ProviderID:     uuid.New(),
		ProposalAmount: 10000,
	})

	require.NoError(t, err)
	assert.Equal(t, "pi_new_secret", out.ClientSecret)
	assert.Equal(t, int64(10000), out.ProposalAmount)
}

func TestCreatePaymentIntent_ExistingRecord(t *testing.T) {
	proposalID := uuid.New()
	existing := newPendingRecord(proposalID, uuid.New(), uuid.New())
	existing.StripePaymentIntentID = "pi_old"

	records := &mockPaymentRecordRepo{
		getByProposalIDFn: func(_ context.Context, id uuid.UUID) (*domain.PaymentRecord, error) {
			if id == proposalID {
				return existing, nil
			}
			return nil, domain.ErrPaymentRecordNotFound
		},
	}
	stripe := &mockStripeService{
		createPaymentIntentFn: func(_ context.Context, _ portservice.CreatePaymentIntentInput) (*portservice.PaymentIntentResult, error) {
			return &portservice.PaymentIntentResult{
				PaymentIntentID: "pi_old",
				ClientSecret:    "pi_old_secret",
			}, nil
		},
	}
	svc := newTestService(&mockPaymentInfoRepo{}, records, stripe)

	out, err := svc.CreatePaymentIntent(context.Background(), portservice.PaymentIntentInput{
		ProposalID:     proposalID,
		ClientID:       existing.ClientID,
		ProviderID:     existing.ProviderID,
		ProposalAmount: existing.ProposalAmount,
	})

	require.NoError(t, err)
	assert.Equal(t, "pi_old_secret", out.ClientSecret)
	assert.Equal(t, existing.ID, out.PaymentRecordID)
}

func TestTransferToProvider_Success(t *testing.T) {
	proposalID := uuid.New()
	providerID := uuid.New()
	record := succeededRecord(proposalID, uuid.New(), providerID)

	records := &mockPaymentRecordRepo{
		getByProposalIDFn: func(_ context.Context, _ uuid.UUID) (*domain.PaymentRecord, error) {
			return record, nil
		},
	}
	info := &mockPaymentInfoRepo{
		getByUserIDFn: func(_ context.Context, _ uuid.UUID) (*domain.PaymentInfo, error) {
			return &domain.PaymentInfo{
				UserID:          providerID,
				StripeAccountID: "acct_provider",
			}, nil
		},
	}
	stripe := &mockStripeService{
		createTransferFn: func(_ context.Context, _ portservice.CreateTransferInput) (string, error) {
			return "tr_ok", nil
		},
	}
	svc := newTestService(info, records, stripe)

	err := svc.TransferToProvider(context.Background(), proposalID)

	require.NoError(t, err)
	assert.Equal(t, domain.TransferCompleted, record.TransferStatus)
}

func TestTransferToProvider_NotSucceeded(t *testing.T) {
	record := newPendingRecord(uuid.New(), uuid.New(), uuid.New())
	records := &mockPaymentRecordRepo{
		getByProposalIDFn: func(_ context.Context, _ uuid.UUID) (*domain.PaymentRecord, error) {
			return record, nil
		},
	}
	svc := newTestService(&mockPaymentInfoRepo{}, records, nil)

	err := svc.TransferToProvider(context.Background(), uuid.New())
	assert.ErrorIs(t, err, domain.ErrPaymentNotSucceeded)
}

func TestVerifyWebhook_StripeNotConfigured(t *testing.T) {
	svc := NewService(&mockPaymentInfoRepo{}, &mockPaymentRecordRepo{}, nil, nil, "")

	_, err := svc.VerifyWebhook([]byte("body"), "sig")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "stripe not configured")
}

func TestStripeConfigured_True(t *testing.T) {
	svc := newTestService(&mockPaymentInfoRepo{}, &mockPaymentRecordRepo{}, &mockStripeService{})
	assert.True(t, svc.StripeConfigured())
}

func TestStripeConfigured_False(t *testing.T) {
	svc := NewService(&mockPaymentInfoRepo{}, &mockPaymentRecordRepo{}, nil, nil, "")
	assert.False(t, svc.StripeConfigured())
}

func TestHandleAccountUpdated_Syncs(t *testing.T) {
	userID := uuid.New()
	var syncCalled bool

	info := &mockPaymentInfoRepo{
		getByStripeAccountFn: func(_ context.Context, _ string) (*domain.PaymentInfo, error) {
			return &domain.PaymentInfo{UserID: userID, StripeAccountID: "acct_upd"}, nil
		},
		updateStripeSyncFieldsFn: func(_ context.Context, _ uuid.UUID, _ repository.StripeSyncInput) error {
			syncCalled = true
			return nil
		},
	}
	stripe := &mockStripeService{
		getFullAccountFn: func(_ context.Context, _ string) (*portservice.StripeAccountInfo, error) {
			return &portservice.StripeAccountInfo{
				ChargesEnabled: true,
				PayoutsEnabled: true,
				Country:        "FR",
				BusinessType:   "individual",
				DisplayName:    "Alice Dupont",
			}, nil
		},
	}
	svc := newTestService(info, &mockPaymentRecordRepo{}, stripe)

	err := svc.HandleAccountUpdated(context.Background(), "acct_upd")

	require.NoError(t, err)
	assert.True(t, syncCalled)
}
