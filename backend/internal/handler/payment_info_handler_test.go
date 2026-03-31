package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	paymentapp "marketplace-backend/internal/app/payment"
	"marketplace-backend/internal/domain/payment"
	"marketplace-backend/internal/handler/middleware"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/port/service"
)

// --- payment-specific mocks ---

type mockPaymentInfoRepo struct {
	getByUserIDFn          func(ctx context.Context, userID uuid.UUID) (*payment.PaymentInfo, error)
	upsertFn               func(ctx context.Context, info *payment.PaymentInfo) error
	updateStripeFieldsFn   func(ctx context.Context, userID uuid.UUID, stripeAccountID string, stripeVerified bool) error
	getByStripeAccountIDFn func(ctx context.Context, stripeAccountID string) (*payment.PaymentInfo, error)
}

func (m *mockPaymentInfoRepo) GetByUserID(ctx context.Context, userID uuid.UUID) (*payment.PaymentInfo, error) {
	if m.getByUserIDFn != nil {
		return m.getByUserIDFn(ctx, userID)
	}
	return nil, payment.ErrNotFound
}

func (m *mockPaymentInfoRepo) Upsert(ctx context.Context, info *payment.PaymentInfo) error {
	if m.upsertFn != nil {
		return m.upsertFn(ctx, info)
	}
	return nil
}

func (m *mockPaymentInfoRepo) UpdateStripeFields(ctx context.Context, userID uuid.UUID, stripeAccountID string, stripeVerified bool) error {
	if m.updateStripeFieldsFn != nil {
		return m.updateStripeFieldsFn(ctx, userID, stripeAccountID, stripeVerified)
	}
	return nil
}

func (m *mockPaymentInfoRepo) GetByStripeAccountID(ctx context.Context, stripeAccountID string) (*payment.PaymentInfo, error) {
	if m.getByStripeAccountIDFn != nil {
		return m.getByStripeAccountIDFn(ctx, stripeAccountID)
	}
	return nil, payment.ErrNotFound
}

type mockPaymentRecordRepo struct {
	createFn               func(ctx context.Context, record *payment.PaymentRecord) error
	getByProposalIDFn      func(ctx context.Context, proposalID uuid.UUID) (*payment.PaymentRecord, error)
	getByPaymentIntentIDFn func(ctx context.Context, paymentIntentID string) (*payment.PaymentRecord, error)
	listByProviderIDFn     func(ctx context.Context, providerID uuid.UUID) ([]*payment.PaymentRecord, error)
	updateFn               func(ctx context.Context, record *payment.PaymentRecord) error
}

func (m *mockPaymentRecordRepo) Create(ctx context.Context, record *payment.PaymentRecord) error {
	if m.createFn != nil {
		return m.createFn(ctx, record)
	}
	return nil
}

func (m *mockPaymentRecordRepo) GetByProposalID(ctx context.Context, proposalID uuid.UUID) (*payment.PaymentRecord, error) {
	if m.getByProposalIDFn != nil {
		return m.getByProposalIDFn(ctx, proposalID)
	}
	return nil, payment.ErrPaymentRecordNotFound
}

func (m *mockPaymentRecordRepo) GetByPaymentIntentID(ctx context.Context, paymentIntentID string) (*payment.PaymentRecord, error) {
	if m.getByPaymentIntentIDFn != nil {
		return m.getByPaymentIntentIDFn(ctx, paymentIntentID)
	}
	return nil, payment.ErrPaymentRecordNotFound
}

func (m *mockPaymentRecordRepo) ListByProviderID(ctx context.Context, providerID uuid.UUID) ([]*payment.PaymentRecord, error) {
	if m.listByProviderIDFn != nil {
		return m.listByProviderIDFn(ctx, providerID)
	}
	return []*payment.PaymentRecord{}, nil
}

func (m *mockPaymentRecordRepo) Update(ctx context.Context, record *payment.PaymentRecord) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, record)
	}
	return nil
}

type mockIdentityDocRepo struct{}

func (m *mockIdentityDocRepo) Create(_ context.Context, _ *payment.IdentityDocument) error { return nil }
func (m *mockIdentityDocRepo) GetByID(_ context.Context, _ uuid.UUID) (*payment.IdentityDocument, error) {
	return nil, payment.ErrDocumentNotFound
}
func (m *mockIdentityDocRepo) ListByUserID(_ context.Context, _ uuid.UUID) ([]*payment.IdentityDocument, error) {
	return nil, nil
}
func (m *mockIdentityDocRepo) Delete(_ context.Context, _ uuid.UUID) error            { return nil }
func (m *mockIdentityDocRepo) UpdateStatus(_ context.Context, _ uuid.UUID, _, _ string) error {
	return nil
}
func (m *mockIdentityDocRepo) UpdateStripeFileID(_ context.Context, _ uuid.UUID, _ string) error {
	return nil
}
func (m *mockIdentityDocRepo) GetByUserAndTypeSide(_ context.Context, _ uuid.UUID, _, _, _ string) (*payment.IdentityDocument, error) {
	return nil, payment.ErrDocumentNotFound
}

type mockBusinessPersonRepo struct{}

func (m *mockBusinessPersonRepo) Create(_ context.Context, _ *payment.BusinessPerson) error {
	return nil
}
func (m *mockBusinessPersonRepo) ListByUserID(_ context.Context, _ uuid.UUID) ([]*payment.BusinessPerson, error) {
	return nil, nil
}
func (m *mockBusinessPersonRepo) DeleteByUserID(_ context.Context, _ uuid.UUID) error { return nil }

type mockStripeService struct{}

func (m *mockStripeService) CreateConnectedAccount(_ context.Context, _ *payment.PaymentInfo, _, _ string) (string, error) {
	return "acct_test", nil
}
func (m *mockStripeService) GetAccountStatus(_ context.Context, _ string) (bool, error) {
	return true, nil
}
func (m *mockStripeService) CreatePaymentIntent(_ context.Context, _ service.CreatePaymentIntentInput) (*service.PaymentIntentResult, error) {
	return &service.PaymentIntentResult{}, nil
}
func (m *mockStripeService) CreateTransfer(_ context.Context, _ service.CreateTransferInput) (string, error) {
	return "tr_test", nil
}
func (m *mockStripeService) ConstructWebhookEvent(_ []byte, _ string) (*service.StripeWebhookEvent, error) {
	return nil, nil
}
func (m *mockStripeService) GetIdentityVerificationStatus(_ context.Context, _ string) (string, string, error) {
	return "verified", "", nil
}
func (m *mockStripeService) UploadIdentityFile(_ context.Context, _ string, _ io.Reader, _ string) (string, error) {
	return "file_test", nil
}
func (m *mockStripeService) UpdateAccountVerification(_ context.Context, _, _, _ string) error {
	return nil
}
func (m *mockStripeService) CreatePerson(_ context.Context, _ string, _ service.CreatePersonInput) (string, error) {
	return "person_test", nil
}
func (m *mockStripeService) UpdateCompanyFlags(_ context.Context, _ string, _, _, _ bool) error {
	return nil
}

var _ repository.PaymentInfoRepository = (*mockPaymentInfoRepo)(nil)
var _ repository.PaymentRecordRepository = (*mockPaymentRecordRepo)(nil)
var _ repository.IdentityDocumentRepository = (*mockIdentityDocRepo)(nil)
var _ repository.BusinessPersonRepository = (*mockBusinessPersonRepo)(nil)

func newTestPaymentService(infoRepo *mockPaymentInfoRepo, recordRepo *mockPaymentRecordRepo) *paymentapp.Service {
	return paymentapp.NewService(paymentapp.ServiceDeps{
		Payments:  infoRepo,
		Records:   recordRepo,
		Documents: &mockIdentityDocRepo{},
		Persons:   &mockBusinessPersonRepo{},
	})
}

func testPaymentInfo(userID uuid.UUID) *payment.PaymentInfo {
	return &payment.PaymentInfo{
		ID: uuid.New(), UserID: userID,
		FirstName: "John", LastName: "Doe",
		DateOfBirth: time.Date(1990, 1, 15, 0, 0, 0, 0, time.UTC),
		Nationality: "FR", Address: "1 Rue Test", City: "Paris", PostalCode: "75001",
		AccountHolder: "John Doe", IBAN: "FR7630006000011234567890189",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
}

func TestPaymentInfoHandler_GetPaymentInfo(t *testing.T) {
	uid := uuid.New()

	tests := []struct {
		name       string
		userID     *uuid.UUID
		setupMock  func(*mockPaymentInfoRepo)
		wantStatus int
	}{
		{
			name: "success", userID: &uid,
			setupMock: func(r *mockPaymentInfoRepo) {
				r.getByUserIDFn = func(_ context.Context, _ uuid.UUID) (*payment.PaymentInfo, error) {
					return testPaymentInfo(uid), nil
				}
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "not configured returns null", userID: &uid,
			wantStatus: http.StatusOK,
		},
		{
			name: "unauthenticated", userID: nil,
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			infoRepo := &mockPaymentInfoRepo{}
			if tc.setupMock != nil {
				tc.setupMock(infoRepo)
			}
			svc := newTestPaymentService(infoRepo, &mockPaymentRecordRepo{})
			h := NewPaymentInfoHandler(svc)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/payment-info", nil)
			if tc.userID != nil {
				ctx := context.WithValue(req.Context(), middleware.ContextKeyUserID, *tc.userID)
				req = req.WithContext(ctx)
			}
			rec := httptest.NewRecorder()
			h.GetPaymentInfo(rec, req)
			assert.Equal(t, tc.wantStatus, rec.Code)
		})
	}
}

func TestPaymentInfoHandler_SavePaymentInfo(t *testing.T) {
	uid := uuid.New()

	validBody := map[string]any{
		"first_name": "John", "last_name": "Doe", "date_of_birth": "1990-01-15",
		"nationality": "FR", "address": "1 Rue Test", "city": "Paris",
		"postal_code": "75001", "account_holder": "John Doe",
		"iban": "FR7630006000011234567890189", "email": "john@test.com",
	}

	tests := []struct {
		name       string
		userID     *uuid.UUID
		body       map[string]any
		setupMock  func(*mockPaymentInfoRepo)
		wantStatus int
	}{
		{
			name: "success", userID: &uid, body: validBody,
			setupMock: func(r *mockPaymentInfoRepo) {
				r.getByUserIDFn = func(_ context.Context, _ uuid.UUID) (*payment.PaymentInfo, error) {
					return nil, payment.ErrNotFound
				}
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "unauthenticated", userID: nil, body: validBody,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "invalid date format", userID: &uid,
			body:       map[string]any{"first_name": "J", "date_of_birth": "not-a-date"},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			infoRepo := &mockPaymentInfoRepo{}
			if tc.setupMock != nil {
				tc.setupMock(infoRepo)
			}
			svc := newTestPaymentService(infoRepo, &mockPaymentRecordRepo{})
			h := NewPaymentInfoHandler(svc)

			body, _ := json.Marshal(tc.body)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/payment-info", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			if tc.userID != nil {
				ctx := context.WithValue(req.Context(), middleware.ContextKeyUserID, *tc.userID)
				req = req.WithContext(ctx)
			}
			rec := httptest.NewRecorder()
			h.SavePaymentInfo(rec, req)
			assert.Equal(t, tc.wantStatus, rec.Code)
		})
	}
}

func TestPaymentInfoHandler_GetPaymentInfoStatus(t *testing.T) {
	uid := uuid.New()

	tests := []struct {
		name       string
		userID     *uuid.UUID
		setupMock  func(*mockPaymentInfoRepo)
		wantStatus int
		wantBody   string
	}{
		{
			name: "complete", userID: &uid,
			setupMock: func(r *mockPaymentInfoRepo) {
				r.getByUserIDFn = func(_ context.Context, _ uuid.UUID) (*payment.PaymentInfo, error) {
					info := testPaymentInfo(uid)
					return info, nil
				}
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "not configured returns incomplete", userID: &uid,
			wantStatus: http.StatusOK,
		},
		{
			name: "unauthenticated", userID: nil,
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			infoRepo := &mockPaymentInfoRepo{}
			if tc.setupMock != nil {
				tc.setupMock(infoRepo)
			}
			svc := newTestPaymentService(infoRepo, &mockPaymentRecordRepo{})
			h := NewPaymentInfoHandler(svc)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/payment-info/status", nil)
			if tc.userID != nil {
				ctx := context.WithValue(req.Context(), middleware.ContextKeyUserID, *tc.userID)
				req = req.WithContext(ctx)
			}
			rec := httptest.NewRecorder()
			h.GetPaymentInfoStatus(rec, req)
			assert.Equal(t, tc.wantStatus, rec.Code)

			if tc.wantStatus == http.StatusOK {
				var resp map[string]any
				require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
				_, exists := resp["complete"]
				assert.True(t, exists, "response should contain 'complete' field")
			}
		})
	}
}
