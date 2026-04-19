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
// When the document does not carry a last_active_at timestamp (doc.LastActiveAt
// == 0) we return 0 : an unknown activity date is indistinguishable from a
// dormant profile and the weight is tiny (2-3 %).
//
// NowUnix comes from the document lite copy — keeping the extractor a pure
// function of its inputs lets the LTR agent replay historical queries later
// without time-travel shenanigans.
func ExtractLastActiveDays(doc SearchDocumentLite, cfg Config) float64 {
	if doc.LastActiveAt <= 0 || doc.NowUnix <= 0 {
		return 0
	}
	decay := float64(cfg.LastActiveDecayDays)
	if decay <= 0 {
		return 0
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
