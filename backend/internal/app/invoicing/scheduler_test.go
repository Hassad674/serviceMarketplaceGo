package invoicing_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	invoicingapp "marketplace-backend/internal/app/invoicing"
	"marketplace-backend/internal/domain/invoicing"
	"marketplace-backend/internal/port/repository"
)

// stubOrgLister is a tiny in-memory OrgLister: returns whatever ids
// were handed to it at construction time.
type stubOrgLister struct {
	ids []uuid.UUID
	err error
}

func (s *stubOrgLister) ListWithStripeAccount(_ context.Context) ([]uuid.UUID, error) {
	return s.ids, s.err
}

// stubRunMarker is an in-memory RunMarker that mirrors what the Redis
// adapter does, minus the network. Records all writes for assertions.
type stubRunMarker struct {
	mu        sync.Mutex
	value     string
	getErr    error
	markErr   error
	markCalls int
}

func (s *stubRunMarker) GetLastMonthlyRun(_ context.Context) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.getErr != nil {
		return "", s.getErr
	}
	return s.value, nil
}

func (s *stubRunMarker) MarkMonthlyRun(_ context.Context, monthKey string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.markCalls++
	if s.markErr != nil {
		return s.markErr
	}
	s.value = monthKey
	return nil
}

func (s *stubRunMarker) Calls() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.markCalls
}

// newSchedulerSvc spins up a Service whose mocks are wired so a tick
// against any orgID succeeds. issuedFor records every (org, year,
// month) the Service was asked to bill so the scheduler tests can
// assert what was processed.
func newSchedulerSvc(t *testing.T) (*invoicingapp.Service, *recordedSvc) {
	t.Helper()
	svc, invRepo, profileRepo, _, _, _, _ := newSvc(t)

	rec := &recordedSvc{}

	profileRepo.findByOrgFn = func(_ context.Context, orgID uuid.UUID) (*invoicing.BillingProfile, error) {
		return frProfile(orgID), nil
	}
	invRepo.listReleasedForOrgFn = func(_ context.Context, orgID uuid.UUID, start, _ time.Time) ([]repository.ReleasedPaymentRecord, error) {
		rec.record(orgID, start)
		// One released record so a real invoice gets issued.
		return []repository.ReleasedPaymentRecord{{
			ID:                  uuid.New(),
			MilestoneID:         uuid.New(),
			ProposalID:          uuid.New(),
			ProposalAmountCents: 100_00,
			PlatformFeeCents:    10_00,
			Currency:            "EUR",
			TransferredAt:       start.Add(2 * 24 * time.Hour),
		}}, nil
	}
	return svc, rec
}

type recordedSvc struct {
	mu    sync.Mutex
	calls []recordedCall
}

type recordedCall struct {
	orgID       uuid.UUID
	periodStart time.Time
}

func (r *recordedSvc) record(orgID uuid.UUID, periodStart time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, recordedCall{orgID: orgID, periodStart: periodStart})
}

func (r *recordedSvc) callCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.calls)
}

func TestScheduler_OutsideWindow_DoesNothing(t *testing.T) {
	svc, rec := newSchedulerSvc(t)
	orgs := &stubOrgLister{ids: []uuid.UUID{uuid.New(), uuid.New()}}
	marker := &stubRunMarker{}

	sched := invoicingapp.NewScheduler(invoicingapp.SchedulerDeps{
		Service:  svc,
		Orgs:     orgs,
		Marker:   marker,
		Interval: time.Hour,
		RunAfter: func(_ time.Time) bool { return false }, // window closed
	})

	sched.Tick(context.Background())

	assert.Equal(t, 0, rec.callCount(), "outside the window the scheduler issues nothing")
	assert.Equal(t, 0, marker.Calls(), "no marker write when no batch ran")
}

func TestScheduler_InWindow_FirstRun_ProcessesOrgs(t *testing.T) {
	svc, rec := newSchedulerSvc(t)
	orgIDs := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}
	orgs := &stubOrgLister{ids: orgIDs}
	marker := &stubRunMarker{} // empty → first run

	sched := invoicingapp.NewScheduler(invoicingapp.SchedulerDeps{
		Service:  svc,
		Orgs:     orgs,
		Marker:   marker,
		Interval: time.Hour,
		RunAfter: func(_ time.Time) bool { return true },
	})

	sched.Tick(context.Background())

	assert.Equal(t, len(orgIDs), rec.callCount(), "every org with a stripe account is processed exactly once")
	assert.Equal(t, 1, marker.Calls(), "marker is bumped after the batch")

	// monthKey reflects the previous calendar month relative to now.
	now := time.Now().UTC()
	prev := now.AddDate(0, -1, 0)
	wantKey := prev.Format("2006-01")
	got, _ := marker.GetLastMonthlyRun(context.Background())
	assert.Equal(t, wantKey, got, "marker stores the period that was just processed")
}

func TestScheduler_InWindow_AlreadyRunThisMonth_Skips(t *testing.T) {
	svc, rec := newSchedulerSvc(t)
	orgs := &stubOrgLister{ids: []uuid.UUID{uuid.New()}}
	marker := &stubRunMarker{}

	// Pre-load the marker with the period the scheduler is about to
	// compute — should short-circuit immediately.
	now := time.Now().UTC()
	prev := now.AddDate(0, -1, 0)
	marker.value = prev.Format("2006-01")

	sched := invoicingapp.NewScheduler(invoicingapp.SchedulerDeps{
		Service:  svc,
		Orgs:     orgs,
		Marker:   marker,
		Interval: time.Hour,
		RunAfter: func(_ time.Time) bool { return true },
	})

	sched.Tick(context.Background())

	assert.Equal(t, 0, rec.callCount(), "scheduler must short-circuit when this month is already done")
	assert.Equal(t, 0, marker.Calls(), "no second mark when we did not re-run")
}

func TestScheduler_DefaultRunWindow_OnlyFirstOfMonthEarlyHours(t *testing.T) {
	// Build a scheduler with the default window so we can probe the
	// real predicate without exposing it.
	svc, rec := newSchedulerSvc(t)
	orgs := &stubOrgLister{ids: []uuid.UUID{uuid.New()}}
	marker := &stubRunMarker{}

	sched := invoicingapp.NewScheduler(invoicingapp.SchedulerDeps{
		Service: svc,
		Orgs:    orgs,
		Marker:  marker,
		// no RunAfter — uses the default
	})

	// We can't time-travel without injecting a clock, but we CAN
	// assert the constructor populated something — and that a tick
	// outside today's window (most days/hours) is a no-op.
	now := time.Now().UTC()
	if !(now.Day() == 1 && now.Hour() >= 2 && now.Hour() < 4) {
		sched.Tick(context.Background())
		assert.Equal(t, 0, rec.callCount(), "default window blocks every tick except day=1, hour in [2,4)")
	} else {
		t.Skip("skipping window-shape probe inside the production window")
	}
	require.NotNil(t, sched)
}
