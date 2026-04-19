package handler_test

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
	domainuser "marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/handler"
	"marketplace-backend/internal/handler/middleware"
	"marketplace-backend/internal/port/repository"
)

// stubUserRepo implements the full UserRepository contract with no-ops
// except GetByID, which is the only method the billing flow exercises.
// We assert the interface satisfaction at compile time so future contract
// changes fail the build instead of panicking at runtime.
var _ repository.UserRepository = (*stubUserRepo)(nil)

type stubUserRepo struct {
	user *domainuser.User
	err  error
}

func (s *stubUserRepo) Create(_ context.Context, _ *domainuser.User) error { return nil }
func (s *stubUserRepo) GetByID(_ context.Context, _ uuid.UUID) (*domainuser.User, error) {
	return s.user, s.err
}
func (s *stubUserRepo) GetByEmail(_ context.Context, _ string) (*domainuser.User, error) {
	return nil, nil
}
func (s *stubUserRepo) Update(_ context.Context, _ *domainuser.User) error { return nil }
func (s *stubUserRepo) Delete(_ context.Context, _ uuid.UUID) error        { return nil }
func (s *stubUserRepo) ExistsByEmail(_ context.Context, _ string) (bool, error) {
	return false, nil
}
func (s *stubUserRepo) ListAdmin(_ context.Context, _ repository.AdminUserFilters) ([]*domainuser.User, string, error) {
	return nil, "", nil
}
func (s *stubUserRepo) CountAdmin(_ context.Context, _ repository.AdminUserFilters) (int, error) {
	return 0, nil
}
func (s *stubUserRepo) CountByRole(_ context.Context) (map[string]int, error)   { return nil, nil }
func (s *stubUserRepo) CountByStatus(_ context.Context) (map[string]int, error) { return nil, nil }
func (s *stubUserRepo) RecentSignups(_ context.Context, _ int) ([]*domainuser.User, error) {
	return nil, nil
}
func (s *stubUserRepo) BumpSessionVersion(_ context.Context, _ uuid.UUID) (int, error) {
	return 0, nil
}
func (s *stubUserRepo) GetSessionVersion(_ context.Context, _ uuid.UUID) (int, error) {
	return 0, nil
}
func (s *stubUserRepo) UpdateEmailNotificationsEnabled(_ context.Context, _ uuid.UUID, _ bool) error {
	return nil
}
func (s *stubUserRepo) TouchLastActive(_ context.Context, _ uuid.UUID) error { return nil }

func withUserID(ctx context.Context, id uuid.UUID) context.Context {
	return context.WithValue(ctx, middleware.ContextKeyUserID, id)
}

func newBillingHandler(t *testing.T, role domainuser.Role) (*handler.BillingHandler, uuid.UUID) {
	t.Helper()
	userID := uuid.New()
	u := &domainuser.User{ID: userID, Role: role}
	svc := paymentapp.NewService(paymentapp.ServiceDeps{Users: &stubUserRepo{user: u}})
	return handler.NewBillingHandler(svc), userID
}

func TestBillingHandler_GetFeePreview_FreelanceTier2(t *testing.T) {
	h, userID := newBillingHandler(t, domainuser.RoleProvider)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/billing/fee-preview?amount=50000", nil)
	req = req.WithContext(withUserID(req.Context(), userID))
	rec := httptest.NewRecorder()

	h.GetFeePreview(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var body struct {
		AmountCents     int64  `json:"amount_cents"`
		FeeCents        int64  `json:"fee_cents"`
		NetCents        int64  `json:"net_cents"`
		Role            string `json:"role"`
		ActiveTierIndex int    `json:"active_tier_index"`
		Tiers           []struct {
			Label    string `json:"label"`
			MaxCents *int64 `json:"max_cents"`
			FeeCents int64  `json:"fee_cents"`
		} `json:"tiers"`
	}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))

	assert.Equal(t, int64(50000), body.AmountCents)
	assert.Equal(t, int64(1500), body.FeeCents)
	assert.Equal(t, int64(48500), body.NetCents)
	assert.Equal(t, "freelance", body.Role)
	assert.Equal(t, 1, body.ActiveTierIndex)
	require.Len(t, body.Tiers, 3)
	assert.Nil(t, body.Tiers[2].MaxCents, "last tier is open-ended")
}

func TestBillingHandler_GetFeePreview_AgencyTier3(t *testing.T) {
	h, userID := newBillingHandler(t, domainuser.RoleAgency)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/billing/fee-preview?amount=500000", nil)
	req = req.WithContext(withUserID(req.Context(), userID))
	rec := httptest.NewRecorder()

	h.GetFeePreview(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var body struct {
		FeeCents        int64  `json:"fee_cents"`
		NetCents        int64  `json:"net_cents"`
		Role            string `json:"role"`
		ActiveTierIndex int    `json:"active_tier_index"`
	}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))

	assert.Equal(t, int64(6900), body.FeeCents)
	assert.Equal(t, int64(493100), body.NetCents)
	assert.Equal(t, "agency", body.Role)
	assert.Equal(t, 2, body.ActiveTierIndex)
}

func TestBillingHandler_GetFeePreview_RejectsMissingAmount(t *testing.T) {
	h, userID := newBillingHandler(t, domainuser.RoleProvider)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/billing/fee-preview", nil)
	req = req.WithContext(withUserID(req.Context(), userID))
	rec := httptest.NewRecorder()

	h.GetFeePreview(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestBillingHandler_GetFeePreview_RejectsNonNumericAmount(t *testing.T) {
	h, userID := newBillingHandler(t, domainuser.RoleProvider)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/billing/fee-preview?amount=abc", nil)
	req = req.WithContext(withUserID(req.Context(), userID))
	rec := httptest.NewRecorder()

	h.GetFeePreview(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestBillingHandler_GetFeePreview_RejectsNegativeAmount(t *testing.T) {
	h, userID := newBillingHandler(t, domainuser.RoleProvider)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/billing/fee-preview?amount=-500", nil)
	req = req.WithContext(withUserID(req.Context(), userID))
	rec := httptest.NewRecorder()

	h.GetFeePreview(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestBillingHandler_GetFeePreview_RejectsUnauthenticated(t *testing.T) {
	h, _ := newBillingHandler(t, domainuser.RoleProvider)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/billing/fee-preview?amount=50000", nil)
	// No WithUserID on the context — simulating an unauthenticated request
	// that somehow reaches the handler. The middleware stops this in
	// production; the defense-in-depth check here keeps a router-wiring
	// bug from leaking fee data to anonymous clients.
	rec := httptest.NewRecorder()

	h.GetFeePreview(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}
