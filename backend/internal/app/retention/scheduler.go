package retention

import (
	"context"
	"errors"
	"log/slog"
	"time"
)

// SchedulerInterval is the cadence at which the retention sweep runs
// in production. One hour is the operational target — the sweep is
// idempotent and incremental, so a missed tick rolls into the next
// without backlog. Aligning on hourly avoids the "everything fires at
// 00:00 UTC" thundering herd while still completing the daily
// retention budget for every policy.
const (
	SchedulerInterval    = 1 * time.Hour
	SchedulerDevInterval = 1 * time.Minute
)

// Scheduler drives Service.Run on a ticker. Owns timing, logging
// frequency, and graceful shutdown. The scheduler does NOT decide
// what to sweep — every retention rule lives in Service via the
// Policy slice it was built with.
type Scheduler struct {
	svc      *Service
	interval time.Duration
}

// NewScheduler returns a scheduler bound to svc. Passing zero falls
// back to SchedulerInterval; tests use the WithInterval helper to
// shrink the cadence.
func NewScheduler(svc *Service, interval time.Duration) *Scheduler {
	if interval == 0 {
		interval = SchedulerInterval
	}
	return &Scheduler{svc: svc, interval: interval}
}

// Run blocks until ctx is cancelled. Runs an immediate tick on start
// so a fresh boot picks up overdue retention without waiting a full
// interval — same pattern as the GDPR scheduler.
func (s *Scheduler) Run(ctx context.Context) {
	if s.svc == nil {
		slog.Error("retention scheduler: refusing to run without a service")
		return
	}
	s.tick(ctx)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("retention scheduler: stopped")
			return
		case <-ticker.C:
			s.tick(ctx)
		}
	}
}

func (s *Scheduler) tick(ctx context.Context) {
	results, errs := s.svc.Run(ctx)
	totalAffected := 0
	totalBatches := 0
	for _, r := range results {
		totalAffected += r.Affected
		totalBatches += r.Batches
	}
	switch {
	case len(errs) == 0 && totalAffected == 0:
		slog.Debug("retention scheduler: tick idle",
			"policies", len(results))
	case len(errs) == 0:
		slog.Info("retention scheduler: tick done",
			"policies", len(results),
			"affected", totalAffected,
			"batches", totalBatches)
	default:
		slog.Warn("retention scheduler: tick partial",
			"policies", len(results),
			"affected", totalAffected,
			"errors", len(errs))
		for _, e := range errs {
			if errors.Is(e, context.Canceled) {
				continue
			}
			slog.Warn("retention scheduler: per-policy error", "error", e.Error())
		}
	}
}
