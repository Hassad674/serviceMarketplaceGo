package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	paymentapp "marketplace-backend/internal/app/payment"
	"marketplace-backend/internal/domain/payment"
	proposalapp "marketplace-backend/internal/app/proposal"
	"marketplace-backend/internal/handler/middleware"
)

func newTestWalletHandler(
	infoRepo *mockPaymentInfoRepo,
	recordRepo *mockPaymentRecordRepo,
) *WalletHandler {
	paymentSvc := paymentapp.NewService(
		infoRepo, recordRepo, &mockIdentityDocRepo{}, &mockBusinessPersonRepo{},
		nil, nil,
	)
	proposalSvc := proposalapp.NewService(proposalapp.ServiceDeps{
		Proposals: &mockProposalRepo{},
		Users:     &mockUserRepo{},
	})
	return NewWalletHandler(paymentSvc, proposalSvc)
}

func TestWalletHandler_GetWallet(t *testing.T) {
	uid := uuid.New()

	tests := []struct {
		name       string
		userID     *uuid.UUID
		setupMock  func(*mockPaymentRecordRepo)
		wantStatus int
	}{
		{
			name: "success empty wallet", userID: &uid,
			wantStatus: http.StatusOK,
		},
		{
			name: "unauthenticated", userID: nil,
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			recordRepo := &mockPaymentRecordRepo{}
			if tc.setupMock != nil {
				tc.setupMock(recordRepo)
			}
			h := newTestWalletHandler(&mockPaymentInfoRepo{}, recordRepo)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/wallet", nil)
			if tc.userID != nil {
				ctx := context.WithValue(req.Context(), middleware.ContextKeyUserID, *tc.userID)
				req = req.WithContext(ctx)
			}
			rec := httptest.NewRecorder()
			h.GetWallet(rec, req)
			assert.Equal(t, tc.wantStatus, rec.Code)

			if tc.wantStatus == http.StatusOK {
				var resp map[string]any
				require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
				assert.NotNil(t, resp)
			}
		})
	}
}

func TestWalletHandler_RequestPayout(t *testing.T) {
	uid := uuid.New()

	tests := []struct {
		name       string
		userID     *uuid.UUID
		setupMocks func(*mockPaymentInfoRepo)
		wantStatus int
	}{
		{
			name: "no stripe account returns error", userID: &uid,
			setupMocks: func(r *mockPaymentInfoRepo) {
				r.getByUserIDFn = func(_ context.Context, _ uuid.UUID) (*payment.PaymentInfo, error) {
					return testPaymentInfo(uid), nil // no StripeAccountID
				}
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name: "unauthenticated", userID: nil,
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			infoRepo := &mockPaymentInfoRepo{}
			if tc.setupMocks != nil {
				tc.setupMocks(infoRepo)
			}
			h := newTestWalletHandler(infoRepo, &mockPaymentRecordRepo{})

			req := httptest.NewRequest(http.MethodPost, "/api/v1/wallet/payout", nil)
			if tc.userID != nil {
				ctx := context.WithValue(req.Context(), middleware.ContextKeyUserID, *tc.userID)
				req = req.WithContext(ctx)
			}
			rec := httptest.NewRecorder()
			h.RequestPayout(rec, req)
			assert.Equal(t, tc.wantStatus, rec.Code)
		})
	}
}
