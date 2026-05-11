package handler

// Test-only exports. Keeps internal handler logic testable from
// handler_test (external) without bloating the public API.

import paymentapp "marketplace-backend/internal/app/payment"

// AuditEntryExposed re-exports the internal auditEntry shape so test
// helpers in handler_test can produce concrete instances. The audit
// logger interface (walletAuditLogger) accepts *auditEntry — pointer
// equality on the underlying struct is what matters, not the alias.
type AuditEntryExposed = auditEntry

// ParseMissionAmountFromMessageForTest exposes the pure helper for
// unit tests in handler_test.
func ParseMissionAmountFromMessageForTest(msg string) int64 {
	return parseMissionAmountFromMessage(msg)
}

// MissionErrCodeForTest exposes the error→code mapping for unit
// tests.
func MissionErrCodeForTest(err error) string {
	return missionErrCode(err)
}

// MissionDrainedFromResultForTest exposes the PayoutResult parser.
func MissionDrainedFromResultForTest(r *paymentapp.PayoutResult) int64 {
	return missionDrainedFromResult(r)
}

// MissionLegForTest exposes the breakdown composer's mission leg.
func MissionLegForTest(w *paymentapp.WalletOverview) summaryBreakdownLeg {
	return missionLeg(w)
}

// MissionTransactionForTest exposes the timeline mapper.
func MissionTransactionForTest(r paymentapp.WalletRecord) summaryTransaction {
	return missionTransaction(r)
}
