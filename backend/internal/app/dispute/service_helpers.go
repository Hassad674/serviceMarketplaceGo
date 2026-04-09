package dispute

import (
	"context"
	"encoding/json"
	"log/slog"

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
