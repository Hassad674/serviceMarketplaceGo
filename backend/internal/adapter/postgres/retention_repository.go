package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"marketplace-backend/internal/domain/retention"
)

// RetentionRepository implements repository.RetentionRepository for
// PostgreSQL. Each Sweep call commits one batch in its own short
// transaction so a long-running retention pass never holds a single
// multi-million-row lock and so a crash in the middle of a sweep
// leaves the database in a consistent state — the next tick will
// pick up where this one left off.
//
// SECURITY: the policy struct carries a Table / AgeColumn / column
// list that this adapter splices into raw SQL. Every spliced value
// is validated against an allowlist (validatePolicy below) before
// the SQL is built. Outside of the allowlist, the adapter REFUSES
// to run — there is no path to inject arbitrary SQL via a Policy
// from the wiring layer.
//
// RLS: the audit_logs / messages / notifications tables are
// RLS-protected (migration 125). The retention sweep tags its
// context with system.WithSystemActor in the calling service, which
// causes the routed DB pool to pick the BYPASSRLS connection. This
// adapter intentionally does not call RunInTxWithTenant — there is
// no per-user tenant for a system-wide retention pass.
type RetentionRepository struct {
	db *sql.DB
}

// NewRetentionRepository wires the repo onto a *sql.DB. Production
// callers pass the admin pool (BYPASSRLS) so the sweep can touch
// every row regardless of RLS context. The retention service tags
// its context as a system actor for clarity, but the underlying
// pool selection is the load-bearing guard.
func NewRetentionRepository(db *sql.DB) *RetentionRepository {
	return &RetentionRepository{db: db}
}

// retentionAllowlist is the single source of truth for which
// (table, age_column, archive_table, anonymize_columns) tuples this
// adapter is willing to accept. Any policy referencing identifiers
// outside this map is rejected before any SQL is built.
//
// Keep this list in sync with domain/retention/policies.go — a
// validation test in retention_repository_test.go (and the policy
// integration tests) will fail loudly if the two drift apart.
var retentionAllowlist = map[string]retentionTableSpec{
	"messages": {
		ageColumns:       map[string]bool{"created_at": true},
		anonymizeColumns: map[string]bool{},
	},
	"notifications": {
		ageColumns:       map[string]bool{"created_at": true},
		anonymizeColumns: map[string]bool{},
	},
	"device_tokens": {
		ageColumns:       map[string]bool{"last_seen_at": true, "created_at": true},
		anonymizeColumns: map[string]bool{},
	},
	"search_queries": {
		ageColumns:       map[string]bool{"created_at": true},
		anonymizeColumns: map[string]bool{"user_id": true, "session_id": true},
	},
	"audit_logs": {
		ageColumns:       map[string]bool{"created_at": true},
		anonymizeColumns: map[string]bool{},
		archiveTables:    map[string]bool{"audit_logs_archive": true},
	},
}

type retentionTableSpec struct {
	ageColumns       map[string]bool
	anonymizeColumns map[string]bool
	archiveTables    map[string]bool
}

// errPolicyNotAllowlisted is the sentinel returned when validatePolicy
// rejects an identifier. Wrapped by Sweep so callers can match on it
// without depending on this private symbol — the public surface is
// the formatted error message.
var errPolicyNotAllowlisted = errors.New("retention: policy references a table or column outside the allowlist")

// validatePolicy enforces the allowlist + the cross-field rules.
// Returns the original sentinel + a formatted reason so logs are
// useful without leaking the SQL template.
func validatePolicy(policy retention.Policy) error {
	if err := policy.Validate(); err != nil {
		return err
	}
	spec, ok := retentionAllowlist[policy.Table]
	if !ok {
		return fmt.Errorf("%w: table=%q", errPolicyNotAllowlisted, policy.Table)
	}
	if !spec.ageColumns[policy.AgeColumn] {
		return fmt.Errorf("%w: table=%q age_column=%q", errPolicyNotAllowlisted, policy.Table, policy.AgeColumn)
	}
	switch policy.Strategy {
	case retention.StrategyArchive:
		if !spec.archiveTables[policy.ArchiveTable] {
			return fmt.Errorf("%w: table=%q archive_table=%q", errPolicyNotAllowlisted, policy.Table, policy.ArchiveTable)
		}
	case retention.StrategyAnonymize:
		for _, col := range policy.AnonymizeColumns {
			if !spec.anonymizeColumns[col] {
				return fmt.Errorf("%w: table=%q anonymize_column=%q", errPolicyNotAllowlisted, policy.Table, col)
			}
		}
	}
	return nil
}

// Sweep dispatches to the strategy-specific helper. Returns the
// number of rows affected by this single batch.
func (r *RetentionRepository) Sweep(ctx context.Context, policy retention.Policy) (int, error) {
	if err := validatePolicy(policy); err != nil {
		return 0, err
	}
	now := time.Now().UTC()
	switch policy.Strategy {
	case retention.StrategyDelete:
		return r.sweepDelete(ctx, policy, now)
	case retention.StrategyArchive:
		return r.sweepArchive(ctx, policy, now)
	case retention.StrategyAnonymize:
		return r.sweepAnonymize(ctx, policy, now)
	default:
		// Unreachable: validatePolicy already accepts only the three
		// supported strategies. Defense in depth — never fall through
		// to a no-op that would silently keep stale rows alive.
		return 0, fmt.Errorf("retention: unsupported strategy %q", policy.Strategy)
	}
}

// sweepDelete handles StrategyDelete: hard-DELETE LIMITed by the
// policy's batch size. Postgres does not support LIMIT on DELETE
// directly. Naive `DELETE … WHERE id IN (SELECT … LIMIT N)` re-runs
// the subquery for every outer row in the planner's nested-loop
// semi-join — the LIMIT is per inner execution, not per statement,
// and the actual rows-deleted count exceeds N.
//
// We sidestep that pitfall with a materialising CTE: the SELECT is
// evaluated exactly once and the DELETE joins against the snapshot.
// FOR UPDATE SKIP LOCKED keeps concurrent sweeps cooperating.
func (r *RetentionRepository) sweepDelete(ctx context.Context, policy retention.Policy, now time.Time) (int, error) {
	cutoff := policy.Cutoff(now)
	batch := policy.EffectiveBatchSize()
	// #nosec G201 -- table and age_column are validated against retentionAllowlist before reaching this template.
	q := fmt.Sprintf(`
        WITH eligible AS MATERIALIZED (
            SELECT id FROM %s
             WHERE %s < $1
             ORDER BY %s ASC
             LIMIT %d
             FOR UPDATE SKIP LOCKED
        )
        DELETE FROM %s
         WHERE id IN (SELECT id FROM eligible)`,
		policy.Table,
		policy.AgeColumn,
		policy.AgeColumn,
		batch,
		policy.Table,
	)
	return r.execBatch(ctx, q, cutoff)
}

// sweepAnonymize handles StrategyAnonymize: UPDATE … SET col = NULL
// for rows older than the cutoff that are not already fully
// anonymized. The "IS NOT NULL" guard makes the sweep idempotent —
// once a row is anonymized the next pass skips it.
//
// Uses the same materialised-CTE shape as sweepDelete to ensure the
// LIMIT applies to the entire UPDATE, not per-outer-row.
func (r *RetentionRepository) sweepAnonymize(ctx context.Context, policy retention.Policy, now time.Time) (int, error) {
	cutoff := policy.Cutoff(now)
	batch := policy.EffectiveBatchSize()
	setExprs := make([]string, 0, len(policy.AnonymizeColumns))
	notNull := make([]string, 0, len(policy.AnonymizeColumns))
	for _, col := range policy.AnonymizeColumns {
		setExprs = append(setExprs, fmt.Sprintf("%s = NULL", col))
		notNull = append(notNull, fmt.Sprintf("%s IS NOT NULL", col))
	}
	// #nosec G201 -- columns validated against retentionAllowlist.
	q := fmt.Sprintf(`
        WITH eligible AS MATERIALIZED (
            SELECT id FROM %s
             WHERE %s < $1
               AND (%s)
             ORDER BY %s ASC
             LIMIT %d
             FOR UPDATE SKIP LOCKED
        )
        UPDATE %s
           SET %s
         WHERE id IN (SELECT id FROM eligible)`,
		policy.Table,
		policy.AgeColumn,
		strings.Join(notNull, " OR "),
		policy.AgeColumn,
		batch,
		policy.Table,
		strings.Join(setExprs, ", "),
	)
	return r.execBatch(ctx, q, cutoff)
}

// sweepArchive handles StrategyArchive: copy a batch into the
// archive table, then delete the same rows from the source. Both
// statements run inside a single short transaction so a crash never
// leaves a row in both tables (duplicates) or neither (data loss).
//
// The source SELECT uses FOR UPDATE SKIP LOCKED so concurrent
// schedulers cooperate without coordination. The DELETE matches on
// the same id list returned from the SELECT.
func (r *RetentionRepository) sweepArchive(ctx context.Context, policy retention.Policy, now time.Time) (int, error) {
	cutoff := policy.Cutoff(now)
	batch := policy.EffectiveBatchSize()

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("retention archive: begin tx: %w", err)
	}
	defer func() {
		// Rollback is a no-op after a successful commit.
		_ = tx.Rollback()
	}()

	// audit_logs has a fixed shape — the archive table mirrors every
	// column. We hard-code the column list here so the INSERT is
	// idempotent and the schema drift is caught at compile time
	// (the test suite asserts the column lists match).
	if policy.Table != "audit_logs" || policy.ArchiveTable != "audit_logs_archive" {
		return 0, fmt.Errorf("retention archive: unsupported archive pair %s -> %s", policy.Table, policy.ArchiveTable)
	}

	const archiveSQL = `
        WITH eligible AS MATERIALIZED (
            SELECT id FROM audit_logs
             WHERE created_at < $1
             ORDER BY created_at ASC
             LIMIT $2
             FOR UPDATE SKIP LOCKED
        ),
        moved AS (
            DELETE FROM audit_logs
             WHERE id IN (SELECT id FROM eligible)
         RETURNING id, user_id, action, resource_type, resource_id, metadata, ip_address, created_at
        )
        INSERT INTO audit_logs_archive
            (id, user_id, action, resource_type, resource_id, metadata, ip_address, created_at)
        SELECT id, user_id, action, resource_type, resource_id, metadata, ip_address, created_at
          FROM moved`
	res, err := tx.ExecContext(ctx, archiveSQL, cutoff, batch)
	if err != nil {
		return 0, fmt.Errorf("retention archive: exec: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("retention archive: rows affected: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("retention archive: commit: %w", err)
	}
	return int(affected), nil
}

// execBatch wraps a single-shot DELETE/UPDATE in a 30s timeout
// context and returns the rows-affected count. Centralised so
// every strategy reports its progress the same way.
func (r *RetentionRepository) execBatch(ctx context.Context, query string, args ...any) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	res, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("retention exec: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("retention exec: rows affected: %w", err)
	}
	return int(n), nil
}
