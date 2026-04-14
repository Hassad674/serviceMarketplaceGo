package dispute

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	disputedomain "marketplace-backend/internal/domain/dispute"
	"marketplace-backend/internal/domain/message"
	milestonedomain "marketplace-backend/internal/domain/milestone"
	proposaldomain "marketplace-backend/internal/domain/proposal"
	portservice "marketplace-backend/internal/port/service"
)

// markMilestoneDisputed transitions the milestone to disputed status
// inside an optimistic-locked update. Mirrors the proposal service's
// withMilestoneLock pattern so the dispute service stays decoupled
// from the proposal app service while still enforcing the same
// concurrency guarantees.
func (s *Service) markMilestoneDisputed(ctx context.Context, milestoneID, disputeID uuid.UUID) error {
	m, err := s.milestones.GetByIDForUpdate(ctx, milestoneID)
	if err != nil {
		return err
	}
	if err := m.OpenDispute(disputeID); err != nil {
		return err
	}
	return s.milestones.Update(ctx, m)
}

// restoreMilestoneFromDispute is the symmetric helper called from
// dispute resolution / cancellation paths. The target status is
// chosen by the dispute service based on the resolution type
// (funded for partial refund, released for full release, refunded
// for full refund).
func (s *Service) restoreMilestoneFromDispute(ctx context.Context, milestoneID uuid.UUID, target milestonedomain.MilestoneStatus) error {
	m, err := s.milestones.GetByIDForUpdate(ctx, milestoneID)
	if err != nil {
		return err
	}
	if err := m.RestoreFromDispute(target); err != nil {
		return err
	}
	return s.milestones.Update(ctx, m)
}

// ---------------------------------------------------------------------------
// Input types
// ---------------------------------------------------------------------------

type OpenDisputeInput struct {
	ProposalID      uuid.UUID
	InitiatorID     uuid.UUID
	Reason          string
	Description     string // Detailed description for admin mediation (private)
	MessageToParty  string // Short message to the other party (visible in conversation)
	RequestedAmount int64
	Attachments     []AttachmentInput
}

type AttachmentInput struct {
	Filename string
	URL      string
	Size     int64
	MimeType string
}

type CounterProposeInput struct {
	DisputeID      uuid.UUID
	ProposerID     uuid.UUID
	AmountClient   int64
	AmountProvider int64
	Message        string
	Attachments    []AttachmentInput
}

type RespondToCounterInput struct {
	DisputeID         uuid.UUID
	CounterProposalID uuid.UUID
	UserID            uuid.UUID
	Accept            bool
}

type CancelDisputeInput struct {
	DisputeID uuid.UUID
	UserID    uuid.UUID
}

type RespondToCancellationInput struct {
	DisputeID uuid.UUID
	UserID    uuid.UUID
	Accept    bool
}

// CancelDisputeResult indicates what happened when a cancellation was attempted.
// When Cancelled is true, the dispute was terminated directly (the respondent
// had not yet engaged). When Cancelled is false and Requested is true, a
// cancellation request was created and now waits for the respondent's consent.
type CancelDisputeResult struct {
	Cancelled bool
	Requested bool
}

type AdminResolveInput struct {
	DisputeID      uuid.UUID
	AdminID        uuid.UUID
	AmountClient   int64
	AmountProvider int64
	Note           string
}

// ---------------------------------------------------------------------------
// OpenDispute
// ---------------------------------------------------------------------------

func (s *Service) OpenDispute(ctx context.Context, in OpenDisputeInput) (*disputedomain.Dispute, error) {
	p, err := s.proposals.GetByID(ctx, in.ProposalID)
	if err != nil {
		return nil, fmt.Errorf("get proposal: %w", err)
	}
	if p.Status != proposaldomain.StatusActive && p.Status != proposaldomain.StatusCompletionRequested {
		return nil, disputedomain.ErrProposalNotDisputable
	}

	existing, err := s.disputes.GetByProposalID(ctx, in.ProposalID)
	if err != nil {
		return nil, fmt.Errorf("check existing dispute: %w", err)
	}
	if existing != nil {
		return nil, disputedomain.ErrAlreadyDisputed
	}

	// Phase 8: a dispute is scoped to a single milestone — the one
	// currently in flight. Resolve it server-side from the proposal
	// so the API surface stays simple (only proposal_id required).
	// Reject if the milestone is not in funded or submitted state:
	// pending_funding milestones have no escrow to dispute, and
	// terminal milestones (released, cancelled, refunded) are
	// immutable.
	current, err := s.milestones.GetCurrentActive(ctx, p.ID)
	if err != nil {
		return nil, fmt.Errorf("get current milestone: %w", err)
	}
	if current.Status != milestonedomain.StatusFunded && current.Status != milestonedomain.StatusSubmitted {
		return nil, disputedomain.ErrProposalNotDisputable
	}

	respondentID := p.ProviderID
	if in.InitiatorID == p.ProviderID {
		respondentID = p.ClientID
	}

	// Resolve both parties' current organizations so the dispute is
	// visible to every operator of either org, not just the user who
	// opened it (Stripe Dashboard shared workspace).
	clientUser, err := s.users.GetByID(ctx, p.ClientID)
	if err != nil {
		return nil, fmt.Errorf("lookup client user: %w", err)
	}
	providerUser, err := s.users.GetByID(ctx, p.ProviderID)
	if err != nil {
		return nil, fmt.Errorf("lookup provider user: %w", err)
	}
	if clientUser.OrganizationID == nil || providerUser.OrganizationID == nil {
		return nil, fmt.Errorf("open dispute: participants must belong to an organization")
	}

	// Phase 8: clamp the requested amount to the milestone amount —
	// a dispute can never claim more than the escrow it targets.
	if in.RequestedAmount > current.Amount {
		return nil, disputedomain.ErrInvalidAmount
	}

	d, err := disputedomain.NewDispute(disputedomain.NewDisputeInput{
		ProposalID:             p.ID,
		MilestoneID:            current.ID,
		ConversationID:         p.ConversationID,
		InitiatorID:            in.InitiatorID,
		RespondentID:           respondentID,
		ClientID:               p.ClientID,
		ProviderID:             p.ProviderID,
		ClientOrganizationID:   *clientUser.OrganizationID,
		ProviderOrganizationID: *providerUser.OrganizationID,
		Reason:                 disputedomain.Reason(in.Reason),
		Description:            in.Description,
		RequestedAmount:        in.RequestedAmount,
		// ProposalAmount carries the milestone amount post-phase-8
		// (the field name is preserved for backward compatibility
		// with existing SQL queries and resolution split logic).
		ProposalAmount: current.Amount,
	})
	if err != nil {
		return nil, err
	}

	if err := s.disputes.Create(ctx, d); err != nil {
		return nil, fmt.Errorf("persist dispute: %w", err)
	}

	for _, att := range in.Attachments {
		e := &disputedomain.Evidence{
			ID:         uuid.New(),
			DisputeID:  d.ID,
			UploaderID: in.InitiatorID,
			Filename:   att.Filename,
			URL:        att.URL,
			Size:       att.Size,
			MimeType:   att.MimeType,
		}
		if err := s.disputes.CreateEvidence(ctx, e); err != nil {
			slog.Warn("dispute: failed to save evidence", "error", err)
		}
	}

	// Phase 8: also mark the milestone disputed in the same flow, so
	// the macro state (proposal-level) and the milestone-level state
	// stay coherent. The milestone repo applies optimistic locking.
	if err := s.markMilestoneDisputed(ctx, current.ID, d.ID); err != nil {
		return nil, fmt.Errorf("mark milestone disputed: %w", err)
	}

	if err := p.MarkDisputed(d.ID); err != nil {
		return nil, fmt.Errorf("mark proposal disputed: %w", err)
	}
	if err := s.proposals.Update(ctx, p); err != nil {
		return nil, fmt.Errorf("update proposal: %w", err)
	}

	// Auto-create the first proposal from the initiator's requested amount.
	// This gives the other party something actionable immediately (accept/reject/counter).
	// All amounts are scoped to the disputed milestone (post-phase-8).
	var clientAmt, providerAmt int64
	if in.InitiatorID == p.ClientID {
		clientAmt = in.RequestedAmount
		providerAmt = current.Amount - in.RequestedAmount
	} else {
		providerAmt = in.RequestedAmount
		clientAmt = current.Amount - in.RequestedAmount
	}
	cp, cpErr := disputedomain.NewCounterProposal(disputedomain.NewCounterProposalInput{
		DisputeID:      d.ID,
		ProposerID:     in.InitiatorID,
		AmountClient:   clientAmt,
		AmountProvider: providerAmt,
		ProposalAmount: current.Amount,
		Message:        in.MessageToParty,
	})
	if cpErr == nil {
		_ = s.disputes.CreateCounterProposal(ctx, cp)
	}

	// Single system message: "Litige ouvert" with the initial proposal embedded
	s.sendSystemMessage(ctx, d.ConversationID, in.InitiatorID,
		message.MessageTypeDisputeOpened, buildOpenedWithProposalMetadata(d, cp, in.MessageToParty))

	s.sendNotification(ctx, respondentID, "dispute_opened",
		"Litige ouvert", "Un litige a ete ouvert sur votre mission.", d.ID)

	return d, nil
}

// ---------------------------------------------------------------------------
// CounterPropose
// ---------------------------------------------------------------------------

func (s *Service) CounterPropose(ctx context.Context, in CounterProposeInput) (*disputedomain.CounterProposal, error) {
	d, err := s.disputes.GetByID(ctx, in.DisputeID)
	if err != nil {
		return nil, err
	}
	// Counter-proposals stay open through admin mediation: parties can still
	// reach an amicable agreement until the admin issues a final decision.
	if d.Status != disputedomain.StatusOpen &&
		d.Status != disputedomain.StatusNegotiation &&
		d.Status != disputedomain.StatusEscalated {
		return nil, disputedomain.ErrInvalidStatus
	}
	if !d.IsParticipant(in.ProposerID) {
		return nil, disputedomain.ErrNotParticipant
	}

	if err := s.disputes.SupersedeAllPending(ctx, d.ID); err != nil {
		return nil, fmt.Errorf("supersede pending: %w", err)
	}

	cp, err := disputedomain.NewCounterProposal(disputedomain.NewCounterProposalInput{
		DisputeID:      d.ID,
		ProposerID:     in.ProposerID,
		AmountClient:   in.AmountClient,
		AmountProvider: in.AmountProvider,
		ProposalAmount: d.ProposalAmount,
		Message:        in.Message,
	})
	if err != nil {
		return nil, err
	}

	if err := s.disputes.CreateCounterProposal(ctx, cp); err != nil {
		return nil, fmt.Errorf("persist counter-proposal: %w", err)
	}

	// Persist attachments linked to this counter-proposal
	for _, att := range in.Attachments {
		e := &disputedomain.Evidence{
			ID:                uuid.New(),
			DisputeID:         d.ID,
			CounterProposalID: &cp.ID,
			UploaderID:        in.ProposerID,
			Filename:          att.Filename,
			URL:               att.URL,
			Size:              att.Size,
			MimeType:          att.MimeType,
		}
		if err := s.disputes.CreateEvidence(ctx, e); err != nil {
			slog.Warn("dispute: failed to save counter-proposal evidence", "error", err)
		}
	}

	if d.Status == disputedomain.StatusOpen {
		_ = d.MarkNegotiation()
	}
	if in.ProposerID == d.RespondentID {
		d.RecordRespondentReply()
	}
	// A new counter-proposal signals active negotiation — any pending
	// cancellation request is implicitly withdrawn.
	d.ClearCancellationRequest()
	d.RecordActivity()
	if err := s.disputes.Update(ctx, d); err != nil {
		return nil, fmt.Errorf("update dispute: %w", err)
	}

	otherID := d.InitiatorID
	if in.ProposerID == d.InitiatorID {
		otherID = d.RespondentID
	}

	s.sendSystemMessage(ctx, d.ConversationID, in.ProposerID,
		message.MessageTypeDisputeCounterProposal, buildCounterMetadata(d, cp))
	s.sendNotification(ctx, otherID, "dispute_counter_proposal",
		"Nouvelle proposition", "Une contre-proposition a ete faite sur votre litige.", d.ID)

	return cp, nil
}

// ---------------------------------------------------------------------------
// RespondToCounter
// ---------------------------------------------------------------------------

func (s *Service) RespondToCounter(ctx context.Context, in RespondToCounterInput) error {
	d, err := s.disputes.GetByID(ctx, in.DisputeID)
	if err != nil {
		return err
	}

	cp, err := s.disputes.GetCounterProposalByID(ctx, in.CounterProposalID)
	if err != nil {
		return err
	}

	if in.Accept {
		if err := cp.Accept(in.UserID); err != nil {
			return err
		}
		if err := s.disputes.UpdateCounterProposal(ctx, cp); err != nil {
			return fmt.Errorf("update counter-proposal: %w", err)
		}

		if err := d.Resolve(disputedomain.ResolveInput{
			AmountClient:   cp.AmountClient,
			AmountProvider: cp.AmountProvider,
			Note:           "Accord amiable entre les parties.",
		}); err != nil {
			return fmt.Errorf("resolve dispute: %w", err)
		}
		if err := s.disputes.Update(ctx, d); err != nil {
			return fmt.Errorf("update dispute: %w", err)
		}

		s.restoreProposalAndDistribute(ctx, d)

		s.sendSystemMessage(ctx, d.ConversationID, in.UserID,
			message.MessageTypeDisputeResolved, buildResolvedMetadata(d))
		s.notifyBothParties(ctx, d, "dispute_resolved",
			"Litige resolu", "Le litige a ete resolu par accord amiable.")
	} else {
		if err := cp.Reject(in.UserID); err != nil {
			return err
		}
		if err := s.disputes.UpdateCounterProposal(ctx, cp); err != nil {
			return fmt.Errorf("update counter-proposal: %w", err)
		}

		// Rejecting a proposal IS engagement with the dispute. Record the
		// respondent's first reply so that subsequent cancellation attempts
		// by the initiator go through the request flow (consent required).
		if in.UserID == d.RespondentID {
			d.RecordRespondentReply()
		}
		d.RecordActivity()
		if err := s.disputes.Update(ctx, d); err != nil {
			return fmt.Errorf("update dispute: %w", err)
		}

		s.sendSystemMessage(ctx, d.ConversationID, in.UserID,
			message.MessageTypeDisputeCounterRejected, buildCounterMetadata(d, cp))
		// Notify the proposer that their amicable resolution proposal was
		// refused so they can react (make a new proposal, escalate, etc).
		s.sendNotification(ctx, cp.ProposerID, "dispute_counter_rejected",
			"Proposition refusee",
			"L'autre partie a refuse votre proposition de resolution du litige.", d.ID)
	}

	return nil
}

// ---------------------------------------------------------------------------
// CancelDispute
// ---------------------------------------------------------------------------

// CancelDispute attempts to cancel a dispute on behalf of any participant.
//
// The domain decides between two paths based on the caller's role and the
// respondent's engagement state:
//   - Initiator + respondent has NOT yet replied → direct cancellation
//     (result.Cancelled=true), proposal restored to active.
//   - Initiator + respondent HAS replied, OR respondent (any time) →
//     cancellation REQUEST is created (result.Requested=true) and the OTHER
//     party is notified for consent.
//
// When a request is created, any pending counter-proposals are superseded.
// This is symmetric with CounterPropose, which clears pending cancellation
// requests: the latest intent always wins, so the two flows never coexist.
func (s *Service) CancelDispute(ctx context.Context, in CancelDisputeInput) (CancelDisputeResult, error) {
	d, err := s.disputes.GetByID(ctx, in.DisputeID)
	if err != nil {
		return CancelDisputeResult{}, err
	}

	cancelled, err := d.Cancel(in.UserID)
	if err != nil {
		return CancelDisputeResult{}, err
	}

	// If a cancellation REQUEST was created, supersede any pending
	// counter-proposals BEFORE persisting the dispute. Doing it in this
	// order means a failure here leaves the dispute untouched in DB
	// (the user can simply retry) instead of leaving an inconsistent
	// state where the request is set but old CPs are still pending.
	if !cancelled {
		if err := s.disputes.SupersedeAllPending(ctx, d.ID); err != nil {
			return CancelDisputeResult{}, fmt.Errorf("supersede pending: %w", err)
		}
	}

	if err := s.disputes.Update(ctx, d); err != nil {
		return CancelDisputeResult{}, fmt.Errorf("update dispute: %w", err)
	}

	if cancelled {
		// Direct cancellation: restore the proposal to active.
		p, err := s.proposals.GetByID(ctx, d.ProposalID)
		if err != nil {
			return CancelDisputeResult{}, fmt.Errorf("get proposal for restore: %w", err)
		}
		if err := p.RestoreFromDispute(proposaldomain.StatusActive); err != nil {
			slog.Warn("dispute: failed to restore proposal", "error", err)
		} else {
			_ = s.proposals.Update(ctx, p)
		}

		s.sendSystemMessage(ctx, d.ConversationID, in.UserID,
			message.MessageTypeDisputeCancelled, buildCancelledMetadata(d))
		s.sendNotification(ctx, d.RespondentID, "dispute_cancelled",
			"Litige annule", "Le litige a ete annule par l'initiateur.", d.ID)
		return CancelDisputeResult{Cancelled: true}, nil
	}

	// Cancellation request created — notify the OTHER participant (not the
	// requester) so they can accept or refuse.
	otherParty := d.RespondentID
	if in.UserID == d.RespondentID {
		otherParty = d.InitiatorID
	}
	s.sendSystemMessage(ctx, d.ConversationID, in.UserID,
		message.MessageTypeDisputeCancellationRequested,
		buildCancellationRequestedMetadata(d, in.UserID))
	s.sendNotification(ctx, otherParty, "dispute_cancellation_requested",
		"Demande d'annulation",
		"L'autre partie demande l'annulation du litige. Votre accord est requis.", d.ID)

	return CancelDisputeResult{Requested: true}, nil
}

// ---------------------------------------------------------------------------
// RespondToCancellation
// ---------------------------------------------------------------------------

// RespondToCancellation processes the other party's decision on a pending
// cancellation request. Either party may have requested the cancellation;
// only the OTHER (not the requester) can respond.
//   - Accept: the dispute is cancelled and the proposal is restored.
//   - Refuse: the cancellation request is cleared and the dispute continues.
func (s *Service) RespondToCancellation(ctx context.Context, in RespondToCancellationInput) error {
	d, err := s.disputes.GetByID(ctx, in.DisputeID)
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
		p, err := s.proposals.GetByID(ctx, d.ProposalID)
		if err != nil {
			return fmt.Errorf("get proposal for restore: %w", err)
		}
		if err := p.RestoreFromDispute(proposaldomain.StatusActive); err != nil {
			slog.Warn("dispute: failed to restore proposal", "error", err)
		} else {
			_ = s.proposals.Update(ctx, p)
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
	d, err := s.disputes.GetByID(ctx, disputeID)
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
	d, err := s.disputes.GetByID(ctx, in.DisputeID)
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
	d, err := s.disputes.GetByID(ctx, disputeID)
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
	d, err := s.disputes.GetByID(ctx, in.DisputeID)
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
	d, err := s.disputes.GetByID(ctx, disputeID)
	if err != nil {
		return nil, err
	}
	if !d.IsParticipant(userID) {
		return nil, disputedomain.ErrNotParticipant
	}
	return s.loadDetail(ctx, d)
}

func (s *Service) GetDisputeForAdmin(ctx context.Context, disputeID uuid.UUID) (*DisputeDetail, error) {
	d, err := s.disputes.GetByID(ctx, disputeID)
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
