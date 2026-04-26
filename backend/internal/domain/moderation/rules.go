// Package moderation holds the business rules that decide how a piece
// of user-generated text should be handled after an automated moderation
// scan. The rules are intentionally separate from the moderation engine
// (OpenAI, Comprehend, etc.) so that swapping providers does not require
// re-deriving the thresholds, and so every consumer (messages, reviews,
// future jobs/proposals/bios) applies the same policy.
//
// The policy is deliberately permissive by default (a simple insult is
// flagged but remains visible to everyone) and strict for five categories
// that the project treats as zero-tolerance because they are illegal or
// severely harmful — explicit threats, hate incitation to violence,
// graphic-violence descriptions, explicit self-harm instructions, and
// any sexual content involving minors.
package moderation

import (
	"marketplace-backend/internal/port/service"
)

// Status is the decided outcome of a moderation scan. The value lands
// directly in the moderation_status column on messages/reviews/etc.
type Status string

const (
	// StatusClean means the content passed moderation and needs no action.
	// The zero value of the column is '' (empty string), so Clean maps to
	// "" when we want to skip writing anything — DecideStatus returns
	// "" in that case.
	StatusClean Status = ""

	// StatusFlagged means the content is visible to everyone but marked
	// for admin review. Typically a borderline insult.
	StatusFlagged Status = "flagged"

	// StatusHidden means the content is hidden from other users (the
	// author still sees it with a "pending review" banner) but kept in
	// the database. Used when the global toxicity is high but no single
	// category crosses the zero-tolerance bar.
	StatusHidden Status = "hidden"

	// StatusDeleted means the content is soft-deleted: invisible to
	// everyone including the author, but the row stays in the database
	// for audit, legal discovery, and one-click admin restoration.
	// Reached only via the zero-tolerance category rules below.
	StatusDeleted Status = "deleted"

	// StatusBlocked means the content was REFUSED at creation time —
	// the row in moderation_results captures the rejected attempt but
	// no row was ever inserted into the source table (user, profile,
	// job, …). Reached only via the synchronous blocking mode used on
	// public-facing surfaces (display_name, job titles, profile bio).
	StatusBlocked Status = "blocked"
)

// Reason codes returned alongside Status so the audit log and admin UI
// can explain why the decision was taken. Codes are stable identifiers;
// user-facing labels are mapped on the frontend.
const (
	ReasonNone                    = ""
	ReasonAutoFlagScore           = "auto_flag_score"
	ReasonAutoHideHighScore       = "auto_hide_high_score"
	ReasonAutoDeleteExtremeScore  = "auto_delete_extreme_score"
	ReasonAutoDeleteMinors        = "auto_delete_minors"
	ReasonAutoDeleteSelfHarmInstr = "auto_delete_selfharm_instr"
	ReasonAutoDeleteThreat        = "auto_delete_threat"
	ReasonAutoDeleteHateThreat    = "auto_delete_hate_threat"
	ReasonAutoDeleteViolenceGfx   = "auto_delete_violence_graphic"
	// ReasonBlockedCreate is set when synchronous blocking mode rejects
	// content at creation time. Distinct from auto_delete because the
	// content was never persisted in its source table.
	ReasonBlockedCreate           = "blocked_create"
)

// Threshold constants. Kept as exported vars — not `const` — so that
// integration tests can tweak them without recompiling, and so a future
// move to env-driven config is a one-file change.
//
// The five category thresholds below are the zero-tolerance bar: any
// one of them crossing its threshold triggers a soft-delete, regardless
// of the global MaxScore. sexual/minors is intentionally very low
// (0.30) because false positives here are vastly preferable to a false
// negative, and admins can restore in one click.
var (
	ThresholdFlag                    = 0.50
	ThresholdHide                    = 0.90
	// ThresholdDeleteExtreme is the "fatal" global threshold. When ANY
	// category score crosses it, the content is soft-deleted regardless
	// of which category triggered it. This catches extreme-but-non-
	// threatening abuse (very strong insults, severe harassment without
	// explicit threats) that the per-category zero-tolerance matrix
	// alone would only hide. 0.95 keeps the policy permissive enough
	// that borderline content stays visible to its author for review.
	ThresholdDeleteExtreme           = 0.95
	ThresholdDeleteSexualMinors      = 0.30
	ThresholdDeleteSelfHarmInstr     = 0.70
	ThresholdDeleteHarassmentThreat  = 0.80
	ThresholdDeleteHateThreat        = 0.80
	ThresholdDeleteViolenceGraphic   = 0.85
)

// OpenAI omni-moderation category names (the moderation adapter maps
// them verbatim into TextModerationResult.Labels). Centralised here so
// the rules engine and the adapter agree on spelling.
const (
	CategorySexual               = "sexual"
	CategorySexualMinors         = "sexual/minors"
	CategoryHarassment           = "harassment"
	CategoryHarassmentThreaten   = "harassment/threatening"
	CategoryHate                 = "hate"
	CategoryHateThreatening      = "hate/threatening"
	CategoryIllicit              = "illicit"
	CategoryIllicitViolent       = "illicit/violent"
	CategorySelfHarm             = "self-harm"
	CategorySelfHarmIntent       = "self-harm/intent"
	CategorySelfHarmInstructions = "self-harm/instructions"
	CategoryViolence             = "violence"
	CategoryViolenceGraphic      = "violence/graphic"
)

// DecideStatus applies the project's moderation policy to a scan result
// and returns the action to take plus a stable reason code.
//
// Returned (StatusClean, ReasonNone) means "do nothing" — the caller
// should not write to the database nor notify admins. Any other status
// means the caller persists the new status+score+labels and emits the
// appropriate audit + admin-notifier events.
//
// Decision order (first match wins):
//  1. Zero-tolerance categories above their threshold -> StatusDeleted.
//  2. Global MaxScore >= ThresholdDeleteExtreme -> StatusDeleted.
//  3. Global MaxScore >= ThresholdHide -> StatusHidden.
//  4. Global MaxScore >= ThresholdFlag -> StatusFlagged.
//  5. Otherwise -> StatusClean.
func DecideStatus(result *service.TextModerationResult) (Status, string) {
	if result == nil {
		return StatusClean, ReasonNone
	}

	if status, reason := decideZeroTolerance(result.Labels); status != StatusClean {
		return status, reason
	}

	switch {
	case result.MaxScore >= ThresholdDeleteExtreme:
		return StatusDeleted, ReasonAutoDeleteExtremeScore
	case result.MaxScore >= ThresholdHide:
		return StatusHidden, ReasonAutoHideHighScore
	case result.MaxScore >= ThresholdFlag:
		return StatusFlagged, ReasonAutoFlagScore
	default:
		return StatusClean, ReasonNone
	}
}

// decideZeroTolerance checks only the 5 categories that force a
// soft-delete regardless of the global score. Kept in a separate helper
// so DecideStatus stays under the project's 50-line function limit and
// so table-driven tests can target the zero-tolerance matrix
// independently from the score-based branches.
func decideZeroTolerance(labels []service.TextModerationLabel) (Status, string) {
	for _, label := range labels {
		switch label.Name {
		case CategorySexualMinors:
			if label.Score >= ThresholdDeleteSexualMinors {
				return StatusDeleted, ReasonAutoDeleteMinors
			}
		case CategorySelfHarmInstructions:
			if label.Score >= ThresholdDeleteSelfHarmInstr {
				return StatusDeleted, ReasonAutoDeleteSelfHarmInstr
			}
		case CategoryHarassmentThreaten:
			if label.Score >= ThresholdDeleteHarassmentThreat {
				return StatusDeleted, ReasonAutoDeleteThreat
			}
		case CategoryHateThreatening:
			if label.Score >= ThresholdDeleteHateThreat {
				return StatusDeleted, ReasonAutoDeleteHateThreat
			}
		case CategoryViolenceGraphic:
			if label.Score >= ThresholdDeleteViolenceGraphic {
				return StatusDeleted, ReasonAutoDeleteViolenceGfx
			}
		}
	}
	return StatusClean, ReasonNone
}
