package subscription_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appsub "marketplace-backend/internal/app/subscription"
	domain "marketplace-backend/internal/domain/subscription"
	domainuser "marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/port/service"
)

// newTestService wires a Service with fresh mocks. Tests mutate the
// returned mock fields to inject specific behaviour.
func newTestService() (*appsub.Service, *mockSubRepo, *mockUserRepo, *mockAmountsReader, *mockStripe) {
	subs := &mockSubRepo{}
	users := &mockUserRepo{
		user: &domainuser.User{
			ID:          uuid.New(),
			Email:       "test@example.com",
			FirstName:   "Test",
			LastName:    "User",
			DisplayName: "Test User",
			Role:        domainuser.RoleProvider,
		},
	}
	amounts := &mockAmountsReader{}
	stripe := &mockStripe{}

	svc := appsub.NewService(appsub.ServiceDeps{
		Subscriptions: subs,
		Users:         users,
		Amounts:       amounts,
		Stripe:        stripe,
		LookupKeys:    appsub.DefaultLookupKeys(),
		URLs: appsub.URLs{
			CheckoutSuccess: "https://app.test/billing/success",
			CheckoutCancel:  "https://app.test/billing/cancel",
			PortalReturn:    "https://app.test/billing",
		},
	})
	return svc, subs, users, amounts, stripe
}

// ---------- Subscribe ----------

func TestSubscribe_HappyPath_FreelanceMonthlyNoAutoRenew(t *testing.T) {
	svc, _, users, _, stripe := newTestService()

	out, err := svc.Subscribe(context.Background(), appsub.SubscribeInput{
		UserID:       users.user.ID,
		Plan:         domain.PlanFreelance,
		BillingCycle: domain.CycleMonthly,
		AutoRenew:    false,
	})

	require.NoError(t, err)
	require.NotEmpty(t, out.CheckoutURL)
	require.NotNil(t, stripe.lastCreateCheckoutInput)
	assert.True(t, stripe.lastCreateCheckoutInput.CancelAtPeriodEnd,
		"AutoRenew=false MUST send cancel_at_period_end=true to Stripe")
	assert.Contains(t, stripe.lastCreateCheckoutInput.PriceID, "premium_freelance_monthly")
}

func TestSubscribe_HappyPath_AgencyAnnualAutoRenewOn(t *testing.T) {
	svc, _, users, _, stripe := newTestService()
	users.user.Role = domainuser.RoleAgency

	out, err := svc.Subscribe(context.Background(), appsub.SubscribeInput{
		UserID:       users.user.ID,
		Plan:         domain.PlanAgency,
		BillingCycle: domain.CycleAnnual,
		AutoRenew:    true,
	})

	require.NoError(t, err)
	require.NotEmpty(t, out.CheckoutURL)
	assert.False(t, stripe.lastCreateCheckoutInput.CancelAtPeriodEnd,
		"AutoRenew=true MUST send cancel_at_period_end=false to Stripe")
	assert.Contains(t, stripe.lastCreateCheckoutInput.PriceID, "premium_agency_annual")
}

func TestSubscribe_RejectsInvalidPlan(t *testing.T) {
	svc, _, users, _, _ := newTestService()

	_, err := svc.Subscribe(context.Background(), appsub.SubscribeInput{
		UserID:       users.user.ID,
		Plan:         "enterprise",
		BillingCycle: domain.CycleMonthly,
	})

	assert.ErrorIs(t, err, domain.ErrInvalidPlan)
}

func TestSubscribe_RejectsInvalidCycle(t *testing.T) {
	svc, _, users, _, _ := newTestService()

	_, err := svc.Subscribe(context.Background(), appsub.SubscribeInput{
		UserID:       users.user.ID,
		Plan:         domain.PlanFreelance,
		BillingCycle: "weekly",
	})

	assert.ErrorIs(t, err, domain.ErrInvalidCycle)
}

func TestSubscribe_RejectsWhenAlreadySubscribed(t *testing.T) {
	svc, subs, users, _, _ := newTestService()
	existing, _ := domain.NewSubscription(freshDomainInput(users.user.ID))
	subs.findOpenByUserFn = func(ctx context.Context, _ uuid.UUID) (*domain.Subscription, error) {
		return existing, nil
	}

	_, err := svc.Subscribe(context.Background(), appsub.SubscribeInput{
		UserID:       users.user.ID,
		Plan:         domain.PlanFreelance,
		BillingCycle: domain.CycleMonthly,
	})

	assert.ErrorIs(t, err, domain.ErrAlreadySubscribed)
}

// ---------- GetStatus ----------

func TestGetStatus_ReturnsSubscription(t *testing.T) {
	svc, subs, users, _, _ := newTestService()
	existing, _ := domain.NewSubscription(freshDomainInput(users.user.ID))
	subs.findOpenByUserFn = func(ctx context.Context, _ uuid.UUID) (*domain.Subscription, error) {
		return existing, nil
	}

	got, err := svc.GetStatus(context.Background(), users.user.ID)

	require.NoError(t, err)
	assert.Equal(t, existing.ID, got.ID)
}

func TestGetStatus_NotFoundWhenFree(t *testing.T) {
	svc, _, users, _, _ := newTestService()

	_, err := svc.GetStatus(context.Background(), users.user.ID)

	assert.ErrorIs(t, err, domain.ErrNotFound)
}

// ---------- ToggleAutoRenew ----------

func TestToggleAutoRenew_BothDirections(t *testing.T) {
	tests := []struct {
		name   string
		turnOn bool
		// What we expect to see as cancel_at_period_end in the persisted row
		// AFTER the toggle has been applied.
		wantCancelAtEnd bool
	}{
		{"turn on", true, false},
		{"turn off", false, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc, subs, users, _, stripe := newTestService()
			existing, _ := domain.NewSubscription(freshDomainInput(users.user.ID))
			_ = existing.Activate()
			// Start in the opposite state of what we're about to toggle to
			// so we observe a real transition.
			existing.CancelAtPeriodEnd = !tc.wantCancelAtEnd

			subs.findOpenByUserFn = func(ctx context.Context, _ uuid.UUID) (*domain.Subscription, error) {
				return existing, nil
			}
			stripe.updateCancelAtPeriodEndFn = func(ctx context.Context, stripeSubID string, cancelAtEnd bool) (service.SubscriptionSnapshot, error) {
				return service.SubscriptionSnapshot{
					ID:                stripeSubID,
					Status:            "active",
					CancelAtPeriodEnd: cancelAtEnd,
					CurrentPeriodStart: time.Now(),
					CurrentPeriodEnd:   time.Now().Add(30 * 24 * time.Hour),
				}, nil
			}

			got, err := svc.ToggleAutoRenew(context.Background(), users.user.ID, tc.turnOn)

			require.NoError(t, err)
			assert.Equal(t, tc.wantCancelAtEnd, got.CancelAtPeriodEnd)
		})
	}
}

func TestToggleAutoRenew_NoSubscription(t *testing.T) {
	svc, _, users, _, _ := newTestService()

	_, err := svc.ToggleAutoRenew(context.Background(), users.user.ID, true)

	assert.ErrorIs(t, err, domain.ErrNotFound)
}

// ---------- ChangeCycle ----------

func TestChangeCycle_MonthlyToAnnual(t *testing.T) {
	svc, subs, users, _, stripe := newTestService()
	in := freshDomainInput(users.user.ID)
	in.BillingCycle = domain.CycleMonthly
	existing, _ := domain.NewSubscription(in)
	_ = existing.Activate()
	subs.findOpenByUserFn = func(ctx context.Context, _ uuid.UUID) (*domain.Subscription, error) {
		return existing, nil
	}
	stripe.changeCycleFn = func(ctx context.Context, stripeSubID, newPriceID string) (service.SubscriptionSnapshot, error) {
		return service.SubscriptionSnapshot{
			ID:                 stripeSubID,
			Status:             "active",
			PriceID:            newPriceID,
			CurrentPeriodStart: time.Now(),
			CurrentPeriodEnd:   time.Now().Add(365 * 24 * time.Hour),
		}, nil
	}

	got, err := svc.ChangeCycle(context.Background(), users.user.ID, domain.CycleAnnual)

	require.NoError(t, err)
	assert.Equal(t, domain.CycleAnnual, got.BillingCycle)
}

func TestChangeCycle_AnnualToMonthly(t *testing.T) {
	svc, subs, users, _, _ := newTestService()
	in := freshDomainInput(users.user.ID)
	in.BillingCycle = domain.CycleAnnual
	existing, _ := domain.NewSubscription(in)
	_ = existing.Activate()
	subs.findOpenByUserFn = func(ctx context.Context, _ uuid.UUID) (*domain.Subscription, error) {
		return existing, nil
	}

	got, err := svc.ChangeCycle(context.Background(), users.user.ID, domain.CycleMonthly)

	require.NoError(t, err)
	assert.Equal(t, domain.CycleMonthly, got.BillingCycle)
}

func TestChangeCycle_SameCycle(t *testing.T) {
	svc, subs, users, _, _ := newTestService()
	existing, _ := domain.NewSubscription(freshDomainInput(users.user.ID))
	_ = existing.Activate()
	subs.findOpenByUserFn = func(ctx context.Context, _ uuid.UUID) (*domain.Subscription, error) {
		return existing, nil
	}

	_, err := svc.ChangeCycle(context.Background(), users.user.ID, existing.BillingCycle)

	assert.ErrorIs(t, err, domain.ErrSameCycle)
}

func TestChangeCycle_InvalidCycle(t *testing.T) {
	svc, _, users, _, _ := newTestService()

	_, err := svc.ChangeCycle(context.Background(), users.user.ID, "weekly")

	assert.ErrorIs(t, err, domain.ErrInvalidCycle)
}

// ---------- GetStats ----------

func TestGetStats_ComputesSavingsAcrossTiers(t *testing.T) {
	svc, subs, users, amounts, _ := newTestService()
	existing, _ := domain.NewSubscription(freshDomainInput(users.user.ID))
	_ = existing.Activate()
	subs.findOpenByUserFn = func(ctx context.Context, _ uuid.UUID) (*domain.Subscription, error) {
		return existing, nil
	}
	// Freelance role — tiers 9€/15€/25€
	amounts.amounts = []int64{
		10000,  // 100€   → tier 1 → 9€ saved
		50000,  // 500€   → tier 2 → 15€ saved
		200000, // 2000€  → tier 3 → 25€ saved
	}

	stats, err := svc.GetStats(context.Background(), users.user.ID)

	require.NoError(t, err)
	assert.Equal(t, int64(900+1500+2500), stats.SavedFeeCents)
	assert.Equal(t, 3, stats.SavedCount)
}

func TestGetStats_EmptyHistory(t *testing.T) {
	svc, subs, users, amounts, _ := newTestService()
	existing, _ := domain.NewSubscription(freshDomainInput(users.user.ID))
	_ = existing.Activate()
	subs.findOpenByUserFn = func(ctx context.Context, _ uuid.UUID) (*domain.Subscription, error) {
		return existing, nil
	}
	amounts.amounts = nil

	stats, err := svc.GetStats(context.Background(), users.user.ID)

	require.NoError(t, err)
	assert.Equal(t, int64(0), stats.SavedFeeCents)
	assert.Equal(t, 0, stats.SavedCount)
}

func TestGetStats_AgencyTierGrid(t *testing.T) {
	svc, subs, users, amounts, _ := newTestService()
	users.user.Role = domainuser.RoleAgency
	existing, _ := domain.NewSubscription(freshDomainInput(users.user.ID))
	_ = existing.Activate()
	subs.findOpenByUserFn = func(ctx context.Context, _ uuid.UUID) (*domain.Subscription, error) {
		return existing, nil
	}
	// Agency role — tiers 19€/39€/69€
	amounts.amounts = []int64{
		30000,  // 300€   → tier 1 → 19€
		100000, // 1000€  → tier 2 → 39€
		500000, // 5000€  → tier 3 → 69€
	}

	stats, err := svc.GetStats(context.Background(), users.user.ID)

	require.NoError(t, err)
	assert.Equal(t, int64(1900+3900+6900), stats.SavedFeeCents)
}

// ---------- GetPortalURL ----------

func TestGetPortalURL_HappyPath(t *testing.T) {
	svc, subs, users, _, _ := newTestService()
	existing, _ := domain.NewSubscription(freshDomainInput(users.user.ID))
	_ = existing.Activate()
	subs.findOpenByUserFn = func(ctx context.Context, _ uuid.UUID) (*domain.Subscription, error) {
		return existing, nil
	}

	url, err := svc.GetPortalURL(context.Background(), users.user.ID)

	require.NoError(t, err)
	assert.Contains(t, url, "portal.stripe.test")
}

func TestGetPortalURL_NoSubscription(t *testing.T) {
	svc, _, users, _, _ := newTestService()

	_, err := svc.GetPortalURL(context.Background(), users.user.ID)

	assert.ErrorIs(t, err, domain.ErrNotFound)
}

// ---------- IsActive (SubscriptionReader implementation) ----------

func TestIsActive_ActiveSubscription(t *testing.T) {
	svc, subs, users, _, _ := newTestService()
	existing, _ := domain.NewSubscription(freshDomainInput(users.user.ID))
	_ = existing.Activate()
	subs.findOpenByUserFn = func(ctx context.Context, _ uuid.UUID) (*domain.Subscription, error) {
		return existing, nil
	}

	active, err := svc.IsActive(context.Background(), users.user.ID)

	require.NoError(t, err)
	assert.True(t, active)
}

func TestIsActive_NoSubscription_ReturnsFalseWithoutError(t *testing.T) {
	svc, _, users, _, _ := newTestService()

	active, err := svc.IsActive(context.Background(), users.user.ID)

	require.NoError(t, err, "free users must NOT be an error path")
	assert.False(t, active)
}

func TestIsActive_FailsOpenOnRepositoryError(t *testing.T) {
	// Transient DB error must surface AND return active=false so the
	// caller applies the normal fee. Never grant Premium under doubt.
	svc, subs, users, _, _ := newTestService()
	subs.findOpenByUserFn = func(ctx context.Context, _ uuid.UUID) (*domain.Subscription, error) {
		return nil, errors.New("db connection lost")
	}

	active, err := svc.IsActive(context.Background(), users.user.ID)

	require.Error(t, err)
	assert.False(t, active, "error MUST fail closed (no Premium granted)")
}

// ---------- HandleSubscriptionSnapshot (webhook entry) ----------

func TestHandleSubscriptionSnapshot_ActivatesFromIncomplete(t *testing.T) {
	svc, subs, users, _, _ := newTestService()
	existing, _ := domain.NewSubscription(freshDomainInput(users.user.ID))
	existing.StripeSubscriptionID = "sub_xyz"
	subs.findByStripeIDFn = func(ctx context.Context, _ string) (*domain.Subscription, error) {
		return existing, nil
	}

	err := svc.HandleSubscriptionSnapshot(context.Background(), service.SubscriptionSnapshot{
		ID:                 "sub_xyz",
		Status:             "active",
		PriceID:            "price_test",
		CurrentPeriodStart: time.Now(),
		CurrentPeriodEnd:   time.Now().Add(30 * 24 * time.Hour),
		CancelAtPeriodEnd:  true,
	}, false)

	require.NoError(t, err)
	assert.Equal(t, domain.StatusActive, existing.Status)
}

func TestHandleSubscriptionSnapshot_PastDueSetsGrace(t *testing.T) {
	svc, subs, users, _, _ := newTestService()
	existing, _ := domain.NewSubscription(freshDomainInput(users.user.ID))
	_ = existing.Activate()
	subs.findByStripeIDFn = func(ctx context.Context, _ string) (*domain.Subscription, error) {
		return existing, nil
	}

	err := svc.HandleSubscriptionSnapshot(context.Background(), service.SubscriptionSnapshot{
		ID:                 existing.StripeSubscriptionID,
		Status:             "past_due",
		CurrentPeriodStart: time.Now(),
		CurrentPeriodEnd:   time.Now().Add(30 * 24 * time.Hour),
	}, false)

	require.NoError(t, err)
	assert.Equal(t, domain.StatusPastDue, existing.Status)
	assert.NotNil(t, existing.GracePeriodEndsAt)
}

func TestHandleSubscriptionSnapshot_DeletedTransitions(t *testing.T) {
	svc, subs, users, _, _ := newTestService()
	existing, _ := domain.NewSubscription(freshDomainInput(users.user.ID))
	_ = existing.Activate()
	subs.findByStripeIDFn = func(ctx context.Context, _ string) (*domain.Subscription, error) {
		return existing, nil
	}

	err := svc.HandleSubscriptionSnapshot(context.Background(), service.SubscriptionSnapshot{
		ID: existing.StripeSubscriptionID,
	}, true)

	require.NoError(t, err)
	assert.Equal(t, domain.StatusCanceled, existing.Status)
	assert.NotNil(t, existing.CanceledAt)
}

func TestHandleSubscriptionSnapshot_UnknownStripeID_NoOp(t *testing.T) {
	svc, _, _, _, _ := newTestService()

	err := svc.HandleSubscriptionSnapshot(context.Background(), service.SubscriptionSnapshot{
		ID:     "sub_never_saw_this",
		Status: "active",
	}, false)

	require.NoError(t, err, "unknown stripe ids must be ignored, not errored")
}

func TestHandleSubscriptionSnapshot_Idempotent(t *testing.T) {
	// Replaying the same webhook event MUST NOT corrupt state. We model
	// this as: a canceled row can receive a canceled snapshot again with
	// no error (no state change either).
	svc, subs, users, _, _ := newTestService()
	existing, _ := domain.NewSubscription(freshDomainInput(users.user.ID))
	_ = existing.Activate()
	_ = existing.MarkCanceled()
	firstCanceledAt := existing.CanceledAt
	subs.findByStripeIDFn = func(ctx context.Context, _ string) (*domain.Subscription, error) {
		return existing, nil
	}

	err := svc.HandleSubscriptionSnapshot(context.Background(), service.SubscriptionSnapshot{
		ID: existing.StripeSubscriptionID,
	}, true)

	require.NoError(t, err)
	assert.Equal(t, firstCanceledAt, existing.CanceledAt, "CanceledAt must not be bumped on replay")
}

// ---------- RegisterFromCheckout ----------

func TestRegisterFromCheckout_CreatesActiveRow(t *testing.T) {
	svc, subs, users, _, _ := newTestService()
	var persisted *domain.Subscription
	subs.createFn = func(ctx context.Context, s *domain.Subscription) error {
		persisted = s
		return nil
	}

	err := svc.RegisterFromCheckout(
		context.Background(),
		users.user.ID,
		domain.PlanFreelance,
		domain.CycleMonthly,
		"cus_123",
		service.SubscriptionSnapshot{
			ID:                 "sub_456",
			Status:             "active",
			PriceID:            "price_789",
			CurrentPeriodStart: time.Now(),
			CurrentPeriodEnd:   time.Now().Add(30 * 24 * time.Hour),
			CancelAtPeriodEnd:  true,
		},
	)

	require.NoError(t, err)
	require.NotNil(t, persisted)
	assert.Equal(t, domain.StatusActive, persisted.Status)
	assert.True(t, persisted.CancelAtPeriodEnd, "cancel_at_period_end mirrored from Stripe snapshot")
	assert.Equal(t, "sub_456", persisted.StripeSubscriptionID)
}

// freshDomainInput builds a valid NewSubscriptionInput for a given user.
func freshDomainInput(userID uuid.UUID) domain.NewSubscriptionInput {
	now := time.Now()
	return domain.NewSubscriptionInput{
		UserID:               userID,
		Plan:                 domain.PlanFreelance,
		BillingCycle:         domain.CycleMonthly,
		StripeCustomerID:     "cus_test",
		StripeSubscriptionID: "sub_test_" + uuid.New().String(),
		StripePriceID:        "price_test",
		CurrentPeriodStart:   now,
		CurrentPeriodEnd:     now.Add(30 * 24 * time.Hour),
		CancelAtPeriodEnd:    true,
	}
}
