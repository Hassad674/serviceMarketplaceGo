package subscription_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/domain/subscription"
)

func baseInput() subscription.NewSubscriptionInput {
	now := time.Now()
	return subscription.NewSubscriptionInput{
		UserID:               uuid.New(),
		Plan:                 subscription.PlanFreelance,
		BillingCycle:         subscription.CycleMonthly,
		StripeCustomerID:     "cus_test",
		StripeSubscriptionID: "sub_test",
		StripePriceID:        "price_test",
		CurrentPeriodStart:   now,
		CurrentPeriodEnd:     now.Add(30 * 24 * time.Hour),
		CancelAtPeriodEnd:    true,
	}
}

func TestNewSubscription(t *testing.T) {
	in := baseInput()
	s, err := subscription.NewSubscription(in)

	require.NoError(t, err)
	require.NotNil(t, s)
	assert.Equal(t, in.UserID, s.UserID)
	assert.Equal(t, subscription.PlanFreelance, s.Plan)
	assert.Equal(t, subscription.CycleMonthly, s.BillingCycle)
	assert.Equal(t, subscription.StatusIncomplete, s.Status, "new sub must start incomplete")
	assert.True(t, s.CancelAtPeriodEnd, "auto-renew OFF by default")
	assert.NotEqual(t, uuid.Nil, s.ID)
	assert.Nil(t, s.GracePeriodEndsAt)
	assert.Nil(t, s.CanceledAt)
}

func TestNewSubscription_ValidationErrors(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*subscription.NewSubscriptionInput)
		want   error
	}{
		{"zero user id", func(in *subscription.NewSubscriptionInput) { in.UserID = uuid.Nil }, subscription.ErrInvalidUser},
		{"invalid plan", func(in *subscription.NewSubscriptionInput) { in.Plan = "enterprise" }, subscription.ErrInvalidPlan},
		{"empty plan", func(in *subscription.NewSubscriptionInput) { in.Plan = "" }, subscription.ErrInvalidPlan},
		{"invalid cycle", func(in *subscription.NewSubscriptionInput) { in.BillingCycle = "weekly" }, subscription.ErrInvalidCycle},
		{"empty cycle", func(in *subscription.NewSubscriptionInput) { in.BillingCycle = "" }, subscription.ErrInvalidCycle},
		{"missing stripe customer", func(in *subscription.NewSubscriptionInput) { in.StripeCustomerID = "" }, subscription.ErrMissingStripeIDs},
		{"missing stripe sub", func(in *subscription.NewSubscriptionInput) { in.StripeSubscriptionID = "" }, subscription.ErrMissingStripeIDs},
		{"missing stripe price", func(in *subscription.NewSubscriptionInput) { in.StripePriceID = "" }, subscription.ErrMissingStripeIDs},
		{"period end before start", func(in *subscription.NewSubscriptionInput) {
			in.CurrentPeriodEnd = in.CurrentPeriodStart.Add(-1 * time.Hour)
		}, subscription.ErrInvalidPeriod},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			in := baseInput()
			tc.mutate(&in)
			_, err := subscription.NewSubscription(in)
			assert.ErrorIs(t, err, tc.want)
		})
	}
}

func TestActivate(t *testing.T) {
	s, _ := subscription.NewSubscription(baseInput())

	err := s.Activate()

	require.NoError(t, err)
	assert.Equal(t, subscription.StatusActive, s.Status)
	assert.WithinDuration(t, time.Now(), s.StartedAt, time.Second, "StartedAt set on first activation")
}

func TestActivate_FromPastDueKeepsStartedAt(t *testing.T) {
	s, _ := subscription.NewSubscription(baseInput())
	require.NoError(t, s.Activate())
	originalStart := s.StartedAt
	time.Sleep(5 * time.Millisecond) // force measurable delta
	require.NoError(t, s.MarkPastDue(time.Now().Add(72*time.Hour)))

	err := s.Activate()

	require.NoError(t, err)
	assert.Equal(t, subscription.StatusActive, s.Status)
	assert.Equal(t, originalStart, s.StartedAt, "StartedAt must survive recovery from past_due")
	assert.Nil(t, s.GracePeriodEndsAt, "grace cleared on recovery")
}

func TestActivate_TerminalStatesRejected(t *testing.T) {
	for _, status := range []subscription.Status{subscription.StatusCanceled, subscription.StatusUnpaid} {
		t.Run(string(status), func(t *testing.T) {
			s, _ := subscription.NewSubscription(baseInput())
			s.Status = status

			err := s.Activate()

			assert.ErrorIs(t, err, subscription.ErrInvalidTransition)
		})
	}
}

func TestMarkPastDue(t *testing.T) {
	s, _ := subscription.NewSubscription(baseInput())
	require.NoError(t, s.Activate())
	graceEnd := time.Now().Add(72 * time.Hour)

	err := s.MarkPastDue(graceEnd)

	require.NoError(t, err)
	assert.Equal(t, subscription.StatusPastDue, s.Status)
	require.NotNil(t, s.GracePeriodEndsAt)
	assert.Equal(t, graceEnd.Unix(), s.GracePeriodEndsAt.Unix())
}

func TestMarkPastDue_FromIncompleteRejected(t *testing.T) {
	s, _ := subscription.NewSubscription(baseInput())

	err := s.MarkPastDue(time.Now().Add(time.Hour))

	assert.ErrorIs(t, err, subscription.ErrInvalidTransition)
}

func TestMarkCanceled(t *testing.T) {
	s, _ := subscription.NewSubscription(baseInput())
	require.NoError(t, s.Activate())

	err := s.MarkCanceled()

	require.NoError(t, err)
	assert.Equal(t, subscription.StatusCanceled, s.Status)
	require.NotNil(t, s.CanceledAt)
	assert.WithinDuration(t, time.Now(), *s.CanceledAt, time.Second)
}

func TestMarkCanceled_Idempotent(t *testing.T) {
	// A second cancel MUST fail — the webhook handler must rely on this
	// to detect replays (see idempotency key path).
	s, _ := subscription.NewSubscription(baseInput())
	require.NoError(t, s.Activate())
	require.NoError(t, s.MarkCanceled())

	err := s.MarkCanceled()

	assert.ErrorIs(t, err, subscription.ErrInvalidTransition)
}

func TestUpdatePeriod(t *testing.T) {
	s, _ := subscription.NewSubscription(baseInput())
	newStart := time.Now()
	newEnd := newStart.Add(365 * 24 * time.Hour)

	err := s.UpdatePeriod(newStart, newEnd)

	require.NoError(t, err)
	assert.Equal(t, newStart.Unix(), s.CurrentPeriodStart.Unix())
	assert.Equal(t, newEnd.Unix(), s.CurrentPeriodEnd.Unix())
}

func TestUpdatePeriod_EndBeforeStart(t *testing.T) {
	s, _ := subscription.NewSubscription(baseInput())

	err := s.UpdatePeriod(time.Now(), time.Now().Add(-1*time.Hour))

	assert.ErrorIs(t, err, subscription.ErrInvalidPeriod)
}

func TestSetAutoRenew(t *testing.T) {
	s, _ := subscription.NewSubscription(baseInput())
	assert.True(t, s.CancelAtPeriodEnd, "default OFF means cancel_at_period_end=true")

	s.SetAutoRenew(true)
	assert.False(t, s.CancelAtPeriodEnd)

	s.SetAutoRenew(false)
	assert.True(t, s.CancelAtPeriodEnd)
}

func TestChangeCycle(t *testing.T) {
	tests := []struct {
		name       string
		fromCycle  subscription.BillingCycle
		toCycle    subscription.BillingCycle
		wantErr    error
	}{
		{"monthly to annual", subscription.CycleMonthly, subscription.CycleAnnual, nil},
		{"annual to monthly", subscription.CycleAnnual, subscription.CycleMonthly, nil},
		{"same cycle", subscription.CycleMonthly, subscription.CycleMonthly, subscription.ErrSameCycle},
		{"invalid target", subscription.CycleMonthly, subscription.BillingCycle("weekly"), subscription.ErrInvalidCycle},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			in := baseInput()
			in.BillingCycle = tc.fromCycle
			s, _ := subscription.NewSubscription(in)
			require.NoError(t, s.Activate())

			newStart := time.Now()
			newEnd := newStart.Add(365 * 24 * time.Hour)
			err := s.ChangeCycle(tc.toCycle, "price_new", newStart, newEnd)

			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.toCycle, s.BillingCycle)
			assert.Equal(t, "price_new", s.StripePriceID)
		})
	}
}

func TestChangeCycle_FromIncompleteRejected(t *testing.T) {
	s, _ := subscription.NewSubscription(baseInput())

	err := s.ChangeCycle(subscription.CycleAnnual, "price_new", time.Now(), time.Now().Add(365*24*time.Hour))

	assert.ErrorIs(t, err, subscription.ErrInvalidTransition)
}

func TestIsPremium(t *testing.T) {
	now := time.Date(2026, 4, 21, 12, 0, 0, 0, time.UTC)
	periodEnd := now.Add(30 * 24 * time.Hour)

	tests := []struct {
		name   string
		status subscription.Status
		within bool
		grace  *time.Time
		want   bool
	}{
		{"active within period", subscription.StatusActive, true, nil, true},
		{"active expired", subscription.StatusActive, false, nil, false},
		{"past_due within grace", subscription.StatusPastDue, true, ptr(now.Add(3 * 24 * time.Hour)), true},
		{"past_due expired grace", subscription.StatusPastDue, true, ptr(now.Add(-1 * time.Hour)), false},
		{"past_due missing grace", subscription.StatusPastDue, true, nil, false},
		{"incomplete never premium", subscription.StatusIncomplete, true, nil, false},
		{"canceled never premium", subscription.StatusCanceled, true, nil, false},
		{"unpaid never premium", subscription.StatusUnpaid, true, nil, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s, _ := subscription.NewSubscription(baseInput())
			s.Status = tc.status
			s.GracePeriodEndsAt = tc.grace
			if tc.within {
				s.CurrentPeriodEnd = periodEnd
			} else {
				s.CurrentPeriodEnd = now.Add(-1 * time.Hour)
			}

			got := s.IsPremium(now)

			assert.Equal(t, tc.want, got)
		})
	}
}

func ptr[T any](v T) *T { return &v }
