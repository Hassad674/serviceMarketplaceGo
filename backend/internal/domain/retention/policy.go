// Package retention encodes the data-retention policies enforced by
// the retention scheduler (see app/retention). The package is pure
// domain — zero dependencies beyond the Go standard library — so it
// can be reused by the scheduler tests, the audit log, and any future
// reporting endpoint without dragging in adapter or app concerns.
//
// A Policy describes ONE table's retention rule:
//
//   - Table:       SQL identifier of the target table (validated by
//                  the adapter layer; this package does not parse SQL).
//   - AgeColumn:   timestamp column the sweep tests against (typically
//                  created_at or last_seen_at).
//   - MaxAge:      anything older than NOW() - MaxAge is eligible for
//                  the policy's strategy (delete / archive / anonymize).
//   - BatchSize:   how many rows the sweep removes per round-trip.
//                  Production-sized tables MUST run in small batches
//                  to avoid lock-storms and oversized WAL records.
//   - Strategy:    StrategyDelete (hard delete), StrategyArchive (move
//                  to a secondary table) or StrategyAnonymize (set the
//                  privacy-bearing column to NULL while keeping the
//                  row for analytics / ML).
//   - ArchiveTable: optional — only for StrategyArchive. The adapter
//                  uses it as the INSERT target when moving a batch.
//   - AnonymizeColumns: optional — only for StrategyAnonymize. The
//                  adapter sets each named column to NULL when the
//                  row crosses the retention boundary.
//
// Validate() enforces the cross-field invariants so a misconfigured
// policy fails fast at boot — never silently sweeps the wrong rows.
package retention

import (
	"errors"
	"fmt"
	"time"
)

// Strategy enumerates the supported retention actions.
type Strategy string

const (
	// StrategyDelete hard-deletes rows older than MaxAge.
	StrategyDelete Strategy = "delete"
	// StrategyArchive copies rows to ArchiveTable, then deletes them
	// from the source. The adapter does this in a single transaction
	// per batch so a crash never leaves the row in both tables or
	// neither.
	StrategyArchive Strategy = "archive"
	// StrategyAnonymize sets every column listed in AnonymizeColumns
	// to NULL for rows older than MaxAge. The row itself stays so
	// analytics aggregates and ML training data remain stable.
	StrategyAnonymize Strategy = "anonymize"
)

// IsValid reports whether s is one of the supported strategies.
func (s Strategy) IsValid() bool {
	switch s {
	case StrategyDelete, StrategyArchive, StrategyAnonymize:
		return true
	default:
		return false
	}
}

// DefaultBatchSize is the row-count cap the scheduler uses when a
// Policy does not override it. Tuned empirically: small enough to
// keep each transaction's WAL record manageable on a hot pool, large
// enough that the sweep makes progress on tables with millions of
// rows. The value is exposed as a constant so tests can reason about
// the same number.
const DefaultBatchSize = 1000

// MaxBatchesPerRun caps the per-tick batch loop. Without it a tick
// could sweep millions of rows in one go and starve the rest of the
// scheduler. The cap is intentionally generous (50 batches × 1000
// rows = 50k rows per policy per tick) — anything beyond that
// indicates a configuration bug or a backlog that needs operator
// attention, not a runtime decision.
const MaxBatchesPerRun = 50

// Sentinel errors. The retention service uses these to flag invalid
// policy declarations at boot so operators see a fail-fast log line
// instead of a silent no-op sweep.
var (
	ErrPolicyNameRequired      = errors.New("retention: policy name required")
	ErrPolicyTableRequired     = errors.New("retention: policy table required")
	ErrPolicyAgeColumnRequired = errors.New("retention: policy age_column required")
	ErrPolicyMaxAgeInvalid     = errors.New("retention: policy max_age must be > 0")
	ErrPolicyStrategyInvalid   = errors.New("retention: policy strategy must be delete, archive or anonymize")
	ErrPolicyArchiveMissing    = errors.New("retention: archive strategy requires archive_table")
	ErrPolicyAnonymizeMissing  = errors.New("retention: anonymize strategy requires anonymize_columns")
)

// Policy is the immutable, value-typed description of a single
// retention rule.
type Policy struct {
	Name             string
	Table            string
	AgeColumn        string
	MaxAge           time.Duration
	BatchSize        int
	Strategy         Strategy
	ArchiveTable     string
	AnonymizeColumns []string
}

// EffectiveBatchSize returns BatchSize when set, DefaultBatchSize
// otherwise. Centralised so the scheduler and the SQL adapter agree
// on the same cap.
func (p Policy) EffectiveBatchSize() int {
	if p.BatchSize > 0 {
		return p.BatchSize
	}
	return DefaultBatchSize
}

// Cutoff returns the timestamp boundary the sweep should compare
// against: rows whose AgeColumn is strictly less than Cutoff(now)
// are eligible. Splitting it as a method (rather than inlining in
// the SQL builder) keeps unit tests trivial — a clock injected at
// the call site lets us assert exact boundaries without sleeping.
func (p Policy) Cutoff(now time.Time) time.Time {
	return now.Add(-p.MaxAge)
}

// Validate enforces the per-strategy invariants. Returns the first
// violation as a wrapped sentinel so callers can errors.Is() against
// the package-level constants.
func (p Policy) Validate() error {
	if p.Name == "" {
		return ErrPolicyNameRequired
	}
	if p.Table == "" {
		return fmt.Errorf("%s: %w", p.Name, ErrPolicyTableRequired)
	}
	if p.AgeColumn == "" {
		return fmt.Errorf("%s: %w", p.Name, ErrPolicyAgeColumnRequired)
	}
	if p.MaxAge <= 0 {
		return fmt.Errorf("%s: %w", p.Name, ErrPolicyMaxAgeInvalid)
	}
	if !p.Strategy.IsValid() {
		return fmt.Errorf("%s: %w", p.Name, ErrPolicyStrategyInvalid)
	}
	switch p.Strategy {
	case StrategyArchive:
		if p.ArchiveTable == "" {
			return fmt.Errorf("%s: %w", p.Name, ErrPolicyArchiveMissing)
		}
	case StrategyAnonymize:
		if len(p.AnonymizeColumns) == 0 {
			return fmt.Errorf("%s: %w", p.Name, ErrPolicyAnonymizeMissing)
		}
	}
	return nil
}

// Result captures what one sweep loop accomplished for a single
// policy. The scheduler logs this so operators can correlate ticks
// with rows-removed counts and spot regressions without parsing SQL.
type Result struct {
	Policy   string
	Affected int
	Batches  int
}
