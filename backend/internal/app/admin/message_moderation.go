package admin

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/audit"
	"marketplace-backend/internal/domain/moderation"
)

// ApproveMessageModeration clears the auto-applied moderation status
// on a message — the admin has reviewed the flag and decided the
// message can stay visible. Phase 7: writes ONLY to moderation_results
// (the legacy messages.moderation_status column was dropped).
func (s *Service) ApproveMessageModeration(ctx context.Context, messageID, adminID uuid.UUID) error {
	return s.markMessageModeration(ctx, messageID, adminID,
		moderation.StatusClean, audit.Action("moderation.manual_approve_message"))
}

// HideMessage marks a message as hidden by admin decision (rather
// than the auto pipeline). Used when reviewing a flagged message and
// deciding it should be hidden after all.
func (s *Service) HideMessage(ctx context.Context, messageID, adminID uuid.UUID) error {
	return s.markMessageModeration(ctx, messageID, adminID,
		moderation.StatusHidden, audit.Action("moderation.manual_hide_message"))
}

// RestoreMessageModeration resets a message back to "clean" — used
// to undo an auto soft-delete or hide that the admin disagrees with.
func (s *Service) RestoreMessageModeration(ctx context.Context, messageID, adminID uuid.UUID) error {
	return s.markMessageModeration(ctx, messageID, adminID,
		moderation.StatusClean, audit.Action("moderation.manual_restore_message"))
}

// markMessageModeration is the shared write path for every manual
// override (Approve, Hide, Restore). Single source of truth =
// moderation_results since Phase 7. ErrResultNotFound is treated as
// "the auto pipeline never wrote a row" which is fine on Approve
// (nothing to clear) but worth flagging on Restore (admin tried to
// restore something that was never moderated). We log+continue —
// idempotency wins over strict pre-conditions.
func (s *Service) markMessageModeration(
	ctx context.Context,
	messageID, adminID uuid.UUID,
	newStatus moderation.Status,
	action audit.Action,
) error {
	if s.moderationResults != nil {
		err := s.moderationResults.MarkReviewed(ctx, moderation.ContentTypeMessage, messageID, adminID, newStatus)
		if err != nil && !errors.Is(err, moderation.ErrResultNotFound) {
			return fmt.Errorf("admin message moderation: results update: %w", err)
		}
	}

	s.auditAdminModerationAction(ctx, adminID, audit.ResourceType("message"), messageID, action)
	return nil
}

// ApproveReviewModeration mirrors ApproveMessageModeration for reviews.
// Reviews never use the "hidden" middleware path — they go directly
// from flagged to clean (visible) or are deleted entirely.
func (s *Service) ApproveReviewModeration(ctx context.Context, reviewID, adminID uuid.UUID) error {
	return s.markReviewModeration(ctx, reviewID, adminID,
		moderation.StatusClean, audit.Action("moderation.manual_approve_review"))
}

// RestoreReviewModeration resets a review back to clean.
func (s *Service) RestoreReviewModeration(ctx context.Context, reviewID, adminID uuid.UUID) error {
	return s.markReviewModeration(ctx, reviewID, adminID,
		moderation.StatusClean, audit.Action("moderation.manual_restore_review"))
}

func (s *Service) markReviewModeration(
	ctx context.Context,
	reviewID, adminID uuid.UUID,
	newStatus moderation.Status,
	action audit.Action,
) error {
	if s.moderationResults != nil {
		err := s.moderationResults.MarkReviewed(ctx, moderation.ContentTypeReview, reviewID, adminID, newStatus)
		if err != nil && !errors.Is(err, moderation.ErrResultNotFound) {
			return fmt.Errorf("admin review moderation: results update: %w", err)
		}
	}

	s.auditAdminModerationAction(ctx, adminID, audit.ResourceType("review"), reviewID, action)
	return nil
}

// RestoreModeration is the generic restore action used by the admin
// frontend for every Phase 2 content type that does not have a
// dedicated route (profile_*, job_*, proposal_*, etc.).
//
// Resolution logic:
//   - Looks up the existing moderation_results row by
//     (content_type, content_id).
//   - Marks it reviewed by the admin with status = clean.
//   - Audits the action so the override has a permanent trail.
//
// Note: this does NOT touch the source table — Phase 2's policy is
// soft-delete only, so the source row was never deleted in the first
// place. Restoring a "blocked" status is a no-op on the source layer
// (no row was ever inserted) — only the admin queue stops listing it.
func (s *Service) RestoreModeration(ctx context.Context, contentType string, contentID, adminID uuid.UUID) error {
	if s.moderationResults == nil {
		return fmt.Errorf("admin restore moderation: results repo not wired")
	}
	err := s.moderationResults.MarkReviewed(ctx,
		moderation.ContentType(contentType), contentID, adminID, moderation.StatusClean)
	if err != nil {
		return fmt.Errorf("admin restore moderation: %w", err)
	}
	s.auditAdminModerationAction(ctx, adminID,
		audit.ResourceType(contentType), contentID,
		audit.Action("moderation.manual_restore_"+contentType))
	return nil
}

// auditAdminModerationAction writes the admin override to audit_logs.
// Failures are logged and swallowed — audit completeness matters but
// the admin's action must succeed regardless.
func (s *Service) auditAdminModerationAction(
	ctx context.Context,
	adminID uuid.UUID,
	resourceType audit.ResourceType,
	resourceID uuid.UUID,
	action audit.Action,
) {
	if s.audit == nil {
		return
	}
	entry, err := audit.NewEntry(audit.NewEntryInput{
		UserID:       &adminID,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   &resourceID,
	})
	if err != nil {
		slog.Error("admin moderation: build audit entry", "error", err)
		return
	}
	if err := s.audit.Log(ctx, entry); err != nil {
		slog.Error("admin moderation: write audit entry", "error", err)
	}
}
