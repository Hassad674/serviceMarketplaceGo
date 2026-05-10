// Package retention orchestrates the periodic enforcement of data
// retention policies declared in domain/retention. The service does
// not own the SQL — it delegates to a RetentionRepository — but it
// owns:
//
//   - validating the policy set at boot (fail-fast on misconfig),
//   - looping the per-policy sweep with a per-run batch cap,
//   - never aborting the loop on a single-policy failure (one bad
//     table must not block the rest of the retention pass),
//   - emitting structured slog events so the operator can graph
//     retention progress without parsing SQL.
//
// The scheduler (Scheduler in scheduler.go) is a thin wrapper that
// drives Run on a ticker. Pulling the loop logic into the service
// keeps the scheduler test-free: the service is the unit of behaviour
// and the scheduler only cares about timing.
package retention

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"marketplace-backend/internal/domain/retention"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/system"
)

// Service runs one retention pass per Run() call.
type Service struct {
	repo     repository.RetentionRepository
	policies []retention.Policy
	maxBatch int
}

// NewService validates every policy in `policies` and returns the
// service bound to `repo`. A misconfigured policy panics at boot —
// this is intentional: the privacy contract must not silently
// regress because a typo crept into the policy struct. Dev workflows
// that want to keep booting with a partial policy set should remove
// the policy from the slice rather than ship it broken.
func NewService(repo repository.RetentionRepository, policies []retention.Policy) (*Service, error) {
	if repo == nil {
		return nil, errors.New("retention service: repo is required")
	}
	if len(policies) == 0 {
		return nil, errors.New("retention service: at least one policy required")
	}
	for i, p := range policies {
		if err := p.Validate(); err != nil {
			return nil, fmt.Errorf("retention service: policy[%d]: %w", i, err)
		}
	}
	return &Service{
		repo:     repo,
		policies: policies,
		maxBatch: retention.MaxBatchesPerRun,
	}, nil
}

// WithMaxBatchesPerRun overrides the per-policy batch cap. Returns the
// receiver for builder-style chaining at wire time. Tests use this to
// shorten the loop without redeclaring constants.
func (s *Service) WithMaxBatchesPerRun(n int) *Service {
	if n > 0 {
		s.maxBatch = n
	}
	return s
}

// Run sweeps every configured policy in declaration order. Per-policy
// errors are logged but do not abort the loop — a transient lock on
// `messages` must not stop `notifications` from being pruned.
//
// Returns one Result per policy plus the slice of errors collected.
// The caller (the scheduler) does not care about the errors beyond
// logging them; the slice is exposed so the unit tests can assert
// "no policy was silently skipped".
func (s *Service) Run(ctx context.Context) ([]retention.Result, []error) {
	// Tag the context as a system actor so the routed DB pool picks
	// the BYPASSRLS connection. Without this tag, the sweep against
	// audit_logs / messages / notifications would hit the per-user
	// RLS policy and find zero rows. Background-job convention,
	// matching wire_dispute / wire_gdpr.
	ctx = system.WithSystemActor(ctx)

	results := make([]retention.Result, 0, len(s.policies))
	errs := make([]error, 0)

	for _, policy := range s.policies {
		select {
		case <-ctx.Done():
			errs = append(errs, fmt.Errorf("retention: cancelled mid-run after %d policies: %w", len(results), ctx.Err()))
			return results, errs
		default:
		}

		res, err := s.runPolicy(ctx, policy)
		results = append(results, res)
		if err != nil {
			errs = append(errs, fmt.Errorf("retention: policy %q: %w", policy.Name, err))
			slog.Warn("retention: policy failed",
				"policy", policy.Name,
				"affected", res.Affected,
				"batches", res.Batches,
				"error", err)
			continue
		}
		if res.Affected > 0 {
			slog.Info("retention: policy swept",
				"policy", policy.Name,
				"affected", res.Affected,
				"batches", res.Batches)
		} else {
			slog.Debug("retention: policy idle",
				"policy", policy.Name)
		}
	}
	return results, errs
}

// runPolicy loops Sweep up to MaxBatchesPerRun times or until the
// repo returns zero. Splitting it out keeps Run() readable and
// testable.
func (s *Service) runPolicy(ctx context.Context, policy retention.Policy) (retention.Result, error) {
	res := retention.Result{Policy: policy.Name}
	for i := 0; i < s.maxBatch; i++ {
		select {
		case <-ctx.Done():
			return res, ctx.Err()
		default:
		}

		affected, err := s.repo.Sweep(ctx, policy)
		if err != nil {
			return res, err
		}
		res.Batches++
		res.Affected += affected
		if affected == 0 {
			return res, nil
		}
		// Yield to the runtime scheduler between batches. A 50ms
		// pause keeps the retention sweep "background" priority and
		// gives the application's main workload room to run on a
		// heavily loaded box. The sleep is interrupt-aware via
		// ctx.Done so a graceful shutdown never waits longer than
		// one batch.
		select {
		case <-ctx.Done():
			return res, ctx.Err()
		case <-time.After(50 * time.Millisecond):
		}
	}
	return res, nil
}
