package features

import "math"

// ExtractAccountAgeBonus implements the log-scaled maturity bonus described
// in `docs/ranking-v1.md` §3.2-9.
//
//	account_age_bonus = min(1.0, log(1 + account_age_days) / log(1 + 365))
//
// Reference table (cap = 365) :
//
//	account_age_days | bonus
//	       0         |  0.00
//	       7         |  0.35
//	      30         |  0.58
//	      90         |  0.77
//	     365         |  1.00
//	    > 365        |  1.00 (capped)
//
// This feature gets a very small weight (1-2 %) so it nudges ranking without
// becoming a "veterans win" signal.
func ExtractAccountAgeBonus(doc SearchDocumentLite, cfg Config) float64 {
	days := int(doc.AccountAgeDays)
	if days < 0 {
		days = 0
	}
	cap := cfg.AccountAgeCapDays
	if cap <= 0 {
		return 0
	}
	if days > cap {
		days = cap
	}
	den := math.Log1p(float64(cap))
	if den == 0 {
		return 0
	}
	return clamp01(math.Log1p(float64(days)) / den)
}
