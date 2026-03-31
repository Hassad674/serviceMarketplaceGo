package payment

import (
	"context"
	"io"
	"time"

	"github.com/google/uuid"

	domain "marketplace-backend/internal/domain/payment"
	portservice "marketplace-backend/internal/port/service"
)

// --- PaymentInfoRepository mock ---

type mockPaymentInfoRepo struct {
	getByUserIDFn        func(ctx context.Context, userID uuid.UUID) (*domain.PaymentInfo, error)
	upsertFn             func(ctx context.Context, info *domain.PaymentInfo) error
	updateStripeFieldsFn func(ctx context.Context, userID uuid.UUID, accountID string, verified bool) error
	getByStripeAccountFn func(ctx context.Context, accountID string) (*domain.PaymentInfo, error)
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
	createPaymentIntentFn      func(ctx context.Context, input portservice.CreatePaymentIntentInput) (*portservice.PaymentIntentResult, error)
	createTransferFn           func(ctx context.Context, input portservice.CreateTransferInput) (string, error)
	getAccountStatusFn         func(ctx context.Context, accountID string) (bool, error)
	uploadIdentityFileFn       func(ctx context.Context, filename string, data io.Reader, purpose string) (string, error)
	updateAccountVerificationFn func(ctx context.Context, accountID, frontID, backID string) error
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

func (m *mockStripeService) CreateConnectedAccount(_ context.Context, _ *domain.PaymentInfo, _, _ string) (string, error) {
	return "", nil
}

func (m *mockStripeService) ConstructWebhookEvent(_ []byte, _ string) (*portservice.StripeWebhookEvent, error) {
	return nil, nil
}

func (m *mockStripeService) GetIdentityVerificationStatus(_ context.Context, _ string) (string, string, error) {
	return "", "", nil
}

func (m *mockStripeService) UploadIdentityFile(ctx context.Context, filename string, data io.Reader, purpose string) (string, error) {
	if m.uploadIdentityFileFn != nil {
		return m.uploadIdentityFileFn(ctx, filename, data, purpose)
	}
	return "", nil
}

func (m *mockStripeService) UpdateAccountVerification(ctx context.Context, accountID, frontID, backID string) error {
	if m.updateAccountVerificationFn != nil {
		return m.updateAccountVerificationFn(ctx, accountID, frontID, backID)
	}
	return nil
}

func (m *mockStripeService) CreatePerson(_ context.Context, _ string, _ portservice.CreatePersonInput) (string, error) {
	return "", nil
}

func (m *mockStripeService) UpdateCompanyFlags(_ context.Context, _ string, _, _, _ bool) error {
	return nil
}

func (m *mockStripeService) GetAccountRequirements(_ context.Context, _ string) ([]string, error) {
	return nil, nil
}

func (m *mockStripeService) CreateAccountLink(_ context.Context, _, _, _ string) (string, error) {
	return "", nil
}

func (m *mockStripeService) GetCountrySpec(_ context.Context, _ string) (*domain.CountryFieldSpec, error) {
	return nil, nil
}

func (m *mockStripeService) ListAllCountrySpecs(_ context.Context) ([]*domain.CountryFieldSpec, error) {
	return nil, nil
}

// --- IdentityDocumentRepository mock ---

type mockIdentityDocRepo struct {
	createFn              func(ctx context.Context, doc *domain.IdentityDocument) error
	getByIDFn             func(ctx context.Context, id uuid.UUID) (*domain.IdentityDocument, error)
	listByUserIDFn        func(ctx context.Context, userID uuid.UUID) ([]*domain.IdentityDocument, error)
	deleteFn              func(ctx context.Context, id uuid.UUID) error
	getByUserAndTypeSideFn func(ctx context.Context, userID uuid.UUID, cat, docType, side string) (*domain.IdentityDocument, error)
}

func (m *mockIdentityDocRepo) Create(ctx context.Context, doc *domain.IdentityDocument) error {
	if m.createFn != nil {
		return m.createFn(ctx, doc)
	}
	return nil
}

func (m *mockIdentityDocRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.IdentityDocument, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, domain.ErrDocumentNotFound
}

func (m *mockIdentityDocRepo) ListByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.IdentityDocument, error) {
	if m.listByUserIDFn != nil {
		return m.listByUserIDFn(ctx, userID)
	}
	return nil, nil
}

func (m *mockIdentityDocRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}

func (m *mockIdentityDocRepo) UpdateStatus(_ context.Context, _ uuid.UUID, _, _ string) error {
	return nil
}

func (m *mockIdentityDocRepo) UpdateStripeFileID(_ context.Context, _ uuid.UUID, _ string) error {
	return nil
}

func (m *mockIdentityDocRepo) GetByUserAndTypeSide(ctx context.Context, userID uuid.UUID, cat, docType, side string) (*domain.IdentityDocument, error) {
	if m.getByUserAndTypeSideFn != nil {
		return m.getByUserAndTypeSideFn(ctx, userID, cat, docType, side)
	}
	return nil, domain.ErrDocumentNotFound
}

// --- BusinessPersonRepository mock ---

type mockBusinessPersonRepo struct{}

func (m *mockBusinessPersonRepo) Create(_ context.Context, _ *domain.BusinessPerson) error {
	return nil
}

func (m *mockBusinessPersonRepo) ListByUserID(_ context.Context, _ uuid.UUID) ([]*domain.BusinessPerson, error) {
	return nil, nil
}

func (m *mockBusinessPersonRepo) DeleteByUserID(_ context.Context, _ uuid.UUID) error { return nil }

// --- StorageService mock ---

type mockStorageService struct {
	uploadFn      func(ctx context.Context, key string, data io.Reader, contentType string, size int64) (string, error)
	deleteFn      func(ctx context.Context, key string) error
	getPublicURLFn func(key string) string
}

func (m *mockStorageService) Upload(ctx context.Context, key string, data io.Reader, contentType string, size int64) (string, error) {
	if m.uploadFn != nil {
		return m.uploadFn(ctx, key, data, contentType, size)
	}
	return key, nil
}

func (m *mockStorageService) Delete(ctx context.Context, key string) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, key)
	}
	return nil
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
