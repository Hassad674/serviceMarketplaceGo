package referral

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/audit"
	"marketplace-backend/internal/domain/notification"
	"marketplace-backend/internal/domain/referral"
)

// Cancel allows the referrer to abort an intro that has not yet activated.
// Forbidden once status is active — use Terminate for that case.
func (s *Service) Cancel(ctx context.Context, referralID, actorID uuid.UUID) (*referral.Referral, error) {
	r, err := s.loadAndAuthorise(ctx, referralID, actorID, referral.ActorReferrer)
	if err != nil {
		return nil, err
	}
	prev := r.Status
	if err := r.Cancel(actorID); err != nil {
		return nil, err
	}
	if err := s.referrals.Update(ctx, r); err != nil {
		return nil, fmt.Errorf("update referral on cancel: %w", err)
	}
	s.notifyStatusTransition(ctx, r, prev)
	s.postTransitionMessages(ctx, r, prev)
	return r, nil
}

// Terminate ends an active referral. Existing attributions and pending
// commissions stay valid — they will pay out as their milestones are released.
// The exclusivity window simply stops generating NEW attributions.
func (s *Service) Terminate(ctx context.Context, referralID, actorID uuid.UUID) (*referral.Referral, error) {
	r, err := s.loadAndAuthorise(ctx, referralID, actorID, referral.ActorReferrer)
	if err != nil {
		return nil, err
	}
	prev := r.Status
	if err := r.Terminate(actorID); err != nil {
		return nil, err
	}
	if err := s.referrals.Update(ctx, r); err != nil {
		return nil, fmt.Errorf("update referral on terminate: %w", err)
	}
	s.notifyStatusTransition(ctx, r, prev)
	s.postTransitionMessages(ctx, r, prev)
	return r, nil
}

// EndIntroAttribution terminates a single referral attribution — the
// "Terminer l'intro" wallet action. After this:
//   - The attribution row is marked ended_at = NOW.
//   - NEW milestones approved on/after that timestamp do NOT generate
//     commissions (gate enforced in commission_distributor).
//   - Milestones already approved before the end remain payable —
//     fair to the apporteur for work delivered during the active
//     window.
//
// Idempotent: a second call with the same attribution id is a no-op
// success — the caller gets the already-ended row back and no
// duplicate audit / notification is emitted.
//
// RBAC: only the apporteur (parent referral.ReferrerID) of the
// attribution can end it. The repository UPDATE is the primary defense
// (SQL JOIN); the service does a cheap pre-check first for a cleaner
// 403 vs 404 distinction at the handler level.
func (s *Service) EndIntroAttribution(
	ctx context.Context,
	attributionID, actorUserID uuid.UUID,
) (*referral.Attribution, error) {
	att, err := s.referrals.FindAttributionByID(ctx, attributionID)
	if err != nil {
		return nil, err
	}
	parent, err := s.referrals.GetByID(ctx, att.ReferralID)
	if err != nil {
		return nil, fmt.Errorf("load parent referral: %w", err)
	}
	if parent.ReferrerID != actorUserID {
		return nil, referral.ErrNotAuthorized
	}

	// Idempotent path — already ended is a successful no-op.
	if att.IsEnded() {
		return att, nil
	}

	endErr := s.referrals.EndAttribution(ctx, attributionID, actorUserID)
	switch {
	case endErr == nil:
		// proceed below
	case errors.Is(endErr, referral.ErrAttributionAlreadyEnded):
		// Race: another caller ended the row between our load and
		// our UPDATE. Reload and return idempotently — no audit, no
		// notifications (the winning caller already emitted them).
		reloaded, rerr := s.referrals.FindAttributionByID(ctx, attributionID)
		if rerr != nil {
			return nil, fmt.Errorf("reload attribution after race: %w", rerr)
		}
		return reloaded, nil
	default:
		return nil, fmt.Errorf("end attribution: %w", endErr)
	}

	// Reload to capture the DB-side ended_at timestamp.
	updated, err := s.referrals.FindAttributionByID(ctx, attributionID)
	if err != nil {
		return nil, fmt.Errorf("reload attribution after end: %w", err)
	}

	s.emitEndAttributionAudit(ctx, updated, parent.ID, actorUserID)
	s.notifyAttributionEnded(ctx, updated, parent)
	return updated, nil
}

// emitEndAttributionAudit writes the audit row recording the
// termination. Best-effort: a persistence error never blocks the
// state change.
func (s *Service) emitEndAttributionAudit(
	ctx context.Context,
	att *referral.Attribution,
	referralID, actorUserID uuid.UUID,
) {
	if s.audits == nil {
		return
	}
	resourceID := att.ID
	metadata := map[string]any{
		"referral_id": referralID.String(),
		"proposal_id": att.ProposalID.String(),
	}
	if att.EndedAt != nil {
		metadata["ended_at"] = att.EndedAt.UTC().Format("2006-01-02T15:04:05Z07:00")
	}
	entry, err := audit.NewEntry(audit.NewEntryInput{
		UserID:       &actorUserID,
		Action:       audit.ActionReferralIntroAttributionEnded,
		ResourceType: audit.ResourceTypeReferralAttribution,
		ResourceID:   &resourceID,
		Metadata:     metadata,
	})
	if err != nil {
		slog.Warn("end attribution audit: NewEntry failed",
			"attribution_id", att.ID, "error", err)
		return
	}
	if err := s.audits.Log(ctx, entry); err != nil {
		slog.Warn("end attribution audit: persist failed",
			"attribution_id", att.ID, "error", err)
	}
}

// notifyAttributionEnded fan-outs the termination event to both
// parties of the proposal. Reuses TypeReferralIntroTerminated — the
// notification copy is identical from the user's perspective ("the
// apporteur ended the intro") so adding a parallel constant would be
// duplication without value. The data payload distinguishes
// attribution-end (carries attribution_id) from referral-terminate
// (no attribution_id) for any consumer that wants to discriminate.
func (s *Service) notifyAttributionEnded(
	ctx context.Context,
	att *referral.Attribution,
	parent *referral.Referral,
) {
	data := map[string]any{
		"referral_id":    parent.ID.String(),
		"attribution_id": att.ID.String(),
		"proposal_id":    att.ProposalID.String(),
	}
	if att.EndedAt != nil {
		data["ended_at"] = att.EndedAt.UTC().Format("2006-01-02T15:04:05Z07:00")
	}
	s.fanOut(ctx, att.ProviderID, notification.TypeReferralIntroTerminated,
		"Mise en relation terminée",
		"L'apporteur a mis fin à la mise en relation pour cette proposition.",
		data)
	s.fanOut(ctx, att.ClientID, notification.TypeReferralIntroTerminated,
		"Mise en relation terminée",
		"L'apporteur a mis fin à la mise en relation pour cette proposition.",
		data)
}

// GetByID is the read-side helper used by the handler. It performs the same
// authorisation check as the respond methods — only the three parties of a
// referral can read it.
func (s *Service) GetByID(ctx context.Context, referralID, viewerID uuid.UUID) (*referral.Referral, error) {
	r, err := s.referrals.GetByID(ctx, referralID)
	if err != nil {
		return nil, err
	}
	if r.ReferrerID != viewerID && r.ProviderID != viewerID && r.ClientID != viewerID {
		return nil, referral.ErrNotAuthorized
	}
	return r, nil
}
