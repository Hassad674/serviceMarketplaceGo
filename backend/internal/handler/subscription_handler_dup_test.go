package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appsub "marketplace-backend/internal/app/subscription"
	domainuser "marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/handler"
	"marketplace-backend/internal/port/service"
)

// idempotencyAwareStripe is a self-contained mock that records
// (idempotencyKey -> client_secret) and replays the cached secret on a
// repeated key. Mirrors Stripe's documented behaviour for the
// `Idempotency-Key` HTTP header — a 24h cache of the response.
type idempotencyAwareStripe struct {
	mu         sync.Mutex
	cache      map[string]string // idempotencyKey -> client_secret
	callCount  int
	lastIdemp  string
	cancelMu   sync.Mutex
	cancelSubs []string
}

func newIdempotencyAwareStripe() *idempotencyAwareStripe {
	return &idempotencyAwareStripe{cache: map[string]string{}}
}

func (s *idempotencyAwareStripe) EnsureCustomer(_ context.Context, _, _, _ string) (string, error) {
	return "cus_idemp", nil
}
func (s *idempotencyAwareStripe) CreateCheckoutSession(_ context.Context, in service.CreateCheckoutSessionInput) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.callCount++
	s.lastIdemp = in.IdempotencyKey
	if cached, ok := s.cache[in.IdempotencyKey]; ok && in.IdempotencyKey != "" {
		return cached, nil
	}
	secret := "cs_" + in.IdempotencyKey + "_" + uuid.NewString()
	if in.IdempotencyKey != "" {
		s.cache[in.IdempotencyKey] = secret
	}
	return secret, nil
}
func (s *idempotencyAwareStripe) EnrichCustomerWithBillingProfile(_ context.Context, _ string, _ service.BillingProfileStripeSnapshot) error {
	return nil
}
func (s *idempotencyAwareStripe) ResolvePriceID(_ context.Context, lookupKey string) (string, error) {
	return "price_" + lookupKey, nil
}
func (s *idempotencyAwareStripe) UpdateCancelAtPeriodEnd(_ context.Context, subID string, cancel bool) (service.SubscriptionSnapshot, error) {
	return service.SubscriptionSnapshot{
		ID: subID, Status: "active", CancelAtPeriodEnd: cancel,
		CurrentPeriodStart: time.Now(), CurrentPeriodEnd: time.Now().Add(30 * 24 * time.Hour),
	}, nil
}
func (s *idempotencyAwareStripe) ChangeCycleImmediate(_ context.Context, subID, newPriceID string) (service.SubscriptionSnapshot, error) {
	return service.SubscriptionSnapshot{
		ID: subID, Status: "active", PriceID: newPriceID,
		CurrentPeriodStart: time.Now(), CurrentPeriodEnd: time.Now().Add(365 * 24 * time.Hour),
	}, nil
}
func (s *idempotencyAwareStripe) ScheduleCycleChange(_ context.Context, subID, _ string) (service.ScheduledCycleChange, error) {
	effectiveAt := time.Now().Add(365 * 24 * time.Hour)
	return service.ScheduledCycleChange{
		ScheduleID: "sched_" + subID, EffectiveAt: effectiveAt,
		Snapshot: service.SubscriptionSnapshot{
			ID: subID, Status: "active",
			CurrentPeriodStart: time.Now(), CurrentPeriodEnd: effectiveAt,
		},
	}, nil
}
func (s *idempotencyAwareStripe) ReleaseSchedule(_ context.Context, _ string) error { return nil }
func (s *idempotencyAwareStripe) PreviewCycleChange(_ context.Context, _ string, _ string, _ bool) (service.InvoicePreview, error) {
	return service.InvoicePreview{
		AmountDueCents: 0, Currency: "eur",
		PeriodStart: time.Now(), PeriodEnd: time.Now().Add(30 * 24 * time.Hour),
	}, nil
}
func (s *idempotencyAwareStripe) CreatePortalSession(_ context.Context, customerID, _ string) (string, error) {
	return "https://portal.stripe.test/" + customerID, nil
}
func (s *idempotencyAwareStripe) CancelSubscription(_ context.Context, subID string) error {
	s.cancelMu.Lock()
	defer s.cancelMu.Unlock()
	s.cancelSubs = append(s.cancelSubs, subID)
	return nil
}

// newIdempotencyHandler is a parallel constructor to newSubTestHandler
// that injects the idempotency-aware Stripe mock. We don't reuse the
// other helper because subHandlerStripe is intentionally dumb — sharing
// state between tests via a single struct would couple cases together.
func newIdempotencyHandler(t *testing.T) (*handler.SubscriptionHandler, *appsub.Service, *idempotencyAwareStripe, uuid.UUID) {
	t.Helper()
	userID := uuid.New()
	orgID := uuid.New()
	userRepo := &subHandlerUserRepo{user: &domainuser.User{
		ID:             userID,
		Email:          "idemp@test.local",
		FirstName:      "Idemp",
		LastName:       "Test",
		DisplayName:    "Idemp Test",
		Role:           domainuser.RoleProvider,
		OrganizationID: &orgID,
	}}
	subRepo := &subHandlerSubRepo{}
	stripeMock := newIdempotencyAwareStripe()
	svc := appsub.NewService(appsub.ServiceDeps{
		Subscriptions: subRepo,
		Users:         userRepo,
		Amounts:       &subHandlerAmounts{},
		Stripe:        stripeMock,
		LookupKeys:    appsub.DefaultLookupKeys(),
		URLs: appsub.URLs{
			CheckoutReturn: "https://app.test/subscribe/return?session_id={CHECKOUT_SESSION_ID}",
			PortalReturn:   "https://app.test/billing",
		},
	})
	// Pin the clock so two consecutive POSTs share the same minute bucket.
	frozen := time.Date(2026, 5, 7, 10, 30, 12, 0, time.UTC)
	svc.SetClock(func() time.Time { return frozen })
	return handler.NewSubscriptionHandler(svc), svc, stripeMock, userID
}

// TestSubscribe_IdempotencyKey_TwoConsecutivePosts_ReturnSameSession
// is the end-to-end contract for the Stripe Idempotency-Key flow: two
// POST /subscriptions calls within the same minute MUST end up calling
// Stripe with the same key, and the cached client_secret is therefore
// returned on the second call. Without this, a double-tap on
// "Subscribe" creates two Checkout sessions (and potentially two paid
// subscriptions on Stripe).
func TestSubscribe_IdempotencyKey_TwoConsecutivePosts_ReturnSameSession(t *testing.T) {
	h, _, stripeMock, userID := newIdempotencyHandler(t)

	body := map[string]any{"plan": "freelance", "billing_cycle": "monthly", "auto_renew": false}

	rec1 := httptest.NewRecorder()
	h.Subscribe(rec1, authReq(http.MethodPost, "/api/v1/subscriptions", body, userID))
	require.Equal(t, http.StatusCreated, rec1.Code)
	var resp1 map[string]string
	require.NoError(t, json.NewDecoder(rec1.Body).Decode(&resp1))

	rec2 := httptest.NewRecorder()
	h.Subscribe(rec2, authReq(http.MethodPost, "/api/v1/subscriptions", body, userID))
	require.Equal(t, http.StatusCreated, rec2.Code)
	var resp2 map[string]string
	require.NoError(t, json.NewDecoder(rec2.Body).Decode(&resp2))

	assert.Equal(t, resp1["client_secret"], resp2["client_secret"],
		"two consecutive Subscribe POSTs in the same minute MUST return the same client_secret (Stripe Idempotency-Key replay)")
	assert.Equal(t, 2, stripeMock.callCount, "we still call the adapter twice — Stripe is the one collapsing on idempotency key")
	assert.True(t, strings.HasPrefix(stripeMock.lastIdemp, "subscription-create-"),
		"adapter received the documented Idempotency-Key prefix")
}

// TestSubscribe_AlreadySubscribed_Returns409WithCanonicalErrorCode
// strengthens the existing 409 test by asserting the JSON envelope's
// machine-readable error code. The web/mobile client switches behaviour
// based on this code (renders "tu es déjà abonné" instead of generic).
func TestSubscribe_AlreadySubscribed_Returns409WithCanonicalErrorCode(t *testing.T) {
	h, subRepo, userID, orgID := newSubTestHandler(t)
	subRepo.existing = seedSubscription(orgID)

	rec := httptest.NewRecorder()
	h.Subscribe(rec, authReq(http.MethodPost, "/api/v1/subscriptions",
		map[string]any{"plan": "freelance", "billing_cycle": "monthly"}, userID))

	require.Equal(t, http.StatusConflict, rec.Code)
	var body map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	// The response envelope is the flat `{error, message}` shape used
	// by pkg/response.Error — the canonical error code rides as the
	// `error` field directly. The mobile/web clients pattern-match on
	// this string to render the correct UX (already-subscribed vs.
	// generic subscribe failure).
	assert.Equal(t, "already_subscribed", body["error"],
		"the canonical error code MUST be `already_subscribed` so the client can branch on it")
	assert.NotEmpty(t, body["message"], "human-readable message must accompany the error code")
}

// Ensure the stub still satisfies the StripeSubscriptionService
// interface — the compile-time check guards against drift.
var _ service.StripeSubscriptionService = (*idempotencyAwareStripe)(nil)
