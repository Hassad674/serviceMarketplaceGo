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
	orgID := uuid.New()
	users := &mockUserRepo{
		user: &domainuser.User{
			ID:             uuid.New(),
			Email:          "test@example.com",
			FirstName:      "Test",
			LastName:       "User",
			DisplayName:    "Test User",
			Role:           domainuser.RoleProvider,
			OrganizationID: &orgID,
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
			CheckoutReturn: "https://app.test/subscribe/return?session_id={CHECKOUT_SESSION_ID}",
			PortalReturn:   "https://app.test/billing",
		},
	})
	return svc, subs, users, amounts, stripe
}

// ---------- Subscribe ----------

func TestSubscribe_HappyPath_FreelanceMonthlyNoAutoRenew(t *testing.T) {
	svc, _, users, _, stripe := newTestService()

	out, err := svc.Subscribe(context.Background(), appsub.SubscribeInput{
		OrganizationID: *users.user.OrganizationID,
		ActorUserID:    users.user.ID,
		Plan:         domain.PlanFreelance,
		BillingCycle: domain.CycleMonthly,
		AutoRenew:    false,
	})

	require.NoError(t, err)
	require.NotEmpty(t, out.ClientSecret)
	require.NotNil(t, stripe.lastCreateCheckoutInput)
	assert.True(t, stripe.lastCreateCheckoutInput.CancelAtPeriodEnd,
		"AutoRenew=false MUST send cancel_at_period_end=true to Stripe")
	assert.Contains(t, stripe.lastCreateCheckoutInput.PriceID, "premium_freelance_monthly")
}

func TestSubscribe_HappyPath_AgencyAnnualAutoRenewOn(t *testing.T) {
	svc, _, users, _, stripe := newTestService()
	users.user.Role = domainuser.RoleAgency

	out, err := svc.Subscribe(context.Background(), appsub.SubscribeInput{
		OrganizationID: *users.user.OrganizationID,
		ActorUserID:    users.user.ID,
		Plan:         domain.PlanAgency,
		BillingCycle: domain.CycleAnnual,
		AutoRenew:    true,
	})

	require.NoError(t, err)
	require.NotEmpty(t, out.ClientSecret)
	assert.False(t, stripe.lastCreateCheckoutInput.CancelAtPeriodEnd,
		"AutoRenew=true MUST send cancel_at_period_end=false to Stripe")
	assert.Contains(t, stripe.lastCreateCheckoutInput.PriceID, "premium_agency_annual")
}

func TestSubscribe_RejectsInvalidPlan(t *testing.T) {
	svc, _, users, _, _ := newTestService()

	_, err := svc.Subscribe(context.Background(), appsub.SubscribeInput{
		OrganizationID: *users.user.OrganizationID,
		ActorUserID:    users.user.ID,
		Plan:         "enterprise",
		BillingCycle: domain.CycleMonthly,
	})

	assert.ErrorIs(t, err, domain.ErrInvalidPlan)
}

func TestSubscribe_RejectsInvalidCycle(t *testing.T) {
	svc, _, users, _, _ := newTestService()

	_, err := svc.Subscribe(context.Background(), appsub.SubscribeInput{
		OrganizationID: *users.user.OrganizationID,
		ActorUserID:    users.user.ID,
		Plan:         domain.PlanFreelance,
		BillingCycle: "weekly",
	})

	assert.ErrorIs(t, err, domain.ErrInvalidCycle)
}

// ---------- Customer enrichment from billing_profile ----------

// fakeBillingReader returns a configurable snapshot to drive the
// enrichment branches inside Subscribe. errFn is set when a test wants
// to assert the error path is logged + swallowed (best-effort).
type fakeBillingReader struct {
	snap service.BillingProfileStripeSnapshot
	err  error
}

func (f *fakeBillingReader) GetBillingProfileSnapshotForStripe(_ context.Context, _ uuid.UUID) (service.BillingProfileStripeSnapshot, error) {
	return f.snap, f.err
}

func TestSubscribe_PushesBillingProfileToStripeBeforeCreatingSession(t *testing.T) {
	// Happy path: a billing profile reader returning a populated
	// snapshot must trigger exactly one EnrichCustomerWithBillingProfile
	// call, with the snapshot as-is, BEFORE CreateCheckoutSession runs.
	svc, _, users, _, stripe := newTestService()
	reader := &fakeBillingReader{
		snap: service.BillingProfileStripeSnapshot{
			LegalName:    "Acme Studio",
			AddressLine1: "1 rue de la Paix",
			PostalCode:   "75001",
			City:         "Paris",
			Country:      "FR",
		},
	}
	svc.SetBillingProfileReader(reader)

	out, err := svc.Subscribe(context.Background(), appsub.SubscribeInput{
		OrganizationID: *users.user.OrganizationID,
		ActorUserID:    users.user.ID,
		Plan:           domain.PlanFreelance,
		BillingCycle:   domain.CycleMonthly,
	})

	require.NoError(t, err)
	require.NotEmpty(t, out.ClientSecret)
	assert.Equal(t, 1, stripe.enrichCallCount, "enrichment must run exactly once")
	require.NotNil(t, stripe.lastEnrichSnapshot)
	assert.Equal(t, "Acme Studio", stripe.lastEnrichSnapshot.LegalName)
	assert.Equal(t, "FR", stripe.lastEnrichSnapshot.Country)
}

func TestSubscribe_SkipsStripeEnrichmentWhenSnapshotIsEmpty(t *testing.T) {
	// First-time user: the org has no billing profile yet. The reader
	// returns a zero-value snapshot — Subscribe must skip the Stripe
	// Customer.Update entirely so we don't push empty strings that
	// would clear the customer's existing fields.
	svc, _, users, _, stripe := newTestService()
	svc.SetBillingProfileReader(&fakeBillingReader{}) // zero-value snapshot

	out, err := svc.Subscribe(context.Background(), appsub.SubscribeInput{
		OrganizationID: *users.user.OrganizationID,
		ActorUserID:    users.user.ID,
		Plan:           domain.PlanFreelance,
		BillingCycle:   domain.CycleMonthly,
	})

	require.NoError(t, err)
	require.NotEmpty(t, out.ClientSecret)
	assert.Zero(t, stripe.enrichCallCount, "must not call enrichment with an empty snapshot")
}

func TestSubscribe_LogsAndContinuesWhenStripeEnrichmentFails(t *testing.T) {
	// Defensive: a failure on the Stripe Customer.Update side MUST NOT
	// block the subscribe. The Embedded Checkout still works without
	// the pre-fill, the user just gets a less-prefilled form.
	svc, _, users, _, stripe := newTestService()
	svc.SetBillingProfileReader(&fakeBillingReader{
		snap: service.BillingProfileStripeSnapshot{LegalName: "Acme"},
	})
	stripe.enrichCustomerFn = func(_ context.Context, _ string, _ service.BillingProfileStripeSnapshot) error {
		return errors.New("stripe API hiccup")
	}

	out, err := svc.Subscribe(context.Background(), appsub.SubscribeInput{
		OrganizationID: *users.user.OrganizationID,
		ActorUserID:    users.user.ID,
		Plan:           domain.PlanFreelance,
		BillingCycle:   domain.CycleMonthly,
	})

	require.NoError(t, err, "enrichment failure MUST NOT block the subscribe")
	require.NotEmpty(t, out.ClientSecret)
}

func TestSubscribe_LogsAndContinuesWhenSnapshotReadFails(t *testing.T) {
	// Defensive: a failure inside the invoicing module's snapshot read
	// is also best-effort — Subscribe continues and Stripe gets no
	// customer enrichment for this attempt.
	svc, _, users, _, stripe := newTestService()
	svc.SetBillingProfileReader(&fakeBillingReader{err: errors.New("db unavailable")})

	out, err := svc.Subscribe(context.Background(), appsub.SubscribeInput{
		OrganizationID: *users.user.OrganizationID,
		ActorUserID:    users.user.ID,
		Plan:           domain.PlanFreelance,
		BillingCycle:   domain.CycleMonthly,
	})

	require.NoError(t, err)
	require.NotEmpty(t, out.ClientSecret)
	assert.Zero(t, stripe.enrichCallCount, "snapshot error must short-circuit before Stripe Customer.Update")
}

func TestSubscribe_NoBillingProfileReaderConfigured(t *testing.T) {
	// Removable-by-design: when invoicing is disabled (reader nil),
	// Subscribe still works — Stripe's customer is whatever it is and
	// the embedded form falls back to what it already has.
	svc, _, users, _, stripe := newTestService() // reader stays nil

	out, err := svc.Subscribe(context.Background(), appsub.SubscribeInput{
		OrganizationID: *users.user.OrganizationID,
		ActorUserID:    users.user.ID,
		Plan:           domain.PlanFreelance,
		BillingCycle:   domain.CycleMonthly,
	})

	require.NoError(t, err)
	require.NotEmpty(t, out.ClientSecret)
	assert.Zero(t, stripe.enrichCallCount)
}

// ---------- Already-subscribed guard ----------

func TestSubscribe_RejectsWhenAlreadySubscribed(t *testing.T) {
	svc, subs, users, _, _ := newTestService()
	existing, _ := domain.NewSubscription(freshDomainInput(*users.user.OrganizationID))
	subs.findOpenByOrgFn = func(ctx context.Context, _ uuid.UUID) (*domain.Subscription, error) {
		return existing, nil
	}

	_, err := svc.Subscribe(context.Background(), appsub.SubscribeInput{
		OrganizationID: *users.user.OrganizationID,
		ActorUserID:    users.user.ID,
		Plan:         domain.PlanFreelance,
		BillingCycle: domain.CycleMonthly,
	})

	assert.ErrorIs(t, err, domain.ErrAlreadySubscribed)
}

// ---------- GetStatus ----------

func TestGetStatus_ReturnsSubscription(t *testing.T) {
	svc, subs, users, _, _ := newTestService()
	existing, _ := domain.NewSubscription(freshDomainInput(*users.user.OrganizationID))
	subs.findOpenByOrgFn = func(ctx context.Context, _ uuid.UUID) (*domain.Subscription, error) {
		return existing, nil
	}

	got, err := svc.GetStatus(context.Background(), *users.user.OrganizationID)

	require.NoError(t, err)
	assert.Equal(t, existing.ID, got.ID)
}

func TestGetStatus_NotFoundWhenFree(t *testing.T) {
	svc, _, users, _, _ := newTestService()

	_, err := svc.GetStatus(context.Background(), *users.user.OrganizationID)

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
			existing, _ := domain.NewSubscription(freshDomainInput(*users.user.OrganizationID))
			_ = existing.Activate()
			// Start in the opposite state of what we're about to toggle to
			// so we observe a real transition.
			existing.CancelAtPeriodEnd = !tc.wantCancelAtEnd

			subs.findOpenByOrgFn = func(ctx context.Context, _ uuid.UUID) (*domain.Subscription, error) {
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

			got, err := svc.ToggleAutoRenew(context.Background(), *users.user.OrganizationID,tc.turnOn)

			require.NoError(t, err)
			assert.Equal(t, tc.wantCancelAtEnd, got.CancelAtPeriodEnd)
		})
	}
}

func TestToggleAutoRenew_NoSubscription(t *testing.T) {
	svc, _, users, _, _ := newTestService()

	_, err := svc.ToggleAutoRenew(context.Background(), *users.user.OrganizationID,true)

	assert.ErrorIs(t, err, domain.ErrNotFound)
}

// TestToggleAutoRenew_OffReleasesPendingSchedule covers the other half of
// the fix that landed with ErrAutoRenewOffBlocksDowngrade: a user who has
// already scheduled a downgrade and then switches auto-renew OFF wants to
// end the subscription at period_end, not transition to a monthly plan.
// Without releasing the Stripe schedule first, Stripe renews anyway — the
// very bug reported on 2026-04-23. The service must release the schedule
// BEFORE flipping the cancel flag so Stripe sees a plain subscription
// it can honour.
func TestToggleAutoRenew_OffReleasesPendingSchedule(t *testing.T) {
	svc, subs, users, _, stripe := newTestService()
	in := freshDomainInput(*users.user.OrganizationID)
	in.BillingCycle = domain.CycleAnnual
	existing, _ := domain.NewSubscription(in)
	_ = existing.Activate()
	existing.SetAutoRenew(true) // auto-renew ON (a pending downgrade needs this)
	require.NoError(t, existing.SchedulePendingCycle(
		domain.CycleMonthly,
		time.Now().Add(30*24*time.Hour),
		"sub_sched_abc",
	))
	subs.findOpenByOrgFn = func(ctx context.Context, _ uuid.UUID) (*domain.Subscription, error) {
		return existing, nil
	}

	var releasedScheduleID string
	stripe.releaseScheduleFn = func(_ context.Context, scheduleID string) error {
		releasedScheduleID = scheduleID
		return nil
	}
	stripe.updateCancelAtPeriodEndFn = func(_ context.Context, subID string, cancelAtEnd bool) (service.SubscriptionSnapshot, error) {
		return service.SubscriptionSnapshot{
			ID:                 subID,
			Status:             "active",
			CancelAtPeriodEnd:  cancelAtEnd,
			CurrentPeriodStart: time.Now(),
			CurrentPeriodEnd:   time.Now().Add(365 * 24 * time.Hour),
		}, nil
	}

	got, err := svc.ToggleAutoRenew(context.Background(), *users.user.OrganizationID, false)

	require.NoError(t, err)
	assert.Equal(t, "sub_sched_abc", releasedScheduleID, "service MUST release the Stripe schedule when turning auto-renew OFF")
	assert.True(t, got.CancelAtPeriodEnd, "cancel flag reflects the user's intent to end at period_end")
	assert.False(t, got.HasPendingCycleChange(), "pending cycle MUST be cleared once the schedule is released")
}

// ---------- ChangeCycle ----------

func TestChangeCycle_MonthlyToAnnual(t *testing.T) {
	svc, subs, users, _, stripe := newTestService()
	in := freshDomainInput(*users.user.OrganizationID)
	in.BillingCycle = domain.CycleMonthly
	existing, _ := domain.NewSubscription(in)
	_ = existing.Activate()
	subs.findOpenByOrgFn = func(ctx context.Context, _ uuid.UUID) (*domain.Subscription, error) {
		return existing, nil
	}
	stripe.changeCycleImmediateFn = func(ctx context.Context, stripeSubID, newPriceID string) (service.SubscriptionSnapshot, error) {
		return service.SubscriptionSnapshot{
			ID:                 stripeSubID,
			Status:             "active",
			PriceID:            newPriceID,
			CurrentPeriodStart: time.Now(),
			CurrentPeriodEnd:   time.Now().Add(365 * 24 * time.Hour),
		}, nil
	}

	got, err := svc.ChangeCycle(context.Background(), *users.user.OrganizationID,domain.CycleAnnual)

	require.NoError(t, err)
	assert.Equal(t, domain.CycleAnnual, got.BillingCycle, "upgrade applies immediately")
	assert.False(t, got.HasPendingCycleChange(), "upgrade MUST NOT leave a pending schedule")
}

func TestChangeCycle_AnnualToMonthly(t *testing.T) {
	// Downgrade = scheduled, current cycle UNCHANGED, pending tuple set.
	// Requires auto-renew ON because a downgrade = "renew at a different
	// cadence next time"; see TestChangeCycle_DowngradeRequiresAutoRenew.
	svc, subs, users, _, _ := newTestService()
	in := freshDomainInput(*users.user.OrganizationID)
	in.BillingCycle = domain.CycleAnnual
	existing, _ := domain.NewSubscription(in)
	_ = existing.Activate()
	existing.SetAutoRenew(true)
	subs.findOpenByOrgFn = func(ctx context.Context, _ uuid.UUID) (*domain.Subscription, error) {
		return existing, nil
	}

	got, err := svc.ChangeCycle(context.Background(), *users.user.OrganizationID,domain.CycleMonthly)

	require.NoError(t, err)
	// CURRENT cycle stays annual — user keeps prepaid access until phase 2.
	assert.Equal(t, domain.CycleAnnual, got.BillingCycle, "downgrade defers; current cycle MUST stay annual")
	require.True(t, got.HasPendingCycleChange())
	assert.Equal(t, domain.CycleMonthly, *got.PendingBillingCycle, "pending target = monthly")
	require.NotNil(t, got.StripeScheduleID)
	assert.NotEmpty(t, *got.StripeScheduleID)
}

// TestChangeCycle_DowngradeRequiresAutoRenew locks in the product rule that
// fixed a billing bug: scheduling a downgrade while auto-renew is OFF used
// to leave Stripe's subscription schedule in charge. At the phase boundary,
// the schedule kept charging monthly invoices even though the user had
// explicitly opted out — "cancel_at_period_end=TRUE" meant to kill the sub
// but Stripe silently honoured the schedule. The service now rejects the
// request with ErrAutoRenewOffBlocksDowngrade so the UI can tell the user
// to re-enable auto-renew first.
func TestChangeCycle_DowngradeRequiresAutoRenew(t *testing.T) {
	svc, subs, users, _, stripe := newTestService()
	in := freshDomainInput(*users.user.OrganizationID)
	in.BillingCycle = domain.CycleAnnual
	existing, _ := domain.NewSubscription(in)
	_ = existing.Activate()
	// Auto-renew OFF by default (CancelAtPeriodEnd=TRUE on freshDomainInput).
	subs.findOpenByOrgFn = func(ctx context.Context, _ uuid.UUID) (*domain.Subscription, error) {
		return existing, nil
	}

	var scheduleCalled bool
	stripe.scheduleCycleChangeFn = func(_ context.Context, _, _ string) (service.ScheduledCycleChange, error) {
		scheduleCalled = true
		return service.ScheduledCycleChange{}, nil
	}

	_, err := svc.ChangeCycle(context.Background(), *users.user.OrganizationID, domain.CycleMonthly)

	assert.ErrorIs(t, err, domain.ErrAutoRenewOffBlocksDowngrade)
	assert.False(t, scheduleCalled, "service MUST NOT reach Stripe when the rule rejects the request")
	assert.False(t, existing.HasPendingCycleChange(), "local sub MUST stay untouched")
}

// TestPreviewCycleChange_DowngradeRequiresAutoRenew mirrors the guard for
// the read-only preview endpoint so the UI never surfaces a misleading
// number for a downgrade the caller won't be allowed to confirm.
func TestPreviewCycleChange_DowngradeRequiresAutoRenew(t *testing.T) {
	svc, subs, users, _, _ := newTestService()
	in := freshDomainInput(*users.user.OrganizationID)
	in.BillingCycle = domain.CycleAnnual
	existing, _ := domain.NewSubscription(in)
	_ = existing.Activate()
	subs.findOpenByOrgFn = func(ctx context.Context, _ uuid.UUID) (*domain.Subscription, error) {
		return existing, nil
	}

	_, err := svc.PreviewCycleChange(context.Background(), *users.user.OrganizationID, domain.CycleMonthly)

	assert.ErrorIs(t, err, domain.ErrAutoRenewOffBlocksDowngrade)
}

func TestChangeCycle_SameCycle(t *testing.T) {
	svc, subs, users, _, _ := newTestService()
	existing, _ := domain.NewSubscription(freshDomainInput(*users.user.OrganizationID))
	_ = existing.Activate()
	subs.findOpenByOrgFn = func(ctx context.Context, _ uuid.UUID) (*domain.Subscription, error) {
		return existing, nil
	}

	_, err := svc.ChangeCycle(context.Background(), *users.user.OrganizationID,existing.BillingCycle)

	assert.ErrorIs(t, err, domain.ErrSameCycle)
}

// TestChangeCycle_DirectionRouting locks in the product rule: upgrading
// monthly → annual calls Stripe with an immediate charge path
// (ChangeCycleImmediate), while downgrading annual → monthly uses the
// scheduled path (ScheduleCycleChange) which preserves the prepaid
// annual period. A regression here either double-charges the user or
// makes them lose their annual access — both are billing-critical bugs.
func TestChangeCycle_DirectionRouting(t *testing.T) {
	t.Run("upgrade monthly->annual calls ChangeCycleImmediate", func(t *testing.T) {
		svc, subs, users, _, stripe := newTestService()
		in := freshDomainInput(*users.user.OrganizationID)
		in.BillingCycle = domain.CycleMonthly
		existing, _ := domain.NewSubscription(in)
		_ = existing.Activate()
		subs.findOpenByOrgFn = func(ctx context.Context, _ uuid.UUID) (*domain.Subscription, error) {
			return existing, nil
		}

		var immediateCalled, scheduleCalled bool
		stripe.changeCycleImmediateFn = func(_ context.Context, subID, newPriceID string) (service.SubscriptionSnapshot, error) {
			immediateCalled = true
			return service.SubscriptionSnapshot{
				ID: subID, Status: "active", PriceID: newPriceID,
				CurrentPeriodStart: time.Now(), CurrentPeriodEnd: time.Now().Add(365 * 24 * time.Hour),
			}, nil
		}
		stripe.scheduleCycleChangeFn = func(_ context.Context, _, _ string) (service.ScheduledCycleChange, error) {
			scheduleCalled = true
			return service.ScheduledCycleChange{}, nil
		}

		_, err := svc.ChangeCycle(context.Background(), *users.user.OrganizationID,domain.CycleAnnual)
		require.NoError(t, err)
		assert.True(t, immediateCalled, "upgrade MUST use immediate-proration path")
		assert.False(t, scheduleCalled, "upgrade MUST NOT create a subscription schedule")
	})

	t.Run("downgrade annual->monthly calls ScheduleCycleChange", func(t *testing.T) {
		svc, subs, users, _, stripe := newTestService()
		in := freshDomainInput(*users.user.OrganizationID)
		in.BillingCycle = domain.CycleAnnual
		existing, _ := domain.NewSubscription(in)
		_ = existing.Activate()
		existing.SetAutoRenew(true)
		subs.findOpenByOrgFn = func(ctx context.Context, _ uuid.UUID) (*domain.Subscription, error) {
			return existing, nil
		}

		var immediateCalled, scheduleCalled bool
		stripe.changeCycleImmediateFn = func(_ context.Context, _, _ string) (service.SubscriptionSnapshot, error) {
			immediateCalled = true
			return service.SubscriptionSnapshot{}, nil
		}
		stripe.scheduleCycleChangeFn = func(_ context.Context, subID, _ string) (service.ScheduledCycleChange, error) {
			scheduleCalled = true
			effectiveAt := time.Now().Add(365 * 24 * time.Hour)
			return service.ScheduledCycleChange{
				ScheduleID:  "sched_abc",
				EffectiveAt: effectiveAt,
				Snapshot: service.SubscriptionSnapshot{
					ID: subID, Status: "active",
					CurrentPeriodStart: time.Now(), CurrentPeriodEnd: effectiveAt,
				},
			}, nil
		}

		_, err := svc.ChangeCycle(context.Background(), *users.user.OrganizationID,domain.CycleMonthly)
		require.NoError(t, err)
		assert.False(t, immediateCalled, "downgrade MUST NOT use immediate-proration path (would lose annual access)")
		assert.True(t, scheduleCalled, "downgrade MUST go through Stripe Subscription Schedules")
	})
}

// TestChangeCycle_UpgradeReleasesStaleSchedule: if a user scheduled a
// downgrade, then changes their mind and upgrades, the existing schedule
// MUST be released before the immediate upgrade runs — Stripe rejects
// subscription.update when a schedule is still managing the subscription.
func TestChangeCycle_UpgradeReleasesStaleSchedule(t *testing.T) {
	svc, subs, users, _, stripe := newTestService()
	in := freshDomainInput(*users.user.OrganizationID)
	in.BillingCycle = domain.CycleMonthly
	existing, _ := domain.NewSubscription(in)
	_ = existing.Activate()
	effectiveAt := time.Now().Add(365 * 24 * time.Hour)
	require.NoError(t, existing.SchedulePendingCycle(domain.CycleAnnual, effectiveAt, "sched_stale"))
	subs.findOpenByOrgFn = func(ctx context.Context, _ uuid.UUID) (*domain.Subscription, error) {
		return existing, nil
	}

	var releasedID string
	stripe.releaseScheduleFn = func(_ context.Context, id string) error {
		releasedID = id
		return nil
	}
	stripe.changeCycleImmediateFn = func(_ context.Context, subID, newPriceID string) (service.SubscriptionSnapshot, error) {
		return service.SubscriptionSnapshot{
			ID: subID, Status: "active", PriceID: newPriceID,
			CurrentPeriodStart: time.Now(), CurrentPeriodEnd: time.Now().Add(365 * 24 * time.Hour),
		}, nil
	}

	// Re-upgrade to annual (the already-scheduled target — but direct now).
	got, err := svc.ChangeCycle(context.Background(), *users.user.OrganizationID,domain.CycleAnnual)
	require.NoError(t, err)
	assert.Equal(t, "sched_stale", releasedID, "the existing schedule MUST be released first")
	assert.False(t, got.HasPendingCycleChange(), "no pending state after a direct upgrade")
}

func TestChangeCycle_InvalidCycle(t *testing.T) {
	svc, _, users, _, _ := newTestService()

	_, err := svc.ChangeCycle(context.Background(), *users.user.OrganizationID,"weekly")

	assert.ErrorIs(t, err, domain.ErrInvalidCycle)
}

// ---------- GetStats ----------

func TestGetStats_ComputesSavingsAcrossTiers(t *testing.T) {
	svc, subs, users, amounts, _ := newTestService()
	existing, _ := domain.NewSubscription(freshDomainInput(*users.user.OrganizationID))
	_ = existing.Activate()
	subs.findOpenByOrgFn = func(ctx context.Context, _ uuid.UUID) (*domain.Subscription, error) {
		return existing, nil
	}
	// Freelance role — tiers 9€/15€/25€
	amounts.amounts = []int64{
		10000,  // 100€   → tier 1 → 9€ saved
		50000,  // 500€   → tier 2 → 15€ saved
		200000, // 2000€  → tier 3 → 25€ saved
	}

	stats, err := svc.GetStats(context.Background(), *users.user.OrganizationID, users.user.ID)

	require.NoError(t, err)
	assert.Equal(t, int64(900+1500+2500), stats.SavedFeeCents)
	assert.Equal(t, 3, stats.SavedCount)
}

func TestGetStats_EmptyHistory(t *testing.T) {
	svc, subs, users, amounts, _ := newTestService()
	existing, _ := domain.NewSubscription(freshDomainInput(*users.user.OrganizationID))
	_ = existing.Activate()
	subs.findOpenByOrgFn = func(ctx context.Context, _ uuid.UUID) (*domain.Subscription, error) {
		return existing, nil
	}
	amounts.amounts = nil

	stats, err := svc.GetStats(context.Background(), *users.user.OrganizationID, users.user.ID)

	require.NoError(t, err)
	assert.Equal(t, int64(0), stats.SavedFeeCents)
	assert.Equal(t, 0, stats.SavedCount)
}

func TestGetStats_AgencyTierGrid(t *testing.T) {
	svc, subs, users, amounts, _ := newTestService()
	users.user.Role = domainuser.RoleAgency
	existing, _ := domain.NewSubscription(freshDomainInput(*users.user.OrganizationID))
	_ = existing.Activate()
	subs.findOpenByOrgFn = func(ctx context.Context, _ uuid.UUID) (*domain.Subscription, error) {
		return existing, nil
	}
	// Agency role — tiers 19€/39€/69€
	amounts.amounts = []int64{
		30000,  // 300€   → tier 1 → 19€
		100000, // 1000€  → tier 2 → 39€
		500000, // 5000€  → tier 3 → 69€
	}

	stats, err := svc.GetStats(context.Background(), *users.user.OrganizationID, users.user.ID)

	require.NoError(t, err)
	assert.Equal(t, int64(1900+3900+6900), stats.SavedFeeCents)
}

// ---------- GetPortalURL ----------

func TestGetPortalURL_HappyPath(t *testing.T) {
	svc, subs, users, _, _ := newTestService()
	existing, _ := domain.NewSubscription(freshDomainInput(*users.user.OrganizationID))
	_ = existing.Activate()
	subs.findOpenByOrgFn = func(ctx context.Context, _ uuid.UUID) (*domain.Subscription, error) {
		return existing, nil
	}

	url, err := svc.GetPortalURL(context.Background(), *users.user.OrganizationID)

	require.NoError(t, err)
	assert.Contains(t, url, "portal.stripe.test")
}

func TestGetPortalURL_NoSubscription(t *testing.T) {
	svc, _, users, _, _ := newTestService()

	_, err := svc.GetPortalURL(context.Background(), *users.user.OrganizationID)

	assert.ErrorIs(t, err, domain.ErrNotFound)
}

// ---------- IsActive (SubscriptionReader implementation) ----------

func TestIsActive_ActiveSubscription(t *testing.T) {
	svc, subs, users, _, _ := newTestService()
	existing, _ := domain.NewSubscription(freshDomainInput(*users.user.OrganizationID))
	_ = existing.Activate()
	subs.findOpenByOrgFn = func(ctx context.Context, _ uuid.UUID) (*domain.Subscription, error) {
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
	subs.findOpenByOrgFn = func(ctx context.Context, _ uuid.UUID) (*domain.Subscription, error) {
		return nil, errors.New("db connection lost")
	}

	active, err := svc.IsActive(context.Background(), users.user.ID)

	require.Error(t, err)
	assert.False(t, active, "error MUST fail closed (no Premium granted)")
}

// ---------- HandleSubscriptionSnapshot (webhook entry) ----------

func TestHandleSubscriptionSnapshot_ActivatesFromIncomplete(t *testing.T) {
	svc, subs, users, _, _ := newTestService()
	existing, _ := domain.NewSubscription(freshDomainInput(*users.user.OrganizationID))
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
	existing, _ := domain.NewSubscription(freshDomainInput(*users.user.OrganizationID))
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
	existing, _ := domain.NewSubscription(freshDomainInput(*users.user.OrganizationID))
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
	existing, _ := domain.NewSubscription(freshDomainInput(*users.user.OrganizationID))
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
		*users.user.OrganizationID,
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

// freshDomainInput builds a valid NewSubscriptionInput for a given org.
// The input used to key by user; it now keys by organization_id — tests
// pass users.user.OrganizationID (pre-set in newTestService).
func freshDomainInput(orgID uuid.UUID) domain.NewSubscriptionInput {
	now := time.Now()
	return domain.NewSubscriptionInput{
		OrganizationID:       orgID,
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
