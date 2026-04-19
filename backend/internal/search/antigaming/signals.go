package antigaming

import "marketplace-backend/internal/search/features"

// RawSignals is the input bundle the anti-gaming pipeline reads in addition
// to the in-flight Features value. It captures the document-side signals
// that the features layer does not expose because they are not themselves
// ranking inputs — only inputs to detection rules (text content, recent
// review events, reviewer user IDs).
//
// The scorer / query service populates this struct once per candidate,
// before handing it to Pipeline.Apply. Keeping the data local (rather than
// re-querying inside a rule) makes the pipeline a pure function of its
// inputs.
type RawSignals struct {
	// ProfileID uniquely identifies the candidate — used for log correlation.
	ProfileID string

	// Persona used for log tagging.
	Persona features.Persona

	// Text content surfaces SkillsText + About, joined by a space, already
	// lowercased by the caller. The stuffing detector tokenises on
	// whitespace + punctuation and counts repetitions.
	Text string

	// RecentReviewTimestamps — list of review.created_at values (Unix
	// seconds) for reviews ≤ 24 h old. Used by rule 2 (velocity).
	// Caller-provided instead of re-queried inside the rule.
	RecentReviewTimestamps []int64

	// TotalReviewCount is the all-time review count (same as
	// Features.RawUniqueReviewers × a multiplicity factor). Used by
	// rule 2 to re-compute the effective count after the velocity cap.
	TotalReviewCount int

	// ReviewerIDs — slice of the unique reviewer user IDs. The default
	// linked-account detector is a no-op, but the interface accepts this
	// slice so a future implementation can check IP / email-domain / device
	// matches.
	ReviewerIDs []string

	// NowUnix is the request time — kept on the struct (not read from
	// clock) so the pipeline stays pure + deterministic under test.
	NowUnix int64

	// AccountAgeDays mirrors Features.RawAccountAgeDays for the new-account
	// rule. Duplicated here so the rule reads from RawSignals exclusively.
	AccountAgeDays int
}
