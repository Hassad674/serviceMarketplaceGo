package referral

import (
	"context"
	"log/slog"
	"time"
)

// DefaultSchedulerInterval is the default tick between two runs of the
// expirer cron. 1 hour is a balance: the 14-day silence expiry and the
// 6-month exclusivity expiry are both coarse enough that an hourly check
// is plenty, while still keeping the worst-case latency between the
// trigger and the state transition at 59 minutes.
const DefaultSchedulerInterval = 1 * time.Hour

// Scheduler wraps the referral Service with a simple time-based loop that
// invokes RunExpirerCycle on a fixed interval. It intentionally does NOT
// use the pending_events queue because the expiry rules are purely
// time-based (no user action triggers them) and there is no benefit to
// persisting the schedule — the cron can drift by a few minutes at
// startup without affecting correctness.
type Scheduler struct {
	svc      *Service
	interval time.Duration
}

// NewScheduler wires the scheduler with a custom interval. Pass 0 to
// use DefaultSchedulerInterval.
func NewScheduler(svc *Service, interval time.Duration) *Scheduler {
	if interval <= 0 {
		interval = DefaultSchedulerInterval
	}
	return &Scheduler{svc: svc, interval: interval}
}

// Run blocks until ctx is cancelled, running one cycle on start and then
// on every tick of the interval. Errors are logged and the loop keeps
// going — a transient DB error must never kill the cron.
func (s *Scheduler) Run(ctx context.Context) {
	// One immediate run at startup so a just-restarted instance catches
	// anything that expired while it was down.
	s.runOnce(ctx)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("referral scheduler stopped", "reason", ctx.Err())
			return
		case <-ticker.C:
			s.runOnce(ctx)
		}
	}
}

func (s *Scheduler) runOnce(ctx context.Context) {
	stale, matured, err := s.svc.RunExpirerCycle(ctx)
	if err != nil {
		slog.Error("referral scheduler cycle failed", "error", err)
		return
	}
	if stale > 0 || matured > 0 {
		slog.Info("referral scheduler cycle", "stale_intros", stale, "matured_referrals", matured)
	}
}
