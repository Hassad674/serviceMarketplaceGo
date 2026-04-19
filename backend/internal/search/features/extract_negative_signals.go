package features

// ExtractNegativeSignals implements the bounded penalty described in
// `docs/ranking-v1.md` §5.3.
//
//	lost_disputes_count = count(disputes with refund outcome where respondent == profile)
//	negative_signals    = min(disputePenaltyCap, lost_disputes_count × disputePenalty)
//
// Defaults : disputePenalty = 0.10, disputePenaltyCap = 0.30 — so three lost
// disputes saturate the penalty at 30 %, and no profile can have more than a
// 30 % penalty from dispute history alone.
//
// Loss-of-dispute is the ONLY negative signal kept in V1. Response-rate
// deficits are already captured in the positive feature (extract_response_rate).
// Cancellations and late deliveries are deferred (too noisy / ambiguous).
func ExtractNegativeSignals(doc SearchDocumentLite, cfg Config) (penalty float64, raw int) {
	raw = int(doc.LostDisputesCount)
	if raw < 0 {
		raw = 0
	}
	if cfg.DisputePenalty <= 0 {
		return 0, raw
	}
	pen := float64(raw) * cfg.DisputePenalty
	cap := cfg.DisputePenaltyCap
	if cap < 0 {
		cap = 0
	}
	if pen > cap {
		pen = cap
	}
	if pen < 0 {
		pen = 0
	}
	return pen, raw
}
