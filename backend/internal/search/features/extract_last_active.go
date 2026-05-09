package features

// ExtractLastActiveDays implements the hyperbolic-decay freshness score
// described in `docs/ranking-v1.md` §3.2-8.
//
//	last_active_days_score = 1 / (1 + last_active_days / decay_days)
//
// Reference table (decay_days = 30, the spec default) :
//
//	days since last active | score
//	           0           | 1.00
//	          15           | 0.67
//	          30           | 0.50
//	          90           | 0.25
//	         180           | 0.14
//	         365           | 0.08
//
// Unknown LastActiveAt
//
// When the document does not carry a last_active_at timestamp the spec
// implies a "dormant baseline" — we genuinely don't know when the
// profile was last active, so we treat it as ≥ 6 months idle rather
// than collapsing the signal to a hard 0 (which would punish profiles
// the indexer simply has not yet caught up on). The 6-month equivalent
// at the default decay (30 days) yields ≈ 1 / (1 + 180/30) = 0.143,
// landing on the same value as a profile last seen six months ago.
//
// This keeps the feature monotonic for the indexer rollout: the moment
// LastActiveAt populates, the score moves continuously rather than
// jumping from 0.143 to whatever the live computation produces.
//
// NowUnix comes from the document lite copy — keeping the extractor a
// pure function of its inputs lets the LTR agent replay historical
// queries later without time-travel shenanigans. When NowUnix itself is
// missing we cannot compute the decay either way and return 0 (no
// reference clock = no signal).
func ExtractLastActiveDays(doc SearchDocumentLite, cfg Config) float64 {
	decay := float64(cfg.LastActiveDecayDays)
	if decay <= 0 {
		return 0
	}
	// Unknown clock → nothing to anchor against. Both the live path
	// and the dormant baseline below need a reference instant.
	if doc.NowUnix <= 0 {
		return 0
	}
	if doc.LastActiveAt <= 0 {
		// Dormant baseline — see function comment for the rationale.
		return clamp01(1.0 / (1.0 + dormantBaselineDays/decay))
	}

	deltaSeconds := doc.NowUnix - doc.LastActiveAt
	if deltaSeconds < 0 {
		// Future timestamps (clock skew) are treated as "right now".
		deltaSeconds = 0
	}
	const secondsPerDay = 86400
	days := float64(deltaSeconds) / secondsPerDay

	return clamp01(1.0 / (1.0 + days/decay))
}

// dormantBaselineDays is the conservative "we don't know" stand-in for
// LastActiveAt — six months. Picked so the resulting score (≈ 0.143 at
// the default decay) matches a profile last seen six months ago,
// without rewarding indexer gaps with a higher value.
const dormantBaselineDays = 180.0
