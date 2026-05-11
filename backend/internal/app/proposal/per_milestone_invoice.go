package proposal

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	"marketplace-backend/internal/system"
)

// emitPerMilestoneInvoice fires the platform_fee invoice for a freshly
// approved milestone. Wraps the call so:
//
//   - A nil invoicer (feature disabled) is a silent no-op — keeps the
//     proposal feature fully decoupled from invoicing.
//   - A non-nil error is LOGGED but NEVER bubbles back to the caller —
//     the approval is already committed, and the monthly safety-net
//     scheduler picks up missed milestones on its next tick. Rolling
//     back the approval over a billing hiccup would be a worse outcome
//     for the user than a brief delay before the invoice appears.
//
// The context is tagged with system actor because the invoicing
// pipeline reaches into payment_records (RLS-gated) without an explicit
// org context — payments are looked up by milestone_id, not by org, so
// the tenant policy would otherwise filter every row.
func (s *Service) emitPerMilestoneInvoice(ctx context.Context, milestoneID uuid.UUID) {
	if s == nil || s.perMilestoneInvoicer == nil {
		return
	}
	if milestoneID == uuid.Nil {
		slog.Warn("proposal: emit per-milestone invoice called with zero milestone id")
		return
	}
	if err := s.perMilestoneInvoicer.IssueFromMilestone(system.WithSystemActor(ctx), milestoneID); err != nil {
		slog.Error("proposal: per-milestone invoice emission failed; safety-net scheduler will retry",
			"milestone_id", milestoneID,
			"error", err,
		)
	}
}
