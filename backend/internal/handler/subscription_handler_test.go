package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appsub "marketplace-backend/internal/app/subscription"
	domain "marketplace-backend/internal/domain/subscription"
	domainuser "marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/handler"
	"marketplace-backend/internal/handler/middleware"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/port/service"
)

// These tests exercise the HTTP surface only; the underlying app service
// is driven through real stubs so we see the full request ↔ response
// round-trip including DTO marshalling and error-to-status mapping.
//
// The mocks here are self-contained (no coupling to app/subscription's
// mocks_test.go) because Go's test package isolation requires it — two
// test binaries cannot share unexported types.

// ---------- shared stubs ----------

type subHandlerUserRepo struct {
	user *domainuser.User
	err  error
}

func (m *subHandlerUserRepo) Create(_ context.Context, _ *domainuser.User) error { return nil }
func (m *subHandlerUserRepo) GetByID(_ context.Context, _ uuid.UUID) (*domainuser.User, error) {
	return m.user, m.err
}
func (m *subHandlerUserRepo) GetByEmail(_ context.Context, _ string) (*domainuser.User, error) {
	return nil, nil
}
func (m *subHandlerUserRepo) Update(_ context.Context, _ *domainuser.User) error { return nil }
func (m *subHandlerUserRepo) Delete(_ context.Context, _ uuid.UUID) error        { return nil }
func (m *subHandlerUserRepo) ExistsByEmail(_ context.Context, _ string) (bool, error) {
	return false, nil
}
func (m *subHandlerUserRepo) ListAdmin(_ context.Context, _ repository.AdminUserFilters) ([]*domainuser.User, string, error) {
	return nil, "", nil
}
func (m *subHandlerUserRepo) CountAdmin(_ context.Context, _ repository.AdminUserFilters) (int, error) {
	return 0, nil
}
func (m *subHandlerUserRepo) CountByRole(_ context.Context) (map[string]int, error) { return nil, nil }
func (m *subHandlerUserRepo) CountByStatus(_ context.Context) (map[string]int, error) {
	return nil, nil
}
func (m *subHandlerUserRepo) RecentSignups(_ context.Context, _ int) ([]*domainuser.User, error) {
	return nil, nil
}
func (m *subHandlerUserRepo) BumpSessionVersion(_ context.Context, _ uuid.UUID) (int, error) {
	return 0, nil
}
func (m *subHandlerUserRepo) GetSessionVersion(_ context.Context, _ uuid.UUID) (int, error) {
	return 0, nil
}
func (m *subHandlerUserRepo) UpdateEmailNotificationsEnabled(_ context.Context, _ uuid.UUID, _ bool) error {
	return nil
}
func (m *subHandlerUserRepo) TouchLastActive(_ context.Context, _ uuid.UUID) error { return nil }

type subHandlerSubRepo struct {
	existing *domain.Subscription
	notFound bool
}

func (r *subHandlerSubRepo) Create(_ context.Context, s *domain.Subscription) error {
	r.existing = s
	return nil
}
func (r *subHandlerSubRepo) FindOpenByOrganization(_ context.Context, _ uuid.UUID) (*domain.Subscription, error) {
	if r.notFound || r.existing == nil {
		return nil, domain.ErrNotFound
	}
	return r.existing, nil
}
func (r *subHandlerSubRepo) FindByStripeID(_ context.Context, _ string) (*domain.Subscription, error) {
	return nil, domain.ErrNotFound
}
func (r *subHandlerSubRepo) Update(_ context.Context, s *domain.Subscription) error {
	r.existing = s
	return nil
}

type subHandlerAmounts struct {
	amounts []int64
}

func (a *subHandlerAmounts) ListProviderMilestoneAmountsSince(
	_ context.Context, _ uuid.UUID, _ time.Time,
) ([]int64, error) {
	return a.amounts, nil
}

type subHandlerStripe struct{}

func (s *subHandlerStripe) EnsureCustomer(_ context.Context, _, _, _ string) (string, error) {
	return "cus_test", nil
}
func (s *subHandlerStripe) CreateCheckoutSession(_ context.Context, in service.CreateCheckoutSessionInput) (string, error) {
	return "https://checkout.stripe.test/" + in.PriceID, nil
}
func (s *subHandlerStripe) ResolvePriceID(_ context.Context, lookupKey string) (string, error) {
	return "price_" + lookupKey, nil
}
func (s *subHandlerStripe) UpdateCancelAtPeriodEnd(_ context.Context, subID string, cancel bool) (service.SubscriptionSnapshot, error) {
	return service.SubscriptionSnapshot{
		ID: subID, Status: "active", CancelAtPeriodEnd: cancel,
		CurrentPeriodStart: time.Now(), CurrentPeriodEnd: time.Now().Add(30 * 24 * time.Hour),
	}, nil
}
func (s *subHandlerStripe) ChangeCycleImmediate(_ context.Context, subID, newPriceID string) (service.SubscriptionSnapshot, error) {
	return service.SubscriptionSnapshot{
		ID: subID, Status: "active", PriceID: newPriceID,
		CurrentPeriodStart: time.Now(), CurrentPeriodEnd: time.Now().Add(365 * 24 * time.Hour),
	}, nil
}
func (s *subHandlerStripe) ScheduleCycleChange(_ context.Context, subID, _ string) (service.ScheduledCycleChange, error) {
	effectiveAt := time.Now().Add(365 * 24 * time.Hour)
	return service.ScheduledCycleChange{
		ScheduleID:  "sched_" + subID,
		EffectiveAt: effectiveAt,
		Snapshot: service.SubscriptionSnapshot{
			ID: subID, Status: "active",
			CurrentPeriodStart: time.Now(), CurrentPeriodEnd: effectiveAt,
		},
	}, nil
}
func (s *subHandlerStripe) ReleaseSchedule(_ context.Context, _ string) error { return nil }
func (s *subHandlerStripe) PreviewCycleChange(_ context.Context, _ string, _ string, prorateImmediately bool) (service.InvoicePreview, error) {
	amount := int64(0)
	if prorateImmediately {
		amount = 41900
	}
	return service.InvoicePreview{
		AmountDueCents: amount, Currency: "eur",
		PeriodStart: time.Now(), PeriodEnd: time.Now().Add(365 * 24 * time.Hour),
	}, nil
}
func (s *subHandlerStripe) CreatePortalSession(_ context.Context, customerID, _ string) (string, error) {
	return "https://portal.stripe.test/" + customerID, nil
}

// ---------- harness ----------

// newSubTestHandler wires a handler against in-memory mocks. Returns
// (handler, repo, userID, orgID). The user is pre-linked to orgID so
// the handler's resolveActorOrg succeeds without extra setup; seed the
// subscription against orgID (that is what the service queries).
func newSubTestHandler(t *testing.T) (*handler.SubscriptionHandler, *subHandlerSubRepo, uuid.UUID, uuid.UUID) {
	t.Helper()
	userID := uuid.New()
	orgID := uuid.New()
	userRepo := &subHandlerUserRepo{user: &domainuser.User{
		ID:             userID,
		Email:          "sub@test.local",
		FirstName:      "Sub",
		LastName:       "Test",
		DisplayName:    "Sub Test",
		Role:           domainuser.RoleProvider,
		OrganizationID: &orgID,
	}}
	subRepo := &subHandlerSubRepo{}
	svc := appsub.NewService(appsub.ServiceDeps{
		Subscriptions: subRepo,
		Users:         userRepo,
		Amounts:       &subHandlerAmounts{},
		Stripe:        &subHandlerStripe{},
		LookupKeys:    appsub.DefaultLookupKeys(),
		URLs: appsub.URLs{
			CheckoutSuccess: "https://app.test/billing/success",
			CheckoutCancel:  "https://app.test/billing/cancel",
			PortalReturn:    "https://app.test/billing",
		},
	})
	return handler.NewSubscriptionHandler(svc), subRepo, userID, orgID
}

func authReq(method, url string, body any, userID uuid.UUID) *http.Request {
	var rdr *strings.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rdr = strings.NewReader(string(b))
	}
	var req *http.Request
	if rdr != nil {
		req = httptest.NewRequest(method, url, rdr)
	} else {
		req = httptest.NewRequest(method, url, nil)
	}
	req = req.WithContext(context.WithValue(req.Context(), middleware.ContextKeyUserID, userID))
	req.Header.Set("Content-Type", "application/json")
	return req
}

// ---------- Subscribe ----------

func TestSubscribe_Unauthenticated_Returns401(t *testing.T) {
	h, _, _, _ := newSubTestHandler(t)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/subscriptions", bytes.NewBufferString(`{"plan":"freelance","billing_cycle":"monthly"}`))
	rec := httptest.NewRecorder()

	h.Subscribe(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestSubscribe_HappyPath_Returns201WithCheckoutURL(t *testing.T) {
	h, _, userID, _ := newSubTestHandler(t)
	req := authReq(http.MethodPost, "/api/v1/subscriptions", map[string]any{
		"plan": "freelance", "billing_cycle": "monthly", "auto_renew": false,
	}, userID)
	rec := httptest.NewRecorder()

	h.Subscribe(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)
	var body map[string]string
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	assert.Contains(t, body["checkout_url"], "checkout.stripe.test")
}

func TestSubscribe_InvalidPlan_Returns400(t *testing.T) {
	h, _, userID, _ := newSubTestHandler(t)
	req := authReq(http.MethodPost, "/api/v1/subscriptions", map[string]any{
		"plan": "enterprise", "billing_cycle": "monthly",
	}, userID)
	rec := httptest.NewRecorder()

	h.Subscribe(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestSubscribe_MalformedJSON_Returns400(t *testing.T) {
	h, _, userID, _ := newSubTestHandler(t)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/subscriptions", strings.NewReader("{not json"))
	req = req.WithContext(context.WithValue(req.Context(), middleware.ContextKeyUserID, userID))
	rec := httptest.NewRecorder()

	h.Subscribe(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestSubscribe_AlreadySubscribed_Returns409(t *testing.T) {
	h, subRepo, userID, orgID := newSubTestHandler(t)
	// Seed an existing subscription for the org.
	subRepo.existing = seedSubscription(orgID)

	req := authReq(http.MethodPost, "/api/v1/subscriptions", map[string]any{
		"plan": "freelance", "billing_cycle": "monthly",
	}, userID)
	rec := httptest.NewRecorder()

	h.Subscribe(rec, req)

	assert.Equal(t, http.StatusConflict, rec.Code)
}

// ---------- GetMine ----------

func TestGetMine_FreeUser_Returns404(t *testing.T) {
	h, _, userID, _ := newSubTestHandler(t)
	req := authReq(http.MethodGet, "/api/v1/subscriptions/me", nil, userID)
	rec := httptest.NewRecorder()

	h.GetMine(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestGetMine_Subscribed_Returns200(t *testing.T) {
	h, subRepo, userID, orgID := newSubTestHandler(t)
	subRepo.existing = seedSubscription(orgID)
	req := authReq(http.MethodGet, "/api/v1/subscriptions/me", nil, userID)
	rec := httptest.NewRecorder()

	h.GetMine(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var body map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	assert.Equal(t, "freelance", body["plan"])
	assert.Equal(t, "monthly", body["billing_cycle"])
}

// ---------- ToggleAutoRenew ----------

func TestToggleAutoRenew_HappyPath(t *testing.T) {
	h, subRepo, userID, orgID := newSubTestHandler(t)
	subRepo.existing = seedSubscription(orgID)
	req := authReq(http.MethodPatch, "/api/v1/subscriptions/me/auto-renew",
		map[string]any{"auto_renew": true}, userID)
	rec := httptest.NewRecorder()

	h.ToggleAutoRenew(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var body map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	// auto_renew=true → cancel_at_period_end MUST be false
	assert.Equal(t, false, body["cancel_at_period_end"])
}

func TestToggleAutoRenew_NoSubscription_Returns404(t *testing.T) {
	h, _, userID, _ := newSubTestHandler(t)
	req := authReq(http.MethodPatch, "/api/v1/subscriptions/me/auto-renew",
		map[string]any{"auto_renew": true}, userID)
	rec := httptest.NewRecorder()

	h.ToggleAutoRenew(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// ---------- ChangeCycle ----------

func TestChangeCycle_MonthlyToAnnual(t *testing.T) {
	h, subRepo, userID, orgID := newSubTestHandler(t)
	subRepo.existing = seedSubscription(orgID) // monthly
	req := authReq(http.MethodPatch, "/api/v1/subscriptions/me/billing-cycle",
		map[string]any{"billing_cycle": "annual"}, userID)
	rec := httptest.NewRecorder()

	h.ChangeCycle(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var body map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	assert.Equal(t, "annual", body["billing_cycle"])
}

func TestChangeCycle_SameCycle_Returns409(t *testing.T) {
	h, subRepo, userID, orgID := newSubTestHandler(t)
	subRepo.existing = seedSubscription(orgID) // monthly
	req := authReq(http.MethodPatch, "/api/v1/subscriptions/me/billing-cycle",
		map[string]any{"billing_cycle": "monthly"}, userID)
	rec := httptest.NewRecorder()

	h.ChangeCycle(rec, req)

	assert.Equal(t, http.StatusConflict, rec.Code)
}

func TestChangeCycle_InvalidCycle_Returns400(t *testing.T) {
	h, subRepo, userID, orgID := newSubTestHandler(t)
	subRepo.existing = seedSubscription(orgID)
	req := authReq(http.MethodPatch, "/api/v1/subscriptions/me/billing-cycle",
		map[string]any{"billing_cycle": "weekly"}, userID)
	rec := httptest.NewRecorder()

	h.ChangeCycle(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// ---------- GetStats ----------

func TestGetStats_HappyPath(t *testing.T) {
	h, subRepo, userID, orgID := newSubTestHandler(t)
	subRepo.existing = seedSubscription(orgID)
	req := authReq(http.MethodGet, "/api/v1/subscriptions/me/stats", nil, userID)
	rec := httptest.NewRecorder()

	h.GetStats(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var body map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	assert.Contains(t, body, "saved_fee_cents")
	assert.Contains(t, body, "saved_count")
	assert.Contains(t, body, "since")
}

func TestGetStats_NoSubscription_Returns404(t *testing.T) {
	h, _, userID, _ := newSubTestHandler(t)
	req := authReq(http.MethodGet, "/api/v1/subscriptions/me/stats", nil, userID)
	rec := httptest.NewRecorder()

	h.GetStats(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// ---------- GetPortal ----------

func TestGetPortal_HappyPath(t *testing.T) {
	h, subRepo, userID, orgID := newSubTestHandler(t)
	subRepo.existing = seedSubscription(orgID)
	req := authReq(http.MethodGet, "/api/v1/subscriptions/portal", nil, userID)
	rec := httptest.NewRecorder()

	h.GetPortal(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var body map[string]string
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	assert.Contains(t, body["url"], "portal.stripe.test")
}

func TestGetPortal_NoSubscription_Returns404(t *testing.T) {
	h, _, userID, _ := newSubTestHandler(t)
	req := authReq(http.MethodGet, "/api/v1/subscriptions/portal", nil, userID)
	rec := httptest.NewRecorder()

	h.GetPortal(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// Compile guard: ensures the package-level error sentinels stay exported.
var _ = errors.Is

// ---------- helpers ----------

// seedSubscription returns an activated subscription tied to the given
// org. Handler tests always resolve the caller's user → org first, so
// this helper takes the org id directly to mirror what the service
// would see.
func seedSubscription(orgID uuid.UUID) *domain.Subscription {
	now := time.Now()
	s, _ := domain.NewSubscription(domain.NewSubscriptionInput{
		OrganizationID:       orgID,
		Plan:                 domain.PlanFreelance,
		BillingCycle:         domain.CycleMonthly,
		StripeCustomerID:     "cus_test",
		StripeSubscriptionID: "sub_test_" + orgID.String()[:8],
		StripePriceID:        "price_freelance_monthly",
		CurrentPeriodStart:   now,
		CurrentPeriodEnd:     now.Add(30 * 24 * time.Hour),
		CancelAtPeriodEnd:    true,
	})
	_ = s.Activate()
	return s
}
