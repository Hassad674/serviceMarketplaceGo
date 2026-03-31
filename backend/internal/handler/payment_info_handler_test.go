package handler

import (
	"bytes"
	"context"
	"encoding/json"
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
	getByUserIDFn            func(ctx context.Context, userID uuid.UUID) (*payment.PaymentInfo, error)
	upsertFn                 func(ctx context.Context, info *payment.PaymentInfo) error
	updateStripeFieldsFn     func(ctx context.Context, userID uuid.UUID, stripeAccountID string, stripeVerified bool) error
	updateStripeSyncFieldsFn func(ctx context.Context, userID uuid.UUID, input repository.StripeSyncInput) error
	getByStripeAccountIDFn   func(ctx context.Context, stripeAccountID string) (*payment.PaymentInfo, error)
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

func (m *mockPaymentInfoRepo) UpdateStripeSyncFields(ctx context.Context, userID uuid.UUID, input repository.StripeSyncInput) error {
	if m.updateStripeSyncFieldsFn != nil {
		return m.updateStripeSyncFieldsFn(ctx, userID, input)
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

type mockStripeService struct{}

func (m *mockStripeService) CreateMinimalAccount(_ context.Context, _, _ string) (string, error) {
	return "acct_test", nil
}
func (m *mockStripeService) CreateAccountSession(_ context.Context, _ string) (string, error) {
	return "cas_test_secret", nil
}
func (m *mockStripeService) GetAccountStatus(_ context.Context, _ string) (bool, error) {
	return true, nil
}
func (m *mockStripeService) GetFullAccount(_ context.Context, _ string) (*service.StripeAccountInfo, error) {
	return &service.StripeAccountInfo{}, nil
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

var _ repository.PaymentInfoRepository = (*mockPaymentInfoRepo)(nil)
var _ repository.PaymentRecordRepository = (*mockPaymentRecordRepo)(nil)

func newTestPaymentService(infoRepo *mockPaymentInfoRepo, recordRepo *mockPaymentRecordRepo) *paymentapp.Service {
	return paymentapp.NewService(infoRepo, recordRepo, nil, nil, "")
}

func testPaymentInfo(userID uuid.UUID) *payment.PaymentInfo {
	return &payment.PaymentInfo{
		ID:              uuid.New(),
		UserID:          userID,
		StripeAccountID: "acct_test",
		StripeVerified:  true,
		ChargesEnabled:  true,
		PayoutsEnabled:  true,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
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
			name:       "not configured returns null",
			userID:     &uid,
			wantStatus: http.StatusOK,
		},
		{
			name:       "unauthenticated",
			userID:     nil,
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

func TestPaymentInfoHandler_GetPaymentInfoStatus(t *testing.T) {
	uid := uuid.New()

	tests := []struct {
		name       string
		userID     *uuid.UUID
		setupMock  func(*mockPaymentInfoRepo)
		wantStatus int
	}{
		{
			name: "verified", userID: &uid,
			setupMock: func(r *mockPaymentInfoRepo) {
				r.getByUserIDFn = func(_ context.Context, _ uuid.UUID) (*payment.PaymentInfo, error) {
					info := testPaymentInfo(uid)
					return info, nil
				}
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "not configured returns incomplete",
			userID:     &uid,
			wantStatus: http.StatusOK,
		},
		{
			name:       "unauthenticated",
			userID:     nil,
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
				var resp map[string]interface{}
				require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
				_, exists := resp["complete"]
				assert.True(t, exists, "response should contain 'complete' field")
			}
		})
	}
}

func TestPaymentInfoHandler_CreateAccountSession(t *testing.T) {
	uid := uuid.New()

	t.Run("unauthenticated", func(t *testing.T) {
		svc := newTestPaymentService(&mockPaymentInfoRepo{}, &mockPaymentRecordRepo{})
		h := NewPaymentInfoHandler(svc)

		body, _ := json.Marshal(map[string]string{"email": "test@example.com"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/payment-info/account-session", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		h.CreateAccountSession(rec, req)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("stripe not configured returns error", func(t *testing.T) {
		svc := newTestPaymentService(&mockPaymentInfoRepo{}, &mockPaymentRecordRepo{})
		h := NewPaymentInfoHandler(svc)

		body, _ := json.Marshal(map[string]string{"email": "test@example.com"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/payment-info/account-session", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		ctx := context.WithValue(req.Context(), middleware.ContextKeyUserID, uid)
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()
		h.CreateAccountSession(rec, req)
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}
