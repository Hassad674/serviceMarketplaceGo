package payment

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domain "marketplace-backend/internal/domain/payment"
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
	return NewService(
		infoRepo,
		recordRepo,
		&mockIdentityDocRepo{},
		&mockBusinessPersonRepo{},
		stripe,
		&mockStorageService{},
	)
}

// --- CreatePaymentIntent tests ---

func TestCreatePaymentIntent_NewPayment(t *testing.T) {
	proposalID := uuid.New()
	clientID := uuid.New()
	providerID := uuid.New()

	records := &mockPaymentRecordRepo{
		getByProposalIDFn: func(_ context.Context, _ uuid.UUID) (*domain.PaymentRecord, error) {
			return nil, domain.ErrPaymentRecordNotFound
		},
		createFn: func(_ context.Context, _ *domain.PaymentRecord) error {
			return nil
		},
	}
	stripe := &mockStripeService{
		createPaymentIntentFn: func(_ context.Context, input portservice.CreatePaymentIntentInput) (*portservice.PaymentIntentResult, error) {
			return &portservice.PaymentIntentResult{
				PaymentIntentID: "pi_new",
				ClientSecret:    "pi_new_secret",
			}, nil
		},
	}
	svc := newTestService(&mockPaymentInfoRepo{}, records, stripe)

	out, err := svc.CreatePaymentIntent(context.Background(), portservice.PaymentIntentInput{
		ProposalID:     proposalID,
		ClientID:       clientID,
		ProviderID:     providerID,
		ProposalAmount: 10000,
	})

	require.NoError(t, err)
	assert.Equal(t, "pi_new_secret", out.ClientSecret)
	assert.Equal(t, int64(10000), out.ProposalAmount)
	assert.NotEqual(t, uuid.Nil, out.PaymentRecordID)
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

func TestCreatePaymentIntent_StripeError(t *testing.T) {
	records := &mockPaymentRecordRepo{
		getByProposalIDFn: func(_ context.Context, _ uuid.UUID) (*domain.PaymentRecord, error) {
			return nil, domain.ErrPaymentRecordNotFound
		},
	}
	stripe := &mockStripeService{
		createPaymentIntentFn: func(_ context.Context, _ portservice.CreatePaymentIntentInput) (*portservice.PaymentIntentResult, error) {
			return nil, errors.New("stripe unavailable")
		},
	}
	svc := newTestService(&mockPaymentInfoRepo{}, records, stripe)

	out, err := svc.CreatePaymentIntent(context.Background(), portservice.PaymentIntentInput{
		ProposalID:     uuid.New(),
		ClientID:       uuid.New(),
		ProviderID:     uuid.New(),
		ProposalAmount: 5000,
	})

	assert.Nil(t, out)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "stripe")
}

// --- MarkPaymentSucceeded tests ---

func TestMarkPaymentSucceeded_Success(t *testing.T) {
	proposalID := uuid.New()
	record := newPendingRecord(proposalID, uuid.New(), uuid.New())
	var updated bool

	records := &mockPaymentRecordRepo{
		getByProposalIDFn: func(_ context.Context, id uuid.UUID) (*domain.PaymentRecord, error) {
			if id == proposalID {
				return record, nil
			}
			return nil, domain.ErrPaymentRecordNotFound
		},
		updateFn: func(_ context.Context, _ *domain.PaymentRecord) error {
			updated = true
			return nil
		},
	}
	svc := newTestService(&mockPaymentInfoRepo{}, records, nil)

	err := svc.MarkPaymentSucceeded(context.Background(), proposalID)

	require.NoError(t, err)
	assert.True(t, updated)
	assert.Equal(t, domain.RecordStatusSucceeded, record.Status)
}

func TestMarkPaymentSucceeded_AlreadySucceeded(t *testing.T) {
	proposalID := uuid.New()
	record := succeededRecord(proposalID, uuid.New(), uuid.New())

	records := &mockPaymentRecordRepo{
		getByProposalIDFn: func(_ context.Context, _ uuid.UUID) (*domain.PaymentRecord, error) {
			return record, nil
		},
	}
	svc := newTestService(&mockPaymentInfoRepo{}, records, nil)

	err := svc.MarkPaymentSucceeded(context.Background(), proposalID)

	assert.NoError(t, err)
}

func TestMarkPaymentSucceeded_NotFound(t *testing.T) {
	records := &mockPaymentRecordRepo{}
	svc := newTestService(&mockPaymentInfoRepo{}, records, nil)

	err := svc.MarkPaymentSucceeded(context.Background(), uuid.New())

	assert.Error(t, err)
}

// --- HandlePaymentSucceeded tests ---

func TestHandlePaymentSucceeded_Success(t *testing.T) {
	proposalID := uuid.New()
	record := newPendingRecord(proposalID, uuid.New(), uuid.New())
	record.StripePaymentIntentID = "pi_test"

	records := &mockPaymentRecordRepo{
		getByPaymentIntentFn: func(_ context.Context, piID string) (*domain.PaymentRecord, error) {
			if piID == "pi_test" {
				return record, nil
			}
			return nil, domain.ErrPaymentRecordNotFound
		},
	}
	svc := newTestService(&mockPaymentInfoRepo{}, records, nil)

	gotID, err := svc.HandlePaymentSucceeded(context.Background(), "pi_test")

	require.NoError(t, err)
	assert.Equal(t, proposalID, gotID)
	assert.Equal(t, domain.RecordStatusSucceeded, record.Status)
}

func TestHandlePaymentSucceeded_NotPending(t *testing.T) {
	proposalID := uuid.New()
	record := newPendingRecord(proposalID, uuid.New(), uuid.New())
	record.Status = domain.RecordStatusFailed

	records := &mockPaymentRecordRepo{
		getByPaymentIntentFn: func(_ context.Context, _ string) (*domain.PaymentRecord, error) {
			return record, nil
		},
	}
	svc := newTestService(&mockPaymentInfoRepo{}, records, nil)

	_, err := svc.HandlePaymentSucceeded(context.Background(), "pi_failed")

	assert.Error(t, err)
}

func TestHandlePaymentSucceeded_NotFound(t *testing.T) {
	records := &mockPaymentRecordRepo{}
	svc := newTestService(&mockPaymentInfoRepo{}, records, nil)

	_, err := svc.HandlePaymentSucceeded(context.Background(), "pi_unknown")

	assert.Error(t, err)
}

func TestHandlePaymentSucceeded_AlreadySucceeded(t *testing.T) {
	proposalID := uuid.New()
	record := succeededRecord(proposalID, uuid.New(), uuid.New())

	records := &mockPaymentRecordRepo{
		getByPaymentIntentFn: func(_ context.Context, _ string) (*domain.PaymentRecord, error) {
			return record, nil
		},
	}
	svc := newTestService(&mockPaymentInfoRepo{}, records, nil)

	gotID, err := svc.HandlePaymentSucceeded(context.Background(), "pi_dup")

	require.NoError(t, err)
	assert.Equal(t, proposalID, gotID)
}

// --- GetWalletOverview tests ---

func TestGetWalletOverview_NoPaymentInfo(t *testing.T) {
	svc := newTestService(&mockPaymentInfoRepo{}, &mockPaymentRecordRepo{}, nil)

	wallet, err := svc.GetWalletOverview(context.Background(), uuid.New())

	require.NoError(t, err)
	assert.Empty(t, wallet.StripeAccountID)
}

func TestGetWalletOverview_EmptyRecords(t *testing.T) {
	userID := uuid.New()
	info := &mockPaymentInfoRepo{
		getByUserIDFn: func(_ context.Context, _ uuid.UUID) (*domain.PaymentInfo, error) {
			return &domain.PaymentInfo{
				UserID:          userID,
				StripeAccountID: "acct_test",
			}, nil
		},
	}
	stripe := &mockStripeService{
		getAccountStatusFn: func(_ context.Context, _ string) (bool, error) {
			return true, nil
		},
	}
	svc := newTestService(info, &mockPaymentRecordRepo{}, stripe)

	wallet, err := svc.GetWalletOverview(context.Background(), userID)

	require.NoError(t, err)
	assert.Equal(t, "acct_test", wallet.StripeAccountID)
	assert.True(t, wallet.ChargesEnabled)
	assert.Empty(t, wallet.Records)
	assert.Equal(t, int64(0), wallet.EscrowAmount)
}

func TestGetWalletOverview_WithRecords(t *testing.T) {
	userID := uuid.New()
	proposalA := uuid.New()
	proposalB := uuid.New()

	escrowRec := succeededRecord(proposalA, uuid.New(), userID)
	transferredRec := succeededRecord(proposalB, uuid.New(), userID)
	_ = transferredRec.MarkTransferred("tr_done")

	info := &mockPaymentInfoRepo{
		getByUserIDFn: func(_ context.Context, _ uuid.UUID) (*domain.PaymentInfo, error) {
			return &domain.PaymentInfo{UserID: userID, StripeAccountID: "acct_x"}, nil
		},
	}
	records := &mockPaymentRecordRepo{
		listByProviderIDFn: func(_ context.Context, _ uuid.UUID) ([]*domain.PaymentRecord, error) {
			return []*domain.PaymentRecord{escrowRec, transferredRec}, nil
		},
	}
	stripe := &mockStripeService{
		getAccountStatusFn: func(_ context.Context, _ string) (bool, error) {
			return true, nil
		},
	}
	svc := newTestService(info, records, stripe)

	wallet, err := svc.GetWalletOverview(context.Background(), userID)

	require.NoError(t, err)
	assert.Len(t, wallet.Records, 2)
	assert.Equal(t, escrowRec.ProviderPayout, wallet.EscrowAmount)
	assert.Equal(t, transferredRec.ProviderPayout, wallet.TransferredAmount)
	assert.Equal(t, wallet.EscrowAmount, wallet.AvailableAmount)
}

// --- TransferToProvider tests ---

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
	proposalID := uuid.New()
	record := newPendingRecord(proposalID, uuid.New(), uuid.New())

	records := &mockPaymentRecordRepo{
		getByProposalIDFn: func(_ context.Context, _ uuid.UUID) (*domain.PaymentRecord, error) {
			return record, nil
		},
	}
	svc := newTestService(&mockPaymentInfoRepo{}, records, nil)

	err := svc.TransferToProvider(context.Background(), proposalID)

	assert.ErrorIs(t, err, domain.ErrPaymentNotSucceeded)
}

func TestTransferToProvider_NoStripeAccount(t *testing.T) {
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
			return &domain.PaymentInfo{UserID: providerID, StripeAccountID: ""}, nil
		},
	}
	svc := newTestService(info, records, nil)

	err := svc.TransferToProvider(context.Background(), proposalID)

	assert.ErrorIs(t, err, domain.ErrStripeAccountNotFound)
}

func TestTransferToProvider_AlreadyTransferred(t *testing.T) {
	proposalID := uuid.New()
	record := succeededRecord(proposalID, uuid.New(), uuid.New())
	_ = record.MarkTransferred("tr_done")

	records := &mockPaymentRecordRepo{
		getByProposalIDFn: func(_ context.Context, _ uuid.UUID) (*domain.PaymentRecord, error) {
			return record, nil
		},
	}
	svc := newTestService(&mockPaymentInfoRepo{}, records, nil)

	err := svc.TransferToProvider(context.Background(), proposalID)

	assert.ErrorIs(t, err, domain.ErrTransferAlreadyDone)
}

// --- RequestPayout tests ---

func TestRequestPayout_Success(t *testing.T) {
	userID := uuid.New()
	record := succeededRecord(uuid.New(), uuid.New(), userID)

	info := &mockPaymentInfoRepo{
		getByUserIDFn: func(_ context.Context, _ uuid.UUID) (*domain.PaymentInfo, error) {
			return &domain.PaymentInfo{
				UserID:          userID,
				StripeAccountID: "acct_pay",
			}, nil
		},
	}
	records := &mockPaymentRecordRepo{
		listByProviderIDFn: func(_ context.Context, _ uuid.UUID) ([]*domain.PaymentRecord, error) {
			return []*domain.PaymentRecord{record}, nil
		},
	}
	stripe := &mockStripeService{
		createTransferFn: func(_ context.Context, _ portservice.CreateTransferInput) (string, error) {
			return "tr_payout", nil
		},
	}
	svc := newTestService(info, records, stripe)

	result, err := svc.RequestPayout(context.Background(), userID)

	require.NoError(t, err)
	assert.Equal(t, "transferred", result.Status)
	assert.Equal(t, domain.TransferCompleted, record.TransferStatus)
}

func TestRequestPayout_NoStripeAccount(t *testing.T) {
	userID := uuid.New()
	info := &mockPaymentInfoRepo{
		getByUserIDFn: func(_ context.Context, _ uuid.UUID) (*domain.PaymentInfo, error) {
			return &domain.PaymentInfo{UserID: userID, StripeAccountID: ""}, nil
		},
	}
	svc := newTestService(info, &mockPaymentRecordRepo{}, nil)

	result, err := svc.RequestPayout(context.Background(), userID)

	assert.Nil(t, result)
	assert.ErrorIs(t, err, domain.ErrStripeAccountNotFound)
}

func TestRequestPayout_NothingToTransfer(t *testing.T) {
	userID := uuid.New()
	info := &mockPaymentInfoRepo{
		getByUserIDFn: func(_ context.Context, _ uuid.UUID) (*domain.PaymentInfo, error) {
			return &domain.PaymentInfo{
				UserID:          userID,
				StripeAccountID: "acct_empty",
			}, nil
		},
	}
	records := &mockPaymentRecordRepo{
		listByProviderIDFn: func(_ context.Context, _ uuid.UUID) ([]*domain.PaymentRecord, error) {
			return nil, nil
		},
	}
	svc := newTestService(info, records, nil)

	result, err := svc.RequestPayout(context.Background(), userID)

	require.NoError(t, err)
	assert.Equal(t, "nothing_to_transfer", result.Status)
}

func TestRequestPayout_OnlyAlreadyTransferred(t *testing.T) {
	userID := uuid.New()
	record := succeededRecord(uuid.New(), uuid.New(), userID)
	_ = record.MarkTransferred("tr_done")

	info := &mockPaymentInfoRepo{
		getByUserIDFn: func(_ context.Context, _ uuid.UUID) (*domain.PaymentInfo, error) {
			return &domain.PaymentInfo{
				UserID:          userID,
				StripeAccountID: "acct_done",
			}, nil
		},
	}
	records := &mockPaymentRecordRepo{
		listByProviderIDFn: func(_ context.Context, _ uuid.UUID) ([]*domain.PaymentRecord, error) {
			return []*domain.PaymentRecord{record}, nil
		},
	}
	svc := newTestService(info, records, nil)

	result, err := svc.RequestPayout(context.Background(), userID)

	require.NoError(t, err)
	assert.Equal(t, "nothing_to_transfer", result.Status)
}

// --- GetPaymentRecord tests ---

func TestGetPaymentRecord_Found(t *testing.T) {
	proposalID := uuid.New()
	expected := newPendingRecord(proposalID, uuid.New(), uuid.New())

	records := &mockPaymentRecordRepo{
		getByProposalIDFn: func(_ context.Context, id uuid.UUID) (*domain.PaymentRecord, error) {
			if id == proposalID {
				return expected, nil
			}
			return nil, domain.ErrPaymentRecordNotFound
		},
	}
	svc := newTestService(&mockPaymentInfoRepo{}, records, nil)

	got, err := svc.GetPaymentRecord(context.Background(), proposalID)

	require.NoError(t, err)
	assert.Equal(t, expected.ID, got.ID)
}

func TestGetPaymentRecord_NotFound(t *testing.T) {
	records := &mockPaymentRecordRepo{}
	svc := newTestService(&mockPaymentInfoRepo{}, records, nil)

	got, err := svc.GetPaymentRecord(context.Background(), uuid.New())

	assert.NoError(t, err)
	assert.Nil(t, got)
}

// --- VerifyWebhook tests ---

func TestVerifyWebhook_StripeNotConfigured(t *testing.T) {
	svc := NewService(
		&mockPaymentInfoRepo{},
		&mockPaymentRecordRepo{},
		&mockIdentityDocRepo{},
		&mockBusinessPersonRepo{},
		nil, // stripe = nil
		&mockStorageService{},
	)

	_, err := svc.VerifyWebhook([]byte("body"), "sig")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "stripe not configured")
}

func TestStripeConfigured_True(t *testing.T) {
	svc := newTestService(&mockPaymentInfoRepo{}, &mockPaymentRecordRepo{}, &mockStripeService{})
	assert.True(t, svc.StripeConfigured())
}

func TestStripeConfigured_False(t *testing.T) {
	svc := NewService(
		&mockPaymentInfoRepo{},
		&mockPaymentRecordRepo{},
		&mockIdentityDocRepo{},
		&mockBusinessPersonRepo{},
		nil,
		&mockStorageService{},
	)
	assert.False(t, svc.StripeConfigured())
}

