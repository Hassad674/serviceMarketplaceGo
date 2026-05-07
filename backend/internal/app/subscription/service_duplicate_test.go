package subscription_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appsub "marketplace-backend/internal/app/subscription"
	"marketplace-backend/internal/domain/audit"
	domain "marketplace-backend/internal/domain/subscription"
	"marketplace-backend/internal/port/service"
)

// ===========================================================================
// Stripe Idempotency-Key contract on POST /subscriptions
// ===========================================================================

// TestSubscribe_SendsIdempotencyKeyToStripe verifies that the Subscribe
// app service derives a deterministic Idempotency-Key from
// (org_id, plan, cycle, minute_truncated) and forwards it to the Stripe
// adapter via CreateCheckoutSessionInput. Stripe will then collapse
// retries within the same minute into a single Checkout session.
func TestSubscribe_SendsIdempotencyKeyToStripe(t *testing.T) {
	svc, _, users, _, stripe := newTestService()

	out, err := svc.Subscribe(context.Background(), appsub.SubscribeInput{
		OrganizationID: *users.user.OrganizationID,
		ActorUserID:    users.user.ID,
		Plan:           domain.PlanFreelance,
		BillingCycle:   domain.CycleMonthly,
	})

	require.NoError(t, err)
	require.NotEmpty(t, out.ClientSecret)
	require.NotNil(t, stripe.lastCreateCheckoutInput)
	assert.NotEmpty(t, stripe.lastCreateCheckoutInput.IdempotencyKey,
		"Subscribe MUST forward an Idempotency-Key to Stripe")
	assert.True(t, strings.HasPrefix(stripe.lastCreateCheckoutInput.IdempotencyKey, "subscription-create-"),
		"Idempotency-Key must follow the documented prefix convention, got %q",
		stripe.lastCreateCheckoutInput.IdempotencyKey)
}

// TestSubscribe_IdempotencyKeyStableWithinMinute verifies that two
// Subscribe calls within the SAME minute yield the SAME Idempotency-Key.
// That is the property Stripe relies on to return the cached Checkout
// session (the user is not double-charged on a double-tap).
func TestSubscribe_IdempotencyKeyStableWithinMinute(t *testing.T) {
	svc, _, users, _, stripe := newTestService()
	frozen := time.Date(2026, 5, 7, 10, 30, 12, 0, time.UTC)
	svc.SetClock(func() time.Time { return frozen })

	in := appsub.SubscribeInput{
		OrganizationID: *users.user.OrganizationID,
		ActorUserID:    users.user.ID,
		Plan:           domain.PlanFreelance,
		BillingCycle:   domain.CycleMonthly,
	}

	_, err := svc.Subscribe(context.Background(), in)
	require.NoError(t, err)
	firstKey := stripe.lastCreateCheckoutInput.IdempotencyKey

	// Second call inside the same minute (same wall-clock minute, +30s).
	svc.SetClock(func() time.Time { return frozen.Add(30 * time.Second) })
	_, err = svc.Subscribe(context.Background(), in)
	require.NoError(t, err)
	secondKey := stripe.lastCreateCheckoutInput.IdempotencyKey

	assert.Equal(t, firstKey, secondKey,
		"two Subscribe calls within the same minute MUST share the same Idempotency-Key")
}

// TestSubscribe_IdempotencyKeyChangesAcrossMinutes verifies that a
// retry one minute later — the user genuinely wants a new attempt
// after a failed first one — produces a fresh Idempotency-Key so
// Stripe accepts a new Checkout session.
func TestSubscribe_IdempotencyKeyChangesAcrossMinutes(t *testing.T) {
	svc, _, users, _, stripe := newTestService()
	frozen := time.Date(2026, 5, 7, 10, 30, 12, 0, time.UTC)
	svc.SetClock(func() time.Time { return frozen })

	in := appsub.SubscribeInput{
		OrganizationID: *users.user.OrganizationID,
		ActorUserID:    users.user.ID,
		Plan:           domain.PlanFreelance,
		BillingCycle:   domain.CycleMonthly,
	}

	_, err := svc.Subscribe(context.Background(), in)
	require.NoError(t, err)
	firstKey := stripe.lastCreateCheckoutInput.IdempotencyKey

	// Bump the clock by 65s — crosses the minute boundary.
	svc.SetClock(func() time.Time { return frozen.Add(65 * time.Second) })
	_, err = svc.Subscribe(context.Background(), in)
	require.NoError(t, err)
	secondKey := stripe.lastCreateCheckoutInput.IdempotencyKey

	assert.NotEqual(t, firstKey, secondKey,
		"two Subscribe calls one minute apart MUST produce different Idempotency-Keys")
}

// TestSubscribe_IdempotencyKeyDifferentForDifferentPlans verifies that
// a user who switches plan/cycle within the same minute still gets a
// fresh session — Stripe would otherwise replay the previous (wrong)
// session from cache.
func TestSubscribe_IdempotencyKeyDifferentForDifferentPlans(t *testing.T) {
	svc, _, users, _, stripe := newTestService()
	frozen := time.Date(2026, 5, 7, 10, 30, 12, 0, time.UTC)
	svc.SetClock(func() time.Time { return frozen })

	_, err := svc.Subscribe(context.Background(), appsub.SubscribeInput{
		OrganizationID: *users.user.OrganizationID,
		ActorUserID:    users.user.ID,
		Plan:           domain.PlanFreelance,
		BillingCycle:   domain.CycleMonthly,
	})
	require.NoError(t, err)
	monthlyKey := stripe.lastCreateCheckoutInput.IdempotencyKey

	_, err = svc.Subscribe(context.Background(), appsub.SubscribeInput{
		OrganizationID: *users.user.OrganizationID,
		ActorUserID:    users.user.ID,
		Plan:           domain.PlanFreelance,
		BillingCycle:   domain.CycleAnnual,
	})
	require.NoError(t, err)
	annualKey := stripe.lastCreateCheckoutInput.IdempotencyKey

	assert.NotEqual(t, monthlyKey, annualKey,
		"different (plan, cycle) MUST produce different Idempotency-Keys within the same minute")
}

// ===========================================================================
// Subscribe duplicate guard — also exercises the audit emission path
// ===========================================================================

// TestSubscribe_DoesNotCallStripeWhenAlreadySubscribed proves the
// short-circuit happens BEFORE any Stripe API call. The bug being
// guarded against is N× billing — so the test asserts the side-effect
// boundary, not just the returned error.
func TestSubscribe_DoesNotCallStripeWhenAlreadySubscribed(t *testing.T) {
	svc, subs, users, _, stripe := newTestService()
	existing, _ := domain.NewSubscription(freshDomainInput(*users.user.OrganizationID))
	subs.findOpenByOrgFn = func(_ context.Context, _ uuid.UUID) (*domain.Subscription, error) {
		return existing, nil
	}

	_, err := svc.Subscribe(context.Background(), appsub.SubscribeInput{
		OrganizationID: *users.user.OrganizationID,
		ActorUserID:    users.user.ID,
		Plan:           domain.PlanFreelance,
		BillingCycle:   domain.CycleMonthly,
	})

	assert.ErrorIs(t, err, domain.ErrAlreadySubscribed)
	assert.Nil(t, stripe.lastCreateCheckoutInput,
		"Stripe MUST NOT be hit when the org already has an open subscription")
}

// captureAuditRepo is a minimal AuditRepository stub that records the
// entries written, ignores the listing methods, and never errors.
type captureAuditRepo struct {
	logged []*audit.Entry
	err    error
}

func (c *captureAuditRepo) Log(_ context.Context, e *audit.Entry) error {
	if c.err != nil {
		return c.err
	}
	c.logged = append(c.logged, e)
	return nil
}
func (c *captureAuditRepo) ListByResource(_ context.Context, _ audit.ResourceType, _ uuid.UUID, _ string, _ int) ([]*audit.Entry, string, error) {
	return nil, "", nil
}
func (c *captureAuditRepo) ListByUser(_ context.Context, _ uuid.UUID, _ string, _ int) ([]*audit.Entry, string, error) {
	return nil, "", nil
}

// TestSubscribe_AlreadySubscribed_EmitsAuditLog verifies the audit
// hook is invoked with the canonical action and the relevant
// identifiers when a duplicate Subscribe is rejected.
func TestSubscribe_AlreadySubscribed_EmitsAuditLog(t *testing.T) {
	svc, subs, users, _, _ := newTestService()
	existing, _ := domain.NewSubscription(freshDomainInput(*users.user.OrganizationID))
	subs.findOpenByOrgFn = func(_ context.Context, _ uuid.UUID) (*domain.Subscription, error) {
		return existing, nil
	}
	auditSink := &captureAuditRepo{}
	svc.SetAuditLogger(auditSink)

	_, err := svc.Subscribe(context.Background(), appsub.SubscribeInput{
		OrganizationID: *users.user.OrganizationID,
		ActorUserID:    users.user.ID,
		Plan:           domain.PlanFreelance,
		BillingCycle:   domain.CycleMonthly,
	})

	assert.ErrorIs(t, err, domain.ErrAlreadySubscribed)
	require.Len(t, auditSink.logged, 1, "an audit row MUST be written when a dup is detected")
	entry := auditSink.logged[0]
	assert.Equal(t, audit.Action("subscription.duplicate_detected"), entry.Action)
	assert.Equal(t, audit.ResourceType("subscription"), entry.ResourceType)
	require.NotNil(t, entry.UserID)
	assert.Equal(t, users.user.ID, *entry.UserID)
	require.NotNil(t, entry.ResourceID)
	assert.Equal(t, existing.ID, *entry.ResourceID)
	assert.Equal(t, "subscribe_blocked", entry.Metadata["stage"])
}

// TestSubscribe_AuditFailureNeverBlocksSubscribe — defence-in-depth:
// a downed audit DB MUST not prevent the legitimate domain error from
// being returned to the caller. (The duplicate guard already short-
// circuits Stripe; we just need to make sure the error pipeline is
// not swallowed by an audit-side panic.)
func TestSubscribe_AuditFailureNeverBlocksSubscribe(t *testing.T) {
	svc, subs, users, _, _ := newTestService()
	existing, _ := domain.NewSubscription(freshDomainInput(*users.user.OrganizationID))
	subs.findOpenByOrgFn = func(_ context.Context, _ uuid.UUID) (*domain.Subscription, error) {
		return existing, nil
	}
	svc.SetAuditLogger(&captureAuditRepo{err: errors.New("audit DB down")})

	_, err := svc.Subscribe(context.Background(), appsub.SubscribeInput{
		OrganizationID: *users.user.OrganizationID,
		ActorUserID:    users.user.ID,
		Plan:           domain.PlanFreelance,
		BillingCycle:   domain.CycleMonthly,
	})

	assert.ErrorIs(t, err, domain.ErrAlreadySubscribed,
		"a failing audit sink MUST NOT change the error returned to the user")
}

// ===========================================================================
// Webhook reconciliation — RegisterFromCheckout race protection
// ===========================================================================

// TestRegisterFromCheckout_NewSubReplacesExistingDuplicate is the
// critical test for the production bug that motivated this work:
// a user ends up with two active Stripe subscriptions for the same
// org (e.g. Stripe CLI on the wrong account, or a dup checkout that
// slipped past the idempotency key). When the second
// customer.subscription.created lands, the older one MUST be
// cancelled in Stripe AND marked canceled locally, then the new one
// MUST be persisted. End state: exactly ONE active subscription.
func TestRegisterFromCheckout_NewSubReplacesExistingDuplicate(t *testing.T) {
	svc, subs, users, _, stripe := newTestService()

	// Pre-existing active subscription on the org (the OLD one).
	old, _ := domain.NewSubscription(freshDomainInput(*users.user.OrganizationID))
	old.StripeSubscriptionID = "sub_old_xxx"
	require.NoError(t, old.Activate())
	subs.findOpenByOrgFn = func(_ context.Context, _ uuid.UUID) (*domain.Subscription, error) {
		return old, nil
	}

	var persistedNew *domain.Subscription
	subs.createFn = func(_ context.Context, s *domain.Subscription) error {
		persistedNew = s
		return nil
	}
	var oldUpdated *domain.Subscription
	subs.updateFn = func(_ context.Context, s *domain.Subscription) error {
		oldUpdated = s
		return nil
	}

	// New subscription event lands.
	newSnap := service.SubscriptionSnapshot{
		ID:                 "sub_new_yyy",
		Status:             "active",
		PriceID:            "price_premium_freelance_monthly",
		CurrentPeriodStart: time.Now(),
		CurrentPeriodEnd:   time.Now().Add(30 * 24 * time.Hour),
	}
	err := svc.RegisterFromCheckout(
		context.Background(),
		*users.user.OrganizationID,
		domain.PlanFreelance,
		domain.CycleMonthly,
		"cus_123",
		newSnap,
	)

	require.NoError(t, err)
	// 1. The OLD Stripe sub was cancelled.
	require.Equal(t, []string{"sub_old_xxx"}, stripe.cancelSubscriptionCalls,
		"the older Stripe subscription MUST be cancelled when a duplicate lands")
	// 2. The OLD local row was persisted as canceled.
	require.NotNil(t, oldUpdated, "the old row must be persisted with canceled status")
	assert.Equal(t, domain.StatusCanceled, oldUpdated.Status)
	// 3. The NEW row was persisted fresh + active.
	require.NotNil(t, persistedNew)
	assert.Equal(t, "sub_new_yyy", persistedNew.StripeSubscriptionID)
	assert.Equal(t, domain.StatusActive, persistedNew.Status)
}

// TestRegisterFromCheckout_SameStripeIDIsNoOpReplay verifies that a
// Stripe webhook RETRY (same event delivered again because we 5xx'd
// on the first attempt) does NOT trigger a cancel-and-replace cycle:
// the existing row is the SAME subscription, not a duplicate.
func TestRegisterFromCheckout_SameStripeIDIsNoOpReplay(t *testing.T) {
	svc, subs, users, _, stripe := newTestService()

	existing, _ := domain.NewSubscription(freshDomainInput(*users.user.OrganizationID))
	existing.StripeSubscriptionID = "sub_same_zzz"
	require.NoError(t, existing.Activate())
	subs.findOpenByOrgFn = func(_ context.Context, _ uuid.UUID) (*domain.Subscription, error) {
		return existing, nil
	}
	// Create returns a unique-violation surrogate to simulate the row
	// being already persisted; the test only asserts cancel was NOT called.
	subs.createFn = func(_ context.Context, _ *domain.Subscription) error {
		return errors.New("duplicate key value violates unique constraint subscriptions_stripe_subscription_id_key")
	}

	_ = svc.RegisterFromCheckout(
		context.Background(),
		*users.user.OrganizationID,
		domain.PlanFreelance,
		domain.CycleMonthly,
		"cus_123",
		service.SubscriptionSnapshot{
			ID: "sub_same_zzz", Status: "active",
			PriceID:            "price_x",
			CurrentPeriodStart: time.Now(),
			CurrentPeriodEnd:   time.Now().Add(30 * 24 * time.Hour),
		},
	)

	assert.Empty(t, stripe.cancelSubscriptionCalls,
		"a Stripe webhook RETRY for the SAME subscription id MUST NOT trigger cancellation")
}

// TestRegisterFromCheckout_NoExistingSub_HappyPath confirms the
// reconciliation path is a true no-op when the org is on free tier.
func TestRegisterFromCheckout_NoExistingSub_HappyPath(t *testing.T) {
	svc, subs, users, _, stripe := newTestService()
	subs.findOpenByOrgFn = func(_ context.Context, _ uuid.UUID) (*domain.Subscription, error) {
		return nil, domain.ErrNotFound
	}
	var persisted *domain.Subscription
	subs.createFn = func(_ context.Context, s *domain.Subscription) error {
		persisted = s
		return nil
	}

	err := svc.RegisterFromCheckout(
		context.Background(),
		*users.user.OrganizationID,
		domain.PlanFreelance,
		domain.CycleMonthly,
		"cus_456",
		service.SubscriptionSnapshot{
			ID: "sub_solo", Status: "active", PriceID: "price_x",
			CurrentPeriodStart: time.Now(),
			CurrentPeriodEnd:   time.Now().Add(30 * 24 * time.Hour),
		},
	)

	require.NoError(t, err)
	require.NotNil(t, persisted)
	assert.Equal(t, "sub_solo", persisted.StripeSubscriptionID)
	assert.Empty(t, stripe.cancelSubscriptionCalls,
		"no cancellation when there is no existing duplicate")
}

// TestRegisterFromCheckout_StripeCancelFailureStillCancelsLocal
// guards a partial-failure path: Stripe rejects the cancellation
// (network blip, sub already canceled, etc.) but the LOCAL row must
// still be marked canceled so the new row can be inserted past the
// partial unique index.
func TestRegisterFromCheckout_StripeCancelFailureStillCancelsLocal(t *testing.T) {
	svc, subs, users, _, stripe := newTestService()
	old, _ := domain.NewSubscription(freshDomainInput(*users.user.OrganizationID))
	old.StripeSubscriptionID = "sub_old_aaa"
	require.NoError(t, old.Activate())
	subs.findOpenByOrgFn = func(_ context.Context, _ uuid.UUID) (*domain.Subscription, error) {
		return old, nil
	}

	stripe.cancelSubscriptionFn = func(_ context.Context, _ string) error {
		return errors.New("stripe API hiccup")
	}
	var oldUpdated *domain.Subscription
	subs.updateFn = func(_ context.Context, s *domain.Subscription) error {
		oldUpdated = s
		return nil
	}
	subs.createFn = func(_ context.Context, _ *domain.Subscription) error { return nil }

	err := svc.RegisterFromCheckout(
		context.Background(),
		*users.user.OrganizationID,
		domain.PlanFreelance,
		domain.CycleMonthly,
		"cus_789",
		service.SubscriptionSnapshot{
			ID: "sub_new_bbb", Status: "active", PriceID: "price_x",
			CurrentPeriodStart: time.Now(),
			CurrentPeriodEnd:   time.Now().Add(30 * 24 * time.Hour),
		},
	)

	require.NoError(t, err, "the new sub registration must still succeed despite Stripe cancel failure")
	require.NotNil(t, oldUpdated)
	assert.Equal(t, domain.StatusCanceled, oldUpdated.Status,
		"local row MUST be canceled even when Stripe cancellation fails — otherwise the partial unique index blocks the new insert")
}

// TestRegisterFromCheckout_DuplicateEmitsAuditLog asserts the audit
// trail is written when the webhook reconciliation kicks in. This is
// the breadcrumb support uses to correlate Stripe-side anomalies with
// "user reports being billed twice".
func TestRegisterFromCheckout_DuplicateEmitsAuditLog(t *testing.T) {
	svc, subs, users, _, _ := newTestService()
	auditSink := &captureAuditRepo{}
	svc.SetAuditLogger(auditSink)

	old, _ := domain.NewSubscription(freshDomainInput(*users.user.OrganizationID))
	old.StripeSubscriptionID = "sub_old_aud"
	require.NoError(t, old.Activate())
	subs.findOpenByOrgFn = func(_ context.Context, _ uuid.UUID) (*domain.Subscription, error) {
		return old, nil
	}
	subs.createFn = func(_ context.Context, _ *domain.Subscription) error { return nil }

	err := svc.RegisterFromCheckout(
		context.Background(),
		*users.user.OrganizationID,
		domain.PlanFreelance,
		domain.CycleMonthly,
		"cus_aud",
		service.SubscriptionSnapshot{
			ID: "sub_new_aud", Status: "active", PriceID: "price_x",
			CurrentPeriodStart: time.Now(),
			CurrentPeriodEnd:   time.Now().Add(30 * 24 * time.Hour),
		},
	)
	require.NoError(t, err)

	require.Len(t, auditSink.logged, 1)
	assert.Equal(t, audit.Action("subscription.duplicate_detected"), auditSink.logged[0].Action)
	assert.Equal(t, "webhook_replace", auditSink.logged[0].Metadata["stage"])
	assert.Equal(t, "sub_old_aud", auditSink.logged[0].Metadata["existing_stripe_subscription_id"])
}
