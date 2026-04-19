package antigaming

import "marketplace-backend/internal/search/features"

// newAccountRule implements `docs/ranking-v1.md` §7.5 — new account cap.
//
// The spec caps the final composite score at the persona median for
// profiles younger than 7 days. Because the features package does not know
// the persona median (that lives in the scorer), this rule sets the
// NewAccount flag on the PipelineResult so the scorer / business-rules
// layer can enforce the final cap.
//
// The rule also sets AccountAgeBonus to 0 so an especially manipulated
// new-account profile cannot accumulate the already-tiny age weight. This
// is a belt-and-suspenders move on top of the extractor's log curve, which
// would return ≈ 0.35 at 7 days.
//
// We detect on RawSignals.AccountAgeDays (which mirrors
// Features.RawAccountAgeDays) so the rule can be unit-tested in isolation.
func newAccountRule(f *features.Features, raw RawSignals, cfg Config) (*Penalty, bool) {
	if cfg.NewAccountAgeDays <= 0 {
		return nil, false
	}
	if raw.AccountAgeDays <= 0 || raw.AccountAgeDays >= cfg.NewAccountAgeDays {
		return nil, false
	}

	before := f.AccountAgeBonus
	f.AccountAgeBonus = 0
	return &Penalty{
		Rule:           RuleNewAccount,
		ProfileID:      raw.ProfileID,
		Persona:        raw.Persona,
		DetectionValue: float64(raw.AccountAgeDays),
		Threshold:      float64(cfg.NewAccountAgeDays),
		PenaltyFactor:  0,
		BeforeValue:    before,
		AfterValue:     0,
	}, true
}
