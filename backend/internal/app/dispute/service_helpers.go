package dispute

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	disputedomain "marketplace-backend/internal/domain/dispute"
	"marketplace-backend/internal/domain/message"
	milestonedomain "marketplace-backend/internal/domain/milestone"
	proposaldomain "marketplace-backend/internal/domain/proposal"
	portservice "marketplace-backend/internal/port/service"
)

// ---------------------------------------------------------------------------
// System message helper
// ---------------------------------------------------------------------------

func (s *Service) sendSystemMessage(ctx context.Context, convID, senderID uuid.UUID, msgType message.MessageType, metadata json.RawMessage) {
	if err := s.messages.SendSystemMessage(ctx, portservice.SystemMessageInput{
		ConversationID: convID,
		SenderID:       senderID,
		Content:        "",
		Type:           string(msgType),
		Metadata:       metadata,
	}); err != nil {
		slog.Warn("dispute: send system message failed", "type", msgType, "error", err)
	}
}

// ---------------------------------------------------------------------------
// Notification helpers
// ---------------------------------------------------------------------------

func (s *Service) sendNotification(ctx context.Context, userID uuid.UUID, notifType, title, body string, disputeID uuid.UUID) {
	data, _ := json.Marshal(map[string]string{"dispute_id": disputeID.String()})
	if err := s.notifications.Send(ctx, portservice.NotificationInput{
		UserID: userID,
		Type:   notifType,
		Title:  title,
		Body:   body,
		Data:   data,
	}); err != nil {
		slog.Warn("dispute: send notification failed", "user_id", userID, "type", notifType, "error", err)
	}
}

func (s *Service) notifyBothParties(ctx context.Context, d *disputedomain.Dispute, notifType, title, body string) {
	s.sendNotification(ctx, d.InitiatorID, notifType, title, body, d.ID)
	s.sendNotification(ctx, d.RespondentID, notifType, title, body, d.ID)
}

// ---------------------------------------------------------------------------
// Fund distribution + proposal restore
// ---------------------------------------------------------------------------

// restoreProposalAndDistribute is the post-resolution cleanup path.
// Once a dispute is settled (amiable, admin or auto), we:
//
//  1. Restore the disputed milestone to a terminal state based on the
//     split — released if the provider keeps any of the escrow, refunded
//     if the client gets 100%.
//  2. Cancel every other pending_funding milestone: a dispute has ended
//     the working relationship on a concrete deliverable, so keeping
//     future milestones alive would create a zombie project. Either
//     party who wants to continue can create a fresh proposal.
//  3. Move the proposal to `completed` so the stepper reflects reality.
//  4. Emit `proposal_completed` + `evaluation_request` system messages
//     so both parties see a clean close and get the 14-day review CTA.
//  5. Distribute funds through the payment processor (unchanged).
//
// Any step failure is logged but does not short-circuit the rest —
// fund distribution MUST run even if a downstream message send fails,
// and vice versa.
func (s *Service) restoreProposalAndDistribute(ctx context.Context, d *disputedomain.Dispute) {
	p, err := s.loadProposalForActor(ctx, d.ProposalID)
	if err != nil {
		slog.Error("dispute: get proposal for resolution", "error", err)
		return
	}

	// 1 — restore the disputed milestone. Full-refund → refunded,
	//     anything else → released (the provider keeps at least part
	//     of the escrow).
	milestoneTarget := milestonedomain.StatusReleased
	if d.ResolutionAmountProvider != nil && *d.ResolutionAmountProvider == 0 {
		milestoneTarget = milestonedomain.StatusRefunded
	}
	if err := s.restoreMilestoneFromDispute(ctx, d.MilestoneID, milestoneTarget); err != nil {
		slog.Error("dispute: restore milestone from dispute",
			"milestone_id", d.MilestoneID, "target", milestoneTarget, "error", err)
	}

	// 2 — cancel every pending_funding milestone so the project can't
	//     linger. Ignores already-terminal milestones by design.
	s.cancelPendingFundingMilestones(ctx, p.ID)

	// 3 — mark the proposal complete.
	if err := p.RestoreFromDispute(proposaldomain.StatusCompleted); err != nil {
		slog.Error("dispute: restore proposal from dispute", "error", err)
	} else {
		if err := s.proposals.Update(ctx, p); err != nil {
			slog.Error("dispute: update proposal after resolution", "error", err)
		}
	}

	// 4 — close-out system messages. Sender = uuid.Nil is the system
	//     actor; SendSystemMessage handles the nil-org branch so the
	//     evaluation prompt reaches both parties.
	completedMeta := buildProposalCompletedMetadata(p)
	s.sendSystemMessage(ctx, p.ConversationID, uuid.Nil,
		message.MessageType("proposal_completed"), completedMeta)
	s.sendSystemMessage(ctx, p.ConversationID, uuid.Nil,
		message.MessageType("evaluation_request"), completedMeta)
	s.sendNotification(ctx, p.ClientID, "proposal_completed",
		"Mission terminée",
		"La mission est marquée comme terminée après résolution du litige. Laissez un avis avant la fin de la fenêtre de 14 jours.",
		d.ID)
	s.sendNotification(ctx, p.ProviderID, "proposal_completed",
		"Mission terminée",
		"La mission est marquée comme terminée après résolution du litige. Laissez un avis avant la fin de la fenêtre de 14 jours.",
		d.ID)

	// 5 — distribute escrow per the resolution split. Unchanged from
	//     pre-fix behaviour except it is now always reached even when
	//     earlier steps logged a non-fatal error.
	if s.payments == nil {
		return
	}

	if d.ResolutionAmountProvider != nil && *d.ResolutionAmountProvider > 0 {
		if err := s.payments.TransferPartialToProvider(ctx, d.ProposalID, *d.ResolutionAmountProvider); err != nil {
			slog.Error("dispute: transfer to provider failed",
				"proposal_id", d.ProposalID, "amount", *d.ResolutionAmountProvider, "error", err)
		}
	}

	if d.ResolutionAmountClient != nil && *d.ResolutionAmountClient > 0 {
		if err := s.payments.RefundToClient(ctx, d.ProposalID, *d.ResolutionAmountClient); err != nil {
			slog.Error("dispute: refund to client failed",
				"proposal_id", d.ProposalID, "amount", *d.ResolutionAmountClient, "error", err)
		}
	}
}

// cancelPendingFundingMilestones iterates the proposal's milestones and
// cancels every one still in pending_funding. Used by the dispute
// auto-complete path and by any future "stop the project early" flow.
// Non-terminal non-pending milestones (funded, submitted, disputed) are
// skipped — they already have a resolution path of their own.
func (s *Service) cancelPendingFundingMilestones(ctx context.Context, proposalID uuid.UUID) {
	milestones, err := s.milestones.ListByProposal(ctx, proposalID)
	if err != nil {
		slog.Error("dispute: list milestones for cancellation",
			"proposal_id", proposalID, "error", err)
		return
	}
	for _, m := range milestones {
		if m.Status != milestonedomain.StatusPendingFunding {
			continue
		}
		// Refetch with lock to apply the optimistic update cleanly.
		locked, err := s.milestones.GetByIDWithVersion(ctx, m.ID)
		if err != nil {
			slog.Error("dispute: lock milestone for cancel",
				"milestone_id", m.ID, "error", err)
			continue
		}
		if err := locked.Cancel(); err != nil {
			slog.Error("dispute: cancel milestone",
				"milestone_id", m.ID, "error", err)
			continue
		}
		if err := s.milestones.Update(ctx, locked); err != nil {
			slog.Error("dispute: update cancelled milestone",
				"milestone_id", m.ID, "error", err)
		}
	}
}

// buildProposalCompletedMetadata mirrors the shape of the proposal
// service's own "proposal_completed" metadata so the frontend message
// component can render it with the same fields (proposal_id,
// proposal_title, client/provider names when available, amount).
//
// We intentionally only include the identifiers the web
// ProposalSystemMessage component uses — no dispute-specific fields —
// so the message looks exactly like a normal end-of-project close.
func buildProposalCompletedMetadata(p *proposaldomain.Proposal) json.RawMessage {
	m := map[string]any{
		"proposal_id":          p.ID.String(),
		"proposal_title":       p.Title,
		"proposal_amount":      p.Amount,
		"proposal_status":      string(p.Status),
		"proposal_client_id":   p.ClientID.String(),
		"proposal_provider_id": p.ProviderID.String(),
		"proposal_version":     p.Version,
	}
	if p.ParentID != nil {
		m["proposal_parent_id"] = p.ParentID.String()
	} else {
		m["proposal_parent_id"] = nil
	}
	data, _ := json.Marshal(m)
	return data
}

// ---------------------------------------------------------------------------
// Metadata builders (on Dispute entity via extension methods)
// ---------------------------------------------------------------------------

// buildOpenedMetadata creates JSON metadata for the dispute_opened system message.
func buildOpenedMetadata(d *disputedomain.Dispute) json.RawMessage {
	return disputedomain.MustJSON(map[string]any{
		"dispute_id":       d.ID.String(),
		"reason":           string(d.Reason),
		"requested_amount": d.RequestedAmount,
		"proposal_amount":  d.ProposalAmount,
		"initiator_role":   d.InitiatorRole(),
	})
}

// buildOpenedWithProposalMetadata creates metadata for dispute_opened that includes
// the initial proposal (amount split + message to party). This avoids sending two
// separate system messages (opened + counter-proposal) for the initial dispute.
func buildOpenedWithProposalMetadata(d *disputedomain.Dispute, cp *disputedomain.CounterProposal, messageToParty string) json.RawMessage {
	m := map[string]any{
		"dispute_id":       d.ID.String(),
		"proposal_id":      d.ProposalID.String(),
		"reason":           string(d.Reason),
		"requested_amount": d.RequestedAmount,
		"proposal_amount":  d.ProposalAmount,
		"initiator_role":   d.InitiatorRole(),
		"message":          messageToParty,
	}
	if cp != nil {
		m["counter_proposal_id"] = cp.ID.String()
		m["amount_client"] = cp.AmountClient
		m["amount_provider"] = cp.AmountProvider
	}
	return disputedomain.MustJSON(m)
}

// buildCounterMetadata creates JSON metadata for counter-proposal messages.
func buildCounterMetadata(d *disputedomain.Dispute, cp *disputedomain.CounterProposal) json.RawMessage {
	return disputedomain.MustJSON(map[string]any{
		"dispute_id":          d.ID.String(),
		"proposal_id":         d.ProposalID.String(),
		"counter_proposal_id": cp.ID.String(),
		"proposer_id":         cp.ProposerID.String(),
		"amount_client":       cp.AmountClient,
		"amount_provider":     cp.AmountProvider,
		"proposal_amount":     d.ProposalAmount,
		"message":             cp.Message,
		"status":              string(cp.Status),
	})
}

// buildResolvedMetadata creates JSON metadata for the dispute_resolved message.
// Includes client_id/provider_id/resolved_at so the chat bubble can render a
// rich resolution card with "your share" highlight, matching the project page
// decision card — without having to refetch the dispute.
func buildResolvedMetadata(d *disputedomain.Dispute) json.RawMessage {
	m := map[string]any{
		"dispute_id":                 d.ID.String(),
		"proposal_id":                d.ProposalID.String(),
		"client_id":                  d.ClientID.String(),
		"provider_id":                d.ProviderID.String(),
		"resolution_type":            "",
		"resolution_amount_client":   int64(0),
		"resolution_amount_provider": int64(0),
	}
	if d.ResolutionType != nil {
		m["resolution_type"] = string(*d.ResolutionType)
	}
	if d.ResolutionAmountClient != nil {
		m["resolution_amount_client"] = *d.ResolutionAmountClient
	}
	if d.ResolutionAmountProvider != nil {
		m["resolution_amount_provider"] = *d.ResolutionAmountProvider
	}
	if d.ResolutionNote != nil {
		m["resolution_note"] = *d.ResolutionNote
	}
	if d.ResolvedAt != nil {
		m["resolved_at"] = d.ResolvedAt.Format(time.RFC3339)
	}
	return disputedomain.MustJSON(m)
}

// buildCancelledMetadata creates JSON metadata for the dispute_cancelled message.
func buildCancelledMetadata(d *disputedomain.Dispute) json.RawMessage {
	return disputedomain.MustJSON(map[string]any{
		"dispute_id": d.ID.String(),
	})
}

// buildCancellationRequestedMetadata creates metadata for the
// dispute_cancellation_requested system message — set when the initiator
// asks to cancel a dispute after the respondent has already replied.
func buildCancellationRequestedMetadata(d *disputedomain.Dispute, requestedBy uuid.UUID) json.RawMessage {
	return disputedomain.MustJSON(map[string]any{
		"dispute_id":   d.ID.String(),
		"proposal_id":  d.ProposalID.String(),
		"requested_by": requestedBy.String(),
	})
}

// buildCancellationRefusedMetadata creates metadata for the
// dispute_cancellation_refused system message — set when the respondent
// refuses the initiator's cancellation request.
func buildCancellationRefusedMetadata(d *disputedomain.Dispute, refusedBy uuid.UUID) json.RawMessage {
	return disputedomain.MustJSON(map[string]any{
		"dispute_id":  d.ID.String(),
		"proposal_id": d.ProposalID.String(),
		"refused_by":  refusedBy.String(),
	})
}

// buildEscalatedMetadata creates JSON metadata for the dispute_escalated message.
func buildEscalatedMetadata(d *disputedomain.Dispute) json.RawMessage {
	return disputedomain.MustJSON(map[string]any{
		"dispute_id": d.ID.String(),
		"reason":     string(d.Reason),
	})
}

// buildAutoResolvedMetadata creates metadata for auto-resolved disputes.
func buildAutoResolvedMetadata(d *disputedomain.Dispute) json.RawMessage {
	m := map[string]any{
		"dispute_id": d.ID.String(),
	}
	if d.ResolutionAmountClient != nil {
		m["resolution_amount_client"] = *d.ResolutionAmountClient
	}
	if d.ResolutionAmountProvider != nil {
		m["resolution_amount_provider"] = *d.ResolutionAmountProvider
	}
	return disputedomain.MustJSON(m)
}

// ---------------------------------------------------------------------------
// AI summary generation
// ---------------------------------------------------------------------------

// generateAISummary builds the analysis input from the dispute, its
// proposal, the post-mission conversation messages, the counter-proposals,
// and the dispute evidence files, then asks the configured AIAnalyzer to
// produce a structured mediation report.
//
// Shared between escalate() (the canonical escalation routine) and the
// scheduler so the result is the same whether escalation is triggered
// automatically or manually.
//
// Scope of what the AI sees:
//   - Conversation messages exchanged AFTER the mission started (i.e. after
//     proposal.PaidAt, when funds went into escrow). Messages from the
//     pre-mission negotiation phase are deliberately excluded by default —
//     they are about negotiating the proposal itself, not its execution,
//     and the admin can request them on demand via the AI chat (later).
//   - Counter-proposals exchanged inside the dispute.
//   - Dispute evidence file METADATA (filename, mime, size, uploader role)
//     — not the file content; that is reserved for a later iteration.
func (s *Service) generateAISummary(ctx context.Context, d *disputedomain.Dispute) (string, portservice.AIUsage, error) {
	input, err := s.buildAIInput(ctx, d)
	if err != nil {
		return "", portservice.AIUsage{}, err
	}

	// The summary call gets the dispute's full summary budget MINUS what
	// has already been consumed by previous summary calls (typically zero
	// since summary is generated once on escalation, but defensive).
	budget := d.AIBudgetSummary() - d.AISummaryUsed()
	if budget <= 0 {
		return "", portservice.AIUsage{}, disputedomain.ErrAIBudgetSummaryExceeded
	}

	return s.ai.AnalyzeDispute(ctx, input, budget)
}

// buildAIInput assembles the DisputeAnalysisInput shared by both summary
// and chat calls. Loads proposal, counter-proposals, post-mission messages,
// and evidence files; the caller decides what budget to enforce.
func (s *Service) buildAIInput(ctx context.Context, d *disputedomain.Dispute) (portservice.DisputeAnalysisInput, error) {
	p, err := s.loadProposalForActor(ctx, d.ProposalID)
	if err != nil {
		return portservice.DisputeAnalysisInput{}, fmt.Errorf("get proposal: %w", err)
	}

	cps, err := s.disputes.ListCounterProposals(ctx, d.ID)
	if err != nil {
		return portservice.DisputeAnalysisInput{}, fmt.Errorf("list counter-proposals: %w", err)
	}

	cpSummaries := make([]portservice.CounterProposalSummary, 0, len(cps))
	for _, cp := range cps {
		role := "provider"
		if cp.ProposerID == d.ClientID {
			role = "client"
		}
		cpSummaries = append(cpSummaries, portservice.CounterProposalSummary{
			ProposerRole:   role,
			AmountClient:   cp.AmountClient,
			AmountProvider: cp.AmountProvider,
			Message:        cp.Message,
			Status:         string(cp.Status),
		})
	}

	// Conversation messages posted after the mission actually started.
	// Fall back to the proposal's accepted/created date if PaidAt is nil
	// (e.g. legacy data or not-yet-paid proposals — should not happen in
	// practice for a disputable mission, but defensive).
	missionStart := time.Time{}
	switch {
	case p.PaidAt != nil:
		missionStart = *p.PaidAt
	case p.AcceptedAt != nil:
		missionStart = *p.AcceptedAt
	default:
		missionStart = p.CreatedAt
	}

	conversationMessages := s.loadPostMissionMessages(ctx, d, missionStart)
	evidenceSummaries := s.loadEvidenceSummaries(ctx, d)

	return portservice.DisputeAnalysisInput{
		DisputeReason:       string(d.Reason),
		DisputeDescription:  d.Description,
		ProposalTitle:       p.Title,
		ProposalDescription: p.Description,
		ProposalAmount:      d.ProposalAmount,
		RequestedAmount:     d.RequestedAmount,
		InitiatorRole:       d.InitiatorRole(),
		Messages:            conversationMessages,
		CounterProposals:    cpSummaries,
		Evidence:            evidenceSummaries,
	}, nil
}

// loadPostMissionMessages fetches conversation messages exchanged after
// the mission started, mapped to the AI input format. Failures are logged
// and degrade gracefully (empty slice) so the AI summary still runs with
// whatever context is available.
func (s *Service) loadPostMissionMessages(ctx context.Context, d *disputedomain.Dispute, since time.Time) []portservice.ConversationMessage {
	if s.messageRepo == nil {
		return nil
	}
	msgs, err := s.messageRepo.ListMessagesSinceTime(ctx, d.ConversationID, since, 200)
	if err != nil {
		slog.Warn("dispute: failed to load conversation messages for AI summary",
			"dispute_id", d.ID, "error", err)
		return nil
	}
	out := make([]portservice.ConversationMessage, 0, len(msgs))
	for _, m := range msgs {
		// Map sender to a role label the AI can reason about. Anything
		// outside the two dispute parties (system messages, admin) is
		// labelled "system" so it does not get confused with a party.
		role := "system"
		switch m.SenderID {
		case d.ClientID:
			role = "client"
		case d.ProviderID:
			role = "provider"
		}
		// Skip system messages with empty content (they only carry
		// metadata for the UI and would just add noise to the prompt).
		if string(m.Type) != "text" && string(m.Type) != "file" && m.Content == "" {
			continue
		}
		out = append(out, portservice.ConversationMessage{
			SenderName: role, // simple label, no DB lookup for display name
			SenderRole: role,
			Content:    m.Content,
			Type:       string(m.Type),
			CreatedAt:  m.CreatedAt.Format(time.RFC3339),
		})
	}
	return out
}

// loadEvidenceSummaries fetches dispute evidence metadata for the AI input.
// Failures degrade gracefully like loadPostMissionMessages.
func (s *Service) loadEvidenceSummaries(ctx context.Context, d *disputedomain.Dispute) []portservice.EvidenceSummary {
	evidence, err := s.disputes.ListEvidence(ctx, d.ID)
	if err != nil {
		slog.Warn("dispute: failed to load evidence for AI summary",
			"dispute_id", d.ID, "error", err)
		return nil
	}
	out := make([]portservice.EvidenceSummary, 0, len(evidence))
	for _, e := range evidence {
		role := "system"
		switch e.UploaderID {
		case d.ClientID:
			role = "client"
		case d.ProviderID:
			role = "provider"
		}
		out = append(out, portservice.EvidenceSummary{
			Filename:     e.Filename,
			MimeType:     e.MimeType,
			Size:         e.Size,
			UploaderRole: role,
		})
	}
	return out
}
