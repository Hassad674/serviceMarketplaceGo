// Package moderation orchestrates the automated text moderation
// pipeline. It is the single funnel through which every consumer
// (messaging, review, profile, job, proposal, auth, application)
// runs user-generated text — the goal being that the moderation
// engine, the threshold matrix, audit logging and admin notifier
// fan-out are all decided in ONE place rather than copy-pasted in
// every domain service.
//
// Two operating modes:
//
//   - Async (BlockingMode=false): the caller has already persisted
//     the content; this service decides + persists the verdict in
//     moderation_results without affecting the caller's flow. Used
//     for messages, reviews, proposals, job applications.
//
//   - Sync blocking (BlockingMode=true): called BEFORE persistence.
//     If the content scores at or above BlockingThreshold, the call
//     returns moderation.ErrContentBlocked so the caller short-
//     circuits with HTTP 422. Used for public-facing surfaces
//     (display_name, profile bio, job title/description).
//
// In both modes, a row in moderation_results is upserted — including
// for blocked attempts — so the admin queue lists every flagged or
// refused contribution.
package moderation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/audit"
	"marketplace-backend/internal/domain/moderation"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/port/service"
)

// Deps groups every collaborator the service needs. A struct rather
// than positional args because the project's 4-arg cap would force
// awkward grouping otherwise. Every field except TextModeration may
// be nil — the service degrades gracefully when downstream sinks are
// missing (CI without Redis, dev without admin notifier, etc.).
type Deps struct {
	TextModeration service.TextModerationService
	Results        repository.ModerationResultsRepository
	Audit          repository.AuditRepository
	AdminNotifier  service.AdminNotifierService
}

// Service is the moderation orchestrator. Concurrent-safe — it carries
// no mutable state.
type Service struct {
	textModeration service.TextModerationService
	results        repository.ModerationResultsRepository
	audit          repository.AuditRepository
	adminNotifier  service.AdminNotifierService
}

// NewService wires the orchestrator. TextModeration + Results are the
// only fields strictly required for production; the others are
// optional sinks.
func NewService(deps Deps) *Service {
	return &Service{
		textModeration: deps.TextModeration,
		results:        deps.Results,
		audit:          deps.Audit,
		adminNotifier:  deps.AdminNotifier,
	}
}

// ModerateInput is the public request shape. ContentID must be set BY
// the caller — even in BlockingMode, where the caller pre-generates
// the UUID it would have used for the source row, so the moderation
// row references something stable that an admin can re-grep later.
type ModerateInput struct {
	ContentType  moderation.ContentType
	ContentID    uuid.UUID
	AuthorUserID *uuid.UUID
	Text         string

	// BlockingMode=true makes this a synchronous gate before
	// persistence. The caller must check the returned error against
	// moderation.ErrContentBlocked and short-circuit accordingly.
	BlockingMode bool

	// BlockingThreshold is consulted only when BlockingMode is true.
	// Values: 0.50 for short public fields (display_name, titles),
	// 0.85 for descriptive long-form fields (bio, descriptions). 0.0
	// is treated as "no threshold" (effectively never blocks).
	BlockingThreshold float64
}

// ModerateResult summarises the verdict for callers that want to
// react beyond the err return — e.g. log the outcome or show it to
// the user. The Status field is the value persisted in
// moderation_results.
type ModerateResult struct {
	Status moderation.Status
	Score  float64
	Reason string
}

// Moderate runs the full pipeline. The contract:
//   - Status=Clean -> nothing persisted, nothing audited (cheap).
//   - Status in {Flagged, Hidden, Deleted} -> upsert + audit + notify
//     admin; nil error.
//   - BlockingMode and Score >= BlockingThreshold -> upsert + audit
//     with Status=Blocked, return moderation.ErrContentBlocked so
//     the caller refuses the create.
//   - Engine error (network, OpenAI 5xx) -> wrapped error returned.
//     The caller decides whether to fail closed (sync blocking) or
//     swallow (async fire-and-forget).
func (s *Service) Moderate(ctx context.Context, in ModerateInput) (ModerateResult, error) {
	if in.ContentID == uuid.Nil {
		return ModerateResult{}, fmt.Errorf("moderate: content_id required")
	}
	if in.ContentType == "" {
		return ModerateResult{}, fmt.Errorf("moderate: content_type required")
	}

	if in.Text == "" {
		return ModerateResult{Status: moderation.StatusClean, Reason: moderation.ReasonNone}, nil
	}

	analysis, err := s.textModeration.AnalyzeText(ctx, in.Text)
	if err != nil {
		return ModerateResult{}, fmt.Errorf("moderate: analyze text: %w", err)
	}

	status, reason := moderation.DecideStatus(analysis)

	// Synchronous gate: turn an otherwise tolerable status into Blocked
	// when the caller is creating a public-facing surface and the
	// score crosses the configured bar. The threshold is checked
	// against MaxScore (any single category passing the bar trips the
	// block) so a borderline-but-multi-category text still gets
	// rejected.
	if in.BlockingMode && in.BlockingThreshold > 0 && analysis.MaxScore >= in.BlockingThreshold {
		status = moderation.StatusBlocked
		reason = moderation.ReasonBlockedCreate
	}

	if status == moderation.StatusClean {
		return ModerateResult{Status: status, Score: analysis.MaxScore, Reason: reason}, nil
	}

	if persistErr := s.persistResult(ctx, in, analysis, status, reason); persistErr != nil {
		// In blocking mode we still return ErrContentBlocked because
		// the user-facing answer is "we refused" regardless of whether
		// our admin queue persisted the trace. In async mode we log
		// and swallow — the worst case is the admin queue misses one
		// entry, which is preferable to bubbling an internal failure
		// back through messaging/review.
		slog.Error("moderation: persist result", "error", persistErr,
			"content_type", string(in.ContentType), "content_id", in.ContentID)
	}

	s.auditDecision(ctx, in, status, reason, analysis.MaxScore)
	s.notifyAdmin(ctx, in.ContentType, status)

	if status == moderation.StatusBlocked {
		return ModerateResult{Status: status, Score: analysis.MaxScore, Reason: reason}, moderation.ErrContentBlocked
	}
	return ModerateResult{Status: status, Score: analysis.MaxScore, Reason: reason}, nil
}

// persistResult upserts the moderation row. Splitting this out of
// Moderate keeps the hot function under the project's 50-line limit
// and isolates the JSON marshalling that would otherwise muddy the
// happy path.
func (s *Service) persistResult(
	ctx context.Context,
	in ModerateInput,
	analysis *service.TextModerationResult,
	status moderation.Status,
	reason string,
) error {
	if s.results == nil {
		return nil
	}
	labelsJSON, err := json.Marshal(analysis.Labels)
	if err != nil {
		return fmt.Errorf("marshal labels: %w", err)
	}
	row, err := moderation.NewResult(moderation.NewResultInput{
		ContentType:  in.ContentType,
		ContentID:    in.ContentID,
		AuthorUserID: in.AuthorUserID,
		Status:       status,
		Score:        analysis.MaxScore,
		Labels:       labelsJSON,
		Reason:       reason,
	})
	if err != nil {
		return fmt.Errorf("build result: %w", err)
	}
	return s.results.Upsert(ctx, row)
}

// auditDecision writes a single row to audit_logs. We do NOT audit
// the Clean status (it would dwarf legitimate audit traffic); only
// the four statuses where a human admin may eventually want to know
// what happened: flagged, hidden, deleted, blocked.
func (s *Service) auditDecision(
	ctx context.Context,
	in ModerateInput,
	status moderation.Status,
	reason string,
	score float64,
) {
	if s.audit == nil {
		return
	}
	action := actionForStatus(in.ContentType, status)
	if action == "" {
		return
	}
	contentID := in.ContentID
	entry, err := audit.NewEntry(audit.NewEntryInput{
		UserID:       in.AuthorUserID,
		Action:       action,
		ResourceType: audit.ResourceType(in.ContentType),
		ResourceID:   &contentID,
		Metadata: map[string]any{
			"reason": reason,
			"score":  score,
		},
	})
	if err != nil {
		slog.Error("moderation: build audit entry", "error", err,
			"content_type", string(in.ContentType), "content_id", contentID)
		return
	}
	if err := s.audit.Log(ctx, entry); err != nil {
		// Audit failures must never break the user flow.
		slog.Error("moderation: write audit entry", "error", err,
			"content_type", string(in.ContentType), "content_id", contentID)
	}
}

// notifyAdmin bumps the relevant per-admin counter so the sidebar
// badge stays in sync. We map every non-clean status to the closest
// existing AdminNotifCategory rather than introduce a new one for
// every (content_type, status) pair — admins triage by content type
// inside /admin/moderation, and the sidebar badge just signals
// "something needs attention".
func (s *Service) notifyAdmin(ctx context.Context, contentType moderation.ContentType, status moderation.Status) {
	if s.adminNotifier == nil {
		return
	}
	if status == moderation.StatusBlocked {
		// Blocked attempts are visible in the admin queue but they do
		// not represent any work the admin must do — no badge bump.
		return
	}
	category := categoryForContentType(contentType, status)
	if category == "" {
		return
	}
	if err := s.adminNotifier.IncrementAll(ctx, category); err != nil {
		slog.Error("moderation: increment admin notifier", "error", err,
			"content_type", string(contentType), "category", category)
	}
}

// actionForStatus maps the (content_type, status) pair to the audit
// action string. Returns "" for statuses we do not audit.
func actionForStatus(contentType moderation.ContentType, status moderation.Status) audit.Action {
	switch status {
	case moderation.StatusFlagged:
		return audit.Action(fmt.Sprintf("moderation.auto_flag_%s", contentType))
	case moderation.StatusHidden:
		return audit.Action(fmt.Sprintf("moderation.auto_hide_%s", contentType))
	case moderation.StatusDeleted:
		return audit.Action(fmt.Sprintf("moderation.auto_delete_%s", contentType))
	case moderation.StatusBlocked:
		return audit.Action(fmt.Sprintf("moderation.block_create_%s", contentType))
	default:
		return ""
	}
}

// categoryForContentType picks an AdminNotifier category. Messages
// and reviews keep their pre-Phase-2 categories; the new content
// types fold into AdminNotifMessagesFlagged / Hidden by default
// (the admin sidebar already groups them under "moderation"). This
// is intentionally coarse — fine-grained badges did not survive the
// Phase 2 review.
func categoryForContentType(contentType moderation.ContentType, status moderation.Status) string {
	switch contentType {
	case moderation.ContentTypeReview:
		return service.AdminNotifReviewsFlagged
	default:
		if status == moderation.StatusHidden || status == moderation.StatusDeleted {
			return service.AdminNotifMessagesHidden
		}
		return service.AdminNotifMessagesFlagged
	}
}

// Compile-time guard against accidental import cycles via the
// (extremely unlikely) ErrContentBlocked re-export.
var _ = errors.Is
