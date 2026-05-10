package repository

import (
	"context"

	"marketplace-backend/internal/domain/retention"
)

// RetentionRepository is the data-access port the retention scheduler
// uses to enforce a single Policy. The interface is intentionally
// thin — one method per round-trip — so the concrete adapter can wrap
// each batch in its own short transaction without leaking SQL details
// into the app layer.
//
// Sweep applies the policy's strategy (delete / archive / anonymize)
// against rows whose AgeColumn is older than `now - policy.MaxAge`,
// up to `policy.EffectiveBatchSize()` rows per call. It returns the
// number of rows affected so the caller can loop "until empty" with a
// safety cap (see retention.MaxBatchesPerRun).
//
// The repository implementation is responsible for:
//   - validating the policy with policy.Validate before any SQL
//   - using parameterised queries everywhere (the strategy decides
//     the SQL shape, not policy.Table verbatim — the adapter has an
//     allowlist)
//   - committing each batch in its own transaction so a long sweep
//     never holds a single multi-million-row lock
//   - returning the stable count of rows touched (rows DELETED for
//     delete + archive, rows UPDATED for anonymize)
type RetentionRepository interface {
	// Sweep applies one batch of the policy at the given clock time
	// and returns how many rows were affected. Callers typically loop
	// until Sweep returns zero or hits the per-run cap.
	Sweep(ctx context.Context, policy retention.Policy) (int, error)
}
