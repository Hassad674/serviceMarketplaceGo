package dispute

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/google/uuid"

	disputedomain "marketplace-backend/internal/domain/dispute"
	"marketplace-backend/internal/domain/message"
	proposaldomain "marketplace-backend/internal/domain/proposal"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/port/service"
)

// SchedulerDeps groups dependencies for the dispute scheduler.
type SchedulerDeps struct {
	Svc           *Service // canonical escalation routine lives here
	Disputes      repository.DisputeRepository
	Proposals     repository.ProposalRepository
	Messages      service.MessageSender
	Notifications service.NotificationSender
	Payments      service.PaymentProcessor
}

// Scheduler periodically checks for disputes that need auto-resolution
// or escalation. Runs as a background goroutine.
//
// Escalation is fully delegated to Service.escalate so that timed and
// manual (force-escalate) escalations produce strictly identical state.
// Auto-resolution (the "ghost" path when the respondent never replies)
// stays here because it has no manual counterpart.
type Scheduler struct {
	svc           *Service
	disputes      repository.DisputeRepository
	proposals     repository.ProposalRepository
	messages      service.MessageSender
	notifications service.NotificationSender
	payments      service.PaymentProcessor
}

func NewScheduler(deps SchedulerDeps) *Scheduler {
	return &Scheduler{
		svc:           deps.Svc,
		disputes:      deps.Disputes,
		proposals:     deps.Proposals,
		messages:      deps.Messages,
		notifications: deps.Notifications,
		payments:      deps.Payments,
	}
}

// Run blocks until ctx is cancelled. Ticks every interval + runs immediately.
func (s *Scheduler) Run(ctx context.Context, interval time.Duration) {
	s.tick(ctx)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.tick(ctx)
		}
	}
}

func (s *Scheduler) tick(ctx context.Context) {
	disputes, err := s.disputes.ListPendingForScheduler(ctx)
	if err != nil {
		slog.Error("dispute scheduler: list pending", "error", err)
		return
	}
	if len(disputes) == 0 {
		return
	}

	slog.Debug("dispute scheduler: processing", "count", len(disputes))

	for _, d := range disputes {
		if d.Status == disputedomain.StatusOpen && d.RespondentFirstReplyAt == nil {
			s.autoResolve(ctx, d)
		} else {
			s.escalate(ctx, d)
		}
	}
}

// autoResolve handles the ghost scenario: respondent never replied within 7 days.
// Funds go to the initiator.
func (s *Scheduler) autoResolve(ctx context.Context, d *disputedomain.Dispute) {
	if err := d.AutoResolveForInitiator(); err != nil {
		slog.Error("dispute scheduler: auto-resolve", "dispute_id", d.ID, "error", err)
		return
	}
	if err := s.disputes.Update(ctx, d); err != nil {
		slog.Error("dispute scheduler: update after auto-resolve", "dispute_id", d.ID, "error", err)
		return
	}

	s.restoreAndDistribute(ctx, d)

	s.broadcastSystemMessage(ctx, d.ConversationID,
		message.MessageTypeDisputeAutoResolved, buildAutoResolvedMetadata(d))
	s.notifyBoth(ctx, d, "dispute_auto_resolved",
		"Litige resolu automatiquement",
		"Le litige a ete resolu automatiquement faute de reponse dans les 7 jours.")

	slog.Info("dispute scheduler: auto-resolved (ghost)",
		"dispute_id", d.ID, "initiator_id", d.InitiatorID)
}

// escalate delegates to Service.escalate so the scheduler and the manual
// force-escalate endpoint share the same code path. The scheduler keeps
// only the "what to log when escalation is triggered by the timer" concern.
func (s *Scheduler) escalate(ctx context.Context, d *disputedomain.Dispute) {
	if err := s.svc.escalate(ctx, d); err != nil {
		slog.Error("dispute scheduler: escalate", "dispute_id", d.ID, "error", err)
		return
	}
	slog.Info("dispute scheduler: escalated to admin",
		"dispute_id", d.ID, "has_ai_summary", d.AISummary != nil)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func (s *Scheduler) restoreAndDistribute(ctx context.Context, d *disputedomain.Dispute) {
	p, err := s.proposals.GetByID(ctx, d.ProposalID)
	if err != nil {
		slog.Error("dispute scheduler: get proposal for restore", "error", err)
		return
	}
	if err := p.RestoreFromDispute(proposaldomain.StatusCompleted); err != nil {
		slog.Error("dispute scheduler: restore proposal", "error", err)
		return
	}
	_ = s.proposals.Update(ctx, p)

	if s.payments != nil {
		if d.ResolutionAmountProvider != nil && *d.ResolutionAmountProvider > 0 {
			if err := s.payments.TransferPartialToProvider(ctx, d.ProposalID, *d.ResolutionAmountProvider); err != nil {
				slog.Error("dispute scheduler: transfer to provider",
					"proposal_id", d.ProposalID, "error", err)
			}
		}
		if d.ResolutionAmountClient != nil && *d.ResolutionAmountClient > 0 {
			if err := s.payments.RefundToClient(ctx, d.ProposalID, *d.ResolutionAmountClient); err != nil {
				slog.Error("dispute scheduler: refund to client",
					"proposal_id", d.ProposalID, "error", err)
			}
		}
	}
}

func (s *Scheduler) broadcastSystemMessage(ctx context.Context, convID uuid.UUID, msgType message.MessageType, metadata json.RawMessage) {
	if err := s.messages.SendSystemMessage(ctx, service.SystemMessageInput{
		ConversationID: convID,
		SenderID:       uuid.Nil,
		Content:        "",
		Type:           string(msgType),
		Metadata:       metadata,
	}); err != nil {
		slog.Warn("dispute scheduler: send system message", "type", msgType, "error", err)
	}
}

func (s *Scheduler) notifyBoth(ctx context.Context, d *disputedomain.Dispute, notifType, title, body string) {
	data, _ := json.Marshal(map[string]string{"dispute_id": d.ID.String()})
	for _, uid := range []uuid.UUID{d.InitiatorID, d.RespondentID} {
		if err := s.notifications.Send(ctx, service.NotificationInput{
			UserID: uid,
			Type:   notifType,
			Title:  title,
			Body:   body,
			Data:   data,
		}); err != nil {
			slog.Warn("dispute scheduler: notify", "user_id", uid, "error", err)
		}
	}
}
