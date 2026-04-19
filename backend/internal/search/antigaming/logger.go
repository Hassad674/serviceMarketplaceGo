package antigaming

import (
	"log/slog"

	"marketplace-backend/internal/search/features"
)

// Rule identifies which detection rule emitted a penalty. Exported so the
// scorer / business-rules layer + tests can match on the enum without
// referencing strings.
type Rule string

const (
	RuleKeywordStuffing Rule = "keyword_stuffing"
	RuleReviewVelocity  Rule = "review_velocity"
	RuleLinkedAccounts  Rule = "linked_accounts"
	RuleReviewerFloor   Rule = "reviewer_floor"
	RuleNewAccount      Rule = "new_account"
)

// Penalty is the structured record emitted by each rule when it fires. The
// pipeline accumulates Penalty values and surfaces them on PipelineResult
// so the scorer + logging layer can process them together.
//
// DetectionValue and Threshold are always positive scalars (count or
// fraction) so the admin dashboard can sort by "worst offenders".
//
// BeforeValue and AfterValue are optional — populated when the rule
// modifies a specific feature so log lines can show the delta. Stuffing
// leaves them at 0 because it multiplies rather than assigns a specific
// value.
type Penalty struct {
	Rule           Rule
	ProfileID      string
	Persona        features.Persona
	DetectionValue float64
	Threshold      float64
	PenaltyFactor  float64
	BeforeValue    float64
	AfterValue     float64
}

// Logger is the interface the pipeline uses to emit penalty events.
// Implementations may do nothing (test), write to slog (production), or
// buffer for later export (future admin dashboard).
type Logger interface {
	LogPenalty(p Penalty)
}

// SlogLogger writes each penalty as a single structured JSON line via
// Go's slog. Output lines match the schema in `docs/ranking-v1.md` §7.6.
type SlogLogger struct{}

// LogPenalty implements Logger on SlogLogger.
func (SlogLogger) LogPenalty(p Penalty) {
	slog.Info("ranking.penalty_applied",
		"rule", string(p.Rule),
		"profile_id", p.ProfileID,
		"persona", string(p.Persona),
		"detection_value", p.DetectionValue,
		"threshold", p.Threshold,
		"penalty_factor", p.PenaltyFactor,
		"before_value", p.BeforeValue,
		"after_value", p.AfterValue,
	)
}

// NoopLogger drops every event — used by property tests to avoid log spam.
type NoopLogger struct{}

// LogPenalty implements Logger on NoopLogger.
func (NoopLogger) LogPenalty(_ Penalty) {}

// RecordingLogger keeps every penalty in an in-memory slice. Used by tests
// to assert the exact set of penalties emitted by a run.
type RecordingLogger struct {
	Penalties []Penalty
}

// LogPenalty implements Logger on RecordingLogger.
func (l *RecordingLogger) LogPenalty(p Penalty) {
	l.Penalties = append(l.Penalties, p)
}
