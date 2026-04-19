package features

// ExtractVerifiedMature implements the binary signal described in
// `docs/ranking-v1.md` §3.2-6.
//
//	is_verified_mature = 1 iff is_verified = true AND account_age_days >= 30
//	                     0 otherwise
//
// Threshold env-tunable via Config.VerifiedMatureMinAgeDays (default 30) —
// long enough to deter disposable accounts, short enough to not penalise
// genuine newcomers.
func ExtractVerifiedMature(doc SearchDocumentLite, cfg Config) float64 {
	if !doc.IsVerified {
		return 0
	}
	if int(doc.AccountAgeDays) < cfg.VerifiedMatureMinAgeDays {
		return 0
	}
	return 1
}
