package gdpr

import (
	"context"
	"errors"
	"log/slog"
	"time"
)

// SchedulerInterval is the cadence at which the GDPR purge cron
// runs in production. Decision 3 of the P5 brief: daily at 03:00 UTC
// is the operational target — the scheduler ticks every interval
// and lets clock alignment land it close to that time. In dev we
// shorten the interval so test scenarios complete in seconds.
const (
	SchedulerInterval    = 24 * time.Hour
	SchedulerDevInterval = 1 * time.Minute
)

// Scheduler is the long-lived goroutine that drives PurgeOnce. The
// service does the actual purge — the scheduler only owns the timing
// + retry policy.
//
// One scheduler per process is enough: the underlying SQL uses FOR
// UPDATE SKIP LOCKED so multiple schedulers (e.g. a horizontally
// scaled deployment) cooperate without coordination, but a single
// instance keeps the operational story simple.
type Scheduler struct {
	svc       *Service
	salt      string
	batchSize int
	interval  time.Duration
}

// NewScheduler builds the cron loop. Passing a zero interval falls
// back to SchedulerInterval so tests can opt-in via NewSchedulerWithInterval.
func NewScheduler(svc *Service, salt string, batchSize int, interval time.Duration) *Scheduler {
	if interval == 0 {
		interval = SchedulerInterval
	}
	if batchSize <= 0 {
		batchSize = 100
	}
	return &Scheduler{
		svc:       svc,
		salt:      salt,
		batchSize: batchSize,
		interval:  interval,
	}
}

// Run blocks until ctx is cancelled. Ticks every interval + runs
// immediately on start so a fresh boot picks up overdue purges
// without waiting a full day.
func (s *Scheduler) Run(ctx context.Context) {
	if s.salt == "" {
		slog.Error("gdpr scheduler: refusing to run without GDPR_ANONYMIZATION_SALT")
		return
	}
	s.tick(ctx)
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("gdpr scheduler: stopped")
			return
		case <-ticker.C:
			s.tick(ctx)
		}
	}
}

func (s *Scheduler) tick(ctx context.Context) {
	res, err := s.svc.PurgeOnce(ctx, s.salt, s.batchSize)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return
		}
		slog.Error("gdpr scheduler: purge batch", "error", err)
		return
	}
	if res.Examined == 0 {
		slog.Debug("gdpr scheduler: tick", "examined", 0)
		return
	}
	slog.Info("gdpr scheduler: tick",
		"examined", res.Examined,
		"purged", res.Purged,
		"errors", len(res.Errors))
	for _, e := range res.Errors {
		slog.Warn("gdpr scheduler: per-row error", "error", e.Error())
	}
}
