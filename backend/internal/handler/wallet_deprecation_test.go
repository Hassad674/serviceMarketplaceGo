package handler_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/handler"
	"marketplace-backend/internal/handler/middleware"
	portservice "marketplace-backend/internal/port/service"
)

// TestRetryCommissionDeprecated_HeadersPresent verifies the
// Deprecation + Sunset + Link headers are emitted on the legacy
// /wallet/commissions/{id}/retry endpoint per the Run B back-compat
// contract.
func TestRetryCommissionDeprecated_HeadersPresent(t *testing.T) {
	retrier := &fakeCommissionRetrier{
		outcome: portservice.ReferralCommissionRetryOutcome{
			Result:        portservice.ReferralCommissionRetryPaid,
			StripeAccount: "acct_test",
		},
	}
	wh := handler.NewWalletHandler(nil, nil).WithCommissionRetrier(retrier)

	req := httptest.NewRequest(http.MethodPost,
		"/api/v1/wallet/commissions/"+uuid.New().String()+"/retry", nil)
	ctx := req.Context()
	ctx = context.WithValue(ctx, middleware.ContextKeyUserID, uuid.New())
	ctx = context.WithValue(ctx, middleware.ContextKeyOrganizationID, uuid.New())
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", uuid.New().String())
	ctx = context.WithValue(ctx, chi.RouteCtxKey, rctx)
	rec := httptest.NewRecorder()
	wh.RetryCommissionDeprecated(rec, req.WithContext(ctx))

	assert.Equal(t, "true", rec.Header().Get("Deprecation"),
		"Deprecation header must be `true`")
	sunset := rec.Header().Get("Sunset")
	require.NotEmpty(t, sunset, "Sunset header must be present")
	// Parse the sunset date and verify it's roughly 30 days in the
	// future (allow ±1 day for clock skew + test runtime).
	parsed, err := time.Parse(http.TimeFormat, sunset)
	require.NoError(t, err)
	delta := parsed.Sub(time.Now().UTC())
	assert.Greater(t, delta, 29*24*time.Hour, "Sunset must be ~30 days out")
	assert.Less(t, delta, 31*24*time.Hour, "Sunset must be ~30 days out, not further")
	assert.Contains(t, rec.Header().Get("Link"), "/api/v1/wallet/withdraw",
		"Link header must point to the successor endpoint")
}

// TestRetryCommissionDeprecated_DelegatesToRetry verifies the
// deprecation wrapper actually delegates to RetryCommission rather
// than skipping it — the existing logic must keep working through
// the deprecation window.
func TestRetryCommissionDeprecated_DelegatesToRetry(t *testing.T) {
	retrier := &fakeCommissionRetrier{
		outcome: portservice.ReferralCommissionRetryOutcome{
			Result:        portservice.ReferralCommissionRetryPaid,
			StripeAccount: "acct_test",
		},
	}
	wh := handler.NewWalletHandler(nil, nil).WithCommissionRetrier(retrier)

	commissionID := uuid.New().String()
	req := httptest.NewRequest(http.MethodPost,
		"/api/v1/wallet/commissions/"+commissionID+"/retry", nil)
	ctx := req.Context()
	ctx = context.WithValue(ctx, middleware.ContextKeyUserID, uuid.New())
	ctx = context.WithValue(ctx, middleware.ContextKeyOrganizationID, uuid.New())
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", commissionID)
	ctx = context.WithValue(ctx, chi.RouteCtxKey, rctx)

	rec := httptest.NewRecorder()
	wh.RetryCommissionDeprecated(rec, req.WithContext(ctx))

	assert.Equal(t, http.StatusOK, rec.Code, "delegated retry must return 200 on success")
	assert.Equal(t, 1, retrier.calls, "retrier must be invoked exactly once")
}
