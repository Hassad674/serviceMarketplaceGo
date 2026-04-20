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
	// users is keyed by UUID — the billing flow may fetch both the caller
	// AND the recipient (when recipient_id is passed), so tests must be
	// able to stage distinct users per id. Fallback `user` is returned
	// when the map is empty (backward-compat with older single-user tests).
	users map[uuid.UUID]*domainuser.User
	user  *domainuser.User
	err   error
}

func (s *stubUserRepo) Create(_ context.Context, _ *domainuser.User) error { return nil }
func (s *stubUserRepo) GetByID(_ context.Context, id uuid.UUID) (*domainuser.User, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.users != nil {
		if u, ok := s.users[id]; ok {
			return u, nil
		}
		return nil, nil
	}
	return s.user, nil
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

// newBillingHandlerWithRecipient stages BOTH the caller and the recipient
// so tests can exercise the proposal.DetermineRoles path driven by the
// recipient_id query parameter.
func newBillingHandlerWithRecipient(t *testing.T, callerRole, recipientRole domainuser.Role) (*handler.BillingHandler, uuid.UUID, uuid.UUID) {
	t.Helper()
	callerID := uuid.New()
	recipientID := uuid.New()
	users := map[uuid.UUID]*domainuser.User{
		callerID:    {ID: callerID, Role: callerRole},
		recipientID: {ID: recipientID, Role: recipientRole},
	}
	svc := paymentapp.NewService(paymentapp.ServiceDeps{Users: &stubUserRepo{users: users}})
	return handler.NewBillingHandler(svc), callerID, recipientID
}

type feePreviewBody struct {
	AmountCents      int64  `json:"amount_cents"`
	FeeCents         int64  `json:"fee_cents"`
	NetCents         int64  `json:"net_cents"`
	Role             string `json:"role"`
	ActiveTierIndex  int    `json:"active_tier_index"`
	ViewerIsProvider bool   `json:"viewer_is_provider"`
}

func decodeFeePreview(t *testing.T, body *httptest.ResponseRecorder) feePreviewBody {
	t.Helper()
	var out feePreviewBody
	require.NoError(t, json.NewDecoder(body.Body).Decode(&out))
	return out
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

// ViewerIsProvider tests cover the visibility flag — the UI hides the
// preview whenever a non-provider viewer queries it, so misclassifying any
// of these combinations would leak the prestataire's cost structure to the
// client side.
func TestBillingHandler_GetFeePreview_ViewerIsProvider_DefaultByRole(t *testing.T) {
	tests := []struct {
		role              domainuser.Role
		wantIsProvider    bool
		wantStatus        int
	}{
		{domainuser.RoleEnterprise, false, http.StatusOK}, // client always
		{domainuser.RoleProvider, true, http.StatusOK},    // provider always
		{domainuser.RoleAgency, true, http.StatusOK},      // ambiguous → default true (happy path)
	}
	for _, tc := range tests {
		t.Run(string(tc.role), func(t *testing.T) {
			h, userID := newBillingHandler(t, tc.role)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/billing/fee-preview?amount=50000", nil)
			req = req.WithContext(withUserID(req.Context(), userID))
			rec := httptest.NewRecorder()

			h.GetFeePreview(rec, req)

			require.Equal(t, tc.wantStatus, rec.Code)
			body := decodeFeePreview(t, rec)
			assert.Equal(t, tc.wantIsProvider, body.ViewerIsProvider,
				"role %q must default to viewer_is_provider=%v", tc.role, tc.wantIsProvider)
		})
	}
}

func TestBillingHandler_GetFeePreview_ViewerIsProvider_WithRecipient(t *testing.T) {
	tests := []struct {
		name             string
		callerRole       domainuser.Role
		recipientRole    domainuser.Role
		wantIsProvider   bool
	}{
		// Enterprise is always the client, regardless of who they message.
		{"enterprise vs provider", domainuser.RoleEnterprise, domainuser.RoleProvider, false},
		{"enterprise vs agency", domainuser.RoleEnterprise, domainuser.RoleAgency, false},

		// Provider is always the provider, regardless of the other party.
		{"provider vs enterprise", domainuser.RoleProvider, domainuser.RoleEnterprise, true},
		{"provider vs agency", domainuser.RoleProvider, domainuser.RoleAgency, true},

		// Agency disambiguation — the whole reason recipient_id exists.
		{"agency vs enterprise → agency is provider", domainuser.RoleAgency, domainuser.RoleEnterprise, true},
		{"agency vs provider → agency is client (no fee shown)", domainuser.RoleAgency, domainuser.RoleProvider, false},

		// Invalid combinations fail CLOSED — the preview MUST be hidden so
		// the UI never leaks fees on a malformed proposal draft.
		{"agency vs agency → invalid, fail closed", domainuser.RoleAgency, domainuser.RoleAgency, false},
		{"enterprise vs enterprise → invalid, fail closed", domainuser.RoleEnterprise, domainuser.RoleEnterprise, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h, callerID, recipientID := newBillingHandlerWithRecipient(t, tc.callerRole, tc.recipientRole)

			url := "/api/v1/billing/fee-preview?amount=50000&recipient_id=" + recipientID.String()
			req := httptest.NewRequest(http.MethodGet, url, nil)
			req = req.WithContext(withUserID(req.Context(), callerID))
			rec := httptest.NewRecorder()

			h.GetFeePreview(rec, req)

			require.Equal(t, http.StatusOK, rec.Code)
			body := decodeFeePreview(t, rec)
			assert.Equal(t, tc.wantIsProvider, body.ViewerIsProvider,
				"%s must return viewer_is_provider=%v", tc.name, tc.wantIsProvider)
		})
	}
}

func TestBillingHandler_GetFeePreview_RejectsInvalidRecipientUUID(t *testing.T) {
	h, userID := newBillingHandler(t, domainuser.RoleProvider)

	url := "/api/v1/billing/fee-preview?amount=50000&recipient_id=not-a-uuid"
	req := httptest.NewRequest(http.MethodGet, url, nil)
	req = req.WithContext(withUserID(req.Context(), userID))
	rec := httptest.NewRecorder()

	h.GetFeePreview(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestBillingHandler_GetFeePreview_UnknownRecipient_FailsClosed(t *testing.T) {
	// When the recipient_id refers to a user that doesn't exist, the safest
	// behaviour is to render no preview — we must never leak fees to an
	// unknown party. The response is 200 (preview still includes the grid
	// for reference) but viewer_is_provider is false.
	callerID := uuid.New()
	users := map[uuid.UUID]*domainuser.User{
		callerID: {ID: callerID, Role: domainuser.RoleAgency},
	}
	svc := paymentapp.NewService(paymentapp.ServiceDeps{Users: &stubUserRepo{users: users}})
	h := handler.NewBillingHandler(svc)

	unknownRecipient := uuid.New().String()
	url := "/api/v1/billing/fee-preview?amount=50000&recipient_id=" + unknownRecipient
	req := httptest.NewRequest(http.MethodGet, url, nil)
	req = req.WithContext(withUserID(req.Context(), callerID))
	rec := httptest.NewRecorder()

	h.GetFeePreview(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	body := decodeFeePreview(t, rec)
	assert.False(t, body.ViewerIsProvider, "unknown recipient must fail closed")
}
