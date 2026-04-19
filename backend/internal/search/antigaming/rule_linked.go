package antigaming

import (
	"context"

	"marketplace-backend/internal/search/features"
)

// LinkedReviewersDetector is the hook implemented by whatever data source
// can tell us, for a given profile's reviewers, which ones share an IP /
// email domain / device fingerprint. For V1 this package ships a no-op
// implementation — production wires in a detector backed by the `users`
// + `sessions` tables outside this package.
//
// The detector returns the count of "linked" reviewers AMONG the supplied
// IDs. A value of 0 means the rule never fires.
//
// Signature kept small + context-scoped so it can eventually hit Redis /
// Postgres without changing the pipeline.
type LinkedReviewersDetector interface {
	LinkedCount(ctx context.Context, reviewerIDs []string) (int, error)
}

// NoopLinkedReviewersDetector is the default V1 implementation — always
// returns 0. Exported so cmd/api/main.go can wire it without referencing
// the unexported type.
type NoopLinkedReviewersDetector struct{}

// LinkedCount implements LinkedReviewersDetector on the no-op detector.
func (NoopLinkedReviewersDetector) LinkedCount(_ context.Context, _ []string) (int, error) {
	return 0, nil
}

// linkedRule implements `docs/ranking-v1.md` §7.3 — linked-account
// discount.
//
//	linked_fraction = linked_count / total_reviewers
//	if linked_fraction > LinkedMaxFraction:
//	    dampen rating_score_diverse by (1 - linked_fraction)
//
// The dampening mirrors the velocity rule's behaviour : reduce rather than
// zero, so a profile with 30 % linked reviews keeps 70 % of its signal.
//
// ctx is accepted to match the detector signature ; the rule itself is
// synchronous + fast once LinkedCount returns.
func linkedRule(ctx context.Context, f *features.Features, raw RawSignals, cfg Config, det LinkedReviewersDetector) (*Penalty, error) {
	if det == nil {
		return nil, nil
	}
	total := len(raw.ReviewerIDs)
	if total == 0 {
		return nil, nil
	}
	linked, err := det.LinkedCount(ctx, raw.ReviewerIDs)
	if err != nil {
		return nil, err
	}
	if linked <= 0 {
		return nil, nil
	}
	fraction := float64(linked) / float64(total)
	if fraction <= cfg.LinkedMaxFraction {
		return nil, nil
	}
	factor := 1.0 - fraction
	if factor < 0 {
		factor = 0
	}
	before := f.RatingScoreDiverse
	f.RatingScoreDiverse *= factor
	return &Penalty{
		Rule:           RuleLinkedAccounts,
		ProfileID:      raw.ProfileID,
		Persona:        raw.Persona,
		DetectionValue: fraction,
		Threshold:      cfg.LinkedMaxFraction,
		PenaltyFactor:  factor,
		BeforeValue:    before,
		AfterValue:     f.RatingScoreDiverse,
	}, nil
}
