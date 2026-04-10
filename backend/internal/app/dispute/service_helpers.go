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

func (s *Service) restoreProposalAndDistribute(ctx context.Context, d *disputedomain.Dispute) {
	p, err := s.proposals.GetByID(ctx, d.ProposalID)
	if err != nil {
		slog.Error("dispute: get proposal for resolution", "error", err)
		return
	}

	// Determine target status: if provider gets 100%, mission is completed.
	// Otherwise, it was partially refunded — consider it completed too.
	target := proposaldomain.StatusCompleted
	if err := p.RestoreFromDispute(target); err != nil {
		slog.Error("dispute: restore proposal from dispute", "error", err)
		return
	}
	if err := s.proposals.Update(ctx, p); err != nil {
		slog.Error("dispute: update proposal after resolution", "error", err)
		return
	}

	if s.payments == nil {
		return
	}

	// Transfer provider's portion (partial or full)
	if d.ResolutionAmountProvider != nil && *d.ResolutionAmountProvider > 0 {
		if err := s.payments.TransferPartialToProvider(ctx, d.ProposalID, *d.ResolutionAmountProvider); err != nil {
			slog.Error("dispute: transfer to provider failed",
				"proposal_id", d.ProposalID, "amount", *d.ResolutionAmountProvider, "error", err)
		}
	}

	// Refund client's portion (partial or full)
	if d.ResolutionAmountClient != nil && *d.ResolutionAmountClient > 0 {
		if err := s.payments.RefundToClient(ctx, d.ProposalID, *d.ResolutionAmountClient); err != nil {
			slog.Error("dispute: refund to client failed",
				"proposal_id", d.ProposalID, "amount", *d.ResolutionAmountClient, "error", err)
		}
	}
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
func buildResolvedMetadata(d *disputedomain.Dispute) json.RawMessage {
	m := map[string]any{
		"dispute_id":                 d.ID.String(),
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
	p, err := s.proposals.GetByID(ctx, d.ProposalID)
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
