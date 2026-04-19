package antigaming

import (
	"context"

	"marketplace-backend/internal/search/features"
)

// Pipeline runs the five anti-gaming rules on a (features, signals) pair.
// Rules execute in a deterministic order + mutate the Features value
// in-place. Penalty records are accumulated, emitted to the Logger, and
// surfaced on PipelineResult for downstream consumers (scorer, dashboard).
//
// The pipeline is safe to share across goroutines — every field is set
// once at construction and read-only afterwards.
type Pipeline struct {
	cfg      Config
	detector LinkedReviewersDetector
	logger   Logger
}

// NewPipeline builds a Pipeline. A nil detector defaults to the no-op
// implementation ; a nil logger defaults to NoopLogger.
func NewPipeline(cfg Config, detector LinkedReviewersDetector, logger Logger) *Pipeline {
	if detector == nil {
		detector = NoopLinkedReviewersDetector{}
	}
	if logger == nil {
		logger = NoopLogger{}
	}
	return &Pipeline{cfg: cfg, detector: detector, logger: logger}
}

// PipelineResult holds the aggregate outcome of a single pipeline pass.
// The scorer reads NewAccountCapped to enforce the final composite cap
// (§7.5 — "at best rank at the persona median").
type PipelineResult struct {
	Penalties        []Penalty
	NewAccountCapped bool
}

// Apply runs the five detection rules in order + returns the aggregate
// result. The Features value is mutated in place.
//
// Ctx is accepted so the linked-account detector can honour timeouts once
// wired to a real data source. Rules that do not need the context ignore
// it — the pipeline never panics on a nil ctx (callers pass context.TODO()
// in unit tests).
func (p *Pipeline) Apply(ctx context.Context, f *features.Features, raw RawSignals) PipelineResult {
	if f == nil {
		return PipelineResult{}
	}
	var res PipelineResult
	res.Penalties = make([]Penalty, 0, 5)

	// Rule 1 — stuffing
	if pen := stuffingRule(f, raw, p.cfg); pen != nil {
		p.logger.LogPenalty(*pen)
		res.Penalties = append(res.Penalties, *pen)
	}

	// Rule 2 — velocity
	if pen := velocityRule(f, raw, p.cfg); pen != nil {
		p.logger.LogPenalty(*pen)
		res.Penalties = append(res.Penalties, *pen)
	}

	// Rule 3 — linked accounts
	// Errors from the detector are swallowed at this layer : the rule
	// is a silent cap and a transient DB error must not take down the
	// search path. The scorer still runs with the (unmodified) rating
	// score. Real errors are logged upstream by the adapter.
	if ctx == nil {
		ctx = context.Background()
	}
	if pen, err := linkedRule(ctx, f, raw, p.cfg, p.detector); err == nil && pen != nil {
		p.logger.LogPenalty(*pen)
		res.Penalties = append(res.Penalties, *pen)
	}

	// Rule 4 — unique reviewer floor
	if pen := reviewerFloorRule(f, raw, p.cfg); pen != nil {
		p.logger.LogPenalty(*pen)
		res.Penalties = append(res.Penalties, *pen)
	}

	// Rule 5 — new account cap. The scorer uses NewAccountCapped to
	// enforce the composite-score cap.
	if pen, capped := newAccountRule(f, raw, p.cfg); capped {
		if pen != nil {
			p.logger.LogPenalty(*pen)
			res.Penalties = append(res.Penalties, *pen)
		}
		res.NewAccountCapped = true
	}

	return res
}

// Config returns a copy of the pipeline's Config for diagnostics / tests.
func (p *Pipeline) Config() Config { return p.cfg }
