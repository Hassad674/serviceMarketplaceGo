package dispute

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	disputedomain "marketplace-backend/internal/domain/dispute"
	"marketplace-backend/internal/domain/message"
	proposaldomain "marketplace-backend/internal/domain/proposal"
	portservice "marketplace-backend/internal/port/service"
)

// RespondToCancellation processes the other party's decision on a pending
// cancellation request. Either party may have requested the cancellation;
// only the OTHER (not the requester) can respond.
//   - Accept: the dispute is cancelled and the proposal is restored.
//   - Refuse: the cancellation request is cleared and the dispute continues.
func (s *Service) RespondToCancellation(ctx context.Context, in RespondToCancellationInput) error {
	d, err := s.loadDisputeForActor(ctx, in.DisputeID)
	if err != nil {
		return err
	}

	// Capture the requester BEFORE the domain method clears it on refuse,
	// so we can notify the right person afterwards.
	var requesterID uuid.UUID
	if d.CancellationRequestedBy != nil {
		requesterID = *d.CancellationRequestedBy
	}

	if err := d.RespondToCancellationRequest(in.UserID, in.Accept); err != nil {
		return err
	}
	if err := s.disputes.Update(ctx, d); err != nil {
		return fmt.Errorf("update dispute: %w", err)
	}

	if in.Accept {
		// BUG-03 (mirror of CancelDispute branch): propagate Update
		// errors rather than swallowing them. The dispute was already
		// persisted as cancelled by the s.disputes.Update call above,
		// so a failure here leaves the proposal in `disputed` while the
		// dispute is `cancelled` — surface the error so the caller can
		// retry, log at ERROR for ops visibility.
		p, err := s.loadProposalForActor(ctx, d.ProposalID)
		if err != nil {
			return fmt.Errorf("get proposal for restore: %w", err)
		}
		if err := p.RestoreFromDispute(proposaldomain.StatusActive); err != nil {
			slog.Error("dispute respond: domain restore rejected",
				"dispute_id", d.ID, "proposal_id", p.ID, "error", err)
			return fmt.Errorf("restore proposal from dispute: %w", err)
		}
		if err := s.proposals.Update(ctx, p); err != nil {
			slog.Error("dispute respond: proposal update failed — pair may be inconsistent until retry",
				"dispute_id", d.ID, "proposal_id", p.ID, "error", err)
			return fmt.Errorf("update proposal after dispute cancellation accepted: %w", err)
		}

		s.sendSystemMessage(ctx, d.ConversationID, in.UserID,
			message.MessageTypeDisputeCancelled, buildCancelledMetadata(d))
		s.notifyBothParties(ctx, d, "dispute_cancelled",
			"Litige annule", "Les deux parties se sont mises d'accord pour annuler le litige.")
		return nil
	}

	s.sendSystemMessage(ctx, d.ConversationID, in.UserID,
		message.MessageTypeDisputeCancellationRefused,
		buildCancellationRefusedMetadata(d, in.UserID))
	if requesterID != uuid.Nil {
		s.sendNotification(ctx, requesterID, "dispute_cancellation_refused",
			"Annulation refusee",
			"L'autre partie a refuse votre demande d'annulation. Le litige continue.", d.ID)
	}

	return nil
}

// ---------------------------------------------------------------------------
// ForceEscalate (dev/testing)
// ---------------------------------------------------------------------------

// ForceEscalate immediately escalates a dispute to admin mediation,
// bypassing the 7-day inactivity window. Intended for development and
// manual testing — the handler that exposes it MUST gate the route on
// non-production environments.
//
// Internally this is a thin wrapper around escalate(), the same code path
// the scheduler uses, so the test result is fully identical to production.
func (s *Service) ForceEscalate(ctx context.Context, disputeID uuid.UUID) error {
	d, err := s.loadDisputeForActor(ctx, disputeID)
	if err != nil {
		return err
	}
	return s.escalate(ctx, d)
}

// escalate is the canonical escalation routine: transition to escalated,
// generate the AI summary (best-effort), persist, broadcast a system
// message to the conversation and notify both parties. It is shared
// between the scheduler (timed escalation after 7 days of inactivity)
// and ForceEscalate (manual dev trigger) so both produce identical state.
func (s *Service) escalate(ctx context.Context, d *disputedomain.Dispute) error {
	if err := d.Escalate(); err != nil {
		return err
	}

	// AI summary is best-effort: a failure here is logged but does not
	// abort the escalation. The admin can still mediate without it.
	if s.ai != nil {
		summary, usage, err := s.generateAISummary(ctx, d)
		if err != nil {
			slog.Warn("dispute: AI analysis failed", "dispute_id", d.ID, "error", err)
		} else {
			d.SetAISummary(summary)
			// Record the actual token usage from the API response so the
			// dispute carries an accurate cost trail for the admin UI.
			d.RecordAISummaryUsage(usage.InputTokens, usage.OutputTokens)
		}
	}

	if err := s.disputes.Update(ctx, d); err != nil {
		return fmt.Errorf("update dispute: %w", err)
	}

	// Use the initiator's ID as sender for the same reason as AdminResolve:
	// the messages.sender_id FK rejects uuid.Nil. Escalation can be triggered
	// by the scheduler (no caller) so the initiator is the best deterministic
	// party. The bubble renders as system regardless.
	s.sendSystemMessage(ctx, d.ConversationID, d.InitiatorID,
		message.MessageTypeDisputeEscalated, buildEscalatedMetadata(d))
	s.notifyBothParties(ctx, d, "dispute_escalated",
		"Litige transmis a la mediation",
		"Votre litige a ete transmis a l'equipe de mediation pour decision.")

	return nil
}

// ---------------------------------------------------------------------------
// AI Chat (admin Q&A on a dispute)
// ---------------------------------------------------------------------------

// AskAIInput groups the parameters for an admin chat question. The chat
// history is NOT in this struct on purpose: the backend loads it from
// dispute_ai_chat_messages so the admin frontend can't tamper with what
// the AI sees, and so multiple admins on the same dispute share state.
type AskAIInput struct {
	DisputeID uuid.UUID
	Question  string
}

// AskAIOutput is what the chat call returns to the handler.
type AskAIOutput struct {
	Answer       string
	InputTokens  int
	OutputTokens int
}

// AskAI processes an admin chat question on an escalated dispute.
//
//   - Loads the dispute, the persisted chat history, and verifies the AI
//     chat budget has not been exceeded (with the +10% overshoot tolerance).
//   - Builds the dispute context and forwards the persisted history (NOT
//     a client-supplied one) so admins cannot tamper with what the AI sees.
//   - Calls the AIAnalyzer with the remaining chat budget.
//   - Persists BOTH the user question and the assistant answer as
//     append-only rows in dispute_ai_chat_messages, then updates the
//     dispute's cumulative chat token counters.
//
// Returns ErrAIBudgetChatExceeded when the cumulative chat usage has
// reached 110% of the dispute's chat budget. The admin can resolve this
// by clicking "Augmenter le budget" which calls IncreaseAIBudget.
func (s *Service) AskAI(ctx context.Context, in AskAIInput) (*AskAIOutput, error) {
	if s.ai == nil {
		return nil, fmt.Errorf("AI analyzer not configured")
	}
	d, err := s.loadDisputeForActor(ctx, in.DisputeID)
	if err != nil {
		return nil, err
	}

	// Hard ceiling at 110% of the chat budget. We allow a small overshoot
	// because the char/4 estimator is imprecise — refusing only past +10%
	// avoids false rejections on borderline cases.
	chatBudget := d.AIBudgetChat()
	overshootCeiling := chatBudget + chatBudget/10
	if d.AIChatUsed() >= overshootCeiling {
		return nil, disputedomain.ErrAIBudgetChatExceeded
	}

	// Load the persisted chat history from DB. This is the source of
	// truth — the admin frontend never sends history in the request body.
	persistedHistory, err := s.disputes.ListChatMessages(ctx, d.ID)
	if err != nil {
		return nil, fmt.Errorf("load chat history: %w", err)
	}

	input, err := s.buildAIInput(ctx, d)
	if err != nil {
		return nil, err
	}

	// Pass the REMAINING chat budget to the adapter so the input
	// truncation logic targets the right ceiling for this specific call.
	remaining := chatBudget - d.AIChatUsed()
	if remaining < 1500 {
		// Floor: even at the very edge, give the adapter enough room to
		// build a minimal prompt + reserve the output cap. The overshoot
		// tolerance above already gates against truly empty budgets.
		remaining = 1500
	}

	// Convert persisted history to the port-layer ChatTurn shape.
	portHistory := make([]portservice.ChatTurn, 0, len(persistedHistory))
	for _, m := range persistedHistory {
		portHistory = append(portHistory, portservice.ChatTurn{
			Role:    string(m.Role),
			Content: m.Content,
		})
	}

	answer, usage, err := s.ai.ChatAboutDispute(ctx, input, portHistory, in.Question, remaining)
	if err != nil {
		return nil, fmt.Errorf("ai chat: %w", err)
	}

	// Persist both turns. The user question records 0 tokens (it isn't
	// charged on its own — the API call that includes it is). The
	// assistant answer carries the actual usage from the API response.
	userMsg := disputedomain.NewChatMessage(d.ID, disputedomain.ChatMessageRoleUser, in.Question, 0, 0)
	if err := s.disputes.CreateChatMessage(ctx, userMsg); err != nil {
		return nil, fmt.Errorf("persist user chat message: %w", err)
	}
	assistantMsg := disputedomain.NewChatMessage(d.ID, disputedomain.ChatMessageRoleAssistant, answer, usage.InputTokens, usage.OutputTokens)
	if err := s.disputes.CreateChatMessage(ctx, assistantMsg); err != nil {
		return nil, fmt.Errorf("persist assistant chat message: %w", err)
	}

	d.RecordAIChatUsage(usage.InputTokens, usage.OutputTokens)
	if err := s.disputes.Update(ctx, d); err != nil {
		return nil, fmt.Errorf("update dispute after ai chat: %w", err)
	}

	return &AskAIOutput{
		Answer:       answer,
		InputTokens:  usage.InputTokens,
		OutputTokens: usage.OutputTokens,
	}, nil
}

// IncreaseAIBudget grants extra AI budget on the dispute via the admin
// "Augmenter le budget" button. Each call adds the configured amount;
// the admin can click multiple times if they need more.
func (s *Service) IncreaseAIBudget(ctx context.Context, disputeID uuid.UUID, amount int) error {
	if amount <= 0 {
		amount = AIBudgetBonusIncrement
	}
	d, err := s.loadDisputeForActor(ctx, disputeID)
	if err != nil {
		return err
	}
	d.AddAIBudgetBonus(amount)
	if err := s.disputes.Update(ctx, d); err != nil {
		return fmt.Errorf("update dispute after budget increase: %w", err)
	}
	slog.Info("dispute AI budget increased",
		"dispute_id", d.ID, "amount_tokens", amount,
		"new_summary_cap", d.AIBudgetSummary(),
		"new_chat_cap", d.AIBudgetChat(),
	)
	return nil
}

// AIBudgetBonusIncrement is the amount of tokens added to a dispute's
// AI budget each time the admin clicks "Augmenter le budget".
const AIBudgetBonusIncrement = 25000

// ---------------------------------------------------------------------------
// AdminResolve
// ---------------------------------------------------------------------------

func (s *Service) AdminResolve(ctx context.Context, in AdminResolveInput) error {
	d, err := s.loadDisputeForActor(ctx, in.DisputeID)
	if err != nil {
		return err
	}
	if d.Status != disputedomain.StatusEscalated {
		return disputedomain.ErrInvalidStatus
	}

	if err := d.Resolve(disputedomain.ResolveInput{
		ResolvedBy:     in.AdminID,
		AmountClient:   in.AmountClient,
		AmountProvider: in.AmountProvider,
		Note:           in.Note,
	}); err != nil {
		return err
	}
	if err := s.disputes.Update(ctx, d); err != nil {
		return fmt.Errorf("update dispute: %w", err)
	}

	s.restoreProposalAndDistribute(ctx, d)

	// Use the admin's user ID as sender: the messages table has a FK on
	// sender_id → users(id), so uuid.Nil silently fails the insert. The
	// chat bubble renders this as a system bubble based on the type, so
	// the visible sender doesn't matter — only the FK does.
	s.sendSystemMessage(ctx, d.ConversationID, in.AdminID,
		message.MessageTypeDisputeResolved, buildResolvedMetadata(d))
	s.notifyBothParties(ctx, d, "dispute_resolved",
		"Litige resolu", "L'equipe de mediation a rendu sa decision.")

	return nil
}

// ---------------------------------------------------------------------------
// Read methods
// ---------------------------------------------------------------------------

type DisputeDetail struct {
	Dispute          *disputedomain.Dispute
	Evidence         []*disputedomain.Evidence
	CounterProposals []*disputedomain.CounterProposal
	ChatMessages     []*disputedomain.ChatMessage
}

func (s *Service) GetDispute(ctx context.Context, userID, disputeID uuid.UUID) (*DisputeDetail, error) {
	d, err := s.loadDisputeForActor(ctx, disputeID)
	if err != nil {
		return nil, err
	}
	if !d.IsParticipant(userID) {
		return nil, disputedomain.ErrNotParticipant
	}
	return s.loadDetail(ctx, d)
}

// GetDisputeForAdmin fetches the dispute under the system-actor
// path: admins are not party to the dispute and the surface is
// gated by the admin role + dedicated admin route, not by org
// tenancy. Callers MUST wrap the request context with
// system.WithSystemActor (the admin handler does this).
func (s *Service) GetDisputeForAdmin(ctx context.Context, disputeID uuid.UUID) (*DisputeDetail, error) {
	d, err := s.loadDisputeForActor(ctx, disputeID)
	if err != nil {
		return nil, err
	}
	return s.loadDetail(ctx, d)
}

func (s *Service) loadDetail(ctx context.Context, d *disputedomain.Dispute) (*DisputeDetail, error) {
	evidence, err := s.disputes.ListEvidence(ctx, d.ID)
	if err != nil {
		return nil, fmt.Errorf("load evidence: %w", err)
	}
	cps, err := s.disputes.ListCounterProposals(ctx, d.ID)
	if err != nil {
		return nil, fmt.Errorf("load counter-proposals: %w", err)
	}
	// Chat history is best-effort: a load failure does not block the
	// detail page (the rest of the dispute is still useful). The admin
	// just sees an empty chat panel they can re-populate.
	chats, err := s.disputes.ListChatMessages(ctx, d.ID)
	if err != nil {
		slog.Warn("dispute: failed to load chat messages", "dispute_id", d.ID, "error", err)
		chats = nil
	}
	return &DisputeDetail{
		Dispute:          d,
		Evidence:         evidence,
		CounterProposals: cps,
		ChatMessages:     chats,
	}, nil
}

// ListOrgDisputes returns the disputes where the caller's organization
// is either the client or the provider side. All operators of the
// same org see the same list (Stripe Dashboard shared workspace).
func (s *Service) ListOrgDisputes(ctx context.Context, orgID uuid.UUID, cursor string, limit int) ([]*disputedomain.Dispute, string, error) {
	return s.disputes.ListByOrganization(ctx, orgID, cursor, limit)
}
