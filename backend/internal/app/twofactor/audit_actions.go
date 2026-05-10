package twofactor

import "marketplace-backend/internal/domain/audit"

// 2FA audit action keys. Defined here rather than in the domain/audit
// package because the rest of the codebase doesn't need them — keeps
// the canonical action list in domain/audit/entity.go free of
// per-feature noise. Cross-feature filters can still match these via
// the "auth.2fa." prefix.
const (
	// AuditActionChallengeIssued is recorded when a challenge row is
	// inserted and the email is dispatched. The metadata carries the
	// challenge_id so admins can correlate with the verify event.
	AuditActionChallengeIssued audit.Action = "auth.2fa.challenge_issued"

	// AuditActionChallengeSuccess is recorded when a verify call
	// matches and used_at is set. Pairs with the issued event for the
	// same challenge_id.
	AuditActionChallengeSuccess audit.Action = "auth.2fa.challenge_success"

	// AuditActionChallengeFailure is recorded on every failed verify
	// — no pending row, expired, attempts exhausted, or code mismatch.
	// The metadata.reason field disambiguates.
	AuditActionChallengeFailure audit.Action = "auth.2fa.challenge_failure"
)
