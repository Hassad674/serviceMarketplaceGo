package payment

import (
	"context"
	"time"

	"github.com/google/uuid"

	domain "marketplace-backend/internal/domain/payment"
	"marketplace-backend/internal/port/repository"
	portservice "marketplace-backend/internal/port/service"
)

// --- PaymentInfoRepository mock ---

type mockPaymentInfoRepo struct {
	getByUserIDFn            func(ctx context.Context, userID uuid.UUID) (*domain.PaymentInfo, error)
	upsertFn                 func(ctx context.Context, info *domain.PaymentInfo) error
	updateStripeFieldsFn     func(ctx context.Context, userID uuid.UUID, accountID string, verified bool) error
	updateStripeSyncFieldsFn func(ctx context.Context, userID uuid.UUID, input repository.StripeSyncInput) error
	getByStripeAccountFn     func(ctx context.Context, accountID string) (*domain.PaymentInfo, error)
}

func (m *mockPaymentInfoRepo) GetByUserID(ctx context.Context, userID uuid.UUID) (*domain.PaymentInfo, error) {
	if m.getByUserIDFn != nil {
		return m.getByUserIDFn(ctx, userID)
	}
	return nil, domain.ErrNotFound
}

func (m *mockPaymentInfoRepo) Upsert(ctx context.Context, info *domain.PaymentInfo) error {
	if m.upsertFn != nil {
		return m.upsertFn(ctx, info)
	}
	return nil
}

func (m *mockPaymentInfoRepo) UpdateStripeFields(ctx context.Context, userID uuid.UUID, accountID string, verified bool) error {
	if m.updateStripeFieldsFn != nil {
		return m.updateStripeFieldsFn(ctx, userID, accountID, verified)
	}
	return nil
}

func (m *mockPaymentInfoRepo) UpdateStripeSyncFields(ctx context.Context, userID uuid.UUID, input repository.StripeSyncInput) error {
	if m.updateStripeSyncFieldsFn != nil {
		return m.updateStripeSyncFieldsFn(ctx, userID, input)
	}
	return nil
}

func (m *mockPaymentInfoRepo) GetByStripeAccountID(ctx context.Context, accountID string) (*domain.PaymentInfo, error) {
	if m.getByStripeAccountFn != nil {
		return m.getByStripeAccountFn(ctx, accountID)
	}
	return nil, domain.ErrNotFound
}

// --- PaymentRecordRepository mock ---

type mockPaymentRecordRepo struct {
	createFn             func(ctx context.Context, record *domain.PaymentRecord) error
	getByProposalIDFn    func(ctx context.Context, proposalID uuid.UUID) (*domain.PaymentRecord, error)
	getByPaymentIntentFn func(ctx context.Context, piID string) (*domain.PaymentRecord, error)
	listByProviderIDFn   func(ctx context.Context, providerID uuid.UUID) ([]*domain.PaymentRecord, error)
	updateFn             func(ctx context.Context, record *domain.PaymentRecord) error
}

func (m *mockPaymentRecordRepo) Create(ctx context.Context, record *domain.PaymentRecord) error {
	if m.createFn != nil {
		return m.createFn(ctx, record)
	}
	return nil
}

func (m *mockPaymentRecordRepo) GetByProposalID(ctx context.Context, proposalID uuid.UUID) (*domain.PaymentRecord, error) {
	if m.getByProposalIDFn != nil {
		return m.getByProposalIDFn(ctx, proposalID)
	}
	return nil, domain.ErrPaymentRecordNotFound
}

func (m *mockPaymentRecordRepo) GetByPaymentIntentID(ctx context.Context, piID string) (*domain.PaymentRecord, error) {
	if m.getByPaymentIntentFn != nil {
		return m.getByPaymentIntentFn(ctx, piID)
	}
	return nil, domain.ErrPaymentRecordNotFound
}

func (m *mockPaymentRecordRepo) ListByProviderID(ctx context.Context, providerID uuid.UUID) ([]*domain.PaymentRecord, error) {
	if m.listByProviderIDFn != nil {
		return m.listByProviderIDFn(ctx, providerID)
	}
	return nil, nil
}

func (m *mockPaymentRecordRepo) Update(ctx context.Context, record *domain.PaymentRecord) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, record)
	}
	return nil
}

// --- StripeService mock ---

type mockStripeService struct {
	createPaymentIntentFn  func(ctx context.Context, input portservice.CreatePaymentIntentInput) (*portservice.PaymentIntentResult, error)
	createTransferFn       func(ctx context.Context, input portservice.CreateTransferInput) (string, error)
	getAccountStatusFn     func(ctx context.Context, accountID string) (bool, error)
	createMinimalAccountFn func(ctx context.Context, country, email string) (string, error)
	createAccountSessionFn func(ctx context.Context, accountID string) (string, error)
	getFullAccountFn       func(ctx context.Context, accountID string) (*portservice.StripeAccountInfo, error)
}

func (m *mockStripeService) CreatePaymentIntent(ctx context.Context, input portservice.CreatePaymentIntentInput) (*portservice.PaymentIntentResult, error) {
	if m.createPaymentIntentFn != nil {
		return m.createPaymentIntentFn(ctx, input)
	}
	return &portservice.PaymentIntentResult{}, nil
}

func (m *mockStripeService) CreateTransfer(ctx context.Context, input portservice.CreateTransferInput) (string, error) {
	if m.createTransferFn != nil {
		return m.createTransferFn(ctx, input)
	}
	return "tr_mock", nil
}

func (m *mockStripeService) GetAccountStatus(ctx context.Context, accountID string) (bool, error) {
	if m.getAccountStatusFn != nil {
		return m.getAccountStatusFn(ctx, accountID)
	}
	return false, nil
}

func (m *mockStripeService) ConstructWebhookEvent(_ []byte, _ string) (*portservice.StripeWebhookEvent, error) {
	return nil, nil
}

func (m *mockStripeService) CreateMinimalAccount(ctx context.Context, country, email string) (string, error) {
	if m.createMinimalAccountFn != nil {
		return m.createMinimalAccountFn(ctx, country, email)
	}
	return "acct_minimal_mock", nil
}

func (m *mockStripeService) CreateAccountSession(ctx context.Context, accountID string) (string, error) {
	if m.createAccountSessionFn != nil {
		return m.createAccountSessionFn(ctx, accountID)
	}
	return "cas_mock_secret", nil
}

func (m *mockStripeService) GetFullAccount(ctx context.Context, accountID string) (*portservice.StripeAccountInfo, error) {
	if m.getFullAccountFn != nil {
		return m.getFullAccountFn(ctx, accountID)
	}
	return &portservice.StripeAccountInfo{}, nil
}

// --- StorageService mock ---

type mockStorageService struct {
	getPublicURLFn func(key string) string
}

func (m *mockStorageService) GetPublicURL(key string) string {
	if m.getPublicURLFn != nil {
		return m.getPublicURLFn(key)
	}
	return "https://storage.example.com/" + key
}

func (m *mockStorageService) GetPresignedUploadURL(_ context.Context, _ string, _ string, _ time.Duration) (string, error) {
	return "", nil
}

// --- NotificationSender mock ---

type mockNotificationSender struct{}

func (m *mockNotificationSender) Send(_ context.Context, _ portservice.NotificationInput) error {
	return nil
}
