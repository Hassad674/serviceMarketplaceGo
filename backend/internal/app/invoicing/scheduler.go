package invoicing

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

// DefaultSchedulerInterval is the default tick between two runs of the
// monthly-consolidation scheduler. 1 hour matches the referral cron —
// the actual processing window (day-of-month 1, 02:00-03:59 UTC) is
// short enough that an hourly check is plenty.
const DefaultSchedulerInterval = 1 * time.Hour

// OrgLister is the narrow projection the scheduler needs of the
// organization repository. Only orgs that completed Stripe Connect
// onboarding (i.e. have a stripe_account_id) ever transact, so the
// scheduler ignores everyone else.
type OrgLister interface {
	ListWithStripeAccount(ctx context.Context) ([]uuid.UUID, error)
}

// RunMarker tracks the last successfully-completed monthly run so a
// re-tick within the same window is a cheap no-op. Implemented in
// adapter/redis as a single key with a 35-day TTL.
type RunMarker interface {
	GetLastMonthlyRun(ctx context.Context) (string, error)
	MarkMonthlyRun(ctx context.Context, monthKey string) error
}

// SchedulerDeps groups the constructor parameters of Scheduler.
//
// Interval and RunAfter are optional — defaults pick DefaultSchedulerInterval
// and the production window (day=1 between 02:00 and 03:59 UTC).
type SchedulerDeps struct {
	Service  *Service
	Orgs     OrgLister
	Marker   RunMarker
	Interval time.Duration
	RunAfter func(now time.Time) bool
}

// Scheduler runs IssueMonthlyConsolidated for every org with a Stripe
// Connect account, once per calendar month. In-process — no
// distributed coordination — because we run a single API instance for
// V1 and the Redis-backed RunMarker is idempotent on retry.
type Scheduler struct {
	svc      *Service
	orgs     OrgLister
	marker   RunMarker
	interval time.Duration
	runAfter func(now time.Time) bool
}

// NewScheduler wires the scheduler. Interval defaults to one hour and
// runAfter defaults to "day-of-month is 1 and the time is in
// [02:00, 04:00) UTC". Tests pass a runAfter that always returns true
// to force the hot path.
func NewScheduler(deps SchedulerDeps) *Scheduler {
	interval := deps.Interval
	if interval <= 0 {
		interval = DefaultSchedulerInterval
	}
	runAfter := deps.RunAfter
	if runAfter == nil {
		runAfter = defaultRunWindow
	}
	return &Scheduler{
		svc:      deps.Service,
		orgs:     deps.Orgs,
		marker:   deps.Marker,
		interval: interval,
		runAfter: runAfter,
	}
}

// Start launches the scheduler in a background goroutine. The goroutine
// terminates when ctx is cancelled. Panics in any single tick are
// recovered so a transient bug never kills the process.
func (s *Scheduler) Start(ctx context.Context) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("invoicing scheduler panic recovered", "panic", r)
			}
		}()

		// One immediate tick on boot so a just-restarted instance
		// catches any window we slept through.
		s.Tick(ctx)

		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				slog.Info("invoicing scheduler stopped", "reason", ctx.Err())
				return
			case <-ticker.C:
				func() {
					defer func() {
						if r := recover(); r != nil {
							slog.Error("invoicing scheduler tick panic recovered", "panic", r)
						}
					}()
					s.Tick(ctx)
				}()
			}
		}
	}()
}

// Tick is exported so tests can drive the scheduler manually without
// having to wait on the real ticker. In production it is called from
// the goroutine spawned by Start.
func (s *Scheduler) Tick(ctx context.Context) {
	now := time.Now().UTC()
	if !s.runAfter(now) {
		return
	}

	// Resolve the period being CONSOLIDATED — i.e. the month
	// immediately before now. On 2026-05-01 we issue invoices for
	// 2026-04, so monthKey = "2026-04".
	periodYear, periodMonth := previousMonth(now)
	monthKey := fmt.Sprintf("%04d-%02d", periodYear, periodMonth)

	logger := slog.With(
		"flow", "invoicing.scheduler",
		"month_key", monthKey,
	)

	if s.marker != nil {
		lastRun, err := s.marker.GetLastMonthlyRun(ctx)
		if err != nil {
			// A Redis blip leaves us guessing — bias toward
			// re-running, the Service-level idempotency probe
			// will short-circuit any org we already processed.
			logger.Warn("invoicing scheduler: marker read failed, proceeding without it", "error", err)
		} else if lastRun == monthKey {
			logger.Info("invoicing scheduler: already ran this month, skipping")
			return
		}
	}

	if s.orgs == nil {
		logger.Warn("invoicing scheduler: org lister missing, skipping tick")
		return
	}
	orgIDs, err := s.orgs.ListWithStripeAccount(ctx)
	if err != nil {
		logger.Error("invoicing scheduler: list orgs failed", "error", err)
		return
	}
	logger.Info("invoicing scheduler: starting batch", "org_count", len(orgIDs))

	var issued, skipped, errored int
	for _, orgID := range orgIDs {
		select {
		case <-ctx.Done():
			logger.Info("invoicing scheduler: aborted mid-batch", "reason", ctx.Err())
			return
		default:
		}

		inv, err := s.svc.IssueMonthlyConsolidated(ctx, IssueMonthlyConsolidatedInput{
			OrganizationID: orgID,
			Year:           periodYear,
			Month:          periodMonth,
		})
		switch {
		case err != nil:
			errored++
			logger.Error("invoicing scheduler: issue failed", "org_id", orgID, "error", err)
		case inv == nil:
			skipped++
		default:
			issued++
		}
	}

	logger.Info("invoicing scheduler: batch done",
		"issued", issued, "skipped", skipped, "errored", errored)

	if s.marker != nil {
		if err := s.marker.MarkMonthlyRun(ctx, monthKey); err != nil {
			logger.Warn("invoicing scheduler: mark run failed", "error", err)
		}
	}
}

// defaultRunWindow returns true on the first day of the month between
// 02:00 (inclusive) and 04:00 (exclusive) UTC. Two-hour window so a
// crash + restart still has time to catch the run.
func defaultRunWindow(now time.Time) bool {
	return now.Day() == 1 && now.Hour() >= 2 && now.Hour() < 4
}

// previousMonth returns the (year, month) of the calendar month
// immediately before the given instant, in [1..12]. Wraps Decembers
// across the year boundary.
func previousMonth(now time.Time) (int, int) {
	prev := now.AddDate(0, -1, 0)
	return prev.Year(), int(prev.Month())
}
